package recogniser

import (
	"gitlab.mdcatapult.io/informatics/software-engineering/entity-recognition/go/gen/pb"
	"gitlab.mdcatapult.io/informatics/software-engineering/entity-recognition/go/lib"
	"sync"
)

type Client interface {
	Recognise(<-chan lib.SnipReaderValue, lib.RecogniserOptions, *sync.WaitGroup) error
	Err() error
	Result() []*pb.RecognizedEntity
}


