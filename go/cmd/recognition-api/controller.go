package main

import (
	"context"
	"io"
	"sync"

	"gitlab.mdcatapult.io/informatics/software-engineering/entity-recognition/go/gen/pb"
	"gitlab.mdcatapult.io/informatics/software-engineering/entity-recognition/go/lib"
	"gitlab.mdcatapult.io/informatics/software-engineering/entity-recognition/go/lib/text"
)

type controller struct {
	clients []pb.RecognizerClient
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

func (c controller) RecognizeInHTML(reader io.Reader) ([]*pb.RecognizedEntity, error) {

	// Instantiate streaming clients for all of our recognisers
	var err error
	recognisers := make([]pb.Recognizer_RecognizeClient, len(c.clients))
	for i, client := range c.clients {
		recognisers[i], err = client.Recognize(context.Background())
		if err != nil {
			return nil, err
		}
	}

	// For each recogniser, instantiate a goroutine which listens for entities
	// and appends them to a slice. The mutex is necessary for thread safety when
	// manipulating the slice. If there is an error, send it to the error channel.
	// When complete, send nil to the error channel.
	errChan := make(chan error, len(c.clients))
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
		return text.Tokenize(snippet, func(snippet *pb.Snippet) error {
			for _, recogniser := range recognisers {
				if err := recogniser.Send(snippet); err != nil {
					return err
				}
			}
			return nil
		})
	}

	if err := lib.HtmlToTextWithCallback(reader, onSnippet); err != nil {
		return nil, err
	}

	// HtmlToText blocks until it is complete, so at this point the callback
	// will not be called again, and it is safe to call CloseSend on the recognisers
	// to close the stream (this doesn't stop us receiving entities).
	for _, recogniser := range recognisers {
		if err := recogniser.CloseSend(); err != nil {
			return nil, err
		}
	}

	// This for loop will block execution until all of the go routines
	// we spawned above have sent either nil or an error on the error
	// channel. If there is no error on any channel, we can continue.
	for range recognisers {
		if err = <-errChan; err != nil {
			return nil, err
		}
	}

	return entities, nil
}
