package testhelpers

import (
	"gitlab.mdcatapult.io/informatics/software-engineering/entity-recognition/go/lib"
	"io"

	mocks "gitlab.mdcatapult.io/informatics/software-engineering/entity-recognition/go/gen/mocks/gen/pb"
	"gitlab.mdcatapult.io/informatics/software-engineering/entity-recognition/go/gen/pb"
)

func CreateSnippets(tokens ...string) []*pb.Snippet {
	snippets := make([]*pb.Snippet, len(tokens))
	for i, tok := range tokens {
		snippets[i] = CreateSnippet(tok, tok, 0, "")
	}
	return snippets
}

func CreateSnippet(originalText, normalisedText string, offset uint32, xpath string) *pb.Snippet {
	return &pb.Snippet{
		Text:           originalText,
		NormalisedText: normalisedText,
		Offset:         offset,
		Xpath:          xpath,
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

func APIEntityFromEntity(entity *pb.Entity) lib.APIEntity {
	return lib.APIEntity{
		Name:        entity.Name,
		Recogniser:  entity.Recogniser,
		Identifiers: entity.Identifiers,
		Metadata:    entity.Metadata,
		Positions: []lib.Position{
			{Xpath: entity.Xpath,
				Position: entity.Position},
		},
	}
}
