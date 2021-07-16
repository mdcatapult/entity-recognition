package lib

import (
	"gitlab.mdcatapult.io/informatics/software-engineering/entity-recognition/go/gen/pb"
	"golang.org/x/text/unicode/norm"
)

var enclosingCharacters = map[byte]struct{}{
	'(': {},
	')': {},
	'{': {},
	'}': {},
	'[': {},
	']': {},
	'"': {},
}

var midSentencePunctuation = map[byte]struct{}{
	':': {},
	';': {},
	',': {},
}

var endSentencePunctuation = map[byte]struct{}{
	'.': {},
	'?': {},
	'!': {},
}

func Normalize(snippet *pb.Snippet) bool {
	// Check length so we dont get a seg fault
	if snippet == nil || len(snippet.GetToken()) == 0 {
		return false
	} else if len(snippet.GetToken()) == 1 {
		_, ok := endSentencePunctuation[snippet.Token[0]]
		return ok
	}
	// remove quotes, brackets etc. from start and increase offset if so.
	if _, ok := enclosingCharacters[snippet.Token[0]]; ok {
		snippet.Offset += 1
		snippet.Token = snippet.Token[1:]
	}

	// Remove mid or end sentence punctuation.
	var sentenceEnd bool
	if _, ok := midSentencePunctuation[snippet.Token[len(snippet.Token)-1]]; ok {
		snippet.Token = snippet.Token[:len(snippet.Token)-1]
	} else if _, ok := endSentencePunctuation[snippet.Token[len(snippet.Token)-1]]; ok {
		sentenceEnd = true
		snippet.Token = snippet.Token[:len(snippet.Token)-1]
	}

	// normalise the bytes to NFKC
	snippet.Token = norm.NFKC.String(snippet.Token)
	return sentenceEnd
}
