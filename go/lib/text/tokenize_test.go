package text

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"gitlab.mdcatapult.io/informatics/software-engineering/entity-recognition/go/gen/pb"
	"testing"
)

func Test_Tokenise2(t *testing.T) {

	for _, test := range []struct {
		name           string
		snippet        *pb.Snippet
		expectedText   []string
		expectedOffset []uint32
		exactMatch     bool
	}{
		{
			name: "text with a special character and preceding/trailing spaces",
			snippet: &pb.Snippet{
				Text: " £ some text ",
			},
			expectedText:   []string{"£", "some", "text"},
			expectedOffset: []uint32{1, 3, 8},
			exactMatch:     true,
		},
		{
			name: "text with alphanumeric and special characters",
			snippet: &pb.Snippet{
				Text: "some-text$",
			},
			expectedText:   []string{"some-text$"},
			expectedOffset: []uint32{0},
			exactMatch:     true,
		},
		{
			name: "text starting with non alpha char, containing alpha and non alpha, ending in space",
			snippet: &pb.Snippet{
				Text: "- apple !@£ pie-face ",
			},
			expectedText:   []string{"-", "apple", "!@£", "pie-face"},
			expectedOffset: []uint32{0, 2, 8, 12},
			exactMatch:     true,
		},
		{
			name: "exact match test",
			snippet: &pb.Snippet{
				Text: "Halogen-bonding-triggered supramolecular gel formation.",
			},
			expectedText:   []string{"Halogen-bonding-triggered", "supramolecular", "gel", "formation."},
			expectedOffset: []uint32{0, 26, 41, 45},
			exactMatch:     true,
		},
		{
			name: "given some greek characters, etc",
			snippet: &pb.Snippet{
				Text: "βωα -νπψ- lamb ανπψ",
			},
			expectedText:   []string{"βωα", "-νπψ-", "lamb", "ανπψ"},
			expectedOffset: []uint32{0, 4, 10, 15},
			exactMatch:     true,
		},
	} {
		var actualSnippets []*pb.Snippet
		callback := func(snippet *pb.Snippet) error {
			actualSnippets = append(actualSnippets, snippet)
			return nil
		}

		tokens := ExactMatch(test.snippet, callback)
		fmt.Println(tokens)

		for i, token := range tokens {
			fmt.Println("-")
			//fmt.Println(token)

			fmt.Println(i)
			fmt.Println(token.Text, test.name)
			//fmt.Println("expected offset: ", test.expectedOffset[i], "actual offset", actualSnippet.Offset)
			assert.Equal(t, test.expectedOffset[i], token.Offset)
			assert.True(t, contains(token.Text, test.expectedText), test.name)
		}

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
