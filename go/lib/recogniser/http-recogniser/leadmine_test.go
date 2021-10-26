package http_recogniser

import (
	"encoding/json"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
	mocks "gitlab.mdcatapult.io/informatics/software-engineering/entity-recognition/go/gen/mocks/lib"
	"gitlab.mdcatapult.io/informatics/software-engineering/entity-recognition/go/gen/pb"
	"gitlab.mdcatapult.io/informatics/software-engineering/entity-recognition/go/lib"
	"io/ioutil"
	"net/http"
	"os"
	"testing"
)

type leadmineSuite struct {
	suite.Suite
	leadmine leadmine
}

func TestLeadmineSuite(t *testing.T) {
	suite.Run(t, new(leadmineSuite))
}

func (s *leadmineSuite) TestRecognise() {
	// Get reader of file to "recognise" in
	sourceHtml, err := os.Open("../../resources/acetylcarnitine.html")
	s.Require().Nil(err)

	// Set up http mock client to return the leadmine response data
	leadmineResponseFile, err := os.Open("../../resources/leadmine-response.json")
	mockHttpClient := &mocks.HttpClient{}
	mockHttpClient.On("Do", mock.AnythingOfType("*http.Request")).Return(&http.Response{
		StatusCode: http.StatusOK,
		Body:       leadmineResponseFile,
	}, nil)
	testLeadmine := leadmine{
		Url:        "https://leadmine.wopr.inf.mdc/chemical-entities/entities",
		httpClient: mockHttpClient,
	}

	// Set up some channels and other function arguments
	testEntityChannel := make(chan []*pb.RecognizedEntity)
	testErrorChannel := make(chan error)
	testOptions := lib.RecogniserOptions{}

	// Call the function we're testing!
	testLeadmine.Recognise(sourceHtml, testOptions, testEntityChannel, testErrorChannel)

	// Get the expected response from resources.
	b, err := ioutil.ReadFile("../../resources/converted-leadmine-response.json")
	s.Require().Nil(err)
	var expectedEntities []*pb.RecognizedEntity
	err = json.Unmarshal(b, &expectedEntities)

	// Wait for both channels to return something and assert they are what we expect.
Loop:
	for {
		select {
		case entities := <-testEntityChannel:
			s.EqualValues(expectedEntities, entities)
		case err := <-testErrorChannel:
			s.Nil(err)
			break Loop
		}
	}
}

func (s *leadmineSuite) TestUrlWithOpts() {
	tests := []struct {
		name     string
		url      string
		opts     lib.RecogniserOptions
		expected string
	}{
		{
			name:     "no query parameters",
			url:      "https://leadmine.wopr.inf.mdc/chemical-entities/entities",
			opts:     lib.RecogniserOptions{},
			expected: "https://leadmine.wopr.inf.mdc/chemical-entities/entities",
		},
		{
			name: "one query parameter",
			url: "https://leadmine.wopr.inf.mdc/chemical-entities/entities",
			opts: lib.RecogniserOptions{
				HttpOptions: lib.HttpOptions{
					QueryParameters: map[string][]string{
						"inchi": {"true"},
					},
				},
			},
			expected: "https://leadmine.wopr.inf.mdc/chemical-entities/entities?inchi=true",
		},
		{
			name: "multiple query parameters",
			url: "https://leadmine.wopr.inf.mdc/chemical-entities/entities",
			opts: lib.RecogniserOptions{
				HttpOptions: lib.HttpOptions{
					QueryParameters: map[string][]string{
						"inchi": {"true", "yes"},
						"hello": {"dave"},
					},
				},
			},
			expected: "https://leadmine.wopr.inf.mdc/chemical-entities/entities?inchi=true&inchi=yes&hello=dave",
		},
	}
	for _, tt := range tests {
		s.T().Log(tt.name)
		leadmine := leadmine{Url: tt.url}
		actual := leadmine.urlWithOpts(tt.opts)
		s.Equal(tt.expected, actual)
	}
}
