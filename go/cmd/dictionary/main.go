package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
	"gitlab.mdcatapult.io/informatics/software-engineering/entity-recognition/go/cmd/dictionary/db"
	"gitlab.mdcatapult.io/informatics/software-engineering/entity-recognition/go/gen/pb"
	"gitlab.mdcatapult.io/informatics/software-engineering/entity-recognition/go/lib"
	"google.golang.org/grpc"
	"io"
	"net"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
)

// Dictionaries sometimes match against multiple words.
// This specifies how many tokens we should concatenate in our lookup.
// The higher this value, the longer this query will take to resolve.
// We should try to lower this as much as possible. (maybe we can set it on a
// per dictionary basis to be as low as the longest key in that dictionary?)
var compoundTokenLength = 5

// This number of operations to pipeline to redis (to save on round trip time).
var pipelineSize = 10000

// config structure
type conf struct {
	LogLevel string `mapstructure:"log_level"`
	DictionaryPath string `mapstructure:"dictionary_path"`
	Server struct{
		GrpcPort int `mapstructure:"grpc_port"`
	}
	Redis db.RedisConfig
}

var config conf

func init() {
	// initialise config with defaults.
	err := lib.InitializeConfig(map[string]interface{}{
		"log_level": "info",
		"dictionary_path": "./dictionaries/henry.tsv",
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
	dbClient := db.NewRedisClient(config.Redis)

	// read all the dictionaries in the dicitonary folder, parse them, and upload the results to redis.
	err := uploadDictionary(config.DictionaryPath, dbClient)
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
		dbClient: dbClient,
	})

	fmt.Println("Serving...")
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
	cache := make(map[*pb.Snippet]*db.Lookup, pipelineSize)

	// populate this when a redis query is queued in the pipeline but the pipeline hasn't
	// been executed yet. We will get these values from the cache after the client stops
	// streaming.
	cacheMisses := make([]*pb.Snippet, pipelineSize)

	// token and key histories are used in combination with the compoundTokenLength
	// to create compound tokens (multiword dictionary keys).
	var tokenHistory []*pb.Snippet
	var keyHistory []string
	var sentenceEnd bool

	pipe := r.dbClient.NewPipeline(pipelineSize)
	onResult := func(snippet *pb.Snippet, lookup *db.Lookup) error {
		cache[snippet] = lookup
		if lookup == nil {
			return nil
		}
		entity := &pb.RecognizedEntity{
			Entity:     string(snippet.GetData()),
			Position:   snippet.GetOffset(),
			Type:       lookup.Dictionary,
			ResolvedTo: lookup.ResolvedEntities[0],
		}
		return stream.Send(entity)
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
					ResolvedTo: lookup.ResolvedEntities[0],
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
		if pipe.Size() > pipelineSize {
			err := pipe.ExecGet(onResult)
			if err != nil {
				return err
			}
			pipe = r.dbClient.NewPipeline(pipelineSize)
		}
	}

	// Check if any of the cacheMisses were populated (nil means redis doesnt have it).
	for _, token := range cacheMisses {
		if lookup := cache[token]; lookup != nil {
			entity := &pb.RecognizedEntity{
				Entity:     string(token.GetData()),
				Position:   token.GetOffset(),
				Type:       lookup.Dictionary,
				ResolvedTo: lookup.ResolvedEntities[0],
			}
			if err := stream.Send(entity); err != nil {
				return err
			}
		}
	}

	return nil
}

func uploadDictionary(dictPath string, dbClient db.Client) error {
	absPath := dictPath
	if !filepath.IsAbs(dictPath) {
		_, thisFile, _, _ := runtime.Caller(0)
		thisDirectory := path.Dir(thisFile)
		absPath = filepath.Join(thisDirectory, dictPath)
	}

	tsv, err := os.Open(absPath)
	if err != nil {
		return err
	}

	dictionaryName := "unichem"

	pipe := dbClient.NewPipeline(pipelineSize)

	scn := bufio.NewScanner(tsv)
	currentId := -1
	row := 0
	var synonyms []string
	var identifiers []string
	for scn.Scan() {
		row++
		if row > 10000000 { break }
		if row % 100000 == 0 {
			log.Info().Int("row", row).Msg("Scanning dictionary...")
		}
		line := scn.Text()
		entries := strings.Split(line, "\t")
		if len(entries) != 2 {
			log.Warn().Int("row", row).Strs("entries", entries).Msg("invalid row in dictionary tsv")
			continue
		}

		pubchemId, err := strconv.Atoi(entries[0])
		if err != nil {
			log.Warn().Int("row", row).Strs("entries", entries).Msg("invalid pubchem id")
			continue
		}

		var synonym string
		var identifier string
		if isIdentifier(entries[1]) {
			identifier = entries[1]
		} else {
			synonym = entries[1]
		}

		if pubchemId != currentId {
			if currentId != -1 {
				// Mid process, some stuff to do
				for _, s := range synonyms {
					b, err := json.Marshal(db.Lookup{
						Dictionary:       dictionaryName,
						ResolvedEntities: identifiers,
					})
					if err != nil {
						return err
					}
					pipe.Set(s, b)
				}

				if pipe.Size() > pipelineSize {
					if err := pipe.ExecSet(); err != nil {
						return err
					}
					pipe = dbClient.NewPipeline(pipelineSize)
				}

				synonyms = []string{}
				identifiers = []string{}
			}


			// Set new current id
			currentId = pubchemId
			if synonym != "" {
				synonyms = append(synonyms, synonym)
			} else {
				identifiers = append(identifiers, fmt.Sprintf("PUBCHEM:%d", pubchemId))
				identifiers = append(identifiers, identifier)
			}
		} else {
			if synonym != "" {
				synonyms = append(synonyms, synonym)
			} else {
				identifiers = append(identifiers, identifier)
			}
		}
	}

	if pipe.Size() > 0 {
		if err := pipe.ExecSet(); err != nil {
			return err
		}
		pipe = dbClient.NewPipeline(pipelineSize)
	}

	return nil
}

func isIdentifier(thing string) bool {
	for _, re := range chemicalIdentifiers {
		if re.MatchString(thing) {
			return true
		}
	}
	return false
}


var chemicalIdentifiers = []*regexp.Regexp{
	regexp.MustCompile(`^SCHEMBL\d+$`),
	regexp.MustCompile(`^DTXSID\d{8}$`),
	regexp.MustCompile(`^CHEMBL\d+$`),
	regexp.MustCompile(`^CHEBI:\d+$`),
	regexp.MustCompile(`^LMFA\d{8}$`),
	regexp.MustCompile(`^HY-\d+?[A-Z]?$`),
	regexp.MustCompile(`^CS-.*$`),
	regexp.MustCompile(`^FT-\d{7}$`),
	regexp.MustCompile(`^Q\d+$`),
	regexp.MustCompile(`^ACMC-\w+$`),
	regexp.MustCompile(`^ALBB-\d{6}$`),
	regexp.MustCompile(`^AKOS\d{9}$`),
	regexp.MustCompile(`^\d+-\d+-\d+$`),
	regexp.MustCompile(`^EINCES\s\d+-\d+-\d+$`),
	regexp.MustCompile(`^EC\s\d+-\d+-\d+$`),
}