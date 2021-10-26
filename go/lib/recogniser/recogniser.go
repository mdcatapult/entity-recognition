package recogniser

import (
	"gitlab.mdcatapult.io/informatics/software-engineering/entity-recognition/go/gen/pb"
	"gitlab.mdcatapult.io/informatics/software-engineering/entity-recognition/go/lib"
	"gitlab.mdcatapult.io/informatics/software-engineering/entity-recognition/go/lib/snippet-reader"
	"sync"
)

type Client interface {
	Recognise(<-chan snippet_reader.Value, lib.RecogniserOptions, *sync.WaitGroup) error
	Err() error
	Result() []*pb.RecognizedEntity
}


