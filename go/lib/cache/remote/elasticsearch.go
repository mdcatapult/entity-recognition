package remote

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"

	"github.com/elastic/go-elasticsearch/v7"
	"gitlab.mdcatapult.io/informatics/software-engineering/entity-recognition/go/gen/pb"
	"gitlab.mdcatapult.io/informatics/software-engineering/entity-recognition/go/lib/cache"
)

type ElasticsearchConfig struct {
	Host  string
	Port  int
	index string
}

type EsLookup struct {
	Dictionary  string            `json:"dictionary"`
	Synonyms    []string          `json:"synonyms"`
	Identifiers map[string]string `json:"identifiers"`
}

type esResponse struct {
	Took      int `json:"took"`
	Responses []struct {
		Took     int  `json:"took"`
		TimedOut bool `json:"timed_out"`
		Shards   struct {
			Total      int `json:"total"`
			Successful int `json:"successful"`
			Skipped    int `json:"skipped"`
			Failed     int `json:"failed"`
		} `json:"_shards"`
		Hits struct {
			Total struct {
				Value    int    `json:"value"`
				Relation string `json:"relation"`
			} `json:"total"`
			MaxScore float64 `json:"max_score"`
			Hits     []struct {
				Index  string   `json:"_index"`
				Type   string   `json:"_type"`
				ID     string   `json:"_id"`
				Score  float64  `json:"_score"`
				Source EsLookup `json:"_source"`
			} `json:"hits"`
		} `json:"hits"`
		Status int `json:"status"`
	} `json:"responses"`
}

func NewElasticsearchClient(conf ElasticsearchConfig) (Client, error) {
	c, err := elasticsearch.NewClient(elasticsearch.Config{
		Addresses: []string{fmt.Sprintf("http://%s:%d", conf.Host, conf.Port)},
	})
	if err != nil {
		return nil, err
	}
	return &esClient{
		Client: c,
		index:  conf.index,
	}, nil
}

type esClient struct {
	*elasticsearch.Client
	index string
}

func (e *esClient) Ready() bool {
	res, err := e.Info()
	if err != nil || res.StatusCode != 200 {
		return false
	}
	return true
}

func (e *esClient) NewGetPipeline(size int) GetPipeline {
	return &esPipeline{
		esClient:     e,
		buf:          bytes.NewBuffer(nil),
		currentQuery: make([]*pb.Snippet, 0, size),
	}
}

func (e *esClient) NewSetPipeline(size int) SetPipeline {
	return &esPipeline{
		esClient:     e,
		buf:          bytes.NewBuffer(nil),
		currentQuery: make([]*pb.Snippet, 0, size),
	}
}

type esPipeline struct {
	*esClient
	buf          *bytes.Buffer
	currentQuery []*pb.Snippet
}

func (p *esPipeline) Set(_ string, data []byte) {
	p.buf.WriteString(fmt.Sprintf(`{"index":{}}%s`, "\n"))
	p.buf.WriteString(fmt.Sprintf("%s%s", string(data), "\n"))
	p.currentQuery = append(p.currentQuery, nil)
}
func (p *esPipeline) ExecSet() error {
	res, err := p.Bulk(p.buf, p.Bulk.WithIndex(p.index))
	if err != nil {
		return err
	} else if res.StatusCode != 200 {
		return errors.New(res.String())
	}
	return nil
}
func (p *esPipeline) Get(token *pb.Snippet) {
	p.buf.WriteString(fmt.Sprintf(`{}%s`, "\n"))
	p.buf.WriteString(fmt.Sprintf(`{"size": 1, "query" : {"match" : { "synonyms": "%s" }}}%s`, jsonEscape(token.GetNormalisedText()), "\n"))
	p.currentQuery = append(p.currentQuery, token)
}

func jsonEscape(i string) string {
	b, err := json.Marshal(i)
	if err != nil {
		panic(err)
	}
	s := string(b)
	return s[1 : len(s)-1]
}

func (p esPipeline) ExecGet(onResult func(*pb.Snippet, *cache.Lookup) error) error {
	//f, err := os.Create(fmt.Sprintf("%s.jsonl", time.Now().Format(time.RFC3339Nano)))
	//if err != nil {
	//	return err
	//}
	//_, err = p.buf.WriteTo(f)
	//if err != nil {
	//	return err
	//}
	res, err := p.Msearch(p.buf, p.Msearch.WithIndex(p.index))
	if err != nil {
		return err
	} else if res.StatusCode != 200 {
		return errors.New(res.String())
	}

	b, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return err
	}
	var esresponse esResponse
	if err := json.Unmarshal(b, &esresponse); err != nil {
		return err
	}

	for i, response := range esresponse.Responses {

		var lookup *cache.Lookup
		if len(response.Hits.Hits) == 0 {
			lookup = nil
		} else {
			lookup = &cache.Lookup{
				Dictionary:  response.Hits.Hits[0].Source.Dictionary,
				Identifiers: response.Hits.Hits[0].Source.Identifiers,
			}
		}
		if err := onResult(p.currentQuery[i], lookup); err != nil {
			return err
		}
	}
	return nil
}
func (p esPipeline) Size() int {
	return len(p.currentQuery)
}
