package main

import (
	"fmt"
	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
	"gitlab.mdcatapult.io/informatics/software-engineering/entity-recognition/go/gen/pb"
	"gitlab.mdcatapult.io/informatics/software-engineering/entity-recognition/go/lib"
	"gitlab.mdcatapult.io/informatics/software-engineering/entity-recognition/go/lib/db"
	"google.golang.org/grpc"
	"net"
)

// config structure
type dictionaryRecogniserConfig struct {
	lib.BaseConfig
	Server struct{
		GrpcPort int `mapstructure:"grpc_port"`
	}
	BackendDatabase     db.DictionaryBackend `mapstructure:"dictionary_backend"`
	PipelineSize        int               `mapstructure:"pipeline_size"`
	Redis               db.RedisConfig
	Elasticsearch       db.ElasticsearchConfig
	CompoundTokenLength int `mapstructure:"compound_token_length"`
}

var config dictionaryRecogniserConfig

func init() {
	// initialise config with defaults.
	err := lib.InitializeConfig("./config/dictionary.yml", map[string]interface{}{
		"log_level": "info",
		"dictionary_backend": db.RedisDictionaryBackend,
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
		log.Fatal().Err(err).Send()
	}

	// unmarshal the viper contents into our config struct
	err = viper.Unmarshal(&config)
	if err != nil {
		log.Fatal().Err(err).Send()
	}
}

func main() {

	// Get a redis client
	var dbClient db.Client
	var err error
	switch config.BackendDatabase {
	case db.RedisDictionaryBackend:
		dbClient = db.NewRedisClient(config.Redis)
	case db.ElasticsearchDictionaryBackend:
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
	})

	log.Info().Int("port", config.Server.GrpcPort).Msg("ready to accept requests")
	err = grpcServer.Serve(lis)
	if err != nil {
		panic(err)
	}
}
