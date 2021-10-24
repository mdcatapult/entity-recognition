package main

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"sync"
	"time"

	"gitlab.mdcatapult.io/informatics/software-engineering/entity-recognition/go/gen/pb"
	"gitlab.mdcatapult.io/informatics/software-engineering/entity-recognition/go/lib"
	http_recogniser "gitlab.mdcatapult.io/informatics/software-engineering/entity-recognition/go/lib/http-recogniser"
	"gitlab.mdcatapult.io/informatics/software-engineering/entity-recognition/go/lib/text"
)

type Options struct {
	HttpOptions http_recogniser.Options `json:"http_options"`
}

type controller struct {
	grpcRecogniserClients map[string]pb.RecognizerClient
	httpRecogniserClients map[string]http_recogniser.Client
}

func (c controller) HTMLToText(reader io.Reader) ([]byte, error) {
	var data []byte
	onSnippet := func(snippet *pb.Snippet) error {
		data = append(data, snippet.GetToken()...)
		return nil
	}
	if err := lib.HtmlToTextWithCallback(reader, onSnippet); err != nil {
		return nil, err
	}

	return data, nil
}

func (c controller) TokenizeHTML(reader io.Reader) ([]*pb.Snippet, error) {
	// This is a callback which is executed when the lib.HtmlToText function reaches some kind
	// of delimiter, e.g. </br>. Here we tokenize the output and append that to our token slice.
	var tokens []*pb.Snippet
	onSnippet := func(snippet *pb.Snippet) error {
		return text.Tokenize(snippet, func(snippet *pb.Snippet) error {
			text.NormalizeSnippet(snippet)
			if len(snippet.Token) > 0 {
				tokens = append(tokens, snippet)
			}
			return nil
		})
	}

	// Call htmlToText with our callback
	if err := lib.HtmlToTextWithCallback(reader, onSnippet); err != nil {
		return nil, err
	}

	return tokens, nil
}

func (c controller) RecognizeInHTML(reader io.Reader, opts map[string]Options) ([]*pb.RecognizedEntity, error) {

	// Instantiate streaming grpcRecogniserClients for all of our recognisers
	var err error
	grpcRecognisers := make(map[string]pb.Recognizer_RecognizeClient)
	httpRecognisers := make(map[string]http_recogniser.Options)
	for name, options := range opts {
		if grpcClient, ok := c.grpcRecogniserClients[name]; ok {
			grpcRecognisers[name], err = grpcClient.Recognize(context.Background())
		} else if _, ok := c.httpRecogniserClients[name]; ok {
			httpRecognisers[name] = options.HttpOptions
		} else {
			return nil, HttpError{
				code:  400,
				error: fmt.Errorf("recogniser '%s' does not exist", name),
			}
		}
		if err != nil {
			return nil, err
		}
	}

	body, err := ioutil.ReadAll(reader)
	if err != nil {
		return nil, err
	}

	// When complete, send nil to the error channel.
	// manipulating the slice. If there is an error, send it to the error channel.
	// and appends them to a slice. The mutex is necessary for thread safety when
	// For each recogniser, instantiate a goroutine which listens for entities
	errChan := make(chan error, len(opts))
	entities := make([]*pb.RecognizedEntity, 0, 1000)
	var mut sync.Mutex
	for _, recogniser := range grpcRecognisers {
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

	httpResponses := make(chan []*pb.RecognizedEntity)
	go func() {
		for range httpRecognisers {
			resp := <-httpResponses
			mut.Lock()
			entities = append(entities, resp...)
			mut.Unlock()
		}
	}()

	for name, options := range httpRecognisers {
		go c.httpRecogniserClients[name].Recognise(bytes.NewReader(body), options, httpResponses, errChan)
	}

	// Callback to the html to text function. Tokenize each block of text
	// and send every token to every recogniser.
	onSnippet := func(snippet *pb.Snippet) error {
		return text.Tokenize(snippet, func(snippet *pb.Snippet) error {
			for _, recogniser := range grpcRecognisers {
				if err := recogniser.Send(snippet); err != nil {
					return err
				}
			}
			return nil
		})
	}

	if err := lib.HtmlToTextWithCallback(bytes.NewReader(body), onSnippet); err != nil {
		return nil, err
	}

	// HtmlToText blocks until it is complete, so at this point the callback
	// will not be called again, and it is safe to call CloseSend on the recognisers
	// to close the stream (this doesn't stop us receiving entities).
	for _, recogniser := range grpcRecognisers {
		if err := recogniser.CloseSend(); err != nil {
			return nil, err
		}
	}

	// This for loop will block execution until all of the go routines
	// we spawned above have sent either nil or an error on the error
	// channel. If there is no error on any channel, we can continue.
	for range opts {
		if err = <-errChan; err != nil {
			return nil, err
		}
	}

	// Http recognisers send "nil" on the error channel immediately after sending their response.
	// This doesn't give the controller quite enough time to unlock the mutex and append the result
	// so we're just sleeping a little to let it catch up.
	time.Sleep(10 * time.Millisecond)

	return entities, nil
}
