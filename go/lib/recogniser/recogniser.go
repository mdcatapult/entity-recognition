package recogniser

import (
	"sync"

	"gitlab.mdcatapult.io/informatics/software-engineering/entity-recognition/go/gen/pb"
	snippet_reader "gitlab.mdcatapult.io/informatics/software-engineering/entity-recognition/go/lib/snippet-reader"
)

// Client
// represents a recogniser client, i.e. a struct which implements functions to
// use a recogniser via HTTP or gRPC. Recognise() must receive snippet_reader.Values, tokenise them, and send them to a configured recogniser.
// It must then either populate result or err depending on what happened.
//
// swagger:model RecogniserClient
type Client interface {
	Recognise(<-chan snippet_reader.Value, *sync.WaitGroup) error
	Err() error
	Result() []*pb.Entity
	SetExactMatch(bool)
}
