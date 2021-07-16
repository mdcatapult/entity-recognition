package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"testing"

	"gitlab.mdcatapult.io/informatics/software-engineering/entity-recognition/go/lib/testhelpers"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
	"gitlab.mdcatapult.io/informatics/software-engineering/entity-recognition/go/gen/mocks"
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
	acetylcarnitineHTML, err := os.Open("resources/acetylcarnitine.html")
	s.Require().Nil(err)
	acetylcarnitineRawFile, err := os.Open("resources/acetylcarnitine.txt")
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
	acetylcarnitineHTML, err := os.Open("resources/acetylcarnitine.html")
	s.Require().Nil(err)
	acetylcarnitineTokensFile, err := os.Open("resources/acetylcarnitine-tokens.json")
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
		tokens, err := s.controller.TokenizeHTML(tt.args.reader)
		s.Equal(tt.wantErr, err)
		s.Equal(fmt.Sprint(tt.want), fmt.Sprint(tokens))
	}
}

func (s *ControllerSuite) Test_controller_RecognizeInHTML() {
	testhelpers.UseOffsets()
	defer testhelpers.DoNotUseOffsets()
	mockRecognizer_RecognizeClient, _ := testhelpers.NewMockRecognizeClientStream("hello", "my", "name", "is", "jeff")
	foundEntity := &pb.RecognizedEntity{
		Entity:     "found entity",
		Position:   2312,
		Type:       "test",
		ResolvedTo: []string{"many", "things"},
	}
	mockRecognizer_RecognizeClient.On("Recv").Return(foundEntity, nil).Once()
	mockRecognizer_RecognizeClient.On("Recv").Return(nil, io.EOF).Once()

	mockRecognizerClient := &mocks.RecognizerClient{}
	mockRecognizerClient.On("Recognize", mock.AnythingOfType("*context.emptyCtx")).Return(mockRecognizer_RecognizeClient, nil).Once()
	s.controller.clients = []pb.RecognizerClient{mockRecognizerClient}

	buf := bytes.NewBuffer([]byte("<p>hello my name is jeff</p>"))
	type args struct {
		reader io.Reader
	}
	tests := []struct {
		name    string
		args    args
		want    []*pb.RecognizedEntity
		wantErr error
	}{
		{
			name: "happy path",
			args: args{
				reader: buf,
			},
			want:    []*pb.RecognizedEntity{foundEntity},
			wantErr: nil,
		},
	}
	for _, tt := range tests {
		s.T().Log(tt.name)
		got, gotErr := s.controller.RecognizeInHTML(tt.args.reader)
		s.ElementsMatch(tt.want, got)
		s.Equal(tt.wantErr, gotErr)
	}
	mockRecognizerClient.AssertExpectations(s.T())
	mockRecognizer_RecognizeClient.AssertExpectations(s.T())
}
