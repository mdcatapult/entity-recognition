package main

import (
	"context"
	"github.com/gin-gonic/gin"
	"gitlab.mdcatapult.io/informatics/software-engineering/entity-recognition/go/gen/pb"
	"gitlab.mdcatapult.io/informatics/software-engineering/entity-recognition/go/lib"
	"io"
	"sync"
)

type server struct {
	clients []pb.RecognizerClient
}

func (s server) RegisterRoutes(r *gin.Engine) {
	r.POST("/html/text", s.HTMLToText)
	r.POST("/html/tokens", s.TokenizeHTML)
	r.POST("/html/entities", s.RecognizeInHTML)
}

func (s server) RecognizeInHTML(c *gin.Context) {


	// Instantiate streaming clients for all of our recognisers
	var err error
	recognisers := make([]pb.Recognizer_RecognizeClient, len(s.clients))
	for i, client := range s.clients {
		recognisers[i], err = client.Recognize(context.Background())
	}

	// For each recogniser, instantiate a goroutine which listens for entities
	// and appends them to a slice. The mutex is necessary for thread safety when
	// manipulating the slice. If there is an error, send it to the error channel.
	// When complete, send nil to the error channel.
	errChan := make(chan error, len(s.clients))
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
}

func (s server) TokenizeHTML(c *gin.Context) {
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
}

func (s server) HTMLToText(c *gin.Context) {
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
}
