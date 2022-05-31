/*
 * Copyright 2022 Medicines Discovery Catapult
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *     http://www.apache.org/licenses/LICENSE-2.0
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package text

import (
	"strings"

	"gitlab.mdcatapult.io/informatics/software-engineering/entity-recognition/go/gen/pb"
	"golang.org/x/text/unicode/norm"
)

// EnclosingCharacters is a map of chars to counterparts, so that when one is removed during normalizing, we can remove
// the counterpart.
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

// TokenDelimiters is a collection of characters which delimit tokens during normalization.
// There is no need to include whitespace because the tokeniser will already have split tokens up
// based on whitespace.
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

// NormalizeSnippet runs NormalizeString on a snippets text, and sets normalizedText and bumps the offset if appropriate.
func NormalizeSnippet(snippet *pb.Snippet) (compoundTokenEnd bool) {
	var removedFirstChar bool
	snippet.NormalisedText, compoundTokenEnd, removedFirstChar = NormalizeString(snippet.Text)

	if removedFirstChar {
		snippet.Offset++
	}

	return compoundTokenEnd
}

//NormalizeAndLowercaseString normalizes the given string and returns the result along with
// whether this is the end of a compound token.
func NormalizeAndLowercaseString(inputString string) (normalizedToken string, compoundTokenEnd bool) {
	normalizedToken, compoundTokenEnd, _ = NormalizeString(inputString)
	normalizedToken = strings.ToLower(normalizedToken)
	return normalizedToken, compoundTokenEnd
}

//NormalizeString
/* NormalizeString normalizes the argument and returns the result, along with whether this is the end of
* a compound token based on TokenDelimiters, and whether the first char was removed (useful for adjusting offsets on snippets).
*
* To 'normalize' is to remove certain trailing and leading characters which are not part of the meaning of the
* wider string, e.g. 'aspirin)' would have the trailing bracket removed.
*
* This is required for sending tokens to recognisers because these trailing / leading chars would cause a
* false-negative in the dictionary lookup. i.e. 'aspirin)' is not going to be in a dictionary whereas 'aspirin' might be.
*
* This should not be required for leadmine, which has its own settings to normalize input tokens.
 */
func NormalizeString(token string) (normalizedToken string, compoundTokenEnd, removedFirstChar bool) {
	// Check length so we dont get a seg fault
	if len(token) == 0 {
		return "", false, false
	} else if _, ok := EnclosingCharacters[token[0]]; ok && len(token) == 1 {
		_, ok := TokenDelimiters[token[0]]
		return "", ok, false
	}

	// remove quotes, brackets etc. from start
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
			token = RemoveFirstChar(token)
		}
		if removeLastChar && len(token) > 0 {
			token = RemoveLastChar(token)
		}
	}

	if len(token) == 0 {
		return "", false, false
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

	return normalizedToken, compoundTokenEnd, removedFirstChar
}
