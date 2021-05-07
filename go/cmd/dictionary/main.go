package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"github.com/go-redis/redis"
	"github.com/spf13/viper"
	"gitlab.mdcatapult.io/informatics/software-engineering/entity-recognition/gen/pb"
	"gitlab.mdcatapult.io/informatics/software-engineering/entity-recognition/lib"
	"google.golang.org/grpc"
	"io"
	"io/ioutil"
	"net"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strings"
)

// Dictionaries sometimes match against multiple words.
// This specifies how many tokens we should concatenate in our lookup.
// The higher this value, the longer this query will take to resolve.
// We should try to lower this as much as possible. (maybe we can set it on a
// per dictionary basis to be as low as the longest key in that dictionary?)
var compoundTokenLength = 5

// This number of operations to pipeline to redis (to save on round trip time).
var pipelineSize = 1000

// Lookup is the value we will store in redis.
type Lookup struct {
	Dictionary string `json:"dictionary"`
	ResolvedEntity string `json:"resolvedEntity,omitempty"`
}

// config structure
type conf struct {
	LogLevel string `mapstructure:"log_level"`
	Server struct{
		GrpcPort int `mapstructure:"grpc_port"`
	}
	Redis struct {
		Host string
		Port int
	}
}

var config conf

func init() {
	// initialise config with defaults.
	err := lib.InitializeConfig(map[string]interface{}{
		"log_level": "info",
		"server": map[string]interface{}{
			"grpc_port": 50052,
		},
		"redis": map[string]interface{}{
			"host": "localhost",
			"port": 6379,
		},
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
	redisClient := redis.NewClient(&redis.Options{
		Addr:               fmt.Sprintf("%s:%d", config.Redis.Host, config.Redis.Port),
	})

	// read all the dictionaries in the dicitonary folder, parse them, and upload the results to redis.
	err := uploadDictionaries(redisClient)
	if err != nil {
		panic(err)
	}

	// start the grpc server
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", config.Server.GrpcPort))
	if err != nil {
		panic(err)
	}
	var opts []grpc.ServerOption
	grpcServer := grpc.NewServer(opts...)
	pb.RegisterRecognizerServer(grpcServer, recogniser{
		redisClient: redisClient,
	})

	fmt.Println("Serving...")
	err = grpcServer.Serve(lis)
	if err != nil {
		panic(err)
	}
}

type recogniser struct {
	pb.UnimplementedRecognizerServer
	redisClient *redis.Client
}

func (r recogniser) Recognize(stream pb.Recognizer_RecognizeServer) error {
	// in memory cache per query. We might be able to be able to combine this
	// with a global in memory cache with a TTL for more speed.
	cache := make(map[*pb.Snippet]*Lookup, pipelineSize)

	// this map allows is to reference the results of a redis pipeline execution.
	results := make(map[*pb.Snippet]*redis.StringCmd, pipelineSize)

	// populate this when a redis query is queued in the pipeline but the pipeline hasn't
	// been executed yet. We will get these values from the cache after the client stops
	// streaming.
	cacheMisses := make([]*pb.Snippet, pipelineSize)

	// token and key histories are used in combination with the compoundTokenLength
	// to create compound tokens (multiword dictionary keys).
	var tokenHistory []*pb.Snippet
	var keyHistory []string
	var sentenceEnd bool

	pipe := r.redisClient.Pipeline()
	for {
		token, err := stream.Recv()
		if err == io.EOF {
			// There are likely some redis queries queued on the pipe. If there are, execute them. Then break.
			if len(results) > 0 {
				err := execPipe(pipe, results, cache, cacheMisses, stream)
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
		if len(tokenHistory) < compoundTokenLength {
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
					ResolvedTo: lookup.ResolvedEntity,
				}
				if err := stream.Send(entity); err != nil {
					return err
				}
			} else {
				// Not in local cache.
				// Queue the redis "GET" in the pipe and set the cache value to an empty lookup
				// (so that future equivalent tokens will be a cache miss).
				results[compoundToken] = pipe.Get(string(compoundToken.GetData()))
				cache[compoundToken] = &Lookup{}
			}
		}

		// If we have enough redis queries in the pipeline, execute it and
		// reset the values of results/cacheMisses.
		if len(results) > pipelineSize {
			err := execPipe(pipe, results, cache, cacheMisses, stream)
			if err != nil {
				return err
			}
			results = make(map[*pb.Snippet]*redis.StringCmd, pipelineSize)
			pipe = r.redisClient.Pipeline()
		}
	}

	// Check if any of the cacheMisses were populated (nil means redis doesnt have it).
	for _, token := range cacheMisses {
		if lookup := cache[token]; lookup != nil {
			entity := &pb.RecognizedEntity{
				Entity:     string(token.GetData()),
				Position:   token.GetOffset(),
				Type:       lookup.Dictionary,
				ResolvedTo: lookup.ResolvedEntity,
			}
			if err := stream.Send(entity); err != nil {
				return err
			}
		}
	}

	return nil
}

func execPipe(pipe redis.Pipeliner, results map[*pb.Snippet]*redis.StringCmd, cache map[*pb.Snippet]*Lookup, cacheMisses []*pb.Snippet, stream pb.Recognizer_RecognizeServer) error {
	_, err := pipe.Exec()
	if err != nil && err != redis.Nil {
		return err
	}
	for key, result := range results {
		b, err := result.Bytes()
		if err == redis.Nil {
			// if nil, not in redis. Put nil in local cache.
			cache[key] = nil
			continue
		} else if err != nil {
			return err
		}
		var lookup Lookup
		err = json.Unmarshal(b, &lookup)
		if err != nil {
			return err
		}

		// At this point we have found an entity in redis. Put it in the cache
		// and send it back to the client.
		cache[key] = &lookup
		entity := &pb.RecognizedEntity{
			Entity:     string(key.GetData()),
			Position:   key.GetOffset(),
			Type:       lookup.Dictionary,
			ResolvedTo: lookup.ResolvedEntity,
		}
		if err := stream.Send(entity); err != nil {
			return err
		}
	}

	return nil
}

func uploadDictionaries(redisClient *redis.Client) error {
	_, thisFile, _, _ := runtime.Caller(0)
	thisDirectory := path.Dir(thisFile)
	dictionaryDir := filepath.Join(thisDirectory, "dictionaries")
	files, err := ioutil.ReadDir(dictionaryDir)
	if err != nil {
		return err
	}

	for _, f := range files {
		values, err := parseDict(path.Join(dictionaryDir, f.Name()))
		if err != nil {
			return err
		}
		pipe := redisClient.Pipeline()
		for key, lookup := range values {
			blob, err := json.Marshal(lookup)
			if err != nil {
				return err
			}
			pipe.Set(key, blob, 0)
		}
		_, err = pipe.Exec()
		if err != nil {
			return err
		}
	}
	return nil
}

func parseDict(fileName string) (map[string]Lookup, error) {
	tsv, err := os.Open(fileName)
	if err != nil {
		return nil, err
	}

	dictionaryName := strings.TrimSuffix(path.Base(fileName), path.Ext(fileName))

	scn := bufio.NewScanner(tsv)
	dictionary := make(map[string]Lookup)
	for scn.Scan() {
		line := scn.Text()
		uncommented := strings.Split(line, "#")
		if len(uncommented[0]) > 0 {
			record := strings.Split(uncommented[0], "\t")
			resolvedEntity := strings.TrimSpace(record[len(record)-1])
			if resolvedEntity == "" {
				continue
			}
			if len(record) == 1 {
				dictionary[strings.TrimSpace(record[0])] = Lookup{
					Dictionary:     dictionaryName,
				}
				continue
			}
			for _, key := range record[:len(record)-1] {
				if key == "" {
					continue
				}
				dictionary[strings.TrimSpace(key)] = Lookup{
					Dictionary:     dictionaryName,
					ResolvedEntity: resolvedEntity,
				}
			}
		}
	}
	return dictionary, nil
}
