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
	"regexp"
	"testing"

	"github.com/stretchr/testify/suite"
	"gitlab.mdcatapult.io/informatics/software-engineering/entity-recognition/go/gen/pb"
	"gitlab.mdcatapult.io/informatics/software-engineering/entity-recognition/go/lib/testhelpers"
)

type RecognizerSuite struct {
	suite.Suite
	recogniser
}

func TestRecognizerSuite(t *testing.T) {
	suite.Run(t, new(RecognizerSuite))
}

func (s *RecognizerSuite) Test_recogniser_Recognize() {
	s.recogniser = recogniser{regexps: map[string]*regexp.Regexp{
		"test_regex": regexp.MustCompile("hello"),
	}}
	mockStream := testhelpers.NewMockRecognizeServerStream(testhelpers.CreateSnippets("hello", "my", "name", "is", "jeff")...)
	foundEntity := &pb.Entity{
		Name: "hello",
		Identifiers: map[string]string{
			"test_regex": "hello",
		},
	}
	mockStream.On("Send", foundEntity).Return(nil).Once()
	type args struct {
		stream pb.Recognizer_GetStreamServer
	}
	tests := []struct {
		name    string
		args    args
		wantErr error
	}{
		{
			name:    "happy path",
			args:    args{stream: mockStream},
			wantErr: nil,
		},
	}
	for _, tt := range tests {
		s.T().Log(tt.name)
		gotErr := s.recogniser.GetStream(tt.args.stream)
		s.Equal(tt.wantErr, gotErr)
	}
	mockStream.AssertExpectations(s.T())
}
