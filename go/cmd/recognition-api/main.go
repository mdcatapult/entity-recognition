package main

import (
	"context"
	"fmt"
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
	Recognizers map[string]struct {
		Host     string
		GrpcPort int `mapstructure:"grpc_port"`
	}
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
	clients := make([]pb.RecognizerClient, len(config.Recognizers))
	connections := make([]*grpc.ClientConn, len(config.Recognizers))
	i := 0
	for name, r := range config.Recognizers {
		log.Info().Str("recognizer", name).Msg("connecting...")
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		conn, err := grpc.DialContext(ctx, fmt.Sprintf("%s:%d", r.Host, r.GrpcPort), opts...)
		if err != nil {
			log.Fatal().Err(err).Send()
		}
		cancel()
		connections[i] = conn
		clients[i] = pb.NewRecognizerClient(conn)
		i++
	}

	r := gin.New()
	r.Use(gin.LoggerWithFormatter(lib.JsonLogFormatter))
	c := controller{clients: clients}
	s := server{controller: c}
	s.RegisterRoutes(r)
	if err := r.Run(fmt.Sprintf(":%d", config.Server.HttpPort)); err != nil {
		for _, conn := range connections {
			if err := conn.Close(); err != nil {
				log.Fatal().Err(err).Send()
			}
		}
		log.Fatal().Err(err).Send()
	}
}
