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

var CompoundTokenLength = 10
var PipelineSize = 10000

type Lookup struct {
	Dictionary string `json:"dictionary"`
	ResolvedEntity string `json:"resolvedEntity,omitempty"`
}

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

	err = viper.Unmarshal(&config)
	if err != nil {
		panic(err)
	}
}

func main() {

	redisClient := redis.NewClient(&redis.Options{
		Addr:               fmt.Sprintf("%s:%d", config.Redis.Host, config.Redis.Port),
	})

	err := uploadDictionaries(redisClient)
	if err != nil {
		panic(err)
	}

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
	cache := make(map[*pb.Snippet]*Lookup, PipelineSize)
	results := make(map[*pb.Snippet]*redis.StringCmd, PipelineSize)
	cacheMisses := make([]*pb.Snippet, PipelineSize)
	pipe := r.redisClient.Pipeline()
	var tokenHistory []*pb.Snippet
	var keyHistory []string

	for {
		token, err := stream.Recv()
		if err == io.EOF {
			if len(results) > 0 {
				err := execPipe(pipe, results, cache, stream)
				if err != nil {
					return err
				}
			}
			break
		} else if err != nil {
			return err
		}

		if sentenceEnd := lib.Normalize(token); sentenceEnd {
			tokenHistory = []*pb.Snippet{}
			keyHistory = []string{}
		}
		if len(tokenHistory) < CompoundTokenLength {
			tokenHistory = append(tokenHistory, token)
			keyHistory = append(keyHistory, string(token.GetData()))
		} else {
			tokenHistory = append(tokenHistory[1:], token)
			keyHistory = append(keyHistory[1:], string(token.GetData()))
		}

		queryTokens := make([]*pb.Snippet, len(tokenHistory))
		for i, historicalToken := range tokenHistory {
			queryTokens[i] = &pb.Snippet{
				Data:   []byte(strings.Join(keyHistory[i:], " ")),
				Offset: historicalToken.GetOffset(),
			}
		}

		for _, compoundToken := range queryTokens {
			if lookup, ok := cache[compoundToken]; ok {
				if lookup == nil {
					continue
				}
				if lookup.Dictionary == "" {
					cacheMisses = append(cacheMisses, compoundToken)
					continue
				}
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
				results[compoundToken] = pipe.Get(string(compoundToken.GetData()))
				cache[compoundToken] = &Lookup{}
			}
		}

		if len(results) > PipelineSize {
			err := execPipe(pipe, results, cache, stream)
			if err != nil {
				return err
			}
			results = make(map[*pb.Snippet]*redis.StringCmd, PipelineSize)
		}
	}

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

func execPipe(pipe redis.Pipeliner, results map[*pb.Snippet]*redis.StringCmd, cache map[*pb.Snippet]*Lookup, stream pb.Recognizer_RecognizeServer) error {
	_, err := pipe.Exec()
	if err != nil && err != redis.Nil {
		return err
	}
	for key, result := range results {
		b, err := result.Bytes()
		if err == redis.Nil {
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
