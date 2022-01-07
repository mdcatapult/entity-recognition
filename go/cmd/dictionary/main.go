package main

import (
	"fmt"
	"github.com/rs/zerolog/log"
	"gitlab.mdcatapult.io/informatics/software-engineering/entity-recognition/go/gen/pb"
	"gitlab.mdcatapult.io/informatics/software-engineering/entity-recognition/go/lib"
	"gitlab.mdcatapult.io/informatics/software-engineering/entity-recognition/go/lib/cache/remote"
	"gitlab.mdcatapult.io/informatics/software-engineering/entity-recognition/go/lib/dict"
	"google.golang.org/grpc"
	"net"
)

// config structure
type dictionaryRecogniserConfig struct {
	lib.BaseConfig
	Dictionary dict.DictConfig
	Server     struct {
		GrpcPort int `mapstructure:"grpc_port"`
	}
	PipelineSize        int `mapstructure:"pipeline_size"`
	Redis               remote.RedisConfig
	CompoundTokenLength int `mapstructure:"compound_token_length"`
}

var config dictionaryRecogniserConfig
var defaultConfig = map[string]interface{}{
	"log_level":     "info",
	"pipeline_size": 10000,
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
	"compound_token_length": 5,
}

func main() {
	if err := lib.InitializeConfig("./config/dictionary.yml", defaultConfig, &config); err != nil {
		log.Fatal().Err(err).Send()
	}

	// Get a redis client
	var redisClient = remote.NewRedisClient(config.Redis)
	var err error

	// start the grpc server
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", config.Server.GrpcPort))
	if err != nil {
		log.Fatal().Err(err).Send()
	}
	var opts []grpc.ServerOption
	grpcServer := grpc.NewServer(opts...)
	if redisClient != nil {
		pb.RegisterRecognizerServer(grpcServer, &recogniser{
			remoteCache: redisClient,
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
