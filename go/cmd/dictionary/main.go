package main

import (
	"fmt"
	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
	"gitlab.mdcatapult.io/informatics/software-engineering/entity-recognition/go/gen/pb"
	"gitlab.mdcatapult.io/informatics/software-engineering/entity-recognition/go/lib"
	"gitlab.mdcatapult.io/informatics/software-engineering/entity-recognition/go/lib/db"
	"google.golang.org/grpc"
	"io"
	"net"
	"strings"
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
	pb.RegisterRecognizerServer(grpcServer, recogniser{
		dbClient: dbClient,
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
}

func (r recogniser) Recognize(stream pb.Recognizer_RecognizeServer) error {
	// in memory cache per query. We might be able to be able to combine this
	// with a global in memory cache with a TTL for more speed.
	cache := make(map[*pb.Snippet]*db.Lookup, config.PipelineSize)

	// populate this when a redis query is queued in the pipeline but the pipeline hasn't
	// been executed yet. We will get these values from the cache after the client stops
	// streaming.
	cacheMisses := make([]*pb.Snippet, config.PipelineSize)

	// token and key histories are used in combination with the compoundTokenLength
	// to create compound tokens (multiword dictionary keys).
	var tokenHistory []*pb.Snippet
	var keyHistory []string
	var sentenceEnd bool

	pipe := r.dbClient.NewGetPipeline(config.PipelineSize)
	onResult := func(snippet *pb.Snippet, lookup *db.Lookup) error {
		cache[snippet] = lookup
		if lookup == nil {
			return nil
		}
		entity := &pb.RecognizedEntity{
			Entity:     string(snippet.GetData()),
			Position:   snippet.GetOffset(),
			Type:       lookup.Dictionary,
			ResolvedTo: lookup.ResolvedEntities,
		}
		if err := stream.Send(entity); err != nil {
			return err
		}

		return nil
	}

	for {
		token, err := stream.Recv()
		if err == io.EOF {
			// There are likely some redis queries queued on the pipe. If there are, execute them. Then break.
			if pipe.Size() > 0 {
				err := pipe.ExecGet(onResult)
				if err != nil {
					return err
				}
			}
			break
		} else if err != nil {
			return err
		}

		// If sentenceEnd is true, we can save some redis queries by resetting the token history..
		if sentenceEnd {
			tokenHistory = []*pb.Snippet{}
			keyHistory = []string{}
			sentenceEnd = false
		}

		// normalise the token (remove enclosing punctuation and enforce NFKC encoding).
		// sentenceEnd is true if the last byte in the token is one of '.', '?', or '!'.
		sentenceEnd = lib.Normalize(token)

		// manage the token history
		if len(tokenHistory) < config.CompoundTokenLength {
			tokenHistory = append(tokenHistory, token)
			keyHistory = append(keyHistory, string(token.GetData()))
		} else {
			tokenHistory = append(tokenHistory[1:], token)
			keyHistory = append(keyHistory[1:], string(token.GetData()))
		}

		// construct the compound tokens to query against redis.
		queryTokens := make([]*pb.Snippet, len(tokenHistory))
		for i, historicalToken := range tokenHistory {
			queryTokens[i] = &pb.Snippet{
				Data:   []byte(strings.Join(keyHistory[i:], " ")),
				Offset: historicalToken.GetOffset(),
			}
		}

		for _, compoundToken := range queryTokens {
			if lookup, ok := cache[compoundToken]; ok {
				// if it's nil, we've already queried redis and it wasn't there
				if lookup == nil {
					continue
				}
				// If it's empty, it's already queued but we don't know if its there or not.
				// Append it to the cacheMisses to be found later.
				if lookup.Dictionary == "" {
					cacheMisses = append(cacheMisses, compoundToken)
					continue
				}
				// Otherwise, construct an entity from the cache value and send it back to the caller.
				entity := &pb.RecognizedEntity{
					Entity:     string(compoundToken.GetData()),
					Position:   compoundToken.GetOffset(),
					Type:       lookup.Dictionary,
					ResolvedTo: lookup.ResolvedEntities,
				}
				if err := stream.Send(entity); err != nil {
					return err
				}
			} else {
				// Not in local cache.
				// Queue the redis "GET" in the pipe and set the cache value to an empty db.Lookup
				// (so that future equivalent tokens will be a cache miss).
				pipe.Get(compoundToken)
				cache[compoundToken] = &db.Lookup{}
			}
		}

		// If we have enough redis queries in the pipeline, execute it and
		// reset the values of results/cacheMisses.
		if pipe.Size() > config.PipelineSize {
			err := pipe.ExecGet(onResult)
			if err != nil {
				return err
			}
			pipe = r.dbClient.NewGetPipeline(config.PipelineSize)
		}
	}

	// Check if any of the cacheMisses were populated (nil means redis doesnt have it).
	for _, token := range cacheMisses {
		if lookup := cache[token]; lookup != nil {
			entity := &pb.RecognizedEntity{
				Entity:     string(token.GetData()),
				Position:   token.GetOffset(),
				Type:       lookup.Dictionary,
				ResolvedTo: lookup.ResolvedEntities,
			}
			if err := stream.Send(entity); err != nil {
				return err
			}
		}
	}

	return nil
}
