package main

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
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

func initConfig() {
	// Set default config values
	err := lib.InitializeConfig("./config/recognition-api.yml", map[string]interface{}{
		"log_level": "info",
		"server": map[string]interface{}{
			"http_port": 8080,
		},
	})
	if err != nil {
		panic(err)
	}

	// Unmarshal the viper config into our struct.
	err = viper.Unmarshal(&config)
	if err != nil {
		log.Fatal().Err(err).Send()
	}
}

func main() {
	initConfig()
	// general grpc options
	var opts []grpc.DialOption
	opts = append(opts, grpc.WithInsecure())
	//opts = append(opts, grpc.WithBlock())

	// for each recogniser in the config, instantiate a client and save the connection
	// so that we can close it later.
	clients := make([]pb.RecognizerClient, len(config.Recognizers))
	connections := make([]*grpc.ClientConn, len(config.Recognizers))
	i := 0
	for _, r := range config.Recognizers {
		conn, err := grpc.Dial(fmt.Sprintf("%s:%d", r.Host, r.GrpcPort), opts...)
		if err != nil {
			log.Fatal().Err(err).Send()
		}
		connections[i] = conn
		clients[i] = pb.NewRecognizerClient(conn)
		i++
	}

	r := gin.Default()
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
