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
	"context"
	"fmt"
	"github.com/gin-contrib/cors"
	"gitlab.mdcatapult.io/informatics/software-engineering/entity-recognition/go/cmd/recognition-api/grpc-recogniser"
	"gitlab.mdcatapult.io/informatics/software-engineering/entity-recognition/go/cmd/recognition-api/http-recogniser"
	"google.golang.org/grpc/credentials/insecure"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
	"gitlab.mdcatapult.io/informatics/software-engineering/entity-recognition/go/gen/pb"
	"gitlab.mdcatapult.io/informatics/software-engineering/entity-recognition/go/lib"
	"gitlab.mdcatapult.io/informatics/software-engineering/entity-recognition/go/lib/blacklist"
	"gitlab.mdcatapult.io/informatics/software-engineering/entity-recognition/go/lib/recogniser"
	"gitlab.mdcatapult.io/informatics/software-engineering/entity-recognition/go/lib/snippet-reader/html"
	"gitlab.mdcatapult.io/informatics/software-engineering/entity-recognition/go/lib/snippet-reader/text"
	"google.golang.org/grpc"
)

// config structure
type recognitionAPIConfig struct {
	LogLevel string `mapstructure:"log_level"`
	Server   struct {
		HttpPort int `mapstructure:"http_port"`
	}
	Blacklist       string `mapstructure:"blacklist"` // global blacklist
	GrpcRecognizers map[string]struct {
		Host      string
		Port      int
		Blacklist string
	} `mapstructure:"grpc_recognisers"`
	HttpRecognisers map[string]struct {
		Type      http_recogniser.Type
		Url       string
		Blacklist string
	} `mapstructure:"http_recognisers"`
}

var config recognitionAPIConfig
var defaultConfig = map[string]interface{}{
	"log_level": "info",
	"server": map[string]interface{}{
		"http_port": 8080,
	},
}

func main() {
	if err := lib.InitializeConfig("./config/recognition-api.yml", defaultConfig, &config); err != nil {
		log.Fatal().Err(err).Send()
	}

	// general grpc options
	var opts []grpc.DialOption
	opts = append(opts, grpc.WithTransportCredentials(insecure.NewCredentials()))
	opts = append(opts, grpc.WithBlock())

	// for each recogniser in the config, instantiate a client and save the connection
	// so that we can close it later.
	recogniserClients := make(map[string]recogniser.Client)
	for name, conf := range config.GrpcRecognizers {
		log.Info().Str("recognizer", name).Msg("connecting...")
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		conn, err := grpc.DialContext(ctx, fmt.Sprintf("%s:%d", conf.Host, conf.Port), opts...)
		if err != nil {
			log.Fatal().Err(err).Send()
		}
		cancel()

		recogniserClients[name] = grpc_recogniser.New(name, pb.NewRecognizerClient(conn), loadBlacklist(conf.Blacklist))
	}

	for name, conf := range config.HttpRecognisers {
		switch conf.Type {
		case http_recogniser.LeadmineType:
			recogniserClients[name] = http_recogniser.NewLeadmineClient(name, conf.Url, loadBlacklist(conf.Blacklist))
		}
	}

	r := gin.Default()
	r.Use(
		gin.LoggerWithFormatter(lib.JsonLogFormatter),
		cors.New(cors.Config{
			AllowOrigins:     []string{"*"},
			AllowMethods:     []string{"GET", "POST"},
			AllowHeaders:     []string{"Origin", "Content-Type", "x-leadmine-chemical-entities"},
			AllowCredentials: true,
			MaxAge:           12 * time.Hour,
		}),
	)

	c := controller{
		recognisers: recogniserClients,
		htmlReader:  html.SnippetReader{},
		textReader:  text.SnippetReader{},
		blacklist:   loadBlacklist(config.Blacklist),
	}

	s := server{controller: &c}
	s.RegisterRoutes(r)
	if err := r.Run(fmt.Sprintf(":%d", config.Server.HttpPort)); err != nil {
		log.Fatal().Err(err).Send()
	}
}

func loadBlacklist(path string) blacklist.Blacklist {
	var bl = blacklist.Blacklist{}
	if path != "" {
		loadedBlacklist, err := blacklist.Load(path)
		if err != nil {
			log.Fatal().Err(err).Send()
		}
		bl = *loadedBlacklist
	}
	return bl
}
