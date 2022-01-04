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
	';':  0, // TODO why are these zero - A they have no 'counterpart'?
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

// NormalizeAndLowercaseSnippet normalizes snippet and returns whether this is the end of a compound token.
func NormalizeAndLowercaseSnippet(snippet *pb.Snippet) (compoundTokenEnd bool) {
	compoundTokenEnd = NormalizeSnippet(snippet)
	snippet.NormalisedText = strings.ToLower(snippet.NormalisedText)
	return compoundTokenEnd
}

// NormalizeSnippet runs NormalizeString on a snippets text, and returns whether this is the end
// of a compound token.
func NormalizeSnippet(snippet *pb.Snippet) (compoundTokenEnd bool) {
	if snippet == nil {
		return false
	}

	var offset uint32
	snippet.NormalisedText, compoundTokenEnd, offset = NormalizeString(snippet.Text)

	//fmt.Println("OFFSET:", offset)
	snippet.Offset += offset

	return compoundTokenEnd
}

//NormalizeAndLowercaseString normalizes the given string and returns the result along with
// whether this is the end of a compound token and
func NormalizeAndLowercaseString(inputString string) (normalizedToken string, compoundTokenEnd bool, offset uint32) {
	normalizedToken, compoundTokenEnd, offset = NormalizeString(inputString)
	normalizedToken = strings.ToLower(normalizedToken)
	return
}

// NormalizeString TODO what is offset?
func NormalizeString(token string) (normalizedToken string, compoundTokenEnd bool, offset uint32) {

	// Check length so we dont get a seg fault
	if len(token) == 0 {
		return "", false, 0
	} else if _, ok := EnclosingCharacters[token[0]]; ok && len(token) == 1 {
		_, ok := TokenDelimiters[token[0]]
		return "", ok, offset
	}

	// remove quotes, brackets etc. from start and increase offset if so. // TODO what is the offset?
	// If we find the counterpart character (e.g. "]" being the counterpart of "[")
	// within the token, we don't remove it.
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
		if removeLastChar && len(token) > 0 {
			token = RemoveLastChar(token)
		}
	}

	if len(token) == 0 {
		return "", false, offset
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
		if removeLastChar && len(token) > 0 {
			_, compoundTokenEnd = TokenDelimiters[LastChar(token)]
			token = RemoveLastChar(token)
		}
	}

	// normalise the bytes to NFKC
	normalizedToken = norm.NFKC.String(token)

	return normalizedToken, compoundTokenEnd, offset
}
