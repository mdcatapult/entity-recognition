package main

import (
	"fmt"
	"net"

	"github.com/rs/zerolog/log"
	"gitlab.mdcatapult.io/informatics/software-engineering/entity-recognition/go/gen/pb"
	"gitlab.mdcatapult.io/informatics/software-engineering/entity-recognition/go/lib"
	"google.golang.org/grpc"
)

// config structure
type regexpRecognizerConfig struct {
	lib.BaseConfig
	Server struct {
		GrpcPort int `mapstructure:"grpc_port"`
	}
	RegexFile string `mapstructure:"regex_file"`
}

// global vars initialised on startup (should never be edited after that).
var config regexpRecognizerConfig
var defaultConfig = map[string]interface{}{
	"log_level": "info",
	"server": map[string]interface{}{
		"grpc_port": 50051,
	},
	"regex_file": "./config/regex_file.yml",
}

func main() {
	if err := lib.InitializeConfig("./config/regexer.yml", defaultConfig, &config); err != nil {
		log.Fatal().Err(err).Send()
	}

	regexps, err := getRegexps()
	if err != nil {
		log.Fatal().Str("path", config.RegexFile).Err(err).Send()
	}

	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", config.Server.GrpcPort))
	if err != nil {
		log.Fatal().Err(err).Send()
	}
	var opts []grpc.ServerOption
	grpcServer := grpc.NewServer(opts...)
	pb.RegisterRecognizerServer(grpcServer, recogniser{regexps: regexps})
	log.Info().Int("port", config.Server.GrpcPort).Msg("ready to accept requests")
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatal().Err(err).Send()
	}
}
