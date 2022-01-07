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
		Synonyms:    []string{"whee"},
		Identifiers: map[string]string{"id key": "id value"},
		Metadata:    map[string]string{"meta": "eyJlbnRpdHlHcm91cCI6IkNoZW1pY2FsIiwiUmVjb2duaXNpbmdEaWN0Ijp7ImVuZm9yY2VCcmFja2V0aW5nIjp0cnVlLCJlbnRpdHlUeXBlIjoiTW9sIiwiaHRtbENvbG9yIjoicGluayIsIm1heENvcnJlY3Rpb25EaXN0YW5jZSI6MCwibWluaW11bUNvcnJlY3RlZEVudGl0eUxlbmd0aCI6OSwibWluaW11bUVudGl0eUxlbmd0aCI6MCwic291cmNlIjoiIn19"},
	}

	// insert data into redis
	err := addToRedis(redisClient, data)
	assert.NoError(t, err)

	synonym := data.Synonyms[0]

	entities, err := readFromRedis(redisClient, synonym)
	assert.NoError(t, err)

	assert.Equal(t, len(entities), 1)
	assert.Equal(t, entities[0].Metadata, data.Metadata)
	assert.Equal(t, entities[0].Identifiers, data.Identifiers)
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

		bytes, err := json.Marshal(cache.Lookup{
			Dictionary:  config.Dictionary.Name,
			Identifiers: entry.Identifiers,
			Metadata:    entry.Metadata,
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
// a grpc_recogniser.
func readFromRedis(client remote.Client, synonym string) (entities []*pb.Entity, err error) {

	grpcServer := grpc.NewServer()
	pb.RegisterRecognizerServer(grpcServer, &recogniser{
		remoteCache: client,
	})

	port := 50053

	lis, err := net.Listen("tcp", ":50053")
	if err != nil {
		fmt.Println("ERROR: failed to serve", err.Error())
	}
	go grpcServer.Serve(lis)

	// set up grpc client
	var clientOptions []grpc.DialOption
	clientOptions = append(clientOptions, grpc.WithInsecure())
	clientOptions = append(clientOptions, grpc.WithBlock())
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)

	conn, err := grpc.DialContext(ctx, fmt.Sprintf("%s:%d", "localhost", port), clientOptions...)

	if err != nil {
		log.Fatal().Err(err).Send()
	}

	cancel()

	// set up grpc client
	recogniser := grpc_recogniser.New("my recogniser", pb.NewRecognizerClient(conn), blacklist.Blacklist{})

	wg := &sync.WaitGroup{}
	snippetChannel := make(chan snippet_reader.Value)
	if err := recogniser.Recognise(snippetChannel, lib.RecogniserOptions{}, wg); err != nil {
		fmt.Println("RECOGNISER ERROR: ", err.Error())
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
