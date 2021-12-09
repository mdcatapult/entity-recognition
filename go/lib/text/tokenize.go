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
	builder := &strings.Builder{}

	var position = uint32(0)
	var snippetOffset = uint32(0)
	var canSetOffset = true

	for segmenter.Segment() {

		if err := segmenter.Err(); err != nil {
			return err
		}

		segmentBytes := segmenter.Bytes()

		switch segmenter.Type() {
		case NonAlphaNumericChar:
			if isWhitespace(segmentBytes[0]) {
				if builder.Len() > 0 { // if we have something in the buffer make a new newSnippet
					if err := onToken(createToken(snippet, snippetOffset, builder.String())); err != nil {
						return err
					}
					builder.Reset()
				}

				canSetOffset = true // after whitespace we can always add a snippet index
			} else {
				if err := writeTextToBufferAndUpdateOffset(&canSetOffset, &snippetOffset, position, segmentBytes, builder); err != nil {
					return err
				}
				if !exactMatch {
					canSetOffset = true // after whitespace we can always add a snippet index
					if err := onToken(createToken(snippet, position, builder.String())); err != nil {
						return err
					}
					builder.Reset()
				}
			}
		default:
			if err := writeTextToBufferAndUpdateOffset(&canSetOffset, &snippetOffset, position, segmentBytes, builder); err != nil {
				return err
			}

			if !exactMatch {
				if err := onToken(createToken(snippet, snippetOffset, builder.String())); err != nil {
					return err
				}
				builder.Reset()
			}
		}

		position += uint32(utf8.RuneCountInString(segmenter.Text()))

	}

	// if we have something in the buffer once the segmenter has finished, make a new snippet
	if builder.Len() > 0 { // if we have something at the buffer make a new newSnippet
		if err := onToken(createToken(snippet, snippetOffset, builder.String())); err != nil {
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

func createToken(
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

func writeTextToBufferAndUpdateOffset(
	canSetOffset *bool,
	snippetOffset *uint32,
	position uint32,
	text []byte,
	builder *strings.Builder) error {

	if *canSetOffset {
		*snippetOffset = position // we will use this as the start position of a snippet
		*canSetOffset = false
	}

	_, err := builder.Write(text)

	return err
}
