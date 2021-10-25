package http_recogniser

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"regexp"
	"strings"

	"gitlab.mdcatapult.io/informatics/software-engineering/entity-recognition/go/gen/pb"
	"gitlab.mdcatapult.io/informatics/software-engineering/entity-recognition/go/lib"
)

func NewLeadmineClient(url string) Client {
	return leadmine{
		Url:        url,
		httpClient: http.DefaultClient,
	}
}

type leadmine struct {
	Url string
	httpClient lib.HttpClient
}

func (d leadmine) UrlWithOpts(opts Options) string {
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

func (d leadmine) Recognise(reader io.Reader, opts Options, entities chan []*pb.RecognizedEntity, errs chan error) {
	snips := make(map[int]*pb.Snippet)
	var text string

	err := lib.HtmlToTextWithCallback(reader, func(snippet *pb.Snippet) error {
		snips[len(text)] = snippet
		text += snippet.GetToken()
		return nil
	})
	if err != nil {
		errs <- err
		return
	}

	req, err := http.NewRequest(http.MethodPost, d.UrlWithOpts(opts), strings.NewReader(text))
	if err != nil {
		errs <- err
		return
	}

	resp, err := d.httpClient.Do(req)
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

	var correctedLeadmineEntities []LeadmineEntity
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
			errs <- err
			return
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

	var recognisedEntities []*pb.RecognizedEntity
	for _, entity := range correctedLeadmineEntities {
		dec := entity.Beg
		position := 0
		var snip *pb.Snippet
		var ok bool
		for {
			snip, ok = snips[dec]
			if ok {
				if strings.Contains(snip.GetToken(), entity.EntityText) {
					break
				} else {
					errs <- errors.New("entity not in snippet - FIX ME")
				}
			}
			dec--
			position++
		}

		metadata, err := json.Marshal(LeadmineMetadata{
			ResolvedEntity:  entity.ResolvedEntity,
			RecognisingDict: entity.RecognisingDict,
		})
		if err != nil {
			errs <- err
			return
		}

		recognisedEntities = append(recognisedEntities, &pb.RecognizedEntity{
			Entity:      entity.EntityText,
			Position:    uint32(position),
			Xpath:       snip.Xpath,
			Dictionary:  entity.EntityGroup,
			Identifiers: nil,
			Metadata:    metadata,
		})
	}

	entities <- recognisedEntities
	errs <- nil
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
	RecognisingDict       RecognisingDict `json:"recognisingDict"`
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
	ResolvedEntity  string `json:"resolvedEntity"`
	RecognisingDict RecognisingDict
}
