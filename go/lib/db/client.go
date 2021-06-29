package db

import "gitlab.mdcatapult.io/informatics/software-engineering/entity-recognition/go/gen/pb"

// Lookup is the value we will store in the db.
type Lookup struct {
	Dictionary       string `json:"dictionary"`
	ResolvedEntities []string `json:"resolvedEntities,omitempty"`
}

type Client interface {
	NewGetPipeline(size int) GetPipeline
	NewSetPipeline(size int) SetPipeline
}

type Pipeline interface {
	Size() int
}

type GetPipeline interface {
	Get(token *pb.Snippet)
	ExecGet(onResult func(*pb.Snippet, *Lookup) error) error
	Pipeline
}

type SetPipeline interface {
	Set(key string, data []byte)
	ExecSet() error
	Pipeline
}