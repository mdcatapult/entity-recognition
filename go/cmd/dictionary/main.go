package main

import (
	"fmt"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
	"gitlab.mdcatapult.io/informatics/software-engineering/entity-recognition/go/gen/pb"
	"gitlab.mdcatapult.io/informatics/software-engineering/entity-recognition/go/lib"
	"gitlab.mdcatapult.io/informatics/software-engineering/entity-recognition/go/lib/db"
	"google.golang.org/grpc"
	"io"
	"net"
	"strings"
	"sync"
)

// config structure
type conf struct {
	LogLevel string `mapstructure:"log_level"`
	Server struct{
		GrpcPort int `mapstructure:"grpc_port"`
	}
	BackendDatabase BackendDatabaseType `mapstructure:"backend_database"`
	PipelineSize   int `mapstructure:"pipeline_size"`
	Redis db.RedisConfig
	Elasticsearch db.ElasticsearchConfig
	CompoundTokenLength int `mapstructure:"compound_token_length"`
}

var config conf

type BackendDatabaseType string

const (
	Redis BackendDatabaseType = "redis"
	Elasticsearch BackendDatabaseType = "elasticsearch"
)

func init() {
	// initialise config with defaults.
	err := lib.InitializeConfig(map[string]interface{}{
		"log_level": "info",
		"backend_database": Redis,
		"pipeline_size": 10000,
		"server": map[string]interface{}{
			"grpc_port": 50051,
		},
		"redis": map[string]interface{}{
			"host": "localhost",
			"port": 6379,
		},
		"elasticsearch": map[string]interface{}{
			"host": "localhost",
			"port": 9200,
		},
		"compound_token_length": 5,
	})
	if err != nil {
		panic(err)
	}

	// unmarshal the viper contents into our config struct
	err = viper.Unmarshal(&config)
	if err != nil {
		panic(err)
	}
}

func main() {

	// Get a redis client
	var dbClient db.Client
	var err error
	switch config.BackendDatabase {
	case Redis:
		dbClient = db.NewRedisClient(config.Redis)
	case Elasticsearch:
		dbClient, err = db.NewElasticsearchClient(config.Elasticsearch)
		if err != nil {
			log.Fatal().Err(err).Send()
		}
	default:
		log.Fatal().Msg("invalid backend database type")
	}

	// start the grpc server
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", config.Server.GrpcPort))
	if err != nil {
		log.Fatal().Err(err).Send()
	}
	var opts []grpc.ServerOption
	grpcServer := grpc.NewServer(opts...)
	pb.RegisterRecognizerServer(grpcServer, &recogniser{
		dbClient: dbClient,
		requestCache: make(map[uuid.UUID]*requestVars),
	})

	log.Info().Int("port", config.Server.GrpcPort).Msg("ready to accept requests")
	err = grpcServer.Serve(lis)
	if err != nil {
		panic(err)
	}
}

type recogniser struct {
	pb.UnimplementedRecognizerServer
	dbClient db.Client
	requestCache map[uuid.UUID]*requestVars
	rwmut sync.RWMutex
}

type requestVars struct {
	tokenCache map[*pb.Snippet]*db.Lookup
	tokenCacheMisses []*pb.Snippet
	tokenHistory []*pb.Snippet
	keyHistory []string
	sentenceEnd bool
	stream pb.Recognizer_RecognizeServer
	pipe db.GetPipeline
}

func (r *recogniser) newResultHandler(requestID uuid.UUID) func(snippet *pb.Snippet, lookup *db.Lookup) error {
	return func(snippet *pb.Snippet, lookup *db.Lookup) error {
		r.rwmut.RLock()
		requestVars, ok := r.requestCache[requestID]
		r.rwmut.RUnlock()
		if !ok {
			log.Fatal().Msg("request not in cache, something went horribly wrong")
		}
		requestVars.tokenCache[snippet] = lookup
		if lookup == nil {
			return nil
		}
		entity := &pb.RecognizedEntity{
			Entity:     string(snippet.GetData()),
			Position:   snippet.GetOffset(),
			Type:       lookup.Dictionary,
			ResolvedTo: lookup.ResolvedEntities,
		}
		err := requestVars.stream.Send(entity)
		if err != nil {
			return err
		}

		return nil
	}
}

func (r *recogniser) getCompoundTokens(requestID uuid.UUID, token *pb.Snippet) []*pb.Snippet {
	r.rwmut.RLock()
	requestVars, ok := r.requestCache[requestID]
	r.rwmut.RUnlock()
	if !ok {
		log.Fatal().Msg("request not in cache, something went horribly wrong")
	}
	// If sentenceEnd is true, we can save some redis queries by resetting the token history..
	if requestVars.sentenceEnd {
		requestVars.tokenHistory = []*pb.Snippet{}
		requestVars.keyHistory = []string{}
		requestVars.sentenceEnd = false
	}

	// normalise the token (remove enclosing punctuation and enforce NFKC encoding).
	// sentenceEnd is true if the last byte in the token is one of '.', '?', or '!'.
	requestVars.sentenceEnd = lib.Normalize(token)

	// manage the token history
	if len(requestVars.tokenHistory) < config.CompoundTokenLength {
		requestVars.tokenHistory = append(requestVars.tokenHistory, token)
		requestVars.keyHistory = append(requestVars.keyHistory, string(token.GetData()))
	} else {
		requestVars.tokenHistory = append(requestVars.tokenHistory[1:], token)
		requestVars.keyHistory = append(requestVars.keyHistory[1:], string(token.GetData()))
	}

	// construct the compound tokens to query against redis.
	queryTokens := make([]*pb.Snippet, len(requestVars.tokenHistory))
	for i, historicalToken := range requestVars.tokenHistory {
		queryTokens[i] = &pb.Snippet{
			Data:   []byte(strings.Join(requestVars.keyHistory[i:], " ")),
			Offset: historicalToken.GetOffset(),
		}
	}
	return queryTokens
}

func (r *recogniser) queryCompoundTokens(requestID uuid.UUID, compoundTokens []*pb.Snippet) error {
	r.rwmut.RLock()
	requestVars, ok := r.requestCache[requestID]
	r.rwmut.RUnlock()
	if !ok {
		log.Fatal().Msg("request not in cache, something went horribly wrong")
	}

	for _, compoundToken := range compoundTokens {
		if lookup, ok := requestVars.tokenCache[compoundToken]; ok {
			// if it's nil, we've already queried redis and it wasn't there
			if lookup == nil {
				continue
			}
			// If it's empty, it's already queued but we don't know if its there or not.
			// Append it to the cacheMisses to be found later.
			if lookup.Dictionary == "" {
				requestVars.tokenCacheMisses = append(requestVars.tokenCacheMisses, compoundToken)
				continue
			}
			// Otherwise, construct an entity from the cache value and send it back to the caller.
			entity := &pb.RecognizedEntity{
				Entity:     string(compoundToken.GetData()),
				Position:   compoundToken.GetOffset(),
				Type:       lookup.Dictionary,
				ResolvedTo: lookup.ResolvedEntities,
			}
			if err := requestVars.stream.Send(entity); err != nil {
				return err
			}
		} else {
			// Not in local cache.
			// Queue the redis "GET" in the pipe and set the cache value to an empty db.Lookup
			// (so that future equivalent tokens will be a cache miss).
			requestVars.pipe.Get(compoundToken)
			requestVars.tokenCache[compoundToken] = &db.Lookup{}
		}
	}
	return nil
}

func (r *recogniser) initializeRequest(stream pb.Recognizer_RecognizeServer) uuid.UUID {
	requestID := uuid.New()
	r.rwmut.Lock()
	r.requestCache[requestID] = &requestVars{
		tokenCache:       make(map[*pb.Snippet]*db.Lookup, config.PipelineSize),
		tokenCacheMisses: make([]*pb.Snippet, config.PipelineSize),
		tokenHistory:     []*pb.Snippet{},
		keyHistory:       []string{},
		sentenceEnd:      false,
		stream:           stream,
		pipe: r.dbClient.NewGetPipeline(config.PipelineSize),
	}
	r.rwmut.Unlock()
	return requestID
}

func (r *recogniser) execPipe(requestID uuid.UUID, onResult func(snippet *pb.Snippet, lookup *db.Lookup) error, threshold int, new bool) error {
	r.rwmut.RLock()
	requestVars, ok := r.requestCache[requestID]
	r.rwmut.RUnlock()
	if !ok {
		log.Fatal().Msg("request not in cache, something went horribly wrong")
	}

	if requestVars.pipe.Size() > threshold {
		if err := requestVars.pipe.ExecGet(onResult); err != nil {
			return err
		}
		if new {
			requestVars.pipe = r.dbClient.NewGetPipeline(config.PipelineSize)
		}
	}
	return nil
}

func (r *recogniser) retryCacheMisses(requestID uuid.UUID) error {
	r.rwmut.RLock()
	requestVars, ok := r.requestCache[requestID]
	r.rwmut.RUnlock()
	if !ok {
		log.Fatal().Msg("request not in cache, something went horribly wrong")
	}

	// Check if any of the cacheMisses were populated (nil means redis doesnt have it).
	for _, token := range requestVars.tokenCacheMisses {
		if lookup := requestVars.tokenCache[token]; lookup != nil {
			entity := &pb.RecognizedEntity{
				Entity:     string(token.GetData()),
				Position:   token.GetOffset(),
				Type:       lookup.Dictionary,
				ResolvedTo: lookup.ResolvedEntities,
			}
			if err := requestVars.stream.Send(entity); err != nil {
				return err
			}
		}
	}
	return nil
}

func (r *recogniser) Recognize(stream pb.Recognizer_RecognizeServer) error {
	requestID := r.initializeRequest(stream)
	onResult := r.newResultHandler(requestID)

	for {
		token, err := stream.Recv()
		if err == io.EOF {
			// There are likely some redis queries queued on the pipe. If there are, execute them. Then break.
			if err := r.execPipe(requestID, onResult, 0, false); err != nil {
				return err
			}
			break
		} else if err != nil {
			return err
		}

		compoundTokens := r.getCompoundTokens(requestID, token)

		if err := r.queryCompoundTokens(requestID, compoundTokens); err != nil {
			return err
		}

		if err := r.execPipe(requestID, onResult, config.PipelineSize, true); err != nil {
			return err
		}
	}

	return nil
}
