package main

import (
	"fmt"
	"io"
	"sync"

	"gitlab.mdcatapult.io/informatics/software-engineering/entity-recognition/go/gen/pb"
	"gitlab.mdcatapult.io/informatics/software-engineering/entity-recognition/go/lib"
	"gitlab.mdcatapult.io/informatics/software-engineering/entity-recognition/go/lib/blacklist"
	"gitlab.mdcatapult.io/informatics/software-engineering/entity-recognition/go/lib/recogniser"
	snippet_reader "gitlab.mdcatapult.io/informatics/software-engineering/entity-recognition/go/lib/snippet-reader"
	"gitlab.mdcatapult.io/informatics/software-engineering/entity-recognition/go/lib/text"
)

type AllowedContentType int

const (
	contentTypeHTML AllowedContentType = iota
	contentTypeRawtext
)

var allowedContentTypeEnumMap = map[string]AllowedContentType{
	"text/html": contentTypeHTML,
	"text/plain": contentTypeRawtext,
}

type controller struct {
	recognisers map[string]recogniser.Client
	htmlReader  snippet_reader.Client
	textReader  snippet_reader.Client
	blacklist   blacklist.Blacklist // a global blacklist to apply against all recognisers
}

func (c controller) HTMLToText(reader io.Reader) ([]byte, error) {
	var data []byte
	onSnippet := func(snippet *pb.Snippet) error {
		data = append(data, snippet.GetText()...)
		return nil
	}
	if err := c.htmlReader.ReadSnippetsWithCallback(reader, onSnippet); err != nil {
		return nil, err
	}

	return data, nil
}

func (c controller) Tokenize(reader io.Reader, contentType AllowedContentType) ([]*pb.Snippet, error) {
	// This is a callback which is executed when the lib.ReadSnippets function reaches some kind
	// of delimiter, e.g. </br>. Here we tokenize the output and append that to our token slice.
	var tokens []*pb.Snippet
	onSnippet := func(snippet *pb.Snippet) error {
		return text.Tokenize(snippet, func(snippet *pb.Snippet) error {
			text.NormalizeSnippet(snippet)
			if len(snippet.NormalisedText) > 0 {
				tokens = append(tokens, snippet)
			}
			return nil
		})
	}

	// Call htmlToText with our callback
	var err error
	switch contentType {
	case contentTypeHTML:
		err = c.htmlReader.ReadSnippetsWithCallback(reader, onSnippet)
	case contentTypeRawtext:
		err = c.textReader.ReadSnippetsWithCallback(reader, onSnippet)
	}
	if err != nil {
		return nil, err
	}

	return tokens, nil
}

func (c controller) ListRecognisers() []string {
	recognisers := make([]string, len(c.recognisers))
	i := 0
	for r := range c.recognisers {
		recognisers[i] = r
		i++
	}
	return recognisers
}

func (c controller) Recognize(reader io.Reader, contentType AllowedContentType, opts map[string]lib.RecogniserOptions) ([]*pb.Entity, error) {

	wg := &sync.WaitGroup{}
	channels := make(map[string]chan snippet_reader.Value)
	for recogniserName, recogniserOptions := range opts {
		validRecogniser, ok := c.recognisers[recogniserName]
		if !ok {
			return nil, HttpError{
				code:  400,
				error: fmt.Errorf("no such recogniser '%s'", recogniserName),
			}
		}

		channels[recogniserName] = make(chan snippet_reader.Value)
		err := validRecogniser.Recognise(channels[recogniserName], recogniserOptions, wg)
		if err != nil {
			return nil, err
		}
	}

	var snippetReaderValues <-chan snippet_reader.Value
	switch contentType {
	case contentTypeHTML:
		snippetReaderValues = c.htmlReader.ReadSnippets(reader)
	case contentTypeRawtext:
		snippetReaderValues = c.textReader.ReadSnippets(reader)
	}

	for snippetReaderValue := range snippetReaderValues {
		SendToAll(snippetReaderValue, channels)
		if snippetReaderValue.Err != nil {
			break
		}
	}

	wg.Wait()
	length := 0
	for recogniserName := range opts {
		if err := c.recognisers[recogniserName].Err(); err != nil {
			return nil, err
		}
		length += len(c.recognisers[recogniserName].Result())
	}

	allowedEntities := make([]*pb.Entity, 0, length)
	for recogniserName := range opts {
		recognisedEntities := c.recognisers[recogniserName].Result()

		// apply global blacklist
		allowedEntities = append(allowedEntities, c.blacklist.FilterEntities(recognisedEntities)...)

	}

	return allowedEntities, nil
}

func SendToAll(snipReaderValue snippet_reader.Value, channels map[string]chan snippet_reader.Value) {
	for _, channel := range channels {
		channel <- snipReaderValue
	}
}
