/*
 * Copyright 2022 Medicines Discovery Catapult
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *     http://www.apache.org/licenses/LICENSE-2.0
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package grpc_recogniser

import (
	"context"
	"io"
	"sync"

	"gitlab.mdcatapult.io/informatics/software-engineering/entity-recognition/go/gen/pb"
	"gitlab.mdcatapult.io/informatics/software-engineering/entity-recognition/go/lib"
	"gitlab.mdcatapult.io/informatics/software-engineering/entity-recognition/go/lib/blocklist"
	"gitlab.mdcatapult.io/informatics/software-engineering/entity-recognition/go/lib/recogniser"
	snippet_reader "gitlab.mdcatapult.io/informatics/software-engineering/entity-recognition/go/lib/snippet-reader"
	"gitlab.mdcatapult.io/informatics/software-engineering/entity-recognition/go/lib/text"
)

func New(name string, client pb.RecognizerClient, blocklist blocklist.Blocklist) recogniser.Client {
	return &grpcRecogniser{
		Name:      name,
		client:    client,
		err:       nil,
		entities:  nil,
		stream:    nil,
		blocklist: blocklist,
	}
}

type grpcRecogniser struct {
	Name       string
	client     pb.RecognizerClient
	err        error
	entities   []*pb.Entity
	stream     pb.Recognizer_GetStreamClient
	blocklist  blocklist.Blocklist
	exactMatch bool
}

func (g *grpcRecogniser) SetExactMatch(exact bool) {
	g.exactMatch = exact
}

func (g *grpcRecogniser) Recognise(snipReaderValues <-chan snippet_reader.Value, wg *sync.WaitGroup, _ lib.HttpOptions) error {
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

			if !g.blocklist.Allowed(entity.Name) {
				continue
			}

			g.entities = append(g.entities, &pb.Entity{
				Name:        entity.Name,
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
		}, g.exactMatch)
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

func (g *grpcRecogniser) Result() []*pb.Entity {
	return g.entities
}
