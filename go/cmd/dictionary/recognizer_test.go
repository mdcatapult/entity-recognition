package main

import (
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
	"gitlab.mdcatapult.io/informatics/software-engineering/entity-recognition/go/gen/mocks"
	"gitlab.mdcatapult.io/informatics/software-engineering/entity-recognition/go/gen/pb"
	"gitlab.mdcatapult.io/informatics/software-engineering/entity-recognition/go/lib/db"
	"io"
	"math/rand"
	"testing"
	"time"
)

var testConfig = dictionaryRecogniserConfig{
	PipelineSize:        100,
	CompoundTokenLength: 5,
}

type RecognizerSuite struct {
	recogniser
	suite.Suite
}

func TestRecognizerSuite(t *testing.T) {
	suite.Run(t, new(RecognizerSuite))
}

func (s *RecognizerSuite) Test_recognizer_Recognize() {
	mockDBClient := &mocks.Client{}
	s.dbClient = mockDBClient
	mockGetPipeline := &mocks.GetPipeline{}
	mockDBClient.On("NewGetPipeline", testConfig.PipelineSize).Return(mockGetPipeline).Once()
	mockStream, snippets := newMockStream("hello", "my", "name", "is", "jeff")
	v := &requestVars{}
	for i, snippet := range snippets {
		compoundTokens := s.getCompoundTokens(v, snippet)
		mockGetPipeline.On("Size").Return(i).Once()
		for _, token := range compoundTokens {
			mockGetPipeline.On("Get", token).Once()
		}
	}
	mockGetPipeline.On("Size").Return(len(snippets)).Once()
	mockGetPipeline.On("ExecGet", mock.Anything).Return(nil)

	err := s.Recognize(mockStream)
	s.Nil(err)
	mockDBClient.AssertExpectations(s.T())
	mockGetPipeline.AssertExpectations(s.T())
	mockStream.AssertExpectations(s.T())
}

func newMockStream(tokens... string) (*mocks.Recognizer_RecognizeServer, []*pb.Snippet) {
	stream := &mocks.Recognizer_RecognizeServer{}
	rand.Seed(time.Now().UnixNano())
	offset := rand.Intn(1000)
	var snippets []*pb.Snippet
	for _, token := range tokens {
		offset += len(token) + 1
		snippet := &pb.Snippet{
			Data:   []byte(token),
			Offset: uint32(offset),
		}
		snippets = append(snippets, snippet)
		stream.On("Recv").Return(snippet, nil).Once()
	}
	stream.On("Recv").Return(nil, io.EOF).Once()
	return stream, snippets
}

func (s *RecognizerSuite) Test_recogniser_queryToken() {

	mockDBClient := &mocks.Client{}
	s.dbClient = mockDBClient
	mockGetPipeline := &mocks.GetPipeline{}
	mockDBClient.On("NewGetPipeline", testConfig.PipelineSize).Return(mockGetPipeline).Once()
	mockStream, _ := newMockStream("hello", "my", "name", "is", "jeff")
	notInDB := &pb.Snippet{
		Data: []byte("not in db"),
	}
	inDB := &pb.Snippet{
		Data: []byte("in db"),
	}
	cacheMiss := &pb.Snippet{
		Data: []byte("cache miss"),
	}
	notInCache := &pb.Snippet{
		Data: []byte("not in cache"),
	}
	tokenCache := map[*pb.Snippet]*db.Lookup{
		notInDB: nil,
		cacheMiss: {},
		inDB: {
			Dictionary: "fake dictionary",
		},
	}
	tokenCacheWithMissingToken := make(map[*pb.Snippet]*db.Lookup)
	for k,v := range tokenCache {
		tokenCacheWithMissingToken[k] = v
	}
	tokenCacheWithMissingToken[notInCache] = &db.Lookup{}
	foundEntity := &pb.RecognizedEntity{
		Type:       "fake dictionary",
		Entity: "in db",
	}
	mockStream.On("Send", foundEntity).Return(nil).Once()
	mockGetPipeline.On("Get", notInCache).Once()
	type args struct {
		vars  *requestVars
		token *pb.Snippet
	}
	tests := []struct {
		name    string
		args    args
		wantErr error
		wantVars *requestVars
	}{
		{
			name: "in cache, not in db",
			args: args{
				vars:  &requestVars{
					tokenCache:       tokenCache,
				},
				token: notInDB,
			},
			wantErr: nil,
			wantVars: &requestVars{
				tokenCache:       tokenCache,
			},
		},
		{
			name: "in cache, not yet queried db (cache miss)",
			args: args{
				vars: &requestVars{
					tokenCache: tokenCache,
				},
				token: cacheMiss,
			},
			wantErr: nil,
			wantVars: &requestVars{
				tokenCache: tokenCache,
				tokenCacheMisses: []*pb.Snippet{cacheMiss},
			},
		},
		{
			name: "in cache with value",
			args: args{
				vars: &requestVars{
					tokenCache: tokenCache,
					stream: mockStream,
				},
				token: inDB,
			},
			wantErr: nil,
			wantVars: &requestVars{
				tokenCache: tokenCache,
				stream: mockStream,
			},
		},
		{
			name: "not in cache",
			args: args{
				vars: &requestVars{
					tokenCache: tokenCache,
					pipe: mockGetPipeline,
				},
				token: notInCache,
			},
			wantErr: nil,
			wantVars: &requestVars{
				tokenCache: tokenCacheWithMissingToken,
				pipe: mockGetPipeline,
			},
		},
	}
	for _, tt := range tests {
		s.T().Log(tt.name)
		gotErr := s.queryToken(tt.args.vars, tt.args.token)
		s.Equal(tt.wantErr, gotErr)
		s.Equal(tt.wantVars, tt.args.vars)
	}
}

func (s *RecognizerSuite) Test_recogniser_getCompoundTokens() {
	type args struct {
		vars  *requestVars
		token *pb.Snippet
	}
	tests := []struct {
		name    string
		args    args
		want    []*pb.Snippet
		wantVars *requestVars
	}{
		{
			name: "end of sentence (for last token)",
			args: args{
				vars: &requestVars{
					tokenHistory:     snips("got", "stuff", "in", "it."),
					keyHistory:       []string{"got", "stuff", "in", "it."},
					sentenceEnd:      true,
				},
				token: snip("Hello"),
			},
			want: snips("Hello"),
			wantVars: &requestVars{
				tokenHistory:     snips("Hello"),
				keyHistory:       []string{"Hello"},
				sentenceEnd:      false,
			},
		},
		{
			name: "detect end of sentence (for current token)",
			args: args{
				vars: &requestVars{
					tokenHistory:     snips("got"),
					keyHistory:       []string{"got"},
				},
				token: snip("Hello."),
			},
			want: snips("Hello", "got Hello"),
			wantVars: &requestVars{
				tokenHistory:     snips("got", "Hello"),
				keyHistory:       []string{"got", "Hello"},
				sentenceEnd:      true,
			},
		},
		{
			name: "less than compound token length",
			args: args{
				vars: &requestVars{
					keyHistory: []string{"old"},
					tokenHistory: snips("old"),
				},
				token: snip("new"),
			},
			want: snips("old new", "new"),
			wantVars: &requestVars{
				keyHistory: []string{"old", "new"},
				tokenHistory: snips("old", "new"),
			},
		},
		{
			name: "at compound token length",
			args: args{
				vars: &requestVars{
					keyHistory: []string{"old", "new", "black", "white", "quavers"},
					tokenHistory: snips("old", "new", "black", "white", "quavers"),
				},
				token: snip("latest"),
			},
			want: snips("latest", 
				"quavers latest",
				"white quavers latest",
				"black white quavers latest",
				"new black white quavers latest"),
			wantVars: &requestVars{
				tokenHistory:     snips("new", "black", "white", "quavers", "latest"),
				keyHistory:       []string{"new", "black", "white", "quavers", "latest"},
			},
		},
	}
		for _, tt := range tests {
			s.T().Log(tt.name)
			got := s.getCompoundTokens(tt.args.vars, tt.args.token)
			s.ElementsMatch(tt.want, got)
			s.ElementsMatch(tt.args.vars.tokenHistory, tt.wantVars.tokenHistory)
			s.ElementsMatch(tt.args.vars.keyHistory, tt.wantVars.keyHistory)
			s.Equal(tt.args.vars.sentenceEnd, tt.wantVars.sentenceEnd)
	}
}

func snips(toks... string) []*pb.Snippet {
	snippets := make([]*pb.Snippet, len(toks))
	for i, tok := range toks {
		snippets[i] = snip(tok)
	}
	return snippets
}

func snip(tok string) *pb.Snippet {
	return &pb.Snippet{
		Data:   []byte(tok),
	}
}