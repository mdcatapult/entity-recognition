package text

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"gitlab.mdcatapult.io/informatics/software-engineering/entity-recognition/go/gen/pb"
	"testing"
)

func Test_Tokenise(t *testing.T) {

	// ε
	//δ
	//Δ
	//γ
	//θ
	//λ
	//τ
	//β
	//ω
	//α
	//ν
	//π
	//ψ

	for _, test := range []struct {
		name           string
		snippet        *pb.Snippet
		expectedText   []string
		expectedOffset []uint32
		exactMatch     bool
	}{
		//{
		//	name: "given the input string sadfads'some-text', with it should return 1 token",
		//	snippet: &pb.Snippet{
		//		Text: "£!£-apples!!",
		//	},
		//	expectedText:   []string{"apples"},
		//	expectedOffset: []uint32{5},
		//	exactMatch: false,
		//},
		//{
		//	name: "given thasdfe input string 'some-text', with it should return 1 token",
		//	snippet: &pb.Snippet{
		//		Text: "-pasta-la vista baby la",
		//	},
		//	expectedText:   []string{"pasta", "la", "vista", "baby", "la"},
		//	expectedOffset: []uint32{1, 7, 10, 16, 21},
		//	exactMatch:     false,
		//},
		//{
		//	name: "given the input string 'some-text', with it should return 1 token",
		//	snippet: &pb.Snippet{
		//		Text: "!@££$%^&*&^%^&pasta-la-vista-baby^%^&^%$%$?????£",
		//	},
		//	expectedText:   []string{"pasta", "la", "vista", "baby"},
		//	expectedOffset: []uint32{15, 21, 24, 30},
		//	exactMatch: false,
		//},
		//{
		//	name: "given some greek characters, etc",
		//	snippet: &pb.Snippet{
		//		Text: "@£$%^βωα*&^νπψ?!@£$%^, ανπψ/!?!?!?",
		//	},
		//	expectedText:   []string{"βωα", "νπψ", "ανπψ"},
		//	expectedOffset: []uint32{5, 11, 23},
		//	exactMatch: false,
		//},
		//{
		//	name: "given the input string 'some-text', it should return 2 tokens",
		//	snippet: &pb.Snippet{
		//		Text: "some-text",
		//	},
		//	expectedText:   []string{"some", "-", "text"},
		//	expectedOffset: []uint32{0, 4, 5},
		//	exactMatch:     false,
		//},
		//{
		//	name: "given the input string 'some-text-', it should return 1 token with exact match",
		//	snippet: &pb.Snippet{
		//		Text: "some-text-",
		//	},
		//	expectedText:   []string{"some-text-"},
		//	expectedOffset: []uint32{0},
		//	exactMatch:     true,
		//},
		//{
		//	name: "given the input string ' some -text', it should return 1 token with exact match",
		//	snippet: &pb.Snippet{
		//		Text: " some -text",
		//	},
		//	expectedText:   []string{"some", "-text"},
		//	expectedOffset: []uint32{1, 6},
		//	exactMatch:     true,
		//},
		{
			name: "given the input string ' soadsfme -text', it should return 1 token with exact match",
			snippet: &pb.Snippet{
				Text: "£ some -text",
			},
			expectedText:   []string{ "£", "some", "-text"},
			expectedOffset: []uint32{0, 2, 7},
			exactMatch:     true,
		},

		//


		//{
		//	name: "given the input string 'some-text', it should return 2 tokens",
		//	snippet: &pb.Snippet{
		//		Text: "some-text",
		//	},
		//	expectedText:   []string{"some-text"},
		//	expectedOffset: []uint32{0},
		//	exactMatch:     true,
		//},

		//{
		//	name: "Text with '-' should not split on '-' by default",
		//	snippet: &pb.Snippet{
		//		Text: "Halogen-bonding-triggered supramolecular gel formation.",
		//	},
		//	expectedText:   []string{"Halogen", "bonding", "triggered", "supramolecular", "gel", "formation"},
		//	expectedOffset: 0,
		//	exactMatch: false,
		//},
		//{
		//	name: "alphabetty spagetty",
		//	snippet: &pb.Snippet{
		//		Text: "AaBbCcDdEeFfGgHhIiJjKkLlMmNnOoPpQqRrSsTtUuVvWwXxYyZzZ0123456789",
		//	},
		//	expectedText:   []string{"AaBbCcDdEeFfGgHhIiJjKkLlMmNnOoPpQqRrSsTtUuVvWwXxYyZzZ0123456789"},
		//	expectedOffset: 0,
		//	exactMatch: false,
		//},
		//{
		//	name: "exact match test",
		//	snippet: &pb.Snippet{
		//		Text: "Halogen-bonding-triggered supramolecular gel formation.",
		//	},
		//	expectedText:   []string{"Halogen-bonding-triggered", "supramolecular", "gel", "formation."},
		//	expectedOffset: 0,
		//	exactMatch: true,
		//},

	} {
		var actualSnippets []*pb.Snippet
		callback := func(snippet *pb.Snippet) error {
			actualSnippets = append(actualSnippets, snippet)
			return nil
		}
		err := Tokenize(test.snippet, callback, test.exactMatch)

		assert.Equal(t, len(test.expectedText), len(actualSnippets), test.name)

		fmt.Println("actual snippets:", actualSnippets, "expected: ", test.expectedText)

		for i, actualSnippet := range actualSnippets {

			fmt.Println(i)
			fmt.Println(actualSnippet.Text, test.name)
			//fmt.Println("expected offset: ", test.expectedOffset[i], "actual offset", actualSnippet.Offset)
			assert.Equal(t, test.expectedOffset[i], actualSnippet.Offset)
			assert.True(t, contains(actualSnippet.Text, test.expectedText), test.name)
		}

		assert.NoError(t, err, test.name)
	}
}

func contains(needle string, haystack []string) bool {
	for _, v := range haystack {
		if v == needle {
			return true
		}
	}
	return false
}
