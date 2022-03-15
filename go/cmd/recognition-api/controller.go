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
	"text/html":  contentTypeHTML,
	"text/plain": contentTypeRawtext,
}

type controller struct {
	recognisers map[string]recogniser.Client
	htmlReader  snippet_reader.Client
	textReader  snippet_reader.Client
	blacklist   blacklist.Blacklist // a global blacklist to apply against all recognisers
	exactMatch  bool
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
		}, c.exactMatch)
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

// Recognize performs entity recognition by calling recognise() on each recogniser in recogniserToOpts.
func (controller controller) Recognize(reader io.Reader, contentType AllowedContentType, requestedRecognisers []lib.RecogniserOptions) ([]*pb.Entity, error) {

	waitGroup := &sync.WaitGroup{}
	channels := make(map[string]chan snippet_reader.Value)

	for _, recogniser := range requestedRecognisers {

		// check that requested recogniser has been configured on controller
		validRecogniser, ok := controller.recognisers[recogniser.Name]
		if !ok {
			return nil, HttpError{
				code:  400,
				error: fmt.Errorf("no such recogniser '%s'", recogniser.Name),
			}
		}

		validRecogniser.SetExactMatch(controller.exactMatch)

		channels[recogniser.Name] = make(chan snippet_reader.Value)
		err := validRecogniser.Recognise(channels[recogniser.Name], waitGroup, recogniser.HttpOptions)
		if err != nil {
			return nil, err
		}
	}

	var snippetReaderValues <-chan snippet_reader.Value
	switch contentType {
	case contentTypeHTML:
		snippetReaderValues = controller.htmlReader.ReadSnippets(reader)
	case contentTypeRawtext:
		snippetReaderValues = controller.textReader.ReadSnippets(reader)
	}

	// all the bits of text as snippets (with an error)
	for snippetReaderValue := range snippetReaderValues {
		// TODO could the snippetReaderValue.Err value be an actual error here?
		SendToAll(snippetReaderValue, channels) // every value goes to every channel (recogniser) which is defined above
		if snippetReaderValue.Err != nil {
			break
		}
	}

	waitGroup.Wait()
	for _, recogniser := range requestedRecognisers {
		if err := controller.recognisers[recogniser.Name].Err(); err != nil {
			return nil, err
		}
	}

	allowedEntities := make([]*pb.Entity, 0)
	// uniqueEntities := make([]*pb.Entity, 0)

	for _, recogniser := range requestedRecognisers {
		recognisedEntities := controller.recognisers[recogniser.Name].Result()

		recogniserUniqueEntities := filterUniqueEntities(recognisedEntities)

		// apply global blacklist
		allowedEntities = append(allowedEntities, controller.blacklist.FilterEntities(recogniserUniqueEntities)...)

	}

	return allowedEntities, nil
}

func filterUniqueEntities(input []*pb.Entity) []*pb.Entity {
	uniqueEntities := make([]*pb.Entity, 0)

	for _, entity := range input {
		isInUnique := false
		uniqueIndex := 0

		for i, uniqueEntity := range uniqueEntities {
			if entity.Name == uniqueEntity.Name {
				isInUnique = true
				uniqueIndex = i
				break
			}
		}

		if !isInUnique {
			entity.Positions = []*pb.Position{
				&pb.Position{
					Xpath:    entity.Xpath,
					Position: entity.Position,
				},
			}
			entity.Xpath = ""
			entity.Position = 0
			uniqueEntities = append(uniqueEntities, entity)
		} else {
			uniqueEntity := uniqueEntities[uniqueIndex]
			positions := uniqueEntity.Positions
			uniqueEntity.Positions = append(positions, &pb.Position{
				Xpath:    entity.Xpath,
				Position: entity.Position,
			})
		}
	}

	return uniqueEntities
}

func SendToAll(snipReaderValue snippet_reader.Value, channels map[string]chan snippet_reader.Value) {
	for _, channel := range channels {
		channel <- snipReaderValue
	}
}
