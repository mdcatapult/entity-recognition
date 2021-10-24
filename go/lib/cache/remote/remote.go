package remote

import (
	"gitlab.mdcatapult.io/informatics/software-engineering/entity-recognition/go/gen/pb"
	"gitlab.mdcatapult.io/informatics/software-engineering/entity-recognition/go/lib/cache"
)

type RemoteCacheClient interface {
	NewGetPipeline(size int) GetPipeline
	NewSetPipeline(size int) SetPipeline
	Ready() bool
}

type Pipeline interface {
	Size() int
}

type GetPipeline interface {
	Get(token *pb.Snippet)
	ExecGet(onResult func(*pb.Snippet, *cache.Lookup) error) error
	Pipeline
}

type SetPipeline interface {
	Set(key string, data []byte)
	ExecSet() error
	Pipeline
}
