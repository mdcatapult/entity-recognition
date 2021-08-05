package text

import (
	"gitlab.mdcatapult.io/informatics/software-engineering/entity-recognition/go/gen/pb"
	"golang.org/x/text/unicode/norm"
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

func Normalize(snippet *pb.Snippet) bool {

	// Check length so we dont get a seg fault
	if snippet == nil || len(snippet.Token) == 0 {
		return false
	} else if len(snippet.Token) == 1 {
		return IsEndSentencePunctuation(snippet.Token[0])
	}

	// remove quotes, brackets etc. from start and increase offset if so.
	if IsEnclosingCharacter(snippet.Token[0]) {
		snippet.Offset += 1
		snippet.Token = RemoveFirstChar(snippet.Token)
	}

	// remove quotes, brackets etc. from end
	if IsEnclosingCharacter(LastChar(snippet.Token)) {
		snippet.Token = RemoveLastChar(snippet.Token)
	}

	// Remove mid or end sentence punctuation.
	var sentenceEnd bool
	if IsMidSentencePunctuation(LastChar(snippet.Token)) {
		snippet.Token = RemoveLastChar(snippet.Token)
	} else if IsEndSentencePunctuation(LastChar(snippet.Token)) {
		sentenceEnd = true
		snippet.Token = RemoveLastChar(snippet.Token)
	}

	// normalise the bytes to NFKC
	snippet.Token = norm.NFKC.String(snippet.Token)
	return sentenceEnd
}
