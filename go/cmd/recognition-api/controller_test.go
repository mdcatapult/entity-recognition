package main

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"strings"
	"sync"
	"testing"

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
	foundEntities := []*pb.Entity{
		{
			Name:        "found entity",
			Position:    2312,
			Recogniser:  "test",
			Identifiers: map[string]string{"many": "", "things": ""},
		},
	}

	sentSnippet := &pb.Snippet{
		Text:   "found entity\n",
		Offset: 3,
		Xpath:  "/p",
	}

	reader := strings.NewReader("<p>found entity</p>")

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
	s.ElementsMatch(foundEntities, entities)
	s.Nil(err)
}
