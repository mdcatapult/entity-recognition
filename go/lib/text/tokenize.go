package text

import (
	"bytes"
	"fmt"
	"github.com/blevesearch/segment"
	"gitlab.mdcatapult.io/informatics/software-engineering/entity-recognition/go/gen/pb"
	"io"
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
	buffer := bytes.NewBuffer([]byte{})

	var position = uint32(0)
	var snippetOffset = uint32(0)
	var canSetOffset = true

	for segmenter.Segment() {
		segmentBytes := segmenter.Bytes()

		switch segmenter.Type() {
		case NonAlphaNumericChar:
			if isWhitespace(segmentBytes[0]) {
				if buffer.Len() > 0 { // if we have something in the buffer make a new newSnippet
					if err := onToken(createSnippet(snippet, &snippetOffset, buffer)); err != nil {
						return err
					}
					buffer.Reset()
				}

				canSetOffset = true // after whitespace we can always add a snippet index
				incrementPosition(&position, segmentBytes)
			} else {
				writeTextToBufferAndUpdateOffset(&canSetOffset, &snippetOffset, &position, &segmentBytes, buffer)

				if !exactMatch {
					canSetOffset = true // after whitespace we can always add a snippet index
					if err := onToken(createSnippet(snippet, &position, buffer)); err != nil {
						return err
					}
					buffer.Reset()
				}
				incrementPosition(&position, segmentBytes)
			}
		default:
			writeTextToBufferAndUpdateOffset(&canSetOffset, &snippetOffset, &position, &segmentBytes, buffer)

			if !exactMatch {
				if err := onToken(createSnippet(snippet, &snippetOffset, buffer)); err != nil {
					return err
				}
				buffer.Reset()
			}
			incrementPosition(&position, segmentBytes)
		}
	}

	// if we have something in the buffer once the segmenter has finished, make a new snippet
	if buffer.Len() > 0 { // if we have something at the buffer make a new newSnippet
		if err := onToken(createSnippet(snippet, &snippetOffset, buffer)); err != nil {
			return err
		}
		buffer.Reset()
	}

	return nil
}

func isWhitespace(b byte) bool {
	whitespaceBoundary := byte(32)
	return b <= whitespaceBoundary
}

func createSnippet(
	snippet *pb.Snippet,
	snippetOffset *uint32,
	buffer *bytes.Buffer,
) *pb.Snippet {

	finalOffset := snippet.GetOffset() + *snippetOffset

	return &pb.Snippet{
		Text:   buffer.String(),
		Offset: finalOffset,
		Xpath:  snippet.GetXpath(),
	}
}

func writeTextToBufferAndUpdateOffset(
	canSetOffset *bool,
	snippetOffset *uint32,
	position *uint32,
	segmentBytes *[]byte,
	buffer *bytes.Buffer) error {

	if *canSetOffset {
		*snippetOffset = *position // we will use this as the start position of a snippet
		*canSetOffset = false
	}

	_, err := buffer.Write(*segmentBytes)

	return err
}

func incrementPosition(position *uint32, textBytes []byte) {

	// get length of string (take account of greek chars) then update position
	numCharsInString := utf8.RuneCountInString(string(textBytes))
	*position += uint32(numCharsInString)

	//fmt.Println(string(textBytes), "bumping pos by", utf8.RuneCountInString(string(textBytes)), "to", *position)

}

func readBufferAndWriteToken(
	currentToken []byte,
	buf *bytes.Buffer,
	snippet *pb.Snippet,
	position **uint32,
	onToken func(*pb.Snippet) error,
	tokenBytes []byte,
) error {

	var err error
	currentToken, err = buf.ReadBytes(0)
	if err != nil && err != io.EOF {
		return err
	}

	//fmt.Println("current token len: ", len(currentToken), "current token: ", string(currentToken))

	// if currentToken has contents, create a snippet and execute the callback.
	if len(currentToken) > 0 {
		token := &pb.Snippet{
			Text:   string(currentToken),
			Offset: snippet.GetOffset() + **position,
			Xpath:  snippet.Xpath,
		}
		err := onToken(token)
		if err != nil {
			return err
		}

		// increment the position
		// reset the currentToken value
		currentToken = []byte{}
	}

	**position += uint32(len(tokenBytes))
	fmt.Println("after write : ", tokenBytes, string(tokenBytes), "bumping position by: ", uint32(len(tokenBytes)), "position now: ", **position)

	return nil
}
