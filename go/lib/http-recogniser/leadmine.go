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
	var text []byte
	err := lib.HtmlToTextWithCallback(reader, func(snippet *pb.Snippet) error {
		snips[len(text)+len([]byte(snippet.GetToken()))] = snippet
		text = append(text, snippet.GetToken()...)
		return nil
	})
	if err != nil {
		errs <- err
		return
	}

	req, err := http.NewRequest(http.MethodPost, d.UrlWithOpts(opts), bytes.NewReader(text))
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

	for _, leadmineEntity := range leadmineResponse.Entities {
		for {
			if text[leadmineEntity.Beg] == leadmineEntity.EntityText[0] {
				textIsEqual := true
				for i := leadmineEntity.Beg; i < leadmineEntity.End; i++ {
					if text[i] != leadmineEntity.EntityText[i - leadmineEntity.Beg]	{
						textIsEqual = false
						break
					}
				}
				if textIsEqual {
					break
				}
			}
			leadmineEntity.Beg++
			leadmineEntity.End++
		}
	}

	var recognisedEntities []*pb.RecognizedEntity
	for _, entity := range leadmineResponse.Entities {
		inc := entity.Beg
		dec := entity.Beg
		var snip *pb.Snippet
		var ok bool
		for {
			snip, ok = snips[inc]
			if ok {
				if strings.Contains(snip.GetToken(), entity.EntityText) {
					break
				}
			}
			snip, ok = snips[dec]
			if ok {
				if strings.Contains(snip.GetToken(), entity.EntityText) {
					break
				}
			}
			inc++
			dec--
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
			Position:    snip.Offset + uint32(entity.Beg) - uint32(inc),
			Xpath: 		 snip.Xpath,
			Dictionary:  entity.EntityGroup,
			Identifiers: nil,
			Metadata:    metadata,
		})
	}

	entities <- recognisedEntities
	errs <- nil
}

type LeadmineResponse struct {
	Created  string `json:"created"`
	Entities []*LeadmineEntity `json:"entities"`
}

type LeadmineEntity struct {
	Beg                   int `json:"beg"`
	BegInNormalizedDoc    int `json:"begInNormalizedDoc"`
	End                   int `json:"end"`
	EndInNormalizedDoc    int `json:"endInNormalizedDoc"`
	EntityText            string `json:"entityText"`
	PossiblyCorrectedText string `json:"possiblyCorrectedText"`
	RecognisingDict       RecognisingDict `json:"recognisingDict"`
	ResolvedEntity string `json:"resolvedEntity"`
	SectionType    string `json:"sectionType"`
	EntityGroup    string `json:"entityGroup"`
}

type RecognisingDict struct {
	EnforceBracketing            bool `json:"enforceBracketing"`
	EntityType                   string `json:"entityType"`
	HtmlColor                    string `json:"htmlColor"`
	MaxCorrectionDistance        int `json:"maxCorrectionDistance"`
	MinimumCorrectedEntityLength int `json:"minimumCorrectedEntityLength"`
	MinimumEntityLength          int `json:"minimumEntityLength"`
	Source                       string `json:"source"`
}

type LeadmineMetadata struct {
	ResolvedEntity string `json:"resolvedEntity"`
	RecognisingDict RecognisingDict
}