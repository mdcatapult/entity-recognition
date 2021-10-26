package grpc_recogniser

import (
	"github.com/stretchr/testify/assert"
	"gitlab.mdcatapult.io/informatics/software-engineering/entity-recognition/go/gen/pb"
	"gitlab.mdcatapult.io/informatics/software-engineering/entity-recognition/go/lib/snippet-reader/html"
	"gitlab.mdcatapult.io/informatics/software-engineering/entity-recognition/go/lib/testhelpers"
	"io"
	"strings"
	"sync"
	"testing"
)

func Test_grpcRecogniser_recognise(t *testing.T) {
	foundEntity := &pb.RecognizedEntity{
		Entity:      "found entity",
		Position:    3,
		Dictionary:  "test",
		Xpath: "/p",
		Identifiers: map[string]string{"many": "", "things": ""},
	}
	foundEntities := []*pb.RecognizedEntity{foundEntity}

	mockRecognizer_RecognizeClient := testhelpers.NewMockRecognizeClientStream(
		testhelpers.Snip("found", 3, "/p"),
		testhelpers.Snip("entity", 9, "/p"),
	)
	mockRecognizer_RecognizeClient.On("Recv").Return(foundEntity, nil).Once()
	mockRecognizer_RecognizeClient.On("Recv").Return(nil, io.EOF).Once()

	testRecogniser := grpcRecogniser{
		err:      nil,
		entities: nil,
		stream:   mockRecognizer_RecognizeClient,
	}

	snipChan := html.SnippetReader{}.ReadSnippets(strings.NewReader("<p>found entity</p>"))
	wg := &sync.WaitGroup{}
	testRecogniser.recognise(snipChan, wg)

	wg.Wait()

	mockRecognizer_RecognizeClient.AssertExpectations(t)
	assert.Nil(t, testRecogniser.err)
	assert.EqualValues(t, foundEntities, testRecogniser.entities)
}

