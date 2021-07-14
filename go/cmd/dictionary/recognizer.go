package main

import (
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
	"gitlab.mdcatapult.io/informatics/software-engineering/entity-recognition/go/gen/pb"
	"gitlab.mdcatapult.io/informatics/software-engineering/entity-recognition/go/lib"
	"gitlab.mdcatapult.io/informatics/software-engineering/entity-recognition/go/lib/db"
	"io"
	"strings"
)

type recogniser struct {
	pb.UnimplementedRecognizerServer
	dbClient db.Client
	requestCache map[uuid.UUID]*requestVars
}

type requestVars struct {
	tokenCache map[*pb.Snippet]*db.Lookup
	tokenCacheMisses []*pb.Snippet
	tokenHistory []*pb.Snippet
	keyHistory []string
	sentenceEnd bool
	stream pb.Recognizer_RecognizeServer
	pipe db.GetPipeline
}

func (r *recogniser) newResultHandler(vars *requestVars) func(snippet *pb.Snippet, lookup *db.Lookup) error {
	return func(snippet *pb.Snippet, lookup *db.Lookup) error {
		vars.tokenCache[snippet] = lookup
		if lookup == nil {
			return nil
		}
		entity := &pb.RecognizedEntity{
			Entity:     string(snippet.GetData()),
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

func (r *recogniser) getCompoundTokens(vars *requestVars, token *pb.Snippet) ([]*pb.Snippet, error) {
	// If sentenceEnd is true, we can save some redis queries by resetting the token history..
	if vars.sentenceEnd {
		vars.tokenHistory = []*pb.Snippet{}
		vars.keyHistory = []string{}
		vars.sentenceEnd = false
	}

	// normalise the token (remove enclosing punctuation and enforce NFKC encoding).
	// sentenceEnd is true if the last byte in the token is one of '.', '?', or '!'.
	vars.sentenceEnd = lib.Normalize(token)

	// manage the token history
	if len(vars.tokenHistory) < config.CompoundTokenLength {
		vars.tokenHistory = append(vars.tokenHistory, token)
		vars.keyHistory = append(vars.keyHistory, string(token.GetData()))
	} else {
		vars.tokenHistory = append(vars.tokenHistory[1:], token)
		vars.keyHistory = append(vars.keyHistory[1:], string(token.GetData()))
	}

	// construct the compound tokens to query against redis.
	queryTokens := make([]*pb.Snippet, len(vars.tokenHistory))
	for i, historicalToken := range vars.tokenHistory {
		queryTokens[i] = &pb.Snippet{
			Data:   []byte(strings.Join(vars.keyHistory[i:], " ")),
			Offset: historicalToken.GetOffset(),
		}
	}
	return queryTokens, nil
}

func (r *recogniser) queryToken(vars *requestVars, token *pb.Snippet) error {
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
			Entity:     string(token.GetData()),
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
		vars.tokenCache[token] = &db.Lookup{}
	}
	return nil
}

func (r *recogniser) initializeRequest(stream pb.Recognizer_RecognizeServer) *requestVars {
	return &requestVars{
		tokenCache:       make(map[*pb.Snippet]*db.Lookup, config.PipelineSize),
		tokenCacheMisses: make([]*pb.Snippet, config.PipelineSize),
		tokenHistory:     []*pb.Snippet{},
		keyHistory:       []string{},
		sentenceEnd:      false,
		stream:           stream,
		pipe: r.dbClient.NewGetPipeline(config.PipelineSize),
	}
}

func (r *recogniser) execPipe(vars *requestVars, onResult func(snippet *pb.Snippet, lookup *db.Lookup) error, threshold int, new bool) error {
	if vars.pipe.Size() > threshold {
		if err := vars.pipe.ExecGet(onResult); err != nil {
			return err
		}
		if new {
			vars.pipe = r.dbClient.NewGetPipeline(config.PipelineSize)
		}
	}
	return nil
}

func (r *recogniser) retryCacheMisses(vars *requestVars) error {
	for _, token := range vars.tokenCacheMisses {
		if lookup := vars.tokenCache[token]; lookup != nil {
			entity := &pb.RecognizedEntity{
				Entity:     string(token.GetData()),
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
			// There are likely some redis queries queued on the pipe. If there are, execute them. Then break.
			if err := r.execPipe(vars, onResult, 0, false); err != nil {
				return err
			}
			break
		} else if err != nil {
			return err
		}

		compoundTokens, err := r.getCompoundTokens(vars, token)
		if err != nil {
			return err
		}

		for _, compoundToken := range compoundTokens {
			if err := r.queryToken(vars, compoundToken); err != nil {
				return err
			}
		}

		if err := r.execPipe(vars, onResult, config.PipelineSize, true); err != nil {
			return err
		}
	}

	return r.retryCacheMisses(vars)
}
