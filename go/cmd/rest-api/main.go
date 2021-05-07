package main

import (
	"context"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/spf13/viper"
	"gitlab.mdcatapult.io/informatics/software-engineering/entity-recognition/gen/pb"
	"gitlab.mdcatapult.io/informatics/software-engineering/entity-recognition/lib"
	"google.golang.org/grpc"
	"io"
	"sync"
)

type conf struct {
	LogLevel string `mapstructure:"log_level"`
	Server struct{
		HttpPort int `mapstructure:"http_port"`
	}
	Recognizers map[string]struct{
		Host     string
		GrpcPort int `mapstructure:"grpc_port"`
	}
}

var config conf

func init() {
	err := lib.InitializeConfig(map[string]interface{}{
		"log_level": "info",
		"server": map[string]interface{}{
			"http_port": 8080,
		},
	})
	if err != nil {
		panic(err)
	}

	err = viper.Unmarshal(&config)
	if err != nil {
		panic(err)
	}
}

func main() {

	var opts []grpc.DialOption
	opts = append(opts, grpc.WithInsecure())
	opts = append(opts, grpc.WithBlock())

	clients := make([]pb.RecognizerClient, len(config.Recognizers))
	connections := make([]*grpc.ClientConn, len(config.Recognizers))
	i := 0
	for _, r := range config.Recognizers {
		conn, err := grpc.Dial(fmt.Sprintf("%s:%d", r.Host, r.GrpcPort), opts...)
		if err != nil {
			panic(err)
		}
		connections[i] = conn
		clients[i] = pb.NewRecognizerClient(conn)
		i++
	}

	startHttpServer(clients...)
	for _, conn := range connections {
		if err := conn.Close(); err != nil {
			panic(err)
		}
	}
}

func startHttpServer(clients ...pb.RecognizerClient) {
	r := gin.Default()
	r.POST("/html/tokens", func(c *gin.Context) {
		type token struct{
			Token string `json:"token"`
			Offset uint32 `json:"offset"`
		}
		var tokens []token
		onSnippet := func(snippet *pb.Snippet) error {
			return lib.Tokenize(snippet, func(snippet *pb.Snippet) error {
				tokens = append(tokens, token{
					Token:  string(snippet.GetData()),
					Offset: snippet.GetOffset(),
				})
				return nil
			})
		}

		if err := lib.HtmlToText(c.Request.Body, onSnippet); err != nil {
			_ = c.AbortWithError(500, err)
			return
		}

		c.JSON(200, tokens)
	})

	r.POST("/html/entities", func(c *gin.Context) {

		var err error
		errChan := make(chan error, len(clients))
		recognisers := make([]pb.Recognizer_RecognizeClient, len(clients))
		for i, client := range clients {
			recognisers[i], err = client.Recognize(context.Background())
		}


		var entities []*pb.RecognizedEntity
		var mut sync.Mutex
		for _, recogniser := range recognisers {
			go func(recogniser pb.Recognizer_RecognizeClient) {
				for {
					entity, err := recogniser.Recv()
					if err == io.EOF {
						errChan <- nil
						return
					} else if err != nil {
						errChan <- err
						return
					}
					mut.Lock()
					entities = append(entities, entity)
					mut.Unlock()
				}
			}(recogniser)
		}

		onSnippet := func(snippet *pb.Snippet) error {
			return lib.Tokenize(snippet, func(snippet *pb.Snippet) error {
				for _, recogniser := range recognisers {
					if err := recogniser.Send(snippet); err != nil {
						return err
					}
				}
				return nil
			})
		}

		if err := lib.HtmlToText(c.Request.Body, onSnippet); err != nil {
			_ = c.AbortWithError(500, err)
			return
		}

		for _, recogniser := range recognisers {
			if err := recogniser.CloseSend(); err != nil {
				_ = c.AbortWithError(500, err)
				return
			}
		}

		for i := 0; i < len(recognisers); i++ {
			if err = <-errChan; err != nil {
				_ = c.AbortWithError(500, err)
				return
			}
		}

		c.JSON(200, entities)
	})

	r.POST("/html/text", func(c *gin.Context) {
		var data []byte
		onSnippet := func(snippet *pb.Snippet) error {
			data = append(data, snippet.GetData()...)
			return nil
		}
		if err := lib.HtmlToText(c.Request.Body, onSnippet); err != nil {
			_ = c.AbortWithError(500, err)
			return
		}

		c.Data(200, "text/plain", data)
	})
	_ = r.Run(":8083")
}
