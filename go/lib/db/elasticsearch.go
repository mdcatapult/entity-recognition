package db

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/elastic/go-elasticsearch/v7"
	"gitlab.mdcatapult.io/informatics/software-engineering/entity-recognition/go/gen/pb"
	"gitlab.mdcatapult.io/informatics/software-engineering/entity-recognition/go/lib"
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

func NewElasticsearchClient (conf ElasticsearchConfig) (Client, error) {
	c, err := elasticsearch.NewClient(elasticsearch.Config{
		Addresses:             []string{fmt.Sprintf("http://%s:%d", conf.Host, conf.Port)},
	})
	if err != nil {
		return nil, err
	}
	index := lib.RandomLowercaseString(7)
	res, err := c.Indices.Create(index)
	if err != nil {
		return nil, err
	} else if res.StatusCode != 200 {
		return nil, errors.New(res.String())
	}

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
	p.buf.WriteString(`{"index":{}}\n`)
	p.buf.Write(data)
	p.buf.WriteString(`\n`)
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
	p.buf.WriteString(`{}\n`)
	p.buf.WriteString(fmt.Sprintf(`{"size": 1, "query" : {"match" : { "synonym": "%s" }}}\n`, string(token.GetData())))
	p.currentQuery = append(p.currentQuery, token)
}

func (p esPipeline) ExecGet(onResult func(*pb.Snippet, *Lookup) error) error {
	res, err := p.Msearch(p.buf, p.Msearch.WithIndex(p.index))
	if err != nil {
		return err
	} else if res.StatusCode != 200 {
		return errors.New(res.String())
	}

	scn := bufio.NewScanner(res.Body)
	i := 0
	for scn.Scan() {
		i++
		var esLookup EsLookup
		if err := json.Unmarshal(scn.Bytes(), &esLookup); err != nil {
			return err
		}

		var lookup *Lookup
		if esLookup.Dictionary == "" {
			lookup = nil
		} else {
			lookup = &Lookup{
				Dictionary:       esLookup.Dictionary,
				ResolvedEntities: esLookup.Identifiers,
			}
		}
		if err := onResult(p.currentQuery[i-1], lookup); err != nil {
			return err
		}
	}
	return nil
}
func (p esPipeline) Size() int {
	return len(p.currentQuery)
}