package grpc_recogniser

import (
	"context"
	"gitlab.mdcatapult.io/informatics/software-engineering/entity-recognition/go/gen/pb"
	"gitlab.mdcatapult.io/informatics/software-engineering/entity-recognition/go/lib"
	"gitlab.mdcatapult.io/informatics/software-engineering/entity-recognition/go/lib/recogniser"
	"gitlab.mdcatapult.io/informatics/software-engineering/entity-recognition/go/lib/text"
	"io"
	"sync"
)

func New(client pb.RecognizerClient) recogniser.Client {
	return &grpcRecogniser{
		client:   client,
		err:      nil,
		entities: nil,
		stream:   nil,
	}
}

type grpcRecogniser struct {
	client pb.RecognizerClient
	err error
	entities []*pb.RecognizedEntity
	stream pb.Recognizer_GetStreamClient
}

func(g *grpcRecogniser) Recognise(snipReaderValues <-chan lib.SnipReaderValue, _ lib.RecogniserOptions, wg *sync.WaitGroup) error {
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

func (g *grpcRecogniser) recognise(snipReaderValues <-chan lib.SnipReaderValue, wg *sync.WaitGroup) {
	wg.Add(1)
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
			g.entities = append(g.entities, entity)
		}
	}()

	err := lib.ReadSnippets(snipReaderValues, func (snippet *pb.Snippet) error {
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


	if err := g.stream.CloseSend(); err != nil {
		g.err = err
		return
	}
}

func(g *grpcRecogniser) Err() error {
	return g.err
}

func(g *grpcRecogniser) Result() []*pb.RecognizedEntity {
	return g.entities
}
