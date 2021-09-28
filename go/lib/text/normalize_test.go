package text

import (
	"github.com/stretchr/testify/assert"
	"gitlab.mdcatapult.io/informatics/software-engineering/entity-recognition/go/gen/pb"
	"testing"
)

func TestNormalizeSnippet(t *testing.T) {
	type args struct {
		snippet *pb.Snippet
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NormalizeSnippet(tt.args.snippet); got != tt.want {
				t.Errorf("NormalizeSnippet() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNormalizeString(t *testing.T) {
	tests := []struct {
		name  				string
		inputToken          string
		expectedToken       string
		expectedSentenceEnd bool
		expectedOffset      uint32
	}{
		{
			name: "empty string",
			inputToken: "",
			expectedToken: "",
			expectedSentenceEnd: false,
			expectedOffset: 0,
		},
		{
			name: "start with end sentence",
			inputToken: ".",
			expectedToken: "",
			expectedSentenceEnd: true,
			expectedOffset: 0,
		},
		{
			name: "start with enclosing character",
			inputToken: "(hello",
			expectedToken: "hello",
			expectedSentenceEnd: false,
			expectedOffset: 1,
		},
		{
			name: "end with enclosing character",
			inputToken: "hello)",
			expectedToken: "hello",
			expectedSentenceEnd: true,
			expectedOffset: 0,
		},
		{
			name: "normalize unicode characters",
			inputToken: "x²",
			expectedToken: "x2",
			expectedSentenceEnd: false,
			expectedOffset: 0,
		},
		{
			name: "lowercase",
			inputToken: "Hello",
			expectedToken: "hello",
			expectedSentenceEnd: false,
			expectedOffset: 0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actualToken, actualSentenceEnd, actualOffset := NormalizeString(tt.inputToken)
			assert.Equal(t, tt.expectedToken, actualToken)
			assert.Equal(t, tt.expectedSentenceEnd, actualSentenceEnd)
			assert.Equal(t, tt.expectedOffset, actualOffset)
		})
	}
}
