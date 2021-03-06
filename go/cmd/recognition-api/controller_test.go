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
	"encoding/json"
	"fmt"
	"gitlab.mdcatapult.io/informatics/software-engineering/entity-recognition/go/lib/testhelpers"
	"io"
	"io/ioutil"
	"os"
	"strings"
	"sync"
	"testing"

	"gitlab.mdcatapult.io/informatics/software-engineering/entity-recognition/go/lib/blocklist"
	"gopkg.in/go-playground/assert.v1"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
	mock_recogniser "gitlab.mdcatapult.io/informatics/software-engineering/entity-recognition/go/gen/mocks/lib/recogniser"
	"gitlab.mdcatapult.io/informatics/software-engineering/entity-recognition/go/gen/pb"
	"gitlab.mdcatapult.io/informatics/software-engineering/entity-recognition/go/lib"
	"gitlab.mdcatapult.io/informatics/software-engineering/entity-recognition/go/lib/recogniser"
	snippet_reader "gitlab.mdcatapult.io/informatics/software-engineering/entity-recognition/go/lib/snippet-reader"
	"gitlab.mdcatapult.io/informatics/software-engineering/entity-recognition/go/lib/snippet-reader/html"
)

type ControllerSuite struct {
	suite.Suite
	controller
}

func TestControllerSuite(t *testing.T) {
	suite.Run(t, new(ControllerSuite))
}

func (s *ControllerSuite) SetupSuite() {
	s.htmlReader = html.SnippetReader{}
}

func (s *ControllerSuite) Test_controller_HTMLToText() {
	acetylcarnitineHTML, err := os.Open("../../resources/acetylcarnitine.html")
	s.Require().Nil(err)
	acetylcarnitineRawFile, err := os.Open("../../resources/acetylcarnitine.txt")
	s.Require().Nil(err)
	acetylcarnitineRAWBytes, err := ioutil.ReadAll(acetylcarnitineRawFile)
	s.Require().Nil(err)

	type args struct {
		reader io.Reader
	}
	tests := []struct {
		name    string
		args    args
		want    []byte
		wantErr error
	}{
		{
			name: "acetylcarnitine wiki page",
			args: args{
				acetylcarnitineHTML,
			},
			want:    acetylcarnitineRAWBytes,
			wantErr: nil,
		},
	}
	for _, tt := range tests {
		s.T().Log(tt.name)
		got, gotErr := s.HTMLToText(tt.args.reader)
		s.Equal(string(tt.want), string(got))
		s.Equal(tt.wantErr, gotErr)
	}
}

func (s *ControllerSuite) Test_controller_TokenizeHTML() {
	acetylcarnitineHTML, err := os.Open("../../resources/acetylcarnitine.html")
	s.Require().Nil(err)
	acetylcarnitineTokensFile, err := os.Open("../../resources/acetylcarnitine-tokens.json")
	s.Require().Nil(err)
	acetylcarnitineTokensBytes, err := ioutil.ReadAll(acetylcarnitineTokensFile)
	s.Require().Nil(err)
	var acetylcarnitineTokens []*pb.Snippet
	err = json.Unmarshal(acetylcarnitineTokensBytes, &acetylcarnitineTokens)
	s.Require().Nil(err)

	type args struct {
		reader io.Reader
	}
	tests := []struct {
		name    string
		args    args
		want    []*pb.Snippet
		wantErr error
	}{
		{
			name:    "acetylcarnitine wiki page",
			args:    args{reader: acetylcarnitineHTML},
			want:    acetylcarnitineTokens,
			wantErr: nil,
		},
	}
	for _, tt := range tests {
		s.T().Log(tt.name)
		s.controller.exactMatch = false
		tokens, err := s.controller.Tokenize(tt.args.reader, contentTypeHTML)

		s.Equal(tt.wantErr, err)
		s.Equal(fmt.Sprint(tt.want), fmt.Sprint(tokens))
	}
}

func (s *ControllerSuite) Test_controller_RecognizeInHTML() {
	entity := &pb.Entity{
		Name:        "found entity",
		Position:    2312,
		Recogniser:  "test",
		Identifiers: map[string]string{"many": "", "things": ""},
	}

	blocklistedEntityName := "blocklisted entity"
	blocklistedEntity := &pb.Entity{
		Name:        blocklistedEntityName,
		Position:    1234,
		Recogniser:  "test",
		Identifiers: map[string]string{"blocklisted": "blocklisted"},
	}

	foundEntities := []*pb.Entity{entity, blocklistedEntity}

	sentSnippet := &pb.Snippet{
		Text:   "found entity\n",
		Offset: 3,
		Xpath:  "/p",
	}

	reader := strings.NewReader("<p>found entity</p>")

	//setup global blocklist on controller
	s.controller.blocklist = blocklist.Blocklist{
		CaseSensitive: map[string]bool{},
		CaseInsensitive: map[string]bool{
			blocklistedEntityName: true,
		},
	}

	// The mock recogniser is a little complicated so read carefully!
	mockRecogniser := &mock_recogniser.Client{}
	mockRecogniser.On("SetExactMatch", true).Return()
	s.controller.exactMatch = true

	mockRecogniser.On("Recognise",
		// Expected arguments
		mock.AnythingOfType("<-chan snippet_reader.Value"),
		mock.AnythingOfType("*sync.WaitGroup"),
		lib.HttpOptions{},
	).Return(
		// Don't error.
		nil,
	).Run(func(args mock.Arguments) {
		// The controller blocks until the recognisers receive the snippet_reader.Values that are being read from the
		// html reader. This is because we are using unbuffered channels. Therefore, our mocked recogniser needs to
		// listen on the channel and receive the values. We also need to listen for these messages in a separate
		// goroutine so that our replacement function doesn't block before read the html! While reading, use this
		// opportunity to make assertions about the snippets that are being sent.
		go func() {
			waitGroup := args[1].(*sync.WaitGroup)
			waitGroup.Add(1)
			err := snippet_reader.ReadChannelWithCallback(args[0].(<-chan snippet_reader.Value), func(snip *pb.Snippet) error {
				s.Equal(sentSnippet, snip)
				return nil
			})
			s.Nil(err)
			waitGroup.Done()
		}()
	})
	mockRecogniser.On("Err").Return(nil)
	mockRecogniser.On("Result").Return(foundEntities)
	s.controller.recognisers = map[string]recogniser.Client{"mock": mockRecogniser}

	opts := []lib.RecogniserOptions{{Name: "mock"}}
	entities, err := s.controller.Recognize(reader, contentTypeHTML, opts)

	// entity should have been found
	s.Equal(testhelpers.APIEntityFromEntity(foundEntities[0]), entities[0])

	// entities should only contain the found entity, not the blocklisted entity
	s.Len(entities, 1)
	s.Nil(err)
}

func TestFilterUniqueEntities(t *testing.T) {

	input := []*pb.Entity{
		{
			Name:     "A",
			Position: 1,
			Xpath:    "<html>",
		},
		{
			Name:     "A",
			Position: 2,
			Xpath:    "<html>[1]",
		},
		{
			Name:     "B",
			Position: 3,
			Xpath:    "<html>",
		},
	}

	expected := []lib.APIEntity{
		{
			Name: "A",
			Positions: []lib.Position{
				{
					Position: 1,
					Xpath:    "<html>",
				},
				{
					Position: 2,
					Xpath:    "<html>[1]",
				},
			},
		},
		{
			Name: "B",
			Positions: []lib.Position{
				{Position: 3,
					Xpath: "<html>",
				},
			},
		},
	}

	actual := filterUniqueEntities(input)

	assert.Equal(t, expected, actual)

}
