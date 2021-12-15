package text

import (
	"github.com/blevesearch/segment"
	"gitlab.mdcatapult.io/informatics/software-engineering/entity-recognition/go/gen/pb"
	"strings"
	"unicode/utf8"
)

const NonAlphaNumericChar = 0

// Tokenize
/**
	Tokenize splits snippet.Text into tokens and calls onToken for each token found.
	onToken can be used to, for example, send tokens to a recognizer.
	Each token's offset (position in snippet.Text) is calculated and set.

	exactMatch controls whether the tokens are split only on whitespace or not.
	E.g. with exactMatch, "some-text" is a single token. Without exact match, it is
	three tokens: "some", "-", "text".
**/
func Tokenize(
	snippet *pb.Snippet,
	onToken func(*pb.Snippet) error,
	exactMatch bool,
) error {

	segmenter := segment.NewWordSegmenterDirect([]byte(snippet.GetText()))

	if exactMatch {
		if err := onExactMatch(segmenter, onToken, snippet); err != nil {
			return err
		}
	} else {
		if err := onNonExactMatch(segmenter, onToken, snippet); err != nil {
			return err
		}
	}

	return nil
}

// onExactMatch combines adjacent non-whitespace tokens in to one token, this behaviour excludes terms like
// 'copper-oxide' if our dictionary has both 'copper' and 'oxide', but not 'copper-oxide'.
// The rationale for this is that 'copper-oxide' is something fundamentally different to 'copper' or 'oxide'.
// Given the text: 'apple-pie' returns one token: 'apple-pie,'
func onExactMatch(
	segmenter *segment.Segmenter,
	onToken func(*pb.Snippet) error,
	snippet *pb.Snippet,
) error {

	// Given the snippet text 'apple-pie', the segmenter will split this in to three segments: 'apple', '-', 'pie'
	// As the segmenter advances through the text, we add these segments to the string builder, but only allow taking an offset
	// at the start of the snippet text (first segment), or after a whitespace character.
	// In the example 'apple-pie' we assign the snippet's start offset to the position of the 'a' character of 'apple'.
	var canSetOffset = true
	var snippetOffset = uint32(0)
	builder := &strings.Builder{}

	var position = uint32(0)

	for segmenter.Segment() {

		if err := segmenter.Err(); err != nil {
			return err
		}

		textBytes := segmenter.Bytes()

		switch segmenter.Type() {
		case NonAlphaNumericChar:
			if isWhitespace(textBytes[0]) {
				if err := sendSnippetAndResetBuilder(snippet, snippetOffset, builder, onToken); err != nil {
					return err
				}

				canSetOffset = true // after whitespace we can always set a snippet's offset
			} else {
				if err := writeTextToBufferAndUpdateOffset(&canSetOffset, &snippetOffset, position, textBytes, builder); err != nil {
					return err
				}
			}
		default:
			if err := writeTextToBufferAndUpdateOffset(&canSetOffset, &snippetOffset, position, textBytes, builder); err != nil {
				return err
			}
		}
		position += numCharsInSegment(segmenter.Text())
	}

	// write any remaining text in the string builder to a new snippet
	if err := sendSnippetAndResetBuilder(snippet, snippetOffset, builder, onToken); err != nil {
		return err
	}

	return nil
}

// We can set a new snippet's offset at the start of the input snippet's text, or after whitespace.
// On every call to this function we add the text from the segmenter to our buffer (string builder).
func writeTextToBufferAndUpdateOffset(
	canSetOffset *bool,
	snippetOffset *uint32,
	position uint32,
	textBytes []byte,
	builder *strings.Builder) error {

	if *canSetOffset {
		*snippetOffset = position // we will use this as the start position of a snippet
		*canSetOffset = false
	}

	_, err := builder.Write(textBytes)

	return err
}

// onNonExactMatch creates a snippet for every non whitespace character.
// Given the snippet text '  Partick Thistle F.C' returns the snippets 'Partick', 'Thistle', 'F', '.', 'C'
func onNonExactMatch(
	segmenter *segment.Segmenter,
	onToken func(*pb.Snippet) error,
	snippet *pb.Snippet,
) error {

	var snippetOffset = uint32(0)

	for segmenter.Segment() {

		if err := segmenter.Err(); err != nil {
			return err
		}

		textBytes := segmenter.Bytes()

		switch segmenter.Type() {
		case NonAlphaNumericChar:
			if !isWhitespace(textBytes[0]) {
				if err := createSnippetAndSendToCallback(textBytes, onToken, snippet, snippetOffset); err != nil {
					return err
				}
			}
		default:
			if err := createSnippetAndSendToCallback(textBytes, onToken, snippet, snippetOffset); err != nil {
				return err
			}
		}

		snippetOffset += numCharsInSegment(segmenter.Text())
	}

	return nil
}

func createSnippetAndSendToCallback(
	textBytes []byte,
	onToken func(*pb.Snippet) error,
	snippet *pb.Snippet,
	snippetOffset uint32,
) error {

	newSnippet := createSnippet(snippet, snippetOffset, string(textBytes))
	if err := onToken(newSnippet); err != nil {
		return err
	}

	return nil
}

func numCharsInSegment(text string) uint32 {
	return uint32(utf8.RuneCountInString(text))
}

func sendSnippetAndResetBuilder(
	snippet *pb.Snippet,
	snippetOffset uint32,
	builder *strings.Builder,
	onToken func(*pb.Snippet) error) error {

	if builder.Len() > 0 {
		newSnippet := createSnippet(snippet, snippetOffset, builder.String())
		if err := onToken(newSnippet); err != nil {
			return err
		}
		builder.Reset()
	}
	return nil
}

func isWhitespace(b byte) bool {
	whitespaceBoundary := byte(32)
	return b <= whitespaceBoundary
}

func createSnippet(
	snippet *pb.Snippet,
	snippetOffset uint32,
	text string,
) *pb.Snippet {
	return &pb.Snippet{
		Text:   text,
		Offset: snippet.GetOffset() + snippetOffset,
		Xpath:  snippet.GetXpath(),
	}
}
