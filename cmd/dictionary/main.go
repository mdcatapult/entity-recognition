package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"github.com/go-redis/redis"
	"gitlab.mdcatapult.io/informatics/software-engineering/entity-recognition/gen/pb"
	"google.golang.org/grpc"
	"io"
	"io/ioutil"
	"net"
	"os"
	"path"
	"strings"
)

var CompoundTokenLength = 5

type recogniser struct {
	pb.UnimplementedRecognizerServer
	redisClient *redis.Client
}

func (r recogniser) Recognize(stream pb.Recognizer_RecognizeServer) error {
	cache := make(map[*pb.Snippet]*Lookup, 1000)
	results := make(map[*pb.Snippet]*redis.StringCmd, 1000)
	cacheMisses := make([]*pb.Snippet, 1000)
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

		if len(results) > 1000 {
			err := execPipe(pipe, results, cache, stream)
			if err != nil {
				return err
			}
			results = make(map[*pb.Snippet]*redis.StringCmd, 1000)
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

func main() {

	redisClient := redis.NewClient(&redis.Options{
		Addr:               "localhost:6379",
	})
	dictionaryDirPath := "cmd/dictionary/dictionaries"
	files, err := ioutil.ReadDir(dictionaryDirPath)
	if err != nil {
		panic(err)
	}

	for _, f := range files {
		values, err := parseDict(path.Join(dictionaryDirPath, f.Name()))
		if err != nil {
			panic(err)
		}
		pipe := redisClient.Pipeline()
		for key, lookup := range values {
			blob, err := json.Marshal(lookup)
			if err != nil {
				panic(err)
			}
			pipe.Set(key, blob, 0)
		}
		_, err = pipe.Exec()
		if err != nil {
			panic(err)
		}
	}

	fmt.Println("Serving...")
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", 50052))
	if err != nil {
		panic(err)
	}
	var opts []grpc.ServerOption
	grpcServer := grpc.NewServer(opts...)
	pb.RegisterRecognizerServer(grpcServer, recogniser{
		redisClient: redisClient,
	})
	err = grpcServer.Serve(lis)
	if err != nil {
		panic(err)
	}
}

type Lookup struct {
	Dictionary string `json:"dictionary"`
	ResolvedEntity string `json:"resolvedEntity,omitempty"`
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
