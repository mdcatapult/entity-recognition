package snippet_reader

import (
	"io"

	"gitlab.mdcatapult.io/informatics/software-engineering/entity-recognition/go/gen/pb"
)

type Client interface {
	ReadSnippets(r io.Reader) <-chan Value
	ReadSnippetsWithCallback(r io.Reader, onSnippet func(*pb.Snippet) error) error
}

type Value struct {
	Snippet *pb.Snippet
	Err     error
}

func ReadChannelWithCallback(snipReaderValues <-chan Value, callback func(snippet *pb.Snippet) error) error {
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
