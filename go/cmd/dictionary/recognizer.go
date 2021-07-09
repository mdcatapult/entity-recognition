package main

import (
	"errors"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
	"gitlab.mdcatapult.io/informatics/software-engineering/entity-recognition/go/gen/pb"
	"gitlab.mdcatapult.io/informatics/software-engineering/entity-recognition/go/lib"
	"gitlab.mdcatapult.io/informatics/software-engineering/entity-recognition/go/lib/db"
	"io"
	"strings"
	"sync"
)

type recogniser struct {
	pb.UnimplementedRecognizerServer
	dbClient db.Client
	requestCache map[uuid.UUID]*requestVars
	rwmut sync.RWMutex
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

func (r *recogniser) newResultHandler(requestID uuid.UUID) func(snippet *pb.Snippet, lookup *db.Lookup) error {
	return func(snippet *pb.Snippet, lookup *db.Lookup) error {
		r.rwmut.RLock()
		requestVars, ok := r.requestCache[requestID]
		r.rwmut.RUnlock()
		if !ok {
			err := errors.New("request not in cache, something went horribly wrong")
			log.Error().Err(err).Send()
			return err
		}
		requestVars.tokenCache[snippet] = lookup
		if lookup == nil {
			return nil
		}
		entity := &pb.RecognizedEntity{
			Entity:     string(snippet.GetData()),
			Position:   snippet.GetOffset(),
			Type:       lookup.Dictionary,
			ResolvedTo: lookup.ResolvedEntities,
		}
		err := requestVars.stream.Send(entity)
		if err != nil {
			return err
		}

		return nil
	}
}

func (r *recogniser) getCompoundTokens(requestID uuid.UUID, token *pb.Snippet) ([]*pb.Snippet, error) {
	r.rwmut.RLock()
	requestVars, ok := r.requestCache[requestID]
	r.rwmut.RUnlock()
	if !ok {
		err := errors.New("request not in cache, something went horribly wrong")
		log.Error().Err(err).Send()
		return nil, err
	}
	// If sentenceEnd is true, we can save some redis queries by resetting the token history..
	if requestVars.sentenceEnd {
		requestVars.tokenHistory = []*pb.Snippet{}
		requestVars.keyHistory = []string{}
		requestVars.sentenceEnd = false
	}

	// normalise the token (remove enclosing punctuation and enforce NFKC encoding).
	// sentenceEnd is true if the last byte in the token is one of '.', '?', or '!'.
	requestVars.sentenceEnd = lib.Normalize(token)

	// manage the token history
	if len(requestVars.tokenHistory) < config.CompoundTokenLength {
		requestVars.tokenHistory = append(requestVars.tokenHistory, token)
		requestVars.keyHistory = append(requestVars.keyHistory, string(token.GetData()))
	} else {
		requestVars.tokenHistory = append(requestVars.tokenHistory[1:], token)
		requestVars.keyHistory = append(requestVars.keyHistory[1:], string(token.GetData()))
	}

	// construct the compound tokens to query against redis.
	queryTokens := make([]*pb.Snippet, len(requestVars.tokenHistory))
	for i, historicalToken := range requestVars.tokenHistory {
		queryTokens[i] = &pb.Snippet{
			Data:   []byte(strings.Join(requestVars.keyHistory[i:], " ")),
			Offset: historicalToken.GetOffset(),
		}
	}
	return queryTokens, nil
}

func (r *recogniser) queryCompoundTokens(requestID uuid.UUID, compoundTokens []*pb.Snippet) error {
	r.rwmut.RLock()
	requestVars, ok := r.requestCache[requestID]
	r.rwmut.RUnlock()
	if !ok {
		err := errors.New("request not in cache, something went horribly wrong")
		log.Error().Err(err).Send()
		return err
	}

	for _, compoundToken := range compoundTokens {
		if lookup, ok := requestVars.tokenCache[compoundToken]; ok {
			// if it's nil, we've already queried redis and it wasn't there
			if lookup == nil {
				continue
			}
			// If it's empty, it's already queued but we don't know if its there or not.
			// Append it to the cacheMisses to be found later.
			if lookup.Dictionary == "" {
				requestVars.tokenCacheMisses = append(requestVars.tokenCacheMisses, compoundToken)
				continue
			}
			// Otherwise, construct an entity from the cache value and send it back to the caller.
			entity := &pb.RecognizedEntity{
				Entity:     string(compoundToken.GetData()),
				Position:   compoundToken.GetOffset(),
				Type:       lookup.Dictionary,
				ResolvedTo: lookup.ResolvedEntities,
			}
			if err := requestVars.stream.Send(entity); err != nil {
				return err
			}
		} else {
			// Not in local cache.
			// Queue the redis "GET" in the pipe and set the cache value to an empty db.Lookup
			// (so that future equivalent tokens will be a cache miss).
			requestVars.pipe.Get(compoundToken)
			requestVars.tokenCache[compoundToken] = &db.Lookup{}
		}
	}
	return nil
}

func (r *recogniser) initializeRequest(stream pb.Recognizer_RecognizeServer) uuid.UUID {
	requestID := uuid.New()
	r.rwmut.Lock()
	r.requestCache[requestID] = &requestVars{
		tokenCache:       make(map[*pb.Snippet]*db.Lookup, config.PipelineSize),
		tokenCacheMisses: make([]*pb.Snippet, config.PipelineSize),
		tokenHistory:     []*pb.Snippet{},
		keyHistory:       []string{},
		sentenceEnd:      false,
		stream:           stream,
		pipe: r.dbClient.NewGetPipeline(config.PipelineSize),
	}
	r.rwmut.Unlock()
	return requestID
}

func (r *recogniser) execPipe(requestID uuid.UUID, onResult func(snippet *pb.Snippet, lookup *db.Lookup) error, threshold int, new bool) error {
	r.rwmut.RLock()
	requestVars, ok := r.requestCache[requestID]
	r.rwmut.RUnlock()
	if !ok {
		err := errors.New("request not in cache, something went horribly wrong")
		log.Error().Err(err).Send()
		return err
	}

	if requestVars.pipe.Size() > threshold {
		if err := requestVars.pipe.ExecGet(onResult); err != nil {
			return err
		}
		if new {
			requestVars.pipe = r.dbClient.NewGetPipeline(config.PipelineSize)
		}
	}
	return nil
}

func (r *recogniser) retryCacheMisses(requestID uuid.UUID) error {
	r.rwmut.RLock()
	requestVars, ok := r.requestCache[requestID]
	r.rwmut.RUnlock()
	if !ok {
		err := errors.New("request not in cache, something went horribly wrong")
		log.Error().Err(err).Send()
		return err
	}

	// Check if any of the cacheMisses were populated (nil means redis doesnt have it).
	for _, token := range requestVars.tokenCacheMisses {
		if lookup := requestVars.tokenCache[token]; lookup != nil {
			entity := &pb.RecognizedEntity{
				Entity:     string(token.GetData()),
				Position:   token.GetOffset(),
				Type:       lookup.Dictionary,
				ResolvedTo: lookup.ResolvedEntities,
			}
			if err := requestVars.stream.Send(entity); err != nil {
				return err
			}
		}
	}
	return nil
}

func (r *recogniser) Recognize(stream pb.Recognizer_RecognizeServer) error {
	requestID := r.initializeRequest(stream)
	log.Info().Str("request_id", requestID.String()).Msg("received request")
	onResult := r.newResultHandler(requestID)

	for {
		token, err := stream.Recv()
		if err == io.EOF {
			// There are likely some redis queries queued on the pipe. If there are, execute them. Then break.
			if err := r.execPipe(requestID, onResult, 0, false); err != nil {
				return err
			}
			break
		} else if err != nil {
			return err
		}

		compoundTokens, err := r.getCompoundTokens(requestID, token)
		if err != nil {
			return err
		}

		if err := r.queryCompoundTokens(requestID, compoundTokens); err != nil {
			return err
		}

		if err := r.execPipe(requestID, onResult, config.PipelineSize, true); err != nil {
			return err
		}
	}

	return nil
}
