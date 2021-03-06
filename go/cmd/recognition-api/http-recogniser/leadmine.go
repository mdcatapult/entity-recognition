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

/*
	http_recogniser provides a client which uses HTTP to communicate with an HTTP server which can perform entity recognition.
	Currently, the only such server is Leadmine Web Service, hence the references to Leadmine.
*/
package http_recogniser

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"sync"

	"gitlab.mdcatapult.io/informatics/software-engineering/entity-recognition/go/gen/pb"
	"gitlab.mdcatapult.io/informatics/software-engineering/entity-recognition/go/lib"
	"gitlab.mdcatapult.io/informatics/software-engineering/entity-recognition/go/lib/blocklist"
	"gitlab.mdcatapult.io/informatics/software-engineering/entity-recognition/go/lib/recogniser"
	snippet_reader "gitlab.mdcatapult.io/informatics/software-engineering/entity-recognition/go/lib/snippet-reader"
)

func NewLeadmineClient(name, url string, blocklist blocklist.Blocklist) recogniser.Client {
	return &leadmine{
		Name:       name,
		Url:        url,
		httpClient: http.DefaultClient,
		blocklist:  blocklist,
	}
}

type leadmine struct {
	Name       string
	Url        string
	httpClient lib.HttpClient
	err        error
	entities   []*pb.Entity
	blocklist  blocklist.Blocklist
	exactMatch bool
}

func (l *leadmine) SetExactMatch(exact bool) {
	l.exactMatch = exact
}

func (l *leadmine) reset() {
	l.err = nil
	l.entities = nil
}

func (l *leadmine) Err() error {
	return l.err
}

func (l *leadmine) Result() []*pb.Entity {
	return l.entities
}

func (l *leadmine) urlWithParams(params url.Values) string {
	if len(params) == 0 {
		return l.Url
	}

	sep := func(key string) string {
		return fmt.Sprintf("&%s=", key)
	}

	paramStr := ""
	for key, values := range params {
		paramStr += sep(key) + strings.Join(values, sep(key))
	}

	return l.Url + "?" + paramStr[1:]
}

func (l *leadmine) handleError(err error) {
	l.err = err
}

// Recognise calls the helper method l.recognise to call Leadmine Web Service to perform entity recognition.
// Found entities are put into l.entities, and errors into l.err.
func (l *leadmine) Recognise(snipReaderValues <-chan snippet_reader.Value, waitGroup *sync.WaitGroup, httpOptions lib.HttpOptions) error {
	l.reset()
	go l.recognise(snipReaderValues, waitGroup, httpOptions)
	return nil
}

func (l *leadmine) recognise(snipReaderValues <-chan snippet_reader.Value, waitGroup *sync.WaitGroup, httpOptions lib.HttpOptions) {
	waitGroup.Add(1)
	defer waitGroup.Done()

	snips := make(map[int]*pb.Snippet)
	var text string

	err := snippet_reader.ReadChannelWithCallback(snipReaderValues, func(snippet *pb.Snippet) error {
		snips[len(text)] = snippet
		text += snippet.GetText()
		return nil
	})

	if err != nil {
		l.handleError(err)
		return
	}

	leadmineResponse, err := l.callLeadmineWebService(httpOptions.QueryParameters, text)
	if err != nil {
		l.handleError(err)
		return
	}

	leadmineResponse.Entities = l.blocklistEntities(leadmineResponse.Entities)

	correctedLeadmineEntities, err := correctLeadmineEntityOffsets(leadmineResponse, text)
	if err != nil {
		l.handleError(err)
		return
	}

	recognisedEntities, err := l.convertLeadmineEntities(correctedLeadmineEntities, snips)
	if err != nil {
		l.handleError(err)
		return
	}

	filteredEntities := lib.FilterSubmatches(recognisedEntities)

	l.entities = filteredEntities
}

func (l *leadmine) blocklistEntities(entities []*LeadmineEntity) []*LeadmineEntity {
	res := make([]*LeadmineEntity, 0, len(entities))
	for _, entity := range entities {
		if l.blocklist.Allowed(entity.EntityText) {
			res = append(res, entity)
		}
	}
	return res
}

func (l *leadmine) convertLeadmineEntities(correctedLeadmineEntities []LeadmineEntity, snips map[int]*pb.Snippet) ([]*pb.Entity, error) {
	var recognisedEntities []*pb.Entity
	for _, entity := range correctedLeadmineEntities {
		dec := entity.Beg
		position := 0
		var snip *pb.Snippet
		var ok bool
		for {
			snip, ok = snips[dec]
			if ok {
				if strings.Contains(snip.GetText(), entity.EntityText) {
					break
				} else {
					return nil, errors.New("LeadmineEntity not in snippet - FIX ME")
				}
			}
			dec--
			position++
		}

		metadata, err := json.Marshal(LeadmineMetadata{
			EntityGroup:     entity.EntityGroup,
			RecognisingDict: entity.RecognisingDict,
		})
		if err != nil {
			return nil, err
		}

		recognisedEntities = append(recognisedEntities, &pb.Entity{
			Name:       entity.EntityText,
			Position:   uint32(position),
			Xpath:      snip.Xpath,
			Recogniser: l.Name,
			Identifiers: map[string]string{
				"resolvedEntity": entity.ResolvedEntity,
			},
			Metadata: string(metadata),
		})
	}
	return recognisedEntities, nil
}

func correctLeadmineEntityOffsets(leadmineResponse *LeadmineResponse, text string) ([]LeadmineEntity, error) {
	var correctedLeadmineEntities []LeadmineEntity
	done := make(map[string]struct{})
	for _, leadmineEntity := range leadmineResponse.Entities {
		if _, ok := done[leadmineEntity.EntityText]; ok {
			continue
		}
		done[leadmineEntity.EntityText] = struct{}{}

		// Only regex for the text (no extra stuff like word boundaries) because
		// it slows things down considerably.
		escapedString := regexp.QuoteMeta(leadmineEntity.EntityText)
		r, err := regexp.Compile(escapedString)
		if err != nil {
			return nil, err
		}

		matches := r.FindAllStringIndex(text, -1)
		for _, match := range matches {
			entity := *leadmineEntity
			entity.Beg = match[0]
			entity.BegInNormalizedDoc = match[0]
			entity.End = match[1]
			entity.EndInNormalizedDoc = match[1]
			correctedLeadmineEntities = append(correctedLeadmineEntities, entity)
		}
	}
	return correctedLeadmineEntities, nil
}

func (l *leadmine) callLeadmineWebService(httpParams url.Values, text string) (*LeadmineResponse, error) {
	req, err := http.NewRequest(http.MethodPost, l.urlWithParams(httpParams), strings.NewReader(text))
	if err != nil {
		return nil, err
	}

	resp, err := l.httpClient.Do(req)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != 200 {
		return nil, err
	}

	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var leadmineResponse *LeadmineResponse
	if err := json.Unmarshal(b, &leadmineResponse); err != nil {
		return nil, err
	}

	return leadmineResponse, nil
}

type LeadmineResponse struct {
	Created  string            `json:"created"`
	Entities []*LeadmineEntity `json:"entities"`
}

type LeadmineEntity struct {
	Beg                   int             `json:"beg"`
	BegInNormalizedDoc    int             `json:"begInNormalizedDoc"`
	End                   int             `json:"end"`
	EndInNormalizedDoc    int             `json:"endInNormalizedDoc"`
	EntityText            string          `json:"entityText"`
	PossiblyCorrectedText string          `json:"possiblyCorrectedText"`
	RecognisingDict       RecognisingDict `json:"RecognisingDict"`
	ResolvedEntity        string          `json:"resolvedEntity"`
	SectionType           string          `json:"sectionType"`
	EntityGroup           string          `json:"entityGroup"`
}

type RecognisingDict struct {
	EnforceBracketing            bool   `json:"enforceBracketing"`
	EntityType                   string `json:"entityType"`
	HtmlColor                    string `json:"htmlColor"`
	MaxCorrectionDistance        int    `json:"maxCorrectionDistance"`
	MinimumCorrectedEntityLength int    `json:"minimumCorrectedEntityLength"`
	MinimumEntityLength          int    `json:"minimumEntityLength"`
	Source                       string `json:"source"`
}

type LeadmineMetadata struct {
	EntityGroup     string `json:"entityGroup"`
	RecognisingDict RecognisingDict
}
