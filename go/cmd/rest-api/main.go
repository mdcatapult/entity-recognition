package main

import (
	"context"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
	"gitlab.mdcatapult.io/informatics/software-engineering/entity-recognition/go/gen/pb"
	"gitlab.mdcatapult.io/informatics/software-engineering/entity-recognition/go/lib"
	"google.golang.org/grpc"
	"io"
	"sync"
)

// config structure
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
	// Set default config values
	err := lib.InitializeConfig(map[string]interface{}{
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

	// general grpc options
	var opts []grpc.DialOption
	opts = append(opts, grpc.WithInsecure())
	opts = append(opts, grpc.WithBlock())

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

	startHttpServer(clients...)
	for _, conn := range connections {
		if err := conn.Close(); err != nil {
			log.Fatal().Err(err).Send()
		}
	}
}

func startHttpServer(clients ...pb.RecognizerClient) {
	r := gin.Default()

	// Get the tokenized text from html
	r.POST("/html/tokens", func(c *gin.Context) {
		type token struct{
			Token string `json:"token"`
			Offset uint32 `json:"offset"`
		}

		// This is a callback which is executed when the lib.HtmlToText function reaches some kind
		// of delimiter, e.g. </br>. Here we tokenize the output and append that to our token slice.
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

		// Call htmlToText with our callback
		if err := lib.HtmlToText(c.Request.Body, onSnippet); err != nil {
			_ = c.AbortWithError(500, err)
			return
		}

		c.JSON(200, tokens)
	})

	// Get the entities from html
	r.POST("/html/entities", func(c *gin.Context) {

		// Instantiate streaming clients for all of our recognisers
		var err error
		recognisers := make([]pb.Recognizer_RecognizeClient, len(clients))
		for i, client := range clients {
			recognisers[i], err = client.Recognize(context.Background())
		}

		// For each recogniser, instantiate a goroutine which listens for entities
		// and appends them to a slice. The mutex is necessary for thread safety when
		// manipulating the slice. If there is an error, send it to the error channel.
		// When complete, send nil to the error channel.
		errChan := make(chan error, len(clients))
		entities := make([]*pb.RecognizedEntity, 0, 1000)
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

		// Callback to the html to text function. Tokenize each block of text
		// and send every token to every recogniser.
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

		// HtmlToText blocks until it is complete, so at this point the callback
		// will not be called again, and it is safe to call CloseSend on the recognisers
		// to close the stream (this doesn't stop us receiving entities).
		for _, recogniser := range recognisers {
			if err := recogniser.CloseSend(); err != nil {
				_ = c.AbortWithError(500, err)
				return
			}
		}

		// This for loop will block execution until all of the go routines
		// we spawned above have sent either nil or an error on the error
		// channel. If there is no error on any channel, we can continue.
		for i := 0; i < len(recognisers); i++ {
			if err = <-errChan; err != nil {
				_ = c.AbortWithError(500, err)
				return
			}
		}

		c.JSON(200, entities)
	})

	// Converts html to text.
	r.POST("/html/text", func(c *gin.Context) {
		// callback to be executed when a html delimiter is reached.
		// Just append to the byte slice.
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
	_ = r.Run(fmt.Sprintf(":%d", config.Server.HttpPort))
}
