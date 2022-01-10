package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"

	"gitlab.mdcatapult.io/informatics/software-engineering/entity-recognition/go/gen/pb"
	"gitlab.mdcatapult.io/informatics/software-engineering/entity-recognition/go/lib"
	"gitlab.mdcatapult.io/informatics/software-engineering/entity-recognition/go/lib/blacklist"
	"gitlab.mdcatapult.io/informatics/software-engineering/entity-recognition/go/lib/cache"
	"gitlab.mdcatapult.io/informatics/software-engineering/entity-recognition/go/lib/cache/remote"
	"gitlab.mdcatapult.io/informatics/software-engineering/entity-recognition/go/lib/dict"
	recogniser_client "gitlab.mdcatapult.io/informatics/software-engineering/entity-recognition/go/lib/recogniser"
	grpc_recogniser "gitlab.mdcatapult.io/informatics/software-engineering/entity-recognition/go/lib/recogniser/grpc-recogniser"
	snippet_reader "gitlab.mdcatapult.io/informatics/software-engineering/entity-recognition/go/lib/snippet-reader"
	"gitlab.mdcatapult.io/informatics/software-engineering/entity-recognition/go/lib/text"
)

// Inserts an entity into redis, then calls grpc_recogniser{}.Recognise() and asserts that it comes out of the recogniser as an entity.
func Test_Redis_Recogniser(t *testing.T) {

	config.CompoundTokenLength = 1 // IF THIS IS < 1, THERE WILL BE A PANIC
	redisClient := remote.NewRedisClient(remote.RedisConfig{
		Host: "localhost",
		Port: 6379,
	})

	data := dict.Entry{
		Synonyms:    []string{"entity"},
		Identifiers: map[string]string{"id key": "id value"},
		Metadata: map[string]interface{}{
			"entityGroup": "Chemical",
			"RecognisingDict": map[string]interface{}{
				"enforceBracketing": true,
				"entityType":        "Mol",
				"htmlColor":         "pink",
			},
		},
	}

	// insert data into redis
	err := addToRedis(redisClient, data)
	assert.NoError(t, err)

	synonym := data.Synonyms[0]

	// set up grpc client
	conn, err := getReadConnection(redisClient)
	assert.NoError(t, err)
	recogniser := grpc_recogniser.New("my recogniser", pb.NewRecognizerClient(conn), blacklist.Blacklist{})

	// perform the read
	entities, err := readFromRedis(recogniser, synonym)
	assert.NoError(t, err)

	assert.Equal(t, 1, len(entities))

	var actualMetadata map[string]interface{}
	err = json.Unmarshal(entities[0].Metadata, &actualMetadata)

	assert.NoError(t, err)
	assert.Equal(t, data.Metadata, actualMetadata)
	assert.Equal(t, data.Identifiers, entities[0].Identifiers)
}

// addToRedis inserts entry into client's pipeline.
func addToRedis(client remote.Client, entry dict.Entry) error {
	pipe := client.NewSetPipeline(config.PipelineSize)

	for i, synonym := range entry.Synonyms {
		tokens := strings.Fields(synonym)
		normalizedTokens := make([]string, 0, len(tokens))
		for _, token := range tokens {
			normalizedToken, _, _ := text.NormalizeAndLowercaseString(token)
			if len(normalizedToken) > 0 {
				normalizedTokens = append(normalizedTokens, normalizedToken)
			}
		}
		entry.Synonyms[i] = strings.Join(normalizedTokens, " ")

		metadata, err := json.Marshal(entry.Metadata)
		if err != nil {
			return err
		}

		bytes, err := json.Marshal(cache.Lookup{
			Dictionary:  config.Dictionary.Name,
			Identifiers: entry.Identifiers,
			Metadata:    metadata,
		})
		if err != nil {
			return err
		}

		// add entry to pipe and immediately exec
		pipe.Set(synonym, bytes)
		if err := pipe.ExecSet(); err != nil {
			return err
		}
	}

	for !client.Ready() {
		log.Info().Msg("database is not ready, waiting...")
		time.Sleep(5 * time.Second)
	}

	return nil
}

// readFromRedis is a helper function which takes a remote.Client and looks up synonym in it using
// a grpc_recogniser. Blocks until lookup is complete.
func readFromRedis(recogniser recogniser_client.Client, synonym string) (entities []*pb.Entity, err error) {

	wg := &sync.WaitGroup{}
	snippetChannel := make(chan snippet_reader.Value)
	if err := recogniser.Recognise(snippetChannel, lib.RecogniserOptions{}, wg); err != nil {
		return nil, err
	}

	time.Sleep(1 * time.Second)
	// send snippet to recogniser channel
	snippetChannel <- snippet_reader.Value{
		Snippet: &pb.Snippet{
			Text: synonym,
		},
	}

	// send EOF to recogniser channel to tell it to stop listening and release waitGroup
	snippetChannel <- snippet_reader.Value{
		Snippet: nil,
		Err:     io.EOF,
	}

	wg.Wait()

	return recogniser.Result(), recogniser.Err()
}

func getReadConnection(client remote.Client) (*grpc.ClientConn, error) {

	grpcServer := grpc.NewServer()
	pb.RegisterRecognizerServer(grpcServer, &recogniser{
		remoteCache: client,
	})

	port := 50053

	lis, err := net.Listen("tcp", ":50053")
	if err != nil {
		return nil, err
	}

	go func() {
		if err := grpcServer.Serve(lis); err != nil {
			fmt.Println("ERROR: failed to serve", err.Error())
		}
	}()

	// set up grpc client
	var clientOptions []grpc.DialOption
	clientOptions = append(clientOptions, grpc.WithInsecure())
	clientOptions = append(clientOptions, grpc.WithBlock())

	return grpc.DialContext(context.Background(), fmt.Sprintf("%s:%d", "localhost", port), clientOptions...)
}
