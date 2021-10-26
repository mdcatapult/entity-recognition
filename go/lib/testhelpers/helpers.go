package testhelpers

import (
	"io"

	mocks "gitlab.mdcatapult.io/informatics/software-engineering/entity-recognition/go/gen/mocks/gen/pb"
	"gitlab.mdcatapult.io/informatics/software-engineering/entity-recognition/go/gen/pb"
)

func Snips(toks ...string) []*pb.Snippet {
	snippets := make([]*pb.Snippet, len(toks))
	for i, tok := range toks {
		snippets[i] = Snip(tok, 0, "")
	}
	return snippets
}

func Snip(tok string, offset uint32, xpath string) *pb.Snippet {
	return &pb.Snippet{
		Token:  tok,
		Offset: offset,
		Xpath:  xpath,
	}
}

func NewMockRecognizeServerStream(snippets ...*pb.Snippet) *mocks.Recognizer_GetStreamServer {
	stream := &mocks.Recognizer_GetStreamServer{}
	for _, snippet := range snippets {
		stream.On("Recv").Return(snippet, nil).Once()
	}
	stream.On("Recv").Return(nil, io.EOF).Once()
	return stream
}

func NewMockRecognizeClientStream(snippets ...*pb.Snippet) *mocks.Recognizer_GetStreamClient {
	stream := &mocks.Recognizer_GetStreamClient{}
	for _, snippet := range snippets {
		stream.On("Send", snippet).Return(nil).Once()
	}
	stream.On("CloseSend").Return(nil).Once()
	return stream
}
