/*
 * Copyright 2022 Medicines Discovery Catapult
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *     http://www.apache.org/licenses/LICENSE-2.0
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

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
