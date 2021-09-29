package text

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNormalizeAndLowercaseString(t *testing.T) {
	tests := []struct {
		name                string
		inputToken          string
		expectedToken       string
		expectedSentenceEnd bool
		expectedOffset      uint32
	}{
		{
			name:                "empty string",
			inputToken:          "",
			expectedToken:       "",
			expectedSentenceEnd: false,
			expectedOffset:      0,
		},
		{
			name:                "start with end sentence",
			inputToken:          ".",
			expectedToken:       "",
			expectedSentenceEnd: true,
			expectedOffset:      0,
		},
		{
			name:                "start with enclosing character",
			inputToken:          "(hello",
			expectedToken:       "hello",
			expectedSentenceEnd: false,
			expectedOffset:      1,
		},
		{
			name:                "end with enclosing character",
			inputToken:          "hello)",
			expectedToken:       "hello",
			expectedSentenceEnd: true,
			expectedOffset:      0,
		},
		{
			name:                "normalize unicode characters",
			inputToken:          "x²",
			expectedToken:       "x2",
			expectedSentenceEnd: false,
			expectedOffset:      0,
		},
		{
			name:                "lowercase",
			inputToken:          "Hello",
			expectedToken:       "hello",
			expectedSentenceEnd: false,
			expectedOffset:      0,
		},
		{
			name: "starts with enclosing character and contains its counterpart",
			inputToken: "(a)-hydroxycarbamide",
			expectedToken: "(a)-hydroxycarbamide",
			expectedSentenceEnd: false,
			expectedOffset: 0,
		},
		{
			name: "starts with enclosing character and ends with its counterpart",
			inputToken: "'hello'",
			expectedToken: "hello",
			expectedSentenceEnd: false,
			expectedOffset: 1,
		},
		{
			name: "ends with enclosing character and contains its counterpart",
			inputToken: "hydroxycarbamide-(a)",
			expectedToken: "hydroxycarbamide-(a)",
			expectedSentenceEnd: false,
			expectedOffset: 0,
		},
		{
			name: "ends with enclosing character that is not a token delimiter",
			inputToken: "hello,",
			expectedToken: "hello",
			expectedSentenceEnd: false,
			expectedOffset: 0,
		},
	}
	for _, tt := range tests {
		t.Log(tt.name)

		actualToken, actualSentenceEnd, actualOffset := NormalizeAndLowercaseString(tt.inputToken)
		assert.Equal(t, tt.expectedToken, actualToken)
		assert.Equal(t, tt.expectedSentenceEnd, actualSentenceEnd)
		assert.Equal(t, tt.expectedOffset, actualOffset)
	}
}
