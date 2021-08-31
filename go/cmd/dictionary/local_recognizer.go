package main

import (
	"gitlab.mdcatapult.io/informatics/software-engineering/entity-recognition/go/lib/cache/local"
	"gitlab.mdcatapult.io/informatics/software-engineering/entity-recognition/go/lib/text"
	"io"
	"strings"

	"github.com/rs/zerolog/log"
	"gitlab.mdcatapult.io/informatics/software-engineering/entity-recognition/go/gen/pb"
)

type localRecogniser struct {
	pb.UnimplementedRecognizerServer
	localCache local.Client
}

func (r *localRecogniser) getCompoundSnippets(vars *requestVars, snippet *pb.Snippet) []*pb.Snippet {
	// If sentenceEnd is true, we can save some redis queries by resetting the token history..
	if vars.sentenceEnd {
		vars.snippetHistory = []*pb.Snippet{}
		vars.tokenHistory = []string{}
		vars.sentenceEnd = false
	}

	// normalise the token (remove enclosing punctuation and enforce NFKC encoding).
	// sentenceEnd is true if the last byte in the token is one of '.', '?', or '!'.
	vars.sentenceEnd = text.Normalize(snippet)

	// manage the token history
	if len(vars.snippetHistory) < config.CompoundTokenLength {
		vars.snippetHistory = append(vars.snippetHistory, snippet)
		vars.tokenHistory = append(vars.tokenHistory, snippet.GetToken())
	} else {
		vars.snippetHistory = append(vars.snippetHistory[1:], snippet)
		vars.tokenHistory = append(vars.tokenHistory[1:], snippet.GetToken())
	}

	// construct the compound tokens to query against redis.
	compoundSnippets := make([]*pb.Snippet, len(vars.snippetHistory))
	for i, historicalToken := range vars.snippetHistory {
		compoundSnippets[i] = &pb.Snippet{
			Token:  strings.Join(vars.tokenHistory[i:], " "),
			Offset: historicalToken.GetOffset(),
		}
	}
	return compoundSnippets
}

func (r *localRecogniser) initializeRequest(stream pb.Recognizer_RecognizeServer) *requestVars {
	return &requestVars{
		snippetHistory:   []*pb.Snippet{},
		tokenHistory:     []string{},
		sentenceEnd:      false,
		stream:           stream,
	}
}

func (r *localRecogniser) Recognize(stream pb.Recognizer_RecognizeServer) error {
	vars := r.initializeRequest(stream)
	log.Info().Msg("received request")

	for {
		token, err := stream.Recv()
		if err == io.EOF {
			break
		} else if err != nil {
			return err
		}

		compoundTokens := r.getCompoundSnippets(vars, token)

		for _, compoundToken := range compoundTokens {
			if lookup := r.localCache.Get(compoundToken.GetToken()); lookup != nil {
				entity := &pb.RecognizedEntity{
					Entity:     compoundToken.GetToken(),
					Position:   compoundToken.GetOffset(),
					Type:       lookup.Dictionary,
					ResolvedTo: lookup.ResolvedEntities,
				}

				if err := vars.stream.Send(entity); err != nil {
					return err
				}
			}
		}
	}

	return nil
}
