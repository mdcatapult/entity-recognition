/*
 * Copyright 2022 Medicines Discovery Catapult
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *     http://www.apache.org/licenses/LICENSE-2.0
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package grpc_recogniser

import (
	"io"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"gitlab.mdcatapult.io/informatics/software-engineering/entity-recognition/go/gen/pb"
	"gitlab.mdcatapult.io/informatics/software-engineering/entity-recognition/go/lib/blocklist"
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
	blocklistedEntity := &pb.Entity{
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

		// this should be blocklisted and therefore does not feature in expectedRecognisedEntities
		testhelpers.CreateSnippet("protein", "", 23, "/p"),
	)

	// mock the grpc server's response
	mockRecognizer_RecognizeClient.On("Recv").Return(foundEntity, nil).Once()
	mockRecognizer_RecognizeClient.On("Recv").Return(blocklistedEntity, nil).Once()
	mockRecognizer_RecognizeClient.On("Recv").Return(nil, io.EOF).Once()

	testRecogniser := grpcRecogniser{
		Name:     "test",
		err:      nil,
		entities: nil,
		stream:   mockRecognizer_RecognizeClient,
		blocklist: blocklist.Blocklist{
			CaseSensitive: map[string]bool{},
			CaseInsensitive: map[string]bool{
				"protein": true,
			},
		},
	}

	waitGroup := &sync.WaitGroup{}
	testRecogniser.recognise(snipChan, waitGroup)

	waitGroup.Wait()

	mockRecognizer_RecognizeClient.AssertExpectations(t)
	assert.Nil(t, testRecogniser.err)
	assert.EqualValues(t, expectedRecognisedEntities, testRecogniser.entities)
}
