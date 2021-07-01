package db

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/elastic/go-elasticsearch/v7"
	"gitlab.mdcatapult.io/informatics/software-engineering/entity-recognition/go/gen/pb"
	"io/ioutil"
)

type ElasticsearchConfig struct {
	Host string
	Port int
}

type EsLookup struct {
	Dictionary string `json:"dictionary"`
	Synonyms    []string `json:"synonyms"`
	Identifiers []string `json:"identifiers"`
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
				Index  string  `json:"_index"`
				Type   string  `json:"_type"`
				ID     string  `json:"_id"`
				Score  float64 `json:"_score"`
				Source EsLookup `json:"_source"`
			} `json:"hits"`
		} `json:"hits"`
		Status int `json:"status"`
	} `json:"responses"`
}

func NewElasticsearchClient (conf ElasticsearchConfig) (Client, error) {
	c, err := elasticsearch.NewClient(elasticsearch.Config{
		Addresses:             []string{fmt.Sprintf("http://%s:%d", conf.Host, conf.Port)},
	})
	if err != nil {
		return nil, err
	}
	//index := lib.RandomLowercaseString(7)
	index := "pubchem"
	//res, err := c.Indices.Create(index)
	//if err != nil {
	//	return nil, err
	//} else if res.StatusCode != 200 {
	//	return nil, errors.New(res.String())
	//}

	return &esClient{
		Client: c,
		index: index,
	}, nil
}

type esClient struct {
	*elasticsearch.Client
	index string
}

func (e *esClient) Ready() bool {
	return true
}

func (e *esClient) NewGetPipeline(size int) GetPipeline {
	return &esPipeline{
		esClient: e,
		buf: bytes.NewBuffer(nil),
		currentQuery: make([]*pb.Snippet, 0, size),
	}
}

func (e *esClient) NewSetPipeline(size int) SetPipeline {
	return &esPipeline{
		esClient: e,
		buf: bytes.NewBuffer(nil),
		currentQuery: make([]*pb.Snippet, 0, size),
	}
}

type esPipeline struct {
	*esClient
	buf *bytes.Buffer
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
	p.buf.WriteString(fmt.Sprintf(`{"size": 1, "query" : {"match" : { "synonyms": "%s" }}}%s`, jsonEscape(string(token.GetData())), "\n"))
	p.currentQuery = append(p.currentQuery, token)
}

func jsonEscape(i string) string {
	b, err := json.Marshal(i)
	if err != nil {
		panic(err)
	}
	s := string(b)
	return s[1:len(s)-1]
}

func (p esPipeline) ExecGet(onResult func(*pb.Snippet, *Lookup) error) error {
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

		var lookup *Lookup
		if len(response.Hits.Hits) == 0 {
			lookup = nil
		} else {
			lookup = &Lookup{
				Dictionary:       response.Hits.Hits[0].Source.Dictionary,
				ResolvedEntities: response.Hits.Hits[0].Source.Identifiers,
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