package db

import (
	"bytes"
	"github.com/elastic/go-elasticsearch/v7"
	"gitlab.mdcatapult.io/informatics/software-engineering/entity-recognition/go/gen/pb"
)

type ElasticsearchConfig struct {
	Host string
	Port int
}

type esLookup struct {
	PubChemId int `json:"pubchem_id"`
	Synonym []string `json:"synonym"`
	Identifier []string `json:"identifier"`
}

//func NewElasticsearchClient (conf ElasticsearchConfig) Client {
//	c, err := elasticsearch.NewClient(elasticsearch.Config{
//		Addresses:             []string{fmt.Sprintf("http://%s:%d", conf.Host, conf.Port)},
//	})
//	if err != nil {
//		panic(err)
//	}
//
//	return &esClient{
//		Client: c,
//	}
//}

type esClient struct {
	*elasticsearch.Client
}

func (e esClient) NewGetPipeline(size int) GetPipeline {
	return nil
}

type esPipeline struct {
	esClient
	buf bytes.Buffer
	size int
}

func (p esPipeline) Set(key string, data []byte) {}
func (p esPipeline) ExecSet() error {
	return nil
}
func (p esPipeline) Get(token *pb.Snippet) {}
func (p esPipeline) ExecGet(onResult func(*pb.Snippet, *Lookup) error) error {
	return nil
}
func (p esPipeline) Size() int {
	return p.size
}