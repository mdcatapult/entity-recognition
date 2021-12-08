package text

import (
	"github.com/stretchr/testify/assert"
	"gitlab.mdcatapult.io/informatics/software-engineering/entity-recognition/go/gen/pb"
	"testing"
)

func Test_ExactMatch(t *testing.T) {

	for _, test := range []struct {
		name           string
		snippet        *pb.Snippet
		expectedText   []string
		expectedOffset []uint32
	}{
		{
			name: "text with a special character and preceding/trailing spaces",
			snippet: &pb.Snippet{
				Text: " £ some text ",
			},
			expectedText:   []string{"£", "some", "text"},
			expectedOffset: []uint32{1, 3, 8},
		},
		{
			name: "text with alphanumeric and special characters",
			snippet: &pb.Snippet{
				Text: "some-text$",
			},
			expectedText:   []string{"some-text$"},
			expectedOffset: []uint32{0},
		},
		{
			name: "text starting with non alpha char, containing alpha and non alpha, ending in space",
			snippet: &pb.Snippet{
				Text: "- apple !@£ pie-face ",
			},
			expectedText:   []string{"-", "apple", "!@£", "pie-face"},
			expectedOffset: []uint32{0, 2, 8, 12},
		},
		{
			name: "exact match test",
			snippet: &pb.Snippet{
				Text: "Halogen-bonding-triggered supramolecular gel formation.",
			},
			expectedText:   []string{"Halogen-bonding-triggered", "supramolecular", "gel", "formation."},
			expectedOffset: []uint32{0, 26, 41, 45},
		},
		{
			name: "exact match test with existing offset",
			snippet: &pb.Snippet{
				Text:   "Halogen-bonding-triggered supramolecular gel formation.",
				Offset: 100,
			},
			expectedText:   []string{"Halogen-bonding-triggered", "supramolecular", "gel", "formation."},
			expectedOffset: []uint32{100, 126, 141, 145},
		},
		{
			name: "given some greek characters, etc",
			snippet: &pb.Snippet{
				Text: "βωα -νπψ- lamb ανπψ",
			},
			expectedText:   []string{"βωα", "-νπψ-", "lamb", "ανπψ"},
			expectedOffset: []uint32{0, 4, 10, 15},
		},
	} {
		var tokens []*pb.Snippet
		callback := func(snippet *pb.Snippet) error {
			tokens = append(tokens, snippet)
			return nil
		}

		err := Tokenize(test.snippet, callback, true)

		assert.NoError(t, err, test.name)
		assert.Equal(t, len(test.expectedText), len(tokens), test.name)

		for i, token := range tokens {
			assert.Equal(t, test.expectedOffset[i], token.Offset, test.name)

			assert.True(t, contains(token.Text, test.expectedText), test.name)
		}

	}
}

func Test_Non_ExactMatch(t *testing.T) {

	for _, test := range []struct {
		name           string
		snippet        *pb.Snippet
		expectedText   []string
		expectedOffset []uint32
	}{
		{
			name: "given non-exact match it should break text on '-'",
			snippet: &pb.Snippet{
				Text: "some-text",
			},
			expectedText:   []string{"some", "-", "text"},
			expectedOffset: []uint32{0, 4, 5},
		},
		{
			name: "given non-exact match it should break text on '-' with existing offset",
			snippet: &pb.Snippet{
				Text:   "some-text",
				Offset: 100,
			},
			expectedText:   []string{"some", "-", "text"},
			expectedOffset: []uint32{100, 104, 105},
		},
		{
			name: "given non-exact match should handle spaces",
			snippet: &pb.Snippet{
				Text: "some text",
			},
			expectedText:   []string{"some", "text"},
			expectedOffset: []uint32{0, 5},
		},
		{
			name: "given non-exact match should handle special chars",
			snippet: &pb.Snippet{
				Text: "βωα βωα hello",
			},
			expectedText:   []string{"βωα", "βωα", "hello"},
			expectedOffset: []uint32{0, 4, 8},
		},
		{
			name: "given non-exact match should handle trailing and leading spaces",
			snippet: &pb.Snippet{
				Text: " some -text some-text ",
			},
			expectedText:   []string{"some", "-", "text", "some", "-", "text"},
			expectedOffset: []uint32{1, 6, 7, 12, 16, 17},
		},
	} {
		var tokens []*pb.Snippet
		callback := func(snippet *pb.Snippet) error {
			tokens = append(tokens, snippet)
			return nil
		}

		err := Tokenize(test.snippet, callback, false)

		assert.NoError(t, err)
		assert.Equal(t, len(test.expectedText), len(tokens), test.name)

		for i, token := range tokens {
			assert.Equal(t, int(test.expectedOffset[i]), int(token.Offset), test.name)

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
