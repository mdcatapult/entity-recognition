package http_recogniser

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"os"
	"sync"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
	mocks "gitlab.mdcatapult.io/informatics/software-engineering/entity-recognition/go/gen/mocks/lib"
	"gitlab.mdcatapult.io/informatics/software-engineering/entity-recognition/go/gen/pb"
	"gitlab.mdcatapult.io/informatics/software-engineering/entity-recognition/go/lib"
	"gitlab.mdcatapult.io/informatics/software-engineering/entity-recognition/go/lib/blacklist"
	"gitlab.mdcatapult.io/informatics/software-engineering/entity-recognition/go/lib/snippet-reader/html"
)

type leadmineSuite struct {
	suite.Suite
}

func TestLeadmineSuite(t *testing.T) {
	suite.Run(t, new(leadmineSuite))
}

func (s *leadmineSuite) TestRecognise() {
	// Get reader of file to "recognise" in
	sourceHtml, err := os.Open("../../../resources/acetylcarnitine.html")
	s.Require().Nil(err)

	// Set up http mock client to return the leadmine response data
	leadmineResponseFile, err := os.Open("../../../resources/leadmine-response.json")
	s.Require().Nil(err)
	mockHttpClient := &mocks.HttpClient{}
	mockHttpClient.On("Do", mock.AnythingOfType("*http.Request")).Return(&http.Response{
		StatusCode: http.StatusOK,
		Body:       leadmineResponseFile,
	}, nil)

	testLeadmine := leadmine{
		Name:       "test-leadmine",
		Url:        "https://leadmine.wopr.inf.mdc/chemical-entities/entities",
		httpClient: mockHttpClient,
		blacklist: blacklist.Blacklist{
			CaseSensitive: map[string]bool{
				"AF-DX 250": true,
			},
			CaseInsensitive: map[string]bool{},
		},
	}

	// Set up function arguments
	snipChan := html.SnippetReader{}.ReadSnippets(sourceHtml)
	testOptions := lib.RecogniserOptions{}
	wg := &sync.WaitGroup{}

	// Call the function we're testing!
	err = testLeadmine.Recognise(snipChan, testOptions, wg)
	s.Nil(err)

	// Get the expected response from resources.
	b, err := ioutil.ReadFile("../../../resources/converted-leadmine-response.json")
	s.Require().Nil(err)
	var expectedEntities []*pb.Entity
	err = json.Unmarshal(b, &expectedEntities)
	s.Require().Nil(err)

	wg.Wait()
	s.Nil(testLeadmine.err)
	s.EqualValues(expectedEntities, testLeadmine.entities)
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
			url:  "https://leadmine.wopr.inf.mdc/chemical-entities/entities",
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
			url:  "https://leadmine.wopr.inf.mdc/chemical-entities/entities",
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

func (s *leadmineSuite) Test_CorrectLeadmineEntityOffsets() {
	for _, test := range []struct {
		name string
		entities []builderEntity
		text string
		expected []builderEntity
	} {
		{
			name: "text with nothing special",
			text: "entity",
			entities: []builderEntity{
				builderEntity{LeadmineEntity{}}.withText("entity"),
			},
			expected: []builderEntity{
				builderEntity{}.withText("entity").withEnd(6),
			},
		},
		{
			name: "empty entityText",
			text: "",
			entities: []builderEntity{
				builderEntity{LeadmineEntity{}}.withText(""),
			},
			expected: []builderEntity{
				builderEntity{}.withText("").withEnd(0),
			},
		},
		{
			name: "text with '-' ",
			text: "an-entity",
			entities: []builderEntity{
				builderEntity{LeadmineEntity{}}.withText("an-entity"),
			},
			expected: []builderEntity{
				builderEntity{}.withText("an-entity").withEnd(9),
			},
		},
		{
			name: "text with multiple '-' ",
			text: "an-entity-text",
			entities: []builderEntity{
				builderEntity{LeadmineEntity{}}.withText("an-entity-text"),
			},
			expected: []builderEntity{
				builderEntity{}.withText("an-entity-text").withEnd(14),
			},
		},
		{
			name: "longer search text than entity text",
			text: "entityText",
			entities: []builderEntity{
				builderEntity{LeadmineEntity{}}.withText("entity"),
			},
			expected: []builderEntity{
				builderEntity{}.withText("entity").withEnd(6),
			},
		},
		{
			name: "all special chars",
			text: "+++---",
			entities: []builderEntity{
				builderEntity{LeadmineEntity{}}.withText("+++---"),
			},
			expected: []builderEntity{
				builderEntity{}.withText("+++---").withEnd(6),
			},
		},
		{
			name: "(+)-(Z)-antazirine",
			text: "(+)-(Z)-antazirine",
			entities: []builderEntity{
				builderEntity{LeadmineEntity{}}.withText("(+)-(Z)-antazirine"),
			},
			expected: []builderEntity{
				builderEntity{}.withText("(+)-(Z)-antazirine").withEnd(18),
			},
		},
	}{

		res, err := correctLeadmineEntityOffsets(&LeadmineResponse{
			Created:  "",
			Entities: getEntityPtrs(test.entities),
		}, test.text)

		s.NoError(err, test.name)
		s.Equal(getEntities(test.expected), res, test.name)
	}
}

type builderEntity struct {
	LeadmineEntity
}

func (b builderEntity) withEnd(end int) builderEntity{
	b.End = end
	b.EndInNormalizedDoc = end
	return b
}

func (b builderEntity) withText(text string) builderEntity {
	b.EntityText = text
	return b
}

func getEntities(b []builderEntity) []LeadmineEntity {
	res := make([]LeadmineEntity, len(b))
	for i, be := range b {
		res[i] = be.LeadmineEntity
	}
	return res
}

func getEntityPtrs(b []builderEntity) []*LeadmineEntity {
	res := make([]*LeadmineEntity, len(b))
	for i, be := range b {
		res[i] = &be.LeadmineEntity
	}
	return res
}
