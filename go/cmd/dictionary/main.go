package main

import (
	"fmt"
	"gitlab.mdcatapult.io/informatics/software-engineering/entity-recognition/go/lib/cache"
	"gitlab.mdcatapult.io/informatics/software-engineering/entity-recognition/go/lib/cache/remote"
	"net"

	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
	"gitlab.mdcatapult.io/informatics/software-engineering/entity-recognition/go/gen/pb"
	"gitlab.mdcatapult.io/informatics/software-engineering/entity-recognition/go/lib"
	"google.golang.org/grpc"
)

// config structure
type dictionaryRecogniserConfig struct {
	lib.BaseConfig
	Server struct {
		GrpcPort int `mapstructure:"grpc_port"`
	}
	BackendDatabase cache.Type `mapstructure:"dictionary_backend"`
	PipelineSize    int        `mapstructure:"pipeline_size"`
	Redis               remote.RedisConfig
	Elasticsearch       remote.ElasticsearchConfig
	CompoundTokenLength int `mapstructure:"compound_token_length"`
}

var config dictionaryRecogniserConfig

func initConfig() {
	// initialise config with defaults.
	err := lib.InitializeConfig("./config/dictionary.yml", map[string]interface{}{
		"log_level":          "info",
		"dictionary_backend": cache.Redis,
		"pipeline_size":      10000,
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
			"index": "pubchem",
		},
		"compound_token_length": 5,
	})
	if err != nil {
		log.Fatal().Err(err).Send()
	}

	// unmarshal the viper contents into our config struct
	err = viper.Unmarshal(&config)
	if err != nil {
		log.Fatal().Err(err).Send()
	}
}

func main() {
	initConfig()
	// Get a redis client
	var dbClient remote.Client
	var err error
	switch config.BackendDatabase {
	case cache.Redis:
		dbClient = remote.NewRedisClient(config.Redis)
	case cache.Elasticsearch:
		dbClient, err = remote.NewElasticsearchClient(config.Elasticsearch)
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
	})

	log.Info().Int("port", config.Server.GrpcPort).Msg("ready to accept requests")
	err = grpcServer.Serve(lis)
	if err != nil {
		panic(err)
	}
}
