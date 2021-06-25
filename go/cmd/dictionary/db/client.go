package db

import "gitlab.mdcatapult.io/informatics/software-engineering/entity-recognition/go/gen/pb"

// Lookup is the value we will store in the db.
type Lookup struct {
	Dictionary string `json:"dictionary"`
	ResolvedEntity string `json:"resolvedEntity,omitempty"`
}

type Client interface {
	NewPipeline(size int) Pipeline
}

type Pipeline interface {
	Set(key string, data []byte)
	ExecSet() error
	Get(token *pb.Snippet)
	ExecGet(onResult func(*pb.Snippet, *Lookup) error) error
	Size() int
}
