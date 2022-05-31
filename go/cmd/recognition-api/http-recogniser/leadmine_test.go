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

package http_recogniser

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	netUrl "net/url"
	"os"
	"sync"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
	mocks "gitlab.mdcatapult.io/informatics/software-engineering/entity-recognition/go/gen/mocks/lib"
	"gitlab.mdcatapult.io/informatics/software-engineering/entity-recognition/go/gen/pb"
	"gitlab.mdcatapult.io/informatics/software-engineering/entity-recognition/go/lib"
	"gitlab.mdcatapult.io/informatics/software-engineering/entity-recognition/go/lib/blocklist"
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
		blocklist: blocklist.Blocklist{
			CaseSensitive: map[string]bool{
				"AF-DX 250": true,
			},
			CaseInsensitive: map[string]bool{},
		},
	}

	// Set up function arguments
	snipChan := html.SnippetReader{}.ReadSnippets(sourceHtml)
	testOptions := lib.RecogniserOptions{}
	waitGroup := &sync.WaitGroup{}

	// Call the function we're testing!
	err = testLeadmine.Recognise(snipChan, waitGroup, testOptions.HttpOptions)
	s.Nil(err)

	// Get the expected response from resources.
	b, err := ioutil.ReadFile("../../../resources/converted-leadmine-response.json")
	s.Require().Nil(err)
	var expectedEntities []*pb.Entity
	err = json.Unmarshal(b, &expectedEntities)
	s.Require().Nil(err)

	waitGroup.Wait()
	s.Nil(testLeadmine.err)
	s.EqualValues(expectedEntities, testLeadmine.entities)
}

func (s *leadmineSuite) TestUrlWithParams() {
	tests := []struct {
		name           string
		url            string
		opts           lib.RecogniserOptions
		expectedPath   string
		expectedParams map[string]string
	}{
		{
			name:         "no query parameters",
			url:          "https://leadmine.wopr.inf.mdc/chemical-entities/entities",
			opts:         lib.RecogniserOptions{},
			expectedPath: "https://leadmine.wopr.inf.mdc/chemical-entities/entities",
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
			expectedPath: "https://leadmine.wopr.inf.mdc/chemical-entities/entities",
			expectedParams: map[string]string{
				"inchi": "true",
			},
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
			expectedPath: "https://leadmine.wopr.inf.mdc/chemical-entities/entities",
			expectedParams: map[string]string{
				"inchi": "true",
				"hello": "dave",
			},
		},
	}
	for _, test := range tests {
		s.T().Log(test.name)
		leadmine := leadmine{Url: test.url}
		url, err := netUrl.Parse(leadmine.urlWithParams(test.opts.HttpOptions.QueryParameters))
		s.NoError(err)
		s.Equal(test.expectedPath, fmt.Sprintf("%v://%v%v", url.Scheme, url.Host, url.Path))

		for paramName, paramValue := range test.expectedParams {
			s.Equal(paramValue, url.Query().Get(paramName))
		}
	}
}

func (s *leadmineSuite) Test_CorrectLeadmineEntityOffsets() {
	for _, test := range []struct {
		name     string
		entities builderEntities
		text     string
		expected builderEntities
	}{
		{
			name: "text with nothing special",
			text: "entity",
			entities: builderEntities{
				builderEntity{LeadmineEntity{}}.withText("entity"),
			},
			expected: builderEntities{
				builderEntity{}.withText("entity").withEnd(6),
			},
		},
		{
			name: "empty entityText",
			text: "",
			entities: builderEntities{
				builderEntity{LeadmineEntity{}}.withText(""),
			},
			expected: builderEntities{
				builderEntity{}.withText("").withEnd(0),
			},
		},
		{
			name: "text with '-' ",
			text: "an-entity",
			entities: builderEntities{
				builderEntity{LeadmineEntity{}}.withText("an-entity"),
			},
			expected: builderEntities{
				builderEntity{}.withText("an-entity").withEnd(9),
			},
		},
		{
			name: "text with multiple '-' ",
			text: "an-entity-text",
			entities: builderEntities{
				builderEntity{LeadmineEntity{}}.withText("an-entity-text"),
			},
			expected: builderEntities{
				builderEntity{}.withText("an-entity-text").withEnd(14),
			},
		},
		{
			name: "longer search text than entity text",
			text: "entityText",
			entities: builderEntities{
				builderEntity{LeadmineEntity{}}.withText("entity"),
			},
			expected: builderEntities{
				builderEntity{}.withText("entity").withEnd(6),
			},
		},
		{
			name: "entityText in middle of search text",
			text: "test foobar test",
			entities: builderEntities{
				builderEntity{LeadmineEntity{}}.withText("foobar"),
			},
			expected: builderEntities{
				builderEntity{}.withText("foobar").withEnd(11).withBeg(5),
			},
		},
		{
			name: "all special chars",
			text: "+++---",
			entities: builderEntities{
				builderEntity{LeadmineEntity{}}.withText("+++---"),
			},
			expected: []builderEntity{
				builderEntity{}.withText("+++---").withEnd(6),
			},
		},
		{
			name: "(+)-(Z)-antazirine",
			text: "(+)-(Z)-antazirine",
			entities: builderEntities{
				builderEntity{LeadmineEntity{}}.withText("(+)-(Z)-antazirine"),
			},
			expected: builderEntities{
				builderEntity{}.withText("(+)-(Z)-antazirine").withEnd(18),
			},
		},
	} {

		res, err := correctLeadmineEntityOffsets(&LeadmineResponse{
			Created:  "",
			Entities: test.entities.toEntityPtrs(),
		}, test.text)

		s.NoError(err, test.name)
		s.Equal(test.expected.toEntities(), res, test.name)
	}
}

type builderEntity struct {
	LeadmineEntity
}

type builderEntities []builderEntity

func (b builderEntity) withEnd(end int) builderEntity {
	b.End = end
	b.EndInNormalizedDoc = end
	return b
}

func (b builderEntity) withBeg(beg int) builderEntity {
	b.Beg = beg
	b.BegInNormalizedDoc = beg
	return b
}

func (b builderEntity) withText(text string) builderEntity {
	b.EntityText = text
	return b
}

func (b builderEntities) toEntities() []LeadmineEntity {
	res := make([]LeadmineEntity, len(b))
	for i, be := range b {
		res[i] = be.LeadmineEntity
	}
	return res
}

func (b builderEntities) toEntityPtrs() []*LeadmineEntity {
	res := make([]*LeadmineEntity, len(b))
	for i, be := range b {
		res[i] = &be.LeadmineEntity
	}
	return res
}
