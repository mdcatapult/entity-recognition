package recogniser

import (
	"gitlab.mdcatapult.io/informatics/software-engineering/entity-recognition/go/lib"
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

	// ALTERNATIVE: instead of passing httpOptions here (they are only used on http implementation, not gRPC), add httpOptions to leadmine recogniser struct.
	// This means exporting the struct and performing a type check on values of this interface which comes with its own issues
	Recognise(<-chan snippet_reader.Value, *sync.WaitGroup, lib.HttpOptions) error
	Err() error
	Result() []*pb.Entity
	SetExactMatch(bool)
}
