package main

import (
	"io"
	"strings"

	"github.com/rs/zerolog/log"
	"gitlab.mdcatapult.io/informatics/software-engineering/entity-recognition/go/gen/pb"
	"gitlab.mdcatapult.io/informatics/software-engineering/entity-recognition/go/lib/cache"
	"gitlab.mdcatapult.io/informatics/software-engineering/entity-recognition/go/lib/cache/remote"
	"gitlab.mdcatapult.io/informatics/software-engineering/entity-recognition/go/lib/text"
)

type recogniser struct {
	pb.UnimplementedRecognizerServer
	remoteCache remote.Client
}

type requestVars struct {
	snippetCache       map[*pb.Snippet]*cache.Lookup
	snippetCacheMisses []*pb.Snippet
	snippetHistory     []*pb.Snippet
	tokenHistory       []string
	stream             pb.Recognizer_GetStreamServer
	pipe               remote.GetPipeline
}

func newEntity(snippet *pb.Snippet, lookup *cache.Lookup) *pb.RecognizedEntity {
	return &pb.RecognizedEntity{
		Entity:      snippet.GetToken(),
		Position:    snippet.GetOffset(),
		Recogniser:  lookup.Dictionary,
		Identifiers: lookup.Identifiers,
		Xpath:       snippet.GetXpath(),
	}
}

func (r *recogniser) newResultHandler(vars *requestVars) func(snippet *pb.Snippet, lookup *cache.Lookup) error {
	return func(snippet *pb.Snippet, lookup *cache.Lookup) error {
		vars.snippetCache[snippet] = lookup
		if lookup == nil {
			return nil
		}
		entity := newEntity(snippet, lookup)

		if err := vars.stream.Send(entity); err != nil {
			return err
		}

		return nil
	}
}

func getCompoundSnippets(vars *requestVars, snippet *pb.Snippet) (snippets []*pb.Snippet, skipToken bool) {
	// normalise the token (remove enclosing punctuation and enforce NFKC encoding).
	// compoundTokenEnd is true if the last byte in the token is one of '.', '?', or '!'.
	compoundTokenEnd := text.NormalizeAndLowercaseSnippet(snippet)
	if len(snippet.Token) == 0 {
		return nil, true
	}

	// manage the token history
	if len(vars.snippetHistory) < config.CompoundTokenLength {
		vars.snippetHistory = append(vars.snippetHistory, snippet)
		vars.tokenHistory = append(vars.tokenHistory, snippet.GetToken())
	} else {
		vars.snippetHistory = append(vars.snippetHistory[1:], snippet)
		vars.tokenHistory = append(vars.tokenHistory[1:], snippet.GetToken())
	}

	// construct the compound tokens to query against redis.
	snippets = make([]*pb.Snippet, len(vars.snippetHistory))
	for i, historicalSnippet := range vars.snippetHistory {
		snippets[i] = &pb.Snippet{
			Token:  strings.Join(vars.tokenHistory[i:], " "),
			Offset: historicalSnippet.GetOffset(),
			Xpath:  historicalSnippet.GetXpath(),
		}
	}

	// If compoundTokenEnd is true, we can save some redis queries by resetting the token history.
	if compoundTokenEnd {
		vars.snippetHistory = []*pb.Snippet{}
		vars.tokenHistory = []string{}
	}

	return snippets, false
}

func (r *recogniser) findOrQueueSnippet(vars *requestVars, snippet *pb.Snippet) error {
	if lookup, ok := vars.snippetCache[snippet]; ok {
		// if it's nil, we've already queried redis and it wasn't there
		if lookup == nil {
			return nil
		}
		// If it's empty, it's already queued but we don't know if its there or not.
		// Append it to the cacheMisses to be found later.
		if lookup.Dictionary == "" {
			vars.snippetCacheMisses = append(vars.snippetCacheMisses, snippet)
			return nil
		}
		// Otherwise, construct an entity from the cache value and send it back to the caller.
		entity := newEntity(snippet, lookup)
		if err := vars.stream.Send(entity); err != nil {
			return err
		}
	} else {
		// Not in local cache.
		// Queue the redis "GET" in the pipe and set the cache value to an empty db.Lookup
		// (so that future equivalent tokens will be a cache miss).
		vars.pipe.Get(snippet)
		vars.snippetCache[snippet] = &cache.Lookup{}
	}
	return nil
}

func (r *recogniser) initializeRequest(stream pb.Recognizer_GetStreamServer) *requestVars {
	return &requestVars{
		snippetCache:       make(map[*pb.Snippet]*cache.Lookup, config.PipelineSize),
		snippetCacheMisses: make([]*pb.Snippet, config.PipelineSize),
		snippetHistory:     []*pb.Snippet{},
		tokenHistory:       []string{},
		stream:             stream,
		pipe:               r.remoteCache.NewGetPipeline(config.PipelineSize),
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
	for _, snippet := range vars.snippetCacheMisses {
		if lookup := vars.snippetCache[snippet]; lookup != nil {
			entity := newEntity(snippet, lookup)
			if err := vars.stream.Send(entity); err != nil {
				return err
			}
		}
	}
	return nil
}

func (r *recogniser) GetStream(stream pb.Recognizer_GetStreamServer) error {
	vars := r.initializeRequest(stream)
	log.Info().Msg("received request")
	onResult := r.newResultHandler(vars)

	for {
		snippet, err := stream.Recv()
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

		compoundSnippets, skip := getCompoundSnippets(vars, snippet)
		if skip {
			continue
		}

		for _, compoundSnippet := range compoundSnippets {
			if err := r.findOrQueueSnippet(vars, compoundSnippet); err != nil {
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
