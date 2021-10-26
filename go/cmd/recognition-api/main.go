package main

import (
	"context"
	"fmt"
	"gitlab.mdcatapult.io/informatics/software-engineering/entity-recognition/go/lib/recogniser"
	grpc_recogniser "gitlab.mdcatapult.io/informatics/software-engineering/entity-recognition/go/lib/recogniser/grpc-recogniser"
	"gitlab.mdcatapult.io/informatics/software-engineering/entity-recognition/go/lib/recogniser/http-recogniser"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
	"gitlab.mdcatapult.io/informatics/software-engineering/entity-recognition/go/gen/pb"
	"gitlab.mdcatapult.io/informatics/software-engineering/entity-recognition/go/lib"
	"google.golang.org/grpc"
)

// config structure
type recognitionAPIConfig struct {
	LogLevel string `mapstructure:"log_level"`
	Server   struct {
		HttpPort int `mapstructure:"http_port"`
	}
	GrpcRecognizers map[string]struct {
		Host string
		Port int
	} `mapstructure:"grpc_recognisers"`
	HttpRecognisers map[string]struct {
		Type http_recogniser.Type
		Url  string
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
	opts = append(opts, grpc.WithInsecure())
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
		recogniserClients[name] = grpc_recogniser.New(pb.NewRecognizerClient(conn))
	}

	for name, conf := range config.HttpRecognisers {
		switch conf.Type {
		case http_recogniser.LeadmineType:
			recogniserClients[name] = http_recogniser.NewLeadmineClient(conf.Url)
		}
	}

	r := gin.New()
	r.Use(gin.LoggerWithFormatter(lib.JsonLogFormatter))
	c := controller{
		recognisers: recogniserClients,
	}
	s := server{controller: c}
	s.RegisterRoutes(r)
	if err := r.Run(fmt.Sprintf(":%d", config.Server.HttpPort)); err != nil {
		log.Fatal().Err(err).Send()
	}
}
