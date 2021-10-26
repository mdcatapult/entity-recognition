package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/stretchr/testify/mock"
	mock_recogniser "gitlab.mdcatapult.io/informatics/software-engineering/entity-recognition/go/gen/mocks/lib/recogniser"
	mock_snippet_reader "gitlab.mdcatapult.io/informatics/software-engineering/entity-recognition/go/gen/mocks/lib/snippet-reader"
	"gitlab.mdcatapult.io/informatics/software-engineering/entity-recognition/go/lib"
	"gitlab.mdcatapult.io/informatics/software-engineering/entity-recognition/go/lib/recogniser"
	"gitlab.mdcatapult.io/informatics/software-engineering/entity-recognition/go/lib/snippet-reader/html"
	"io"
	"io/ioutil"
	"os"
	"testing"

	"github.com/stretchr/testify/suite"
	"gitlab.mdcatapult.io/informatics/software-engineering/entity-recognition/go/gen/pb"
)

type ControllerSuite struct {
	suite.Suite
	controller
}

func TestControllerSuite(t *testing.T) {
	suite.Run(t, new(ControllerSuite))
}

func (s *ControllerSuite) Test_controller_HTMLToText() {
	acetylcarnitineHTML, err := os.Open("../../resources/acetylcarnitine.html")
	s.Require().Nil(err)
	acetylcarnitineRawFile, err := os.Open("../../resources/acetylcarnitine.txt")
	s.Require().Nil(err)
	acetylcarnitineRAWBytes, err := ioutil.ReadAll(acetylcarnitineRawFile)
	s.Require().Nil(err)
	s.html = html.SnippetReader{}

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
	s.html = html.SnippetReader{}

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
		tokens, err := s.controller.TokenizeHTML(tt.args.reader)
		s.Equal(tt.wantErr, err)
		s.Equal(fmt.Sprint(tt.want), fmt.Sprint(tokens))
	}
}

func (s *ControllerSuite) Test_controller_RecognizeInHTML() {
	foundEntities := []*pb.RecognizedEntity{
		{
			Entity:      "found entity",
			Position:    2312,
			Dictionary:  "test",
			Identifiers: map[string]string{"many": "", "things": ""},
		},
	}

	mockRecogniser := &mock_recogniser.Client{}
	mockRecogniser.On("Recognise",
		mock.AnythingOfType("<-chan snippet_reader.Value"),
		lib.RecogniserOptions{},
		mock.AnythingOfType("*sync.WaitGroup"),
	).Return(nil)
	mockRecogniser.On("Err").Return(nil)
	mockRecogniser.On("Result").Return(foundEntities)
	s.controller.recognisers = map[string]recogniser.Client{"mock": mockRecogniser}

	mockSnippetReader := &mock_snippet_reader.Client{}
	mockSnippetReader.On("ReadSnippetsWithCallback", mock.Anything, mock.Anything).Return(nil)
	s.html = mockSnippetReader

	buf := bytes.NewBuffer([]byte("<p>hello my name is jeff</p>"))
	opts := map[string]lib.RecogniserOptions{
		"mock": {},
	}
	entities, err := s.controller.RecognizeInHTML(buf, opts)
	s.ElementsMatch(foundEntities, entities)
	s.Nil(err)
}
