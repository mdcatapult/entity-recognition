package testhelpers

import (
	"io"

	"gitlab.mdcatapult.io/informatics/software-engineering/entity-recognition/go/gen/mocks"
	"gitlab.mdcatapult.io/informatics/software-engineering/entity-recognition/go/gen/pb"
)

var useOffset = false

func UseOffsets() {
	useOffset = true
}

func DoNotUseOffsets() {
	useOffset = false
}

func Snips(toks ...string) []*pb.Snippet {
	snippets := make([]*pb.Snippet, len(toks))
	var offset uint32
	for i, tok := range toks {
		snippets[i] = Snip(tok, offset)
		if useOffset {
			offset += uint32(len(tok) + 1)
		}
	}
	return snippets
}

func Snip(tok string, offset uint32) *pb.Snippet {
	return &pb.Snippet{
		Token:  tok,
		Offset: offset,
	}
}

func NewMockRecognizeServerStream(tokens ...string) (*mocks.Recognizer_RecognizeServer, []*pb.Snippet) {
	stream := &mocks.Recognizer_RecognizeServer{}
	snippets := Snips(tokens...)
	for _, snippet := range snippets {
		stream.On("Recv").Return(snippet, nil).Once()
	}
	stream.On("Recv").Return(nil, io.EOF).Once()
	return stream, snippets
}


func NewMockRecognizeClientStream(tokens ...string) (*mocks.Recognizer_RecognizeClient, []*pb.Snippet) {
	stream := &mocks.Recognizer_RecognizeClient{}
	snippets := Snips(tokens...)
	for _, snippet := range snippets {
		stream.On("Send", snippet).Return(nil).Once()
	}
	stream.On("CloseSend").Return(nil).Once()
	return stream, snippets
}
