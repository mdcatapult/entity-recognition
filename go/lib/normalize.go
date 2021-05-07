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
	if snippet == nil || len(snippet.GetData()) == 0 {
		return false
	} else if len(snippet.GetData()) == 1 {
		_, ok := endSentencePunctuation[snippet.Data[0]]
		return ok
	}
	// remove quotes, brackets etc. from start and increase offset if so.
	if _, ok := enclosingCharacters[snippet.Data[0]]; ok {
		snippet.Offset += 1
		snippet.Data = snippet.Data[1:]
	}

	// Remove mid or end sentence punctuation.
	var sentenceEnd bool
	if _, ok := midSentencePunctuation[snippet.Data[len(snippet.Data)-1]]; ok {
		snippet.Data = snippet.Data[:len(snippet.Data)-1]
	} else if _, ok := endSentencePunctuation[snippet.Data[len(snippet.Data)-1]]; ok {
		sentenceEnd = true
		snippet.Data = snippet.Data[:len(snippet.Data)-1]
	}

	// normalise the bytes to NFKC
	snippet.Data = norm.NFKC.Bytes(snippet.Data)
	return sentenceEnd
}