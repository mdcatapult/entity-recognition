package grpc_recogniser

import (
	"io"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"gitlab.mdcatapult.io/informatics/software-engineering/entity-recognition/go/gen/pb"
	"gitlab.mdcatapult.io/informatics/software-engineering/entity-recognition/go/lib/blacklist"
	"gitlab.mdcatapult.io/informatics/software-engineering/entity-recognition/go/lib/snippet-reader/html"
	"gitlab.mdcatapult.io/informatics/software-engineering/entity-recognition/go/lib/testhelpers"
)

func Test_grpcRecogniser_recognise(t *testing.T) {
	foundEntity := &pb.Entity{
		Name:        "found entity",
		Position:    3,
		Recogniser:  "test",
		Xpath:       "/p",
		Identifiers: map[string]string{"many": "", "things": ""},
	}
	blacklistedEntity := &pb.Entity{
		Name:        "protein",
		Position:    99999,
		Recogniser:  "test",
		Xpath:       "/p",
		Identifiers: map[string]string{"many": "", "things": ""},
	}

	expectedRecognisedEntities := []*pb.Entity{foundEntity}

	// This text will be fed to the recogniser
	snipChan := html.SnippetReader{}.ReadSnippets(strings.NewReader("" +
		"<p>found entity</p> <p>protein</p>"))

	// This mock stream must match the text that has been supplied to the recogniser
	// in the snipChan
	mockRecognizer_RecognizeClient := testhelpers.NewMockRecognizeClientStream(
		testhelpers.CreateSnippet("found", "", 3, "/p"),
		testhelpers.CreateSnippet("entity", "", 9, "/p"),

		// this should be blacklisted and therefore does not feature in expectedRecognisedEntities
		testhelpers.CreateSnippet("protein", "", 23, "/p"),
	)

	// mock the grpc server's response
	mockRecognizer_RecognizeClient.On("Recv").Return(foundEntity, nil).Once()
	mockRecognizer_RecognizeClient.On("Recv").Return(blacklistedEntity, nil).Once()
	mockRecognizer_RecognizeClient.On("Recv").Return(nil, io.EOF).Once()

	testRecogniser := grpcRecogniser{
		Name:     "test",
		err:      nil,
		entities: nil,
		stream:   mockRecognizer_RecognizeClient,
		blacklist: blacklist.Blacklist{
			CaseSensitive: map[string]bool{},
			CaseInsensitive: map[string]bool{
				"protein": true,
			},
		},
	}

	wg := &sync.WaitGroup{}
	testRecogniser.recognise(snipChan, wg)

	wg.Wait()

	mockRecognizer_RecognizeClient.AssertExpectations(t)
	assert.Nil(t, testRecogniser.err)
	assert.EqualValues(t, expectedRecognisedEntities, testRecogniser.entities)
}
