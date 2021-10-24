package http_recogniser

import (
	"io"
	"net/url"

	"gitlab.mdcatapult.io/informatics/software-engineering/entity-recognition/go/gen/pb"
)

type RecogniserType string

const (
	LeadmineType RecogniserType = "leadmine"
	DummyType    RecogniserType = "dummy"
)

type Options struct {
	QueryParameters url.Values `json:"query_parameters"`
}

type Client interface {
	Recognise(reader io.Reader, opts Options, snippets chan []*pb.RecognizedEntity, errors chan error)
}

type DummyClient struct{}

func (d DummyClient) Recognise(reader io.Reader, opts Options, snippets chan []*pb.RecognizedEntity, errs chan error) {
	snippets <- []*pb.RecognizedEntity{
		{
			Entity: "dummy entity",
		},
	}
	errs <- nil
}
