package lib

import (
	"gitlab.mdcatapult.io/informatics/software-engineering/entity-recognition/go/gen/pb"
	"io"
)

type SnipReaderValue struct {
	Snippet *pb.Snippet
	Err error
}

func ReadSnippets(snipReaderValues <-chan SnipReaderValue, callback func(snippet *pb.Snippet) error) error {
	for readerValue := range snipReaderValues {
		if readerValue.Err == io.EOF {
			break
		} else if readerValue.Err != nil {
			return readerValue.Err
		}
		if err := callback(readerValue.Snippet); err != nil {
			return err
		}
	}
	return nil
}

