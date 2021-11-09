package grpc_recogniser

import (
	"context"
	"gitlab.mdcatapult.io/informatics/software-engineering/entity-recognition/go/lib/blacklist"
	"io"
	"sync"

	"gitlab.mdcatapult.io/informatics/software-engineering/entity-recognition/go/gen/pb"
	"gitlab.mdcatapult.io/informatics/software-engineering/entity-recognition/go/lib"
	"gitlab.mdcatapult.io/informatics/software-engineering/entity-recognition/go/lib/recogniser"
	snippet_reader "gitlab.mdcatapult.io/informatics/software-engineering/entity-recognition/go/lib/snippet-reader"
	"gitlab.mdcatapult.io/informatics/software-engineering/entity-recognition/go/lib/text"
)

func New(name string, client pb.RecognizerClient) recogniser.Client {
	return &grpcRecogniser{
		Name:     name,
		client:   client,
		err:      nil,
		entities: nil,
		stream:   nil,
	}
}

type grpcRecogniser struct {
	Name     string
	client   pb.RecognizerClient
	err      error
	entities []*pb.RecognizedEntity
	stream   pb.Recognizer_GetStreamClient
}

func (g *grpcRecogniser) Recognise(snipReaderValues <-chan snippet_reader.Value, _ lib.RecogniserOptions, wg *sync.WaitGroup) error {
	g.reset()

	var err error
	g.stream, err = g.client.GetStream(context.Background())
	if err != nil {
		return err
	}

	go g.recognise(snipReaderValues, wg)

	return nil
}

func (g *grpcRecogniser) reset() {
	g.err = nil
	g.entities = nil
	g.stream = nil
}

// This function doesn't return anything. Instead we expect the caller to check `recogniser.Err()` when the
// wait group completes.
func (g *grpcRecogniser) recognise(snipReaderValues <-chan snippet_reader.Value, wg *sync.WaitGroup) {
	// Add to the work group - makes the caller wait.
	wg.Add(1)

	// In a separate goroutine, listen on the stream for entities and append to the entities field of the receiver.
	// The stream will exit successfully with an io.EOF. Only call wg.Done() when we've finished listening to the response.
	go func() {
		defer wg.Done()
		for {
			entity, err := g.stream.Recv()
			if err == io.EOF {
				return
			} else if err != nil {
				g.err = err
				return
			}
			if !blacklist.SnippetAllowed(entity.Entity) {
				continue
			}

			g.entities = append(g.entities, &pb.RecognizedEntity{
				Entity:      entity.Entity,
				Position:    entity.Position,
				Xpath:       entity.Xpath,
				Recogniser:  g.Name,
				Identifiers: entity.Identifiers,
				Metadata:    entity.Metadata,
			})
		}
	}()

	// Read from the input channel, tokenise the snippets we read and send them on the stream.
	err := snippet_reader.ReadChannelWithCallback(snipReaderValues, func(snippet *pb.Snippet) error {
		return text.Tokenize(snippet, func(snippet *pb.Snippet) error {

			if err := g.stream.Send(snippet); err != nil {
				return err
			}
			return nil
		})
	})
	if err != nil {
		g.err = err
	}

	// Close the stream. This lets the server know we've stopped sending, then it will know to send an io.EOF
	// back to us.
	if err := g.stream.CloseSend(); err != nil {
		g.err = err
		return
	}
}

func (g *grpcRecogniser) Err() error {
	return g.err
}

func (g *grpcRecogniser) Result() []*pb.RecognizedEntity {
	return g.entities
}
