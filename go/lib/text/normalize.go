package text

import (
	"strings"

	"gitlab.mdcatapult.io/informatics/software-engineering/entity-recognition/go/gen/pb"
	"golang.org/x/text/unicode/norm"
)

var TokenDelimiters = map[byte]struct{}{
	'(':  {},
	')':  {},
	'{':  {},
	'}':  {},
	'[':  {},
	']':  {},
	'"':  {},
	'\'': {},
	':':  {},
	';':  {},
	',':  {},
	'.':  {},
	'?':  {},
	'!':  {},
}

func IsTokenDelimiter(b byte) bool {
	_, ok := TokenDelimiters[b]
	return ok
}

func LastChar(in string) byte {
	return in[len(in)-1]
}

func RemoveLastChar(in string) string {
	return in[:len(in)-1]
}

func RemoveFirstChar(in string) string {
	return in[1:]
}

func NormalizeSnippet(snippet *pb.Snippet) bool {
	if snippet == nil {
		return false
	}

	var sentenceEnd bool
	var offset uint32
	snippet.Token, sentenceEnd, offset = NormalizeString(snippet.Token)
	snippet.Offset += offset

	return sentenceEnd
}

func NormalizeString(token string) (string, bool, uint32) {
	var offset uint32 = 0
	var sentenceEnd = false

	// Check length so we dont get a seg fault
	if len(token) == 0 {
		return "", false, 0
	} else if len(token) == 1 {
		return "", IsTokenDelimiter(token[0]), offset
	}

	// remove quotes, brackets etc. from start and increase offset if so.
	if IsTokenDelimiter(token[0]) {
		offset += 1
		token = RemoveFirstChar(token)
	}

	// remove quotes, brackets etc. from end
	if IsTokenDelimiter(LastChar(token)) {
		token = RemoveLastChar(token)
		sentenceEnd = true
	}

	// normalise the bytes to NFKC
	token = norm.NFKC.String(token)
	token = strings.ToLower(token)

	return token, sentenceEnd, offset
}
