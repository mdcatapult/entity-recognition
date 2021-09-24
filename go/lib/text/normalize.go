package text

import (
	"gitlab.mdcatapult.io/informatics/software-engineering/entity-recognition/go/gen/pb"
	"golang.org/x/text/unicode/norm"
	"strings"
)

var EnclosingCharacters = map[byte]struct{}{
	'(': {},
	')': {},
	'{': {},
	'}': {},
	'[': {},
	']': {},
	'"': {},
}

var MidSentencePunctuation = map[byte]struct{}{
	':': {},
	';': {},
	',': {},
}

var EndSentencePunctuation = map[byte]struct{}{
	'.': {},
	'?': {},
	'!': {},
}

func IsEndSentencePunctuation(b byte) bool {
	_, ok := EndSentencePunctuation[b]
	return ok
}

func IsMidSentencePunctuation(b byte) bool {
	_, ok := MidSentencePunctuation[b]
	return ok
}

func IsEnclosingCharacter(b byte) bool {
	_, ok := EnclosingCharacters[b]
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
		return token, IsEndSentencePunctuation(token[0]), offset
	}

	// remove quotes, brackets etc. from start and increase offset if so.
	if IsEnclosingCharacter(token[0]) {
		offset += 1
		token = RemoveFirstChar(token)
	}

	// remove quotes, brackets etc. from end
	if IsEnclosingCharacter(LastChar(token)) {
		token = RemoveLastChar(token)
	}

	// Remove mid or end sentence punctuation.
	if IsMidSentencePunctuation(LastChar(token)) {
		token = RemoveLastChar(token)
	} else if IsEndSentencePunctuation(LastChar(token)) {
		sentenceEnd = true
		token = RemoveLastChar(token)
	}

	// normalise the bytes to NFKC
	token = norm.NFKC.String(token)
	token = strings.ToLower(token)

	return token, sentenceEnd, offset
}

