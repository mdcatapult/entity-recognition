package http_recogniser

import (
	"encoding/json"
	"errors"
	"fmt"
	"gitlab.mdcatapult.io/informatics/software-engineering/entity-recognition/go/lib/blacklist"
	"io/ioutil"
	"net/http"
	"regexp"
	"strings"
	"sync"

	"gitlab.mdcatapult.io/informatics/software-engineering/entity-recognition/go/gen/pb"
	"gitlab.mdcatapult.io/informatics/software-engineering/entity-recognition/go/lib"
	"gitlab.mdcatapult.io/informatics/software-engineering/entity-recognition/go/lib/recogniser"
	snippet_reader "gitlab.mdcatapult.io/informatics/software-engineering/entity-recognition/go/lib/snippet-reader"
	"gitlab.mdcatapult.io/informatics/software-engineering/entity-recognition/go/lib/types/leadmine"
)

func NewLeadmineClient(name, url string) recogniser.Client {
	return &leadminer{
		Name:       name,
		Url:        url,
		httpClient: http.DefaultClient,
	}
}

type leadminer struct {
	Name	   string
	Url        string
	httpClient lib.HttpClient
	err        error
	entities   []*pb.RecognizedEntity
}

func (l *leadminer) reset() {
	l.err = nil
	l.entities = nil
}

func (l *leadminer) Err() error {
	return l.err
}

func (l *leadminer) Result() []*pb.RecognizedEntity {
	return l.entities
}

func (l *leadminer) urlWithOpts(opts lib.RecogniserOptions) string {
	if len(opts.QueryParameters) == 0 {
		return l.Url
	}

	sep := func(key string) string {
		return fmt.Sprintf("&%s=", key)
	}

	paramStr := ""
	for key, values := range opts.QueryParameters {
		paramStr += sep(key) + strings.Join(values, sep(key))
	}

	return l.Url + "?" + paramStr[1:]
}

func (l *leadminer) handleError(err error) {
	l.err = err
}

func (l *leadminer) Recognise(snipReaderValues <-chan snippet_reader.Value, opts lib.RecogniserOptions, wg *sync.WaitGroup) error {
	l.reset()
	go l.recognise(snipReaderValues, opts, wg)
	return nil
}

func (l *leadminer) recognise(snipReaderValues <-chan snippet_reader.Value, opts lib.RecogniserOptions, wg *sync.WaitGroup) {
	wg.Add(1)
	defer wg.Done()

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

	leadmineResponse, err := l.callLeadmineWebService(opts, text)
	if err != nil {
		l.handleError(err)
		return
	}

	leadmineResponse.Entities = blacklist.FilterLeadmineEntities(leadmineResponse.Entities)

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

func (l *leadminer) convertLeadmineEntities(correctedLeadmineEntities []leadmine.Entity, snips map[int]*pb.Snippet) ([]*pb.RecognizedEntity, error) {
	var recognisedEntities []*pb.RecognizedEntity
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
					return nil, errors.New("entity not in snippet - FIX ME")
				}
			}
			dec--
			position++
		}

		metadata, err := json.Marshal(leadmine.Metadata{
			EntityGroup:     entity.EntityGroup,
			RecognisingDict: entity.RecognisingDict,
		})
		if err != nil {
			return nil, err
		}

		recognisedEntities = append(recognisedEntities, &pb.RecognizedEntity{
			Entity:     entity.EntityText,
			Position:   uint32(position),
			Xpath:      snip.Xpath,
			Recogniser: l.Name,
			Identifiers: map[string]string{
				"resolvedEntity": entity.ResolvedEntity,
			},
			Metadata: metadata,
		})
	}
	return recognisedEntities, nil
}

func correctLeadmineEntityOffsets(leadmineResponse *leadmine.Response, text string) ([]leadmine.Entity, error) {
	var correctedLeadmineEntities []leadmine.Entity
	done := make(map[string]struct{})
	for _, leadmineEntity := range leadmineResponse.Entities {
		if _, ok := done[leadmineEntity.EntityText]; ok {
			continue
		}
		done[leadmineEntity.EntityText] = struct{}{}

		// Only regex for the text (no extra stuff like word boundaries) because
		// it slows things down considerably.
		r, err := regexp.Compile(leadmineEntity.EntityText)
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

func (l *leadminer) callLeadmineWebService(opts lib.RecogniserOptions, text string) (*leadmine.Response, error) {
	req, err := http.NewRequest(http.MethodPost, l.urlWithOpts(opts), strings.NewReader(text))
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

	var leadmineResponse *leadmine.Response
	if err := json.Unmarshal(b, &leadmineResponse); err != nil {
		return nil, err
	}

	return leadmineResponse, nil
}
