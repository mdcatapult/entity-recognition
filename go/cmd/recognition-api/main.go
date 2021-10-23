package main

import (
	"context"
	"fmt"
	http_recogniser "gitlab.mdcatapult.io/informatics/software-engineering/entity-recognition/go/lib/http-recogniser"
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
	HttpRecognisers map[string]struct{
		Type string
		Host string
		Port int
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
	clients := make(map[string]pb.RecognizerClient, len(config.GrpcRecognizers))
	connections := make(map[string]*grpc.ClientConn, len(config.GrpcRecognizers))
	for name, conf := range config.GrpcRecognizers {
		log.Info().Str("recognizer", name).Msg("connecting...")
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		conn, err := grpc.DialContext(ctx, fmt.Sprintf("%s:%d", conf.Host, conf.Port), opts...)
		if err != nil {
			log.Fatal().Err(err).Send()
		}
		cancel()
		connections[name] = conn
		clients[name] = pb.NewRecognizerClient(conn)
	}

	httpClients := make(map[string]http_recogniser.Client)
	for name, conf := range config.HttpRecognisers {
		switch conf.Type {
		case "dummy":
			httpClients[name] = http_recogniser.DummyClient{}
		}
	}


	r := gin.New()
	r.Use(gin.LoggerWithFormatter(lib.JsonLogFormatter))
	c := controller{
		grpcRecogniserClients: clients,
		httpRecogniserClients: httpClients,
	}
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
