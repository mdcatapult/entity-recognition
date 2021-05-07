package lib

import (
	"gitlab.mdcatapult.io/informatics/software-engineering/entity-recognition/gen/pb"
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
	if snippet == nil || len(snippet.GetData()) == 0 {
		return false
	} else if len(snippet.GetData()) == 1 {
		_, ok := endSentencePunctuation[snippet.Data[0]]
		return ok
	}
	if _, ok := enclosingCharacters[snippet.Data[0]]; ok {
		snippet.Offset += 1
		snippet.Data = snippet.Data[1:]
	}
	var sentenceEnd bool
	if _, ok := midSentencePunctuation[snippet.Data[len(snippet.Data)-1]]; ok {
		snippet.Data = snippet.Data[:len(snippet.Data)-1]
	} else if _, ok := endSentencePunctuation[snippet.Data[len(snippet.Data)-1]]; ok {
		sentenceEnd = true
		snippet.Data = snippet.Data[:len(snippet.Data)-1]
	}
	snippet.Data = norm.NFKD.Bytes(snippet.Data)
	return sentenceEnd
}