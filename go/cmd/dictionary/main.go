package main

import (
	"fmt"
	"net"
	"os"

	"gitlab.mdcatapult.io/informatics/software-engineering/entity-recognition/go/lib/cache"
	"gitlab.mdcatapult.io/informatics/software-engineering/entity-recognition/go/lib/cache/local"
	"gitlab.mdcatapult.io/informatics/software-engineering/entity-recognition/go/lib/cache/remote"
	"gitlab.mdcatapult.io/informatics/software-engineering/entity-recognition/go/lib/dict"

	"github.com/rs/zerolog/log"
	"gitlab.mdcatapult.io/informatics/software-engineering/entity-recognition/go/gen/pb"
	"gitlab.mdcatapult.io/informatics/software-engineering/entity-recognition/go/lib"
	"google.golang.org/grpc"
)

// config structure
type dictionaryRecogniserConfig struct {
	lib.BaseConfig
	Dictionary dict.DictConfig
	Server     struct {
		GrpcPort int `mapstructure:"grpc_port"`
	}
	CacheType           cache.Type `mapstructure:"cache_type"`
	PipelineSize        int        `mapstructure:"pipeline_size"`
	Redis               remote.RedisConfig
	Elasticsearch       remote.ElasticsearchConfig
	CompoundTokenLength int `mapstructure:"compound_token_length"`
}

var config dictionaryRecogniserConfig
var defaultConfig = map[string]interface{}{
	"log_level":          "info",
	"dictionary_backend": cache.Redis,
	"pipeline_size":      10000,
	"dictionary": map[string]interface{}{
		"type": "pubchem",
		"name": "pubchem_data",
	},
	"server": map[string]interface{}{
		"grpc_port": 50051,
	},
	"redis": map[string]interface{}{
		"host": "localhost",
		"port": 6379,
	},
	"elasticsearch": map[string]interface{}{
		"host":  "localhost",
		"port":  9200,
		"index": "pubchem",
	},
	"compound_token_length": 5,
}

func main() {
	if err := lib.InitializeConfig("./config/dictionary.yml", defaultConfig, &config); err != nil {
		log.Fatal().Err(err).Send()
	}

	// Get a redis client
	var remoteCache remote.RemoteCacheClient
	var localCache local.LocalCacheClient
	var err error
	switch config.CacheType {
	case cache.Redis:
		remoteCache = remote.NewRedisClient(config.Redis)
	case cache.Elasticsearch:
		remoteCache, err = remote.NewElasticsearchClient(config.Elasticsearch)
		if err != nil {
			log.Fatal().Err(err).Send()
		}
	case cache.Local:
		localCache = local.New()
		dictFile, err := os.Open(config.Dictionary.Path)
		if err != nil {
			log.Fatal().Err(err).Send()
		}

		nSynonyms := 0
		nLookups := 0
		callback := func(entry dict.Entry) error {
			nLookups++
			lookup := &cache.Lookup{
				Dictionary:  config.Dictionary.Name,
				Identifiers: entry.Identifiers,
			}

			for _, synonym := range entry.Synonyms {
				localCache.Set(synonym, lookup)
				nSynonyms++
			}

			if nLookups%100000 == 0 {
				log.Info().Int("identifiers", nLookups).Int("synonyms", nSynonyms).Msg("importing data")
			}

			return nil
		}

		if err := dict.ReadWithCallback(dictFile, config.Dictionary.Format, callback, nil); err != nil {
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
	if remoteCache != nil {
		pb.RegisterRecognizerServer(grpcServer, &recogniser{
			remoteCache: remoteCache,
		})
	} else if localCache != nil {
		pb.RegisterRecognizerServer(grpcServer, &localRecogniser{
			localCache: localCache,
		})
	} else {
		log.Fatal().Msg("no cache configured")
	}

	log.Info().Int("port", config.Server.GrpcPort).Msg("ready to accept requests")
	err = grpcServer.Serve(lis)
	if err != nil {
		panic(err)
	}
}
