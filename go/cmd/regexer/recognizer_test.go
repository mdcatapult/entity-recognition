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
	mockStream, _ := testhelpers.NewMockRecognizeServerStream("hello", "my", "name", "is", "jeff")
	foundEntity := &pb.RecognizedEntity{
		Entity:     "hello",
		Dictionary: "test_regex",
	}
	mockStream.On("Send", foundEntity).Return(nil).Once()
	type args struct {
		stream pb.Recognizer_RecognizeServer
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
		gotErr := s.recogniser.Recognize(tt.args.stream)
		s.Equal(tt.wantErr, gotErr)
	}
	mockStream.AssertExpectations(s.T())
}
