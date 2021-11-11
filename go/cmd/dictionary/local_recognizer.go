package main

import (
	"io"

	"github.com/rs/zerolog/log"
	"gitlab.mdcatapult.io/informatics/software-engineering/entity-recognition/go/gen/pb"
	"gitlab.mdcatapult.io/informatics/software-engineering/entity-recognition/go/lib/cache/local"
)

type localRecogniser struct {
	pb.UnimplementedRecognizerServer
	localCache local.Client
}

func initializeRequest(stream pb.Recognizer_GetStreamServer) *requestVars {
	return &requestVars{
		snippetHistory: []*pb.Snippet{},
		stream:         stream,
	}
}

func (r *localRecogniser) Recognize(stream pb.Recognizer_GetStreamServer) error {
	vars := initializeRequest(stream)
	log.Info().Msg("received request")

	for {
		token, err := stream.Recv()
		if err == io.EOF {
			break
		} else if err != nil {
			return err
		}

		compoundTokens, skip := getCompoundSnippets(vars, token)
		if skip {
			continue
		}

		for _, compoundToken := range compoundTokens {
			if lookup := r.localCache.Get(compoundToken.GetText()); lookup != nil {
				entity := &pb.Entity{
					Name:      compoundToken.GetText(),
					Position:    compoundToken.GetOffset(),
					Recogniser:  lookup.Dictionary,
					Identifiers: lookup.Identifiers,
					Metadata:    lookup.Metadata,
				}

				if err := vars.stream.Send(entity); err != nil {
					return err
				}
			}
		}
	}

	return nil
}
