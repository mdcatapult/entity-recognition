package main

import (
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
	"gitlab.mdcatapult.io/informatics/software-engineering/entity-recognition/go/gen/mocks"
	"gitlab.mdcatapult.io/informatics/software-engineering/entity-recognition/go/gen/pb"
	"gitlab.mdcatapult.io/informatics/software-engineering/entity-recognition/go/lib/cache"
	"gitlab.mdcatapult.io/informatics/software-engineering/entity-recognition/go/lib/testhelpers"
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

func (s *RecognizerSuite) SetupSuite() {
	config = testConfig
}

func (s *RecognizerSuite) Test_recognizer_Recognize() {
	mockDBClient := &mocks.Client{}
	s.dbClient = mockDBClient
	mockGetPipeline := &mocks.GetPipeline{}
	mockDBClient.On("NewGetPipeline", testConfig.PipelineSize).Return(mockGetPipeline).Times(2)
	mockStream, snippets := testhelpers.NewMockRecognizeServerStream("hello", "my", "name", "is", "jeff")
	v := &requestVars{}
	for i, snippet := range snippets {
		compoundTokens := s.getCompoundSnippets(v, snippet)
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

func (s *RecognizerSuite) Test_recogniser_queryToken() {

	mockDBClient := &mocks.Client{}
	s.dbClient = mockDBClient
	mockGetPipeline := &mocks.GetPipeline{}
	mockDBClient.On("NewGetPipeline", testConfig.PipelineSize).Return(mockGetPipeline).Once()
	mockStream, _ := testhelpers.NewMockRecognizeServerStream("hello", "my", "name", "is", "jeff")
	notInDB := &pb.Snippet{
		Token: "not in db",
	}
	inDB := &pb.Snippet{
		Token: "in db",
	}
	cacheMiss := &pb.Snippet{
		Token: "cache miss",
	}
	notInCache := &pb.Snippet{
		Token: "not in cache",
	}
	tokenCache := map[*pb.Snippet]*cache.Lookup{
		notInDB:   nil,
		cacheMiss: {},
		inDB: {
			Dictionary: "fake dictionary",
		},
	}
	tokenCacheWithMissingToken := make(map[*pb.Snippet]*cache.Lookup)
	for k, v := range tokenCache {
		tokenCacheWithMissingToken[k] = v
	}
	tokenCacheWithMissingToken[notInCache] = &cache.Lookup{}
	foundEntity := &pb.RecognizedEntity{
		Type:   "fake dictionary",
		Entity: "in db",
	}
	mockStream.On("Send", foundEntity).Return(nil).Once()
	mockGetPipeline.On("Get", notInCache).Once()
	type args struct {
		vars  *requestVars
		token *pb.Snippet
	}
	tests := []struct {
		name     string
		args     args
		wantErr  error
		wantVars *requestVars
	}{
		{
			name: "in cache, not in db",
			args: args{
				vars: &requestVars{
					tokenCache: tokenCache,
				},
				token: notInDB,
			},
			wantErr: nil,
			wantVars: &requestVars{
				tokenCache: tokenCache,
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
				tokenCache:       tokenCache,
				tokenCacheMisses: []*pb.Snippet{cacheMiss},
			},
		},
		{
			name: "in cache with value",
			args: args{
				vars: &requestVars{
					tokenCache: tokenCache,
					stream:     mockStream,
				},
				token: inDB,
			},
			wantErr: nil,
			wantVars: &requestVars{
				tokenCache: tokenCache,
				stream:     mockStream,
			},
		},
		{
			name: "not in cache",
			args: args{
				vars: &requestVars{
					tokenCache: tokenCache,
					pipe:       mockGetPipeline,
				},
				token: notInCache,
			},
			wantErr: nil,
			wantVars: &requestVars{
				tokenCache: tokenCacheWithMissingToken,
				pipe:       mockGetPipeline,
			},
		},
	}
	for _, tt := range tests {
		s.T().Log(tt.name)
		gotErr := s.findOrQueueSnippet(tt.args.vars, tt.args.token)
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
		name     string
		args     args
		want     []*pb.Snippet
		wantVars *requestVars
	}{
		{
			name: "end of sentence (for last token)",
			args: args{
				vars: &requestVars{
					snippetHistory: testhelpers.Snips("got", "stuff", "in", "it."),
					tokenHistory:   []string{"got", "stuff", "in", "it."},
					sentenceEnd:    true,
				},
				token: testhelpers.Snip("Hello", 0),
			},
			want: testhelpers.Snips("Hello"),
			wantVars: &requestVars{
				snippetHistory: testhelpers.Snips("Hello"),
				tokenHistory:   []string{"Hello"},
				sentenceEnd:    false,
			},
		},
		{
			name: "detect end of sentence (for current token)",
			args: args{
				vars: &requestVars{
					snippetHistory: testhelpers.Snips("got"),
					tokenHistory:   []string{"got"},
				},
				token: testhelpers.Snip("Hello.", 0),
			},
			want: testhelpers.Snips("Hello", "got Hello"),
			wantVars: &requestVars{
				snippetHistory: testhelpers.Snips("got", "Hello"),
				tokenHistory:   []string{"got", "Hello"},
				sentenceEnd:    true,
			},
		},
		{
			name: "less than compound token length",
			args: args{
				vars: &requestVars{
					tokenHistory:   []string{"old"},
					snippetHistory: testhelpers.Snips("old"),
				},
				token: testhelpers.Snip("new", 0),
			},
			want: testhelpers.Snips("old new", "new"),
			wantVars: &requestVars{
				tokenHistory:   []string{"old", "new"},
				snippetHistory: testhelpers.Snips("old", "new"),
			},
		},
		{
			name: "at compound token length",
			args: args{
				vars: &requestVars{
					tokenHistory:   []string{"old", "new", "black", "white", "quavers"},
					snippetHistory: testhelpers.Snips("old", "new", "black", "white", "quavers"),
				},
				token: testhelpers.Snip("latest", 0),
			},
			want: testhelpers.Snips("latest",
				"quavers latest",
				"white quavers latest",
				"black white quavers latest",
				"new black white quavers latest"),
			wantVars: &requestVars{
				snippetHistory: testhelpers.Snips("new", "black", "white", "quavers", "latest"),
				tokenHistory:   []string{"new", "black", "white", "quavers", "latest"},
			},
		},
	}
	for _, tt := range tests {
		s.T().Log(tt.name)
		got := s.getCompoundSnippets(tt.args.vars, tt.args.token)
		s.ElementsMatch(tt.want, got)
		s.ElementsMatch(tt.args.vars.snippetHistory, tt.wantVars.snippetHistory)
		s.ElementsMatch(tt.args.vars.tokenHistory, tt.wantVars.tokenHistory)
		s.Equal(tt.args.vars.sentenceEnd, tt.wantVars.sentenceEnd)
	}
}
