package main

import (
	"gitlab.mdcatapult.io/informatics/software-engineering/entity-recognition/go/lib/cache"
	"gitlab.mdcatapult.io/informatics/software-engineering/entity-recognition/go/lib/text"
	"io"
	"strings"

	"github.com/rs/zerolog/log"
	"gitlab.mdcatapult.io/informatics/software-engineering/entity-recognition/go/gen/pb"
	"gitlab.mdcatapult.io/informatics/software-engineering/entity-recognition/go/lib/cache/remote"
)

type recogniser struct {
	pb.UnimplementedRecognizerServer
	remoteCache remote.Client
}

type requestVars struct {
	tokenCache       map[*pb.Snippet]*cache.Lookup
	tokenCacheMisses []*pb.Snippet
	snippetHistory   []*pb.Snippet
	tokenHistory     []string
	sentenceEnd      bool
	stream           pb.Recognizer_RecognizeServer
	pipe             remote.GetPipeline
}

func (r *recogniser) newResultHandler(vars *requestVars) func(snippet *pb.Snippet, lookup *cache.Lookup) error {
	return func(snippet *pb.Snippet, lookup *cache.Lookup) error {
		vars.tokenCache[snippet] = lookup
		if lookup == nil {
			return nil
		}
		entity := &pb.RecognizedEntity{
			Entity:     snippet.GetToken(),
			Position:   snippet.GetOffset(),
			Type:       lookup.Dictionary,
			ResolvedTo: lookup.ResolvedEntities,
		}

		if err := vars.stream.Send(entity); err != nil {
			return err
		}

		return nil
	}
}

func (r *recogniser) getCompoundSnippets(vars *requestVars, snippet *pb.Snippet) []*pb.Snippet {
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

func (r *recogniser) findOrQueueSnippet(vars *requestVars, token *pb.Snippet) error {
	if lookup, ok := vars.tokenCache[token]; ok {
		// if it's nil, we've already queried redis and it wasn't there
		if lookup == nil {
			return nil
		}
		// If it's empty, it's already queued but we don't know if its there or not.
		// Append it to the cacheMisses to be found later.
		if lookup.Dictionary == "" {
			vars.tokenCacheMisses = append(vars.tokenCacheMisses, token)
			return nil
		}
		// Otherwise, construct an entity from the cache value and send it back to the caller.
		entity := &pb.RecognizedEntity{
			Entity:     token.GetToken(),
			Position:   token.GetOffset(),
			Type:       lookup.Dictionary,
			ResolvedTo: lookup.ResolvedEntities,
		}
		if err := vars.stream.Send(entity); err != nil {
			return err
		}
	} else {
		// Not in local cache.
		// Queue the redis "GET" in the pipe and set the cache value to an empty db.Lookup
		// (so that future equivalent tokens will be a cache miss).
		vars.pipe.Get(token)
		vars.tokenCache[token] = &cache.Lookup{}
	}
	return nil
}

func (r *recogniser) initializeRequest(stream pb.Recognizer_RecognizeServer) *requestVars {
	return &requestVars{
		tokenCache:       make(map[*pb.Snippet]*cache.Lookup, config.PipelineSize),
		tokenCacheMisses: make([]*pb.Snippet, config.PipelineSize),
		snippetHistory:   []*pb.Snippet{},
		tokenHistory:     []string{},
		sentenceEnd:      false,
		stream:           stream,
		pipe:             r.remoteCache.NewGetPipeline(config.PipelineSize),
	}
}

func (r *recogniser) runPipeline(vars *requestVars, onResult func(snippet *pb.Snippet, lookup *cache.Lookup) error) error {
	if err := vars.pipe.ExecGet(onResult); err != nil {
		return err
	}
	vars.pipe = r.remoteCache.NewGetPipeline(config.PipelineSize)
	return nil
}

func (r *recogniser) retryCacheMisses(vars *requestVars) error {
	for _, token := range vars.tokenCacheMisses {
		if lookup := vars.tokenCache[token]; lookup != nil {
			entity := &pb.RecognizedEntity{
				Entity:     token.GetToken(),
				Position:   token.GetOffset(),
				Type:       lookup.Dictionary,
				ResolvedTo: lookup.ResolvedEntities,
			}
			if err := vars.stream.Send(entity); err != nil {
				return err
			}
		}
	}
	return nil
}

func (r *recogniser) Recognize(stream pb.Recognizer_RecognizeServer) error {
	vars := r.initializeRequest(stream)
	log.Info().Msg("received request")
	onResult := r.newResultHandler(vars)

	for {
		token, err := stream.Recv()
		if err == io.EOF {
			// Number of tokens is unlikely to be a multiple of the pipeline size. There will still be tokens on the
			// pipeline. Execute it now, then break.
			if vars.pipe.Size() > 0 {
				if err := r.runPipeline(vars, onResult); err != nil {
					return err
				}
			}
			break
		} else if err != nil {
			return err
		}

		compoundTokens := r.getCompoundSnippets(vars, token)

		for _, compoundToken := range compoundTokens {
			if err := r.findOrQueueSnippet(vars, compoundToken); err != nil {
				return err
			}
		}

		if vars.pipe.Size() > config.PipelineSize {
			if err := r.runPipeline(vars, onResult); err != nil {
				return err
			}
		}
	}

	return r.retryCacheMisses(vars)
}
