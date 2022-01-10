/*
grpc_recogniser provides a client which uses gRPC to communicate with a gRPC server which can perform entity recognition.
This is used by the recognition API
*/

package grpc_recogniser

import (
	"context"
	"gitlab.mdcatapult.io/informatics/software-engineering/entity-recognition/go/lib"
	"io"
	"sync"

	"gitlab.mdcatapult.io/informatics/software-engineering/entity-recognition/go/gen/pb"
	"gitlab.mdcatapult.io/informatics/software-engineering/entity-recognition/go/lib/blacklist"
	"gitlab.mdcatapult.io/informatics/software-engineering/entity-recognition/go/lib/recogniser"
	snippet_reader "gitlab.mdcatapult.io/informatics/software-engineering/entity-recognition/go/lib/snippet-reader"
	"gitlab.mdcatapult.io/informatics/software-engineering/entity-recognition/go/lib/text"
)

func New(name string, client pb.RecognizerClient, blacklist blacklist.Blacklist) recogniser.Client {
	return &grpcRecogniser{
		Name:      name,
		client:    client,
		err:       nil,
		entities:  nil,
		stream:    nil,
		blacklist: blacklist,
	}
}

type grpcRecogniser struct {
	Name       string
	client     pb.RecognizerClient
	err        error
	entities   []*pb.Entity
	stream     pb.Recognizer_GetStreamClient
	blacklist  blacklist.Blacklist
	exactMatch bool
}

func (grpcRecogniser *grpcRecogniser) SetExactMatch(exact bool) {
	grpcRecogniser.exactMatch = exact
}

// Recognise calls the helper function recognise. This listens for snippets on the given channel, and blocks with waitGroup 
// until the gRPC recogniser has returned results for every snippet.
func (grpcRecogniser *grpcRecogniser) Recognise(snipReaderValues <-chan snippet_reader.Value, waitGroup *sync.WaitGroup, _ lib.HttpOptions) error {
	grpcRecogniser.reset()

	var err error
	grpcRecogniser.stream, err = grpcRecogniser.client.GetStream(context.Background())
	if err != nil {
		return err
	}

	go grpcRecogniser.recognise(snipReaderValues, waitGroup )

	return nil
}

func (grpcRecogniser *grpcRecogniser) reset() {
	grpcRecogniser.err = nil
	grpcRecogniser.entities = nil
	grpcRecogniser.stream = nil
}

// This function doesn't return anything. Instead, we expect the caller to check `recogniser.Err()` when the
// wait group completes.
func (grpcRecogniser *grpcRecogniser) recognise(snipReaderValues <-chan snippet_reader.Value, waitGroup *sync.WaitGroup) {
	// Add to the work group - makes the caller wait.
	waitGroup.Add(1)

	// In a separate goroutine, listen on the stream for entities and append to the entities field of the receiver.
	// The stream will exit successfully with an io.EOF. Only call waitGroup.Done() when we've finished listening to the response.
	go func() {
		defer waitGroup.Done()
		for {
			entity, err := grpcRecogniser.stream.Recv()
			if err == io.EOF {
				return
			} else if err != nil {
				grpcRecogniser.err = err
				return
			}

			if !grpcRecogniser.blacklist.Allowed(entity.Name) {
				continue
			}

			grpcRecogniser.entities = append(grpcRecogniser.entities, &pb.Entity{
				Name:        entity.Name,
				Position:    entity.Position,
				Xpath:       entity.Xpath,
				Recogniser:  grpcRecogniser.Name,
				Identifiers: entity.Identifiers,
				Metadata:    entity.Metadata,
			})
		}
	}()

	// Send token to stream when a token is found
	onTokenizeCallback := func(snippet *pb.Snippet) error {
		return text.Tokenize(snippet, func(snippet *pb.Snippet) error {

			if err := grpcRecogniser.stream.Send(snippet); err != nil {
				return err
			}
			return nil
		}, grpcRecogniser.exactMatch)
	}

	// Read from the input channel, tokenise the snippets we read and send them on the stream.
	err := snippet_reader.ReadChannelWithCallback(snipReaderValues, onTokenizeCallback)
	if err != nil {
		grpcRecogniser.err = err
		return
	}

	// Close the stream. This lets the server know we've stopped sending, then it will know to send an io.EOF
	// back to us.
	if err := grpcRecogniser.stream.CloseSend(); err != nil {
		grpcRecogniser.err = err
		return
	}
}

func (grpcRecogniser *grpcRecogniser) Err() error {
	return grpcRecogniser.err
}

func (grpcRecogniser *grpcRecogniser) Result() []*pb.Entity {
	return grpcRecogniser.entities
}
