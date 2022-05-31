/*
 * Copyright 2022 Medicines Discovery Catapult
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *     http://www.apache.org/licenses/LICENSE-2.0
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package main

import (
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
	mocks "gitlab.mdcatapult.io/informatics/software-engineering/entity-recognition/go/gen/mocks/lib/cache/remote"
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
	s.remoteCache = mockDBClient
	mockGetPipeline := &mocks.GetPipeline{}
	mockDBClient.On("NewGetPipeline", testConfig.PipelineSize).Return(mockGetPipeline).Times(2)
	snippets := testhelpers.CreateSnippets("hello", "my", "name", "is", "jeff")
	mockStream := testhelpers.NewMockRecognizeServerStream(snippets...)
	v := &requestVars{}
	for i, snippet := range snippets {
		compoundTokens, _ := getCompoundSnippets(v, snippet)
		mockGetPipeline.On("Size").Return(i).Once()
		for _, token := range compoundTokens {
			mockGetPipeline.On("Get", token).Once()
		}
	}
	mockGetPipeline.On("Size").Return(len(snippets)).Once()
	mockGetPipeline.On("ExecGet", mock.Anything).Return(nil)

	err := s.GetStream(mockStream)
	s.Nil(err)
	mockDBClient.AssertExpectations(s.T())
	mockGetPipeline.AssertExpectations(s.T())
	mockStream.AssertExpectations(s.T())
}

func (s *RecognizerSuite) Test_recogniser_queryToken() {

	mockDBClient := &mocks.Client{}
	s.remoteCache = mockDBClient
	mockGetPipeline := &mocks.GetPipeline{}
	mockDBClient.On("NewGetPipeline", testConfig.PipelineSize).Return(mockGetPipeline).Once()
	mockStream := testhelpers.NewMockRecognizeServerStream(testhelpers.CreateSnippets("hello", "my", "name", "is", "jeff")...)
	notInDB := &pb.Snippet{
		Text: "not in db",
	}
	inDB := &pb.Snippet{
		Text: "in db",
	}
	// "in cache, not yet queried db (cache miss)"
	cacheMiss := &pb.Snippet{
		Text: "cache miss",
	}
	notInCache := &pb.Snippet{
		Text: "not in cache",
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
	foundEntity := &pb.Entity{
		Recogniser:  "fake dictionary",
		Name:        "in db",
		Identifiers: make(map[string]string),
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
					snippetCache: tokenCache,
				},
				token: notInDB,
			},
			wantErr: nil,
			wantVars: &requestVars{
				snippetCache: tokenCache,
			},
		},
		{
			name: "in cache, not yet queried db (cache miss)",
			args: args{
				vars: &requestVars{
					snippetCache: tokenCache,
				},
				token: cacheMiss,
			},
			wantErr: nil,
			wantVars: &requestVars{
				snippetCache:       tokenCache,
				snippetCacheMisses: []*pb.Snippet{cacheMiss},
			},
		},
		{
			name: "in cache with value",
			args: args{
				vars: &requestVars{
					snippetCache: tokenCache,
					stream:       mockStream,
				},
				token: inDB,
			},
			wantErr: nil,
			wantVars: &requestVars{
				snippetCache: tokenCache,
				stream:       mockStream,
			},
		},
		{
			name: "not in cache",
			args: args{
				vars: &requestVars{
					snippetCache: tokenCache,
					pipeline:     mockGetPipeline,
				},
				token: notInCache,
			},
			wantErr: nil,
			wantVars: &requestVars{
				snippetCache: tokenCacheWithMissingToken,
				pipeline:     mockGetPipeline,
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
					snippetHistory: []*pb.Snippet{},
				},
				token: testhelpers.CreateSnippet("Hello", "hello", 0, ""),
			},
			want: []*pb.Snippet{
				{
					Text:           "Hello",
					NormalisedText: "hello",
				},
			},
			wantVars: &requestVars{
				snippetHistory: []*pb.Snippet{
					{
						Text:           "Hello",
						NormalisedText: "hello",
					},
				},
			},
		},
		{

			name: "detect end of sentence (for current token)",
			args: args{
				vars: &requestVars{
					snippetHistory: testhelpers.CreateSnippets("got"),
				},
				token: testhelpers.CreateSnippet("Hello.", "hello", 0, ""),
			},
			want: []*pb.Snippet{
				{
					Text:           "got Hello.",
					NormalisedText: "got hello",
				},
				{
					Text:           "Hello.",
					NormalisedText: "hello",
				},
			},
			wantVars: &requestVars{
				snippetHistory: []*pb.Snippet{},
			},
		},
		{
			name: "less than compound token length",
			args: args{
				vars: &requestVars{
					snippetHistory: testhelpers.CreateSnippets("old"),
				},
				token: testhelpers.CreateSnippet("new", "new", 0, ""),
			},
			want: testhelpers.CreateSnippets("old new", "new"),
			wantVars: &requestVars{
				snippetHistory: testhelpers.CreateSnippets("old", "new"),
			},
		},
		{
			name: "at compound token length",
			args: args{
				vars: &requestVars{
					snippetHistory: testhelpers.CreateSnippets("old", "new", "black", "white", "quavers"),
				},
				token: testhelpers.CreateSnippet("latest", "latest", 0, ""),
			},
			want: testhelpers.CreateSnippets(
				"new black white quavers latest",
				"black white quavers latest",
				"white quavers latest",
				"quavers latest",
				"latest",
			),
			wantVars: &requestVars{
				snippetHistory: testhelpers.CreateSnippets("new", "black", "white", "quavers", "latest"),
			},
		},
	}
	for i, tt := range tests {
		s.T().Logf("Case %d: %s", i, tt.name)
		got, _ := getCompoundSnippets(tt.args.vars, tt.args.token)
		s.Len(got, len(tt.want))
		for j, snip := range tt.want {
			s.Equal(snip, got[j])
		}

		s.Len(tt.wantVars.snippetHistory, len(tt.args.vars.snippetHistory))
		for j, snip := range tt.wantVars.snippetHistory {
			s.Equal(snip, tt.args.vars.snippetHistory[j])
		}
	}
}
