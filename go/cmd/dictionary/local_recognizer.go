package main

import (
	"github.com/rs/zerolog/log"
	"gitlab.mdcatapult.io/informatics/software-engineering/entity-recognition/go/gen/pb"
	"gitlab.mdcatapult.io/informatics/software-engineering/entity-recognition/go/lib/cache/local"
	"io"
)

type localRecogniser struct {
	pb.UnimplementedRecognizerServer
	localCache local.Client
}

func initializeRequest(stream pb.Recognizer_RecognizeServer) *requestVars {
	return &requestVars{
		snippetHistory:   []*pb.Snippet{},
		tokenHistory:     []string{},
		sentenceEnd:      false,
		stream:           stream,
	}
}

func (r *localRecogniser) Recognize(stream pb.Recognizer_RecognizeServer) error {
	vars := initializeRequest(stream)
	log.Info().Msg("received request")

	for {
		token, err := stream.Recv()
		if err == io.EOF {
			break
		} else if err != nil {
			return err
		}

		compoundTokens := getCompoundSnippets(vars, token)

		for _, compoundToken := range compoundTokens {
			if lookup := r.localCache.Get(compoundToken.GetToken()); lookup != nil {
				entity := &pb.RecognizedEntity{
					Entity:     compoundToken.GetToken(),
					Position:   compoundToken.GetOffset(),
					Dictionary:       lookup.Dictionary,
					Identifiers: lookup.Identifiers,
					Metadata: lookup.Metadata,
				}

				if err := vars.stream.Send(entity); err != nil {
					return err
				}
			}
		}
	}

	return nil
}
