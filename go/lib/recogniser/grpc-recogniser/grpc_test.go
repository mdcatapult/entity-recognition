package grpc_recogniser

import (
	"io"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"gitlab.mdcatapult.io/informatics/software-engineering/entity-recognition/go/gen/pb"
	"gitlab.mdcatapult.io/informatics/software-engineering/entity-recognition/go/lib/snippet-reader/html"
	"gitlab.mdcatapult.io/informatics/software-engineering/entity-recognition/go/lib/testhelpers"
)

func Test_grpcRecogniser_recognise(t *testing.T) {
	foundEntity := &pb.RecognizedEntity{
		Entity:      "found entity",
		Position:    3,
		Recogniser:  "test",
		Xpath:       "/p",
		Identifiers: map[string]string{"many": "", "things": ""},
	}
	blacklistedEntity := &pb.RecognizedEntity{
		Entity:      "protein",
		Position:    22,
		Recogniser:  "test",
		Xpath:       "/p",
		Identifiers: map[string]string{"many": "", "things": ""},
	}
	foundEntities := []*pb.RecognizedEntity{foundEntity, blacklistedEntity}

	mockRecognizer_RecognizeClient := testhelpers.NewMockRecognizeClientStream(
		testhelpers.Snip("found", "", 3, "/p"),
		testhelpers.Snip("entity", "", 9, "/p"),
		testhelpers.Snip("protein", "", 22, "/p"),

	)
	mockRecognizer_RecognizeClient.On("Recv").Return(foundEntity, nil).Once()
	mockRecognizer_RecognizeClient.On("Recv").Return(nil, io.EOF).Once()

	testRecogniser := grpcRecogniser{
		Name:     "test",
		err:      nil,
		entities: nil,
		stream:   mockRecognizer_RecognizeClient,
	}

	snipChan := html.SnippetReader{}.ReadSnippets(strings.NewReader("" +
		"<p>found entity</p> <p>protein</p>"))
	wg := &sync.WaitGroup{}
	testRecogniser.recognise(snipChan, wg)

	wg.Wait()

	mockRecognizer_RecognizeClient.AssertExpectations(t)
	assert.Nil(t, testRecogniser.err)
	assert.EqualValues(t, foundEntities, testRecogniser.entities)
}
