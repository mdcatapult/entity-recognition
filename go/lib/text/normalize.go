package text

import (
	"strings"

	"gitlab.mdcatapult.io/informatics/software-engineering/entity-recognition/go/gen/pb"
	"golang.org/x/text/unicode/norm"
)

var EnclosingCharacters = map[byte]byte{
	'(':  ')',
	')':  '(',
	'{':  '}',
	'}':  '{',
	'[':  ']',
	']':  '[',
	'"':  '"',
	'\'': '\'',
	':':  0,
	';':  0,
	',':  0,
	'.':  0,
	'?':  0,
	'!':  0,
}

var TokenDelimiters = map[byte]struct{}{
	')': {},
	']': {},
	'}': {},
	'?': {},
	'!': {},
	'.': {},
	':': {},
	';': {},
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

func NormalizeAndLowercaseSnippet(snippet *pb.Snippet) bool {
	sentenceEnd := NormalizeSnippet(snippet)
	snippet.Token = strings.ToLower(snippet.Token)
	return sentenceEnd
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

func NormalizeAndLowercaseString(token string) (normalizedToken string, sentenceEnd bool, offset uint32) {
	normalizedToken, sentenceEnd, offset = NormalizeString(token)
	normalizedToken = strings.ToLower(normalizedToken)
	return
}

func NormalizeString(token string) (normalizedToken string, sentenceEnd bool, offset uint32) {

	// Check length so we dont get a seg fault
	if len(token) == 0 {
		return "", false, 0
	} else if _, ok := EnclosingCharacters[token[0]]; ok && len(token) == 1 {
		_, ok := TokenDelimiters[token[0]]
		return "", ok, offset
	}

	// remove quotes, brackets etc. from start and increase offset if so.
	if counterpart, ok := EnclosingCharacters[token[0]]; ok {
		removeLastChar := false
		removeFirstChar := true
		if counterpart != 0 {
			for i := 1; i < len(token); i++ {
				b := token[i]
				if b != counterpart {
					continue
				}
				if i == len(token)-1 {
					removeLastChar = true
					break
				}
				removeFirstChar = false
			}
		}
		if removeFirstChar {
			offset += 1
			token = RemoveFirstChar(token)
		}
		if removeLastChar {
			token = RemoveLastChar(token)
		}
	}

	// remove quotes, brackets etc. from end
	if counterpart, ok := EnclosingCharacters[LastChar(token)]; ok {
		removeLastChar := true
		if counterpart != 0 {
			for _, b := range token {
				if byte(b) == counterpart {
					removeLastChar = false
					break
				}
			}
		}
		if removeLastChar {
			_, sentenceEnd = TokenDelimiters[LastChar(token)]
			token = RemoveLastChar(token)
		}
	}

	// normalise the bytes to NFKC
	normalizedToken = norm.NFKC.String(token)

	return normalizedToken, sentenceEnd, offset
}
