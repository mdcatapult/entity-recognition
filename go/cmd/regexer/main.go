package main

import (
	"fmt"
	"net"

	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
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

func init() {
	// Initialize config with default values
	err := lib.InitializeConfig("./config/regexer.yml", map[string]interface{}{
		"log_level": "info",
		"server": map[string]interface{}{
			"grpc_port": 50051,
		},
		"regex_file": "./config/regex_file.yml",
	})
	if err != nil {
		panic(err)
	}

	// unmarshal viper contents into our struct
	err = viper.Unmarshal(&config)
	if err != nil {
		log.Fatal().Err(err).Send()
	}
}

func main() {

	regexps, err := getRegexps()
	if err != nil {
		log.Fatal().Err(err).Send()
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
