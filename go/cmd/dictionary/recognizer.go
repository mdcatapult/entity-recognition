package main

import (
	"encoding/json"
	"fmt"
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
	stream             pb.Recognizer_GetStreamServer
	pipeline           remote.GetPipeline
}

func newEntityWithNormalisedText(snippet *pb.Snippet, lookup *cache.Lookup) *pb.Entity {

	normalisedText, _, _ := text.NormalizeString(snippet.GetText())

	return &pb.Entity{
		Name:        normalisedText,
		Position:    snippet.GetOffset(),
		Recogniser:  lookup.Dictionary,
		Identifiers: convertIdentifiers(lookup.Identifiers),
		Xpath:       snippet.GetXpath(),
		Metadata:    string(lookup.Metadata),
	}
}

func (recogniser *recogniser) newResultHandler(vars *requestVars) func(snippet *pb.Snippet, lookup *cache.Lookup) error {
	return func(snippet *pb.Snippet, lookup *cache.Lookup) error {
		vars.snippetCache[snippet] = lookup
		if lookup == nil {
			return nil
		}
		entity := newEntityWithNormalisedText(snippet, lookup)
		if err := vars.stream.Send(entity); err != nil {
			return err
		}

		return nil
	}
}

func joinSnippets(snips []*pb.Snippet) (originalText, normalisedText string) {
	for _, snip := range snips {
		originalText += snip.Text + " "
		normalisedText += snip.NormalisedText + " "
	}
	return strings.TrimRight(originalText, " "), strings.TrimRight(normalisedText, " ")
}

func getCompoundSnippets(vars *requestVars, snippet *pb.Snippet) (snippets []*pb.Snippet, skipToken bool) {
	// normalise the token (remove enclosing punctuation and enforce NFKC encoding).
	// compoundTokenEnd is true if the last byte in the token is one of '.', '?', or '!'.
	compoundTokenEnd := text.NormalizeAndLowercaseSnippet(snippet)
	if len(snippet.NormalisedText) == 0 {
		return nil, true
	}

	// manage the token history
	if len(vars.snippetHistory) < config.CompoundTokenLength {
		vars.snippetHistory = append(vars.snippetHistory, snippet)
	} else {
		vars.snippetHistory = append(vars.snippetHistory[1:], snippet)
	}

	// construct the compound tokens to query against redis.
	snippets = make([]*pb.Snippet, len(vars.snippetHistory))
	for i, historicalSnippet := range vars.snippetHistory {
		originalText, normalisedText := joinSnippets(vars.snippetHistory[i:])
		snippets[i] = &pb.Snippet{
			Text:           originalText,
			NormalisedText: normalisedText,
			Offset:         historicalSnippet.GetOffset(),
			Xpath:          historicalSnippet.GetXpath(),
		}
	}

	// If compoundTokenEnd is true, we can save some redis queries by resetting the token history.
	if compoundTokenEnd {
		vars.snippetHistory = []*pb.Snippet{}
	}

	return snippets, false
}

func (recogniser *recogniser) findOrQueueSnippet(vars *requestVars, snippet *pb.Snippet) error {
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
		entity := newEntityWithNormalisedText(snippet, lookup)
		if err := vars.stream.Send(entity); err != nil {
			return err
		}
	} else {
		// Not in local cache.
		// Queue the redis "GET" in the pipeline and set the cache value to an empty db.Lookup
		// (so that future equivalent tokens will be a cache miss).
		vars.pipeline.Get(snippet)
		vars.snippetCache[snippet] = &cache.Lookup{}
	}
	return nil
}

func (recogniser *recogniser) initializeRequest(stream pb.Recognizer_GetStreamServer) *requestVars {
	return &requestVars{
		snippetCache:       make(map[*pb.Snippet]*cache.Lookup, config.PipelineSize),
		snippetCacheMisses: make([]*pb.Snippet, config.PipelineSize),
		snippetHistory:     []*pb.Snippet{},
		stream:             stream,
		pipeline:           recogniser.remoteCache.NewGetPipeline(config.PipelineSize),
	}
}

func (recogniser *recogniser) runPipeline(vars *requestVars, onResult func(snippet *pb.Snippet, lookup *cache.Lookup) error) error {
	if err := vars.pipeline.ExecGet(onResult); err != nil {
		return err
	}
	vars.pipeline = recogniser.remoteCache.NewGetPipeline(config.PipelineSize)
	return nil
}

func (recogniser *recogniser) retryCacheMisses(vars *requestVars) error {
	for _, snippet := range vars.snippetCacheMisses {
		if lookup := vars.snippetCache[snippet]; lookup != nil {
			entity := newEntityWithNormalisedText(snippet, lookup)
			if err := vars.stream.Send(entity); err != nil {
				return err
			}
		}
	}
	return nil
}

func (recogniser *recogniser) GetStream(stream pb.Recognizer_GetStreamServer) error {
	vars := recogniser.initializeRequest(stream)
	log.Info().Msg("received request")
	onResult := recogniser.newResultHandler(vars)

	for {
		snippet, err := stream.Recv()
		if err == io.EOF {
			// Number of tokens is unlikely to be a multiple of the pipeline size. There will still be tokens on the
			// pipeline. Execute it now, then break.
			if vars.pipeline.Size() > 0 {
				if err := recogniser.runPipeline(vars, onResult); err != nil {
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
			if err = recogniser.findOrQueueSnippet(vars, compoundSnippet); err != nil {
				return err
			}
		}

		if vars.pipeline.Size() > config.PipelineSize {
			if err = recogniser.runPipeline(vars, onResult); err != nil {
				return err
			}
		}
	}

	return recogniser.retryCacheMisses(vars)
}

func convertIdentifiers(identifiers map[string]interface{}) map[string]string {
	res := make(map[string]string, len(identifiers))
	for k := range identifiers {
		jsonIdentifierBytes, err := json.Marshal(identifiers[k])

		if err != nil {
			fmt.Println("failed to serialize entry", identifiers[k], err)
		}
		res[k] = string(jsonIdentifierBytes)
	}

	return res
}
