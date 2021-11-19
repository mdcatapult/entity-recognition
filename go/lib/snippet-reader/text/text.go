package text

import (
	"bufio"
	"io"

	"gitlab.mdcatapult.io/informatics/software-engineering/entity-recognition/go/gen/pb"
	snippet_reader "gitlab.mdcatapult.io/informatics/software-engineering/entity-recognition/go/lib/snippet-reader"
)

type SnippetReader struct{}

func (t SnippetReader) ReadSnippets(r io.Reader) <-chan snippet_reader.Value {
	snips := make(chan snippet_reader.Value)
	go readLines(r, snips)
	return snips
}

func (t SnippetReader) ReadSnippetsWithCallback(r io.Reader, onSnippet func(*pb.Snippet) error) error {
	snips := ReadSnippets(r)
	return snippet_reader.ReadChannelWithCallback(snips, onSnippet)
}

func ReadSnippets(r io.Reader) <-chan snippet_reader.Value {
	snips := make(chan snippet_reader.Value)
	go readLines(r, snips)
	return snips
}

func readLines(r io.Reader, values chan snippet_reader.Value) {
	scanner := bufio.NewScanner(r)
	offset := 0
	for scanner.Scan() {
		values <- snippet_reader.Value{
			Snippet: &pb.Snippet{
				Text:   scanner.Text(),
				Offset: uint32(offset),
			},
			Err: nil,
		}
		offset += len(scanner.Text()) + 1 // +1 for newline character
	}
	values <- snippet_reader.Value{
		Snippet: nil,
		Err:     io.EOF,
	}
}
