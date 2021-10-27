package recogniser

import (
	"sync"

	"gitlab.mdcatapult.io/informatics/software-engineering/entity-recognition/go/gen/pb"
	"gitlab.mdcatapult.io/informatics/software-engineering/entity-recognition/go/lib"
	snippet_reader "gitlab.mdcatapult.io/informatics/software-engineering/entity-recognition/go/lib/snippet-reader"
)

type Client interface {
	Recognise(<-chan snippet_reader.Value, lib.RecogniserOptions, *sync.WaitGroup) error
	Err() error
	Result() []*pb.RecognizedEntity
}
