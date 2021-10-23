package http_recogniser

import (
	"bytes"
	"encoding/json"
	"fmt"
	"gitlab.mdcatapult.io/informatics/software-engineering/entity-recognition/go/lib"
	"io"
	"io/ioutil"
	"net/http"
	"strings"

	"gitlab.mdcatapult.io/informatics/software-engineering/entity-recognition/go/gen/pb"
)

type Leadmine struct {
	Url string
}

func (d Leadmine) UrlWithOpts(opts Options) string {
	if len(opts.QueryParameters) == 0 {
		return d.Url
	}

	sep := func(key string) string {
		return fmt.Sprintf("&%s=", key)
	}

	paramStr := ""
	for key, values := range opts.QueryParameters {
		paramStr += sep(key) + strings.Join(values, sep(key))
	}

	return d.Url + "?" + paramStr[1:]
}

func (d Leadmine) Recognise(reader io.Reader, opts Options, entities chan []*pb.RecognizedEntity, errs chan error) {
	snips := make(map[int]*pb.Snippet)
	var text bytes.Buffer
	err := lib.HtmlToTextWithCallback(reader, func(snippet *pb.Snippet) error {
		snips[len(snippet.GetToken())] = snippet
		text.WriteString(snippet.GetToken())
		return nil
	})
	if err != nil {
		errs <- err
		return
	}

	req, err := http.NewRequest(http.MethodGet, d.UrlWithOpts(opts), &text)
	if err != nil {
		errs <- err
		return
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		errs <- err
		return
	}

	if resp.StatusCode != 200 {
		errs <- err
		return
	}

	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		errs <- err
		return
	}

	var leadmineResponse LeadmineResponse
	if err := json.Unmarshal(b, &leadmineResponse); err != nil {
		errs <- err
		return
	}
	var recognisedEntities []*pb.RecognizedEntity
	for _, entity := range leadmineResponse.entities {
		position := entity.beg
		var snip *pb.Snippet
		var ok bool
		for snip, ok = snips[position]; !ok; position-- {}

		metadata, err := json.Marshal(LeadmineMetadata{
			resolvedEntity:  entity.resolvedEntity,
			recognisingDict: entity.recognisingDict,
		})
		if err != nil {
			errs <- err
			return
		}

		recognisedEntities = append(recognisedEntities, &pb.RecognizedEntity{
			Entity:      entity.entityText,
			Position:    snip.Offset + uint32(entity.beg) - uint32(position),
			Xpath: 		 snip.Xpath,
			Dictionary:  entity.entityGroup,
			Identifiers: nil,
			Metadata:    metadata,
		})
	}

	entities <- recognisedEntities
	errs <- nil
}

type LeadmineResponse struct {
	created  string
	entities []struct {
		beg                   int
		begInNormalizedDoc    int
		end                   int
		endInNormalizedDoc    int
		entityText            string
		possiblyCorrectedText string
		recognisingDict       RecognisingDict
		resolvedEntity string
		sectionType    string
		entityGroup    string
	}
}

type RecognisingDict struct {
	enforceBracketing            bool
	entityType                   string
	htmlColor                    string
	maxCorrectionDistance        int
	minimumCorrectedEntityLength int
	minimumEntityLength          int
	source                       string
}

type LeadmineMetadata struct {
	resolvedEntity string
	recognisingDict RecognisingDict
}