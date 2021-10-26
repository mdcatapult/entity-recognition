package main

import (
	"gitlab.mdcatapult.io/informatics/software-engineering/entity-recognition/go/gen/pb"
	"gitlab.mdcatapult.io/informatics/software-engineering/entity-recognition/go/lib"
	"gitlab.mdcatapult.io/informatics/software-engineering/entity-recognition/go/lib/recogniser"
	"gitlab.mdcatapult.io/informatics/software-engineering/entity-recognition/go/lib/text"
	"io"
	"sync"
)

type controller struct {
	recognisers map[string]recogniser.Client
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

func (c controller) RecognizeInHTML(reader io.Reader, opts map[string]lib.RecogniserOptions) ([]*pb.RecognizedEntity, error) {

	wg := &sync.WaitGroup{}
	channels := make(map[string]chan lib.SnipReaderValue)
	for recogniserName, recogniserOptions := range opts {
		channels[recogniserName] = make(chan lib.SnipReaderValue)
		err := c.recognisers[recogniserName].Recognise(channels[recogniserName], recogniserOptions, wg)
		if err != nil {
			return nil, err
		}
	}

	err := lib.HtmlToTextWithCallback(reader, func (snippet *pb.Snippet) error {
		SendToAll(lib.SnipReaderValue{Snippet: snippet}, opts, channels)
		return nil
	})
	if err != nil {
		// Send the error to all recognisers. They should clean themselves up and call
		// done on the waitgroup, then we'll return the error after wg.Wait.
		SendToAll(lib.SnipReaderValue{Err: err}, opts, channels)
	}

	// Send io.EOF so the recognisers know they've received the last value.
	SendToAll(lib.SnipReaderValue{Err: io.EOF}, opts, channels)

	wg.Wait()
	length := 0
	for recogniserName := range opts {
		if err := c.recognisers[recogniserName].Err(); err != nil {
			return nil, err
		}
		length += len(c.recognisers[recogniserName].Result())
	}

	recognisedEntities := make([]*pb.RecognizedEntity, 0, length)
	for recogniserName := range opts {
		recognisedEntities = append(recognisedEntities, c.recognisers[recogniserName].Result()...)
	}

	return recognisedEntities, nil
}

func SendToAll(snipReaderValue lib.SnipReaderValue, opts map[string]lib.RecogniserOptions, channels map[string]chan lib.SnipReaderValue) {
	for recogniserName := range opts {
		channels[recogniserName] <- snipReaderValue
	}
}
