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

// Tokenize delimits text by whitespace and executes a callback for every token it
// finds.
func Tokenize(snippet *pb.Snippet, onToken func(*pb.Snippet) error, exactMatch bool) error {

	// token	exact_match		non-exact
	// 'some'	add				add, read
	// '-'		add				add, read
	// 'text	add				add, read
	//  end		read			add, read

	// token	exact_match		non-exact	position index
	// ' '		don't add 		don't add	+ 1
	// 'some'	add				add, read	+ len(some)
	// '-'		add				add, read	+ 1
	// 'text	add				add, read	+ len(text)
	//  end		read			add, read 	+ len(end)

	// segmenter is a utf8 word boundary segmenter. Instead of splitting on all
	// word boundaries, we check first that the boundary is whitespace.
	segmenter := segment.NewWordSegmenterDirect([]byte(snippet.GetText()))
	buf := bytes.NewBuffer([]byte{})
	var currentToken []byte
	var position uint32 = 0

	if exactMatch {
		//onExactMatch(snippet, onToken, segmenter, buf, &currentToken, &position)
	} else {
		//onNonExactMatch(snippet, onToken, segmenter, buf, &currentToken, &position)
	}

	if err := segmenter.Err(); err != nil {
		return err
	}

	currentToken, err := buf.ReadBytes(0)

	if err != nil && err != io.EOF {
		return err
	}

	if len(currentToken) > 0 {

		pbEntity := &pb.Snippet{
			Text:   string(currentToken),
			Offset: snippet.GetOffset() + position,
			Xpath:  snippet.Xpath,
		}

		err := onToken(pbEntity)
		if err != nil {
			return err
		}

		position += uint32(len(currentToken)) // + len(currentToken))

	}
	return nil
}

//////////////// <----------- ----------->

var verbose = false

func ExactMatch(
	snippet *pb.Snippet,
	onToken func(*pb.Snippet) error,
	exactMatch bool,
) []*pb.Snippet {

	segmenter := segment.NewWordSegmenterDirect([]byte(snippet.GetText()))
	buffer := bytes.NewBuffer([]byte{})

	var snippets []*pb.Snippet
	var position = uint32(0)
	var snippetOffset = uint32(0)
	var canSetOffset = true

	for segmenter.Segment() {
		segmentBytes := segmenter.Bytes()

		switch segmenter.Type() {
		case NonAlphaNumericChar:
			if isWhitespace(segmentBytes[0]) {
				if buffer.Len() > 0 { // if we have something in the buffer make a new newSnippet

					newSnippet := createSnippet(snippet, &snippetOffset, buffer)
					snippets = append(snippets, newSnippet)
					buffer.Reset()
				}

				canSetOffset = true // after whitespace we can always add a snippet index
				incrementPosition(&position, segmentBytes)
			} else {
				writeTextToBufferAndUpdateOffset(&canSetOffset, &snippetOffset, &position, &segmentBytes, buffer)

				if !exactMatch {
					canSetOffset = true // after whitespace we can always add a snippet index
					newSnippet := createSnippet(snippet, &position, buffer)
					snippets = append(snippets, newSnippet)
					buffer.Reset()
				}
				incrementPosition(&position, segmentBytes)
			}
		default:
			writeTextToBufferAndUpdateOffset(&canSetOffset, &snippetOffset, &position, &segmentBytes, buffer)

			if !exactMatch {
				newSnippet := createSnippet(snippet, &snippetOffset, buffer)
				snippets = append(snippets, newSnippet)
				buffer.Reset()
			}
			incrementPosition(&position, segmentBytes)
		}
	}

	// if we have something in the buffer once the segmenter has finished, make a new snippet
	if buffer.Len() > 0 { // if we have something at the buffer make a new newSnippet
		newSnippet := createSnippet(snippet, &snippetOffset, buffer)
		snippets = append(snippets, newSnippet)
		buffer.Reset()
	}

	return snippets
}

func NonExactMatch(snippet *pb.Snippet, onToken func(snippet2 *pb.Snippet) error) []*pb.Snippet {
	segmenter := segment.NewWordSegmenterDirect([]byte(snippet.GetText()))
	buffer := bytes.NewBuffer([]byte{})

	var snippets []*pb.Snippet
	var position = uint32(0)
	var snippetOffset = uint32(0)
	var canSetOffset = true

	for segmenter.Segment() {
		segmentBytes := segmenter.Bytes()

		switch segmenter.Type() {
		case NonAlphaNumericChar:
			if isWhitespace(segmentBytes[0]) {
				if buffer.Len() > 0 { // if we have something in the buffer make a new newSnippet
					newSnippet := createSnippet(snippet, &snippetOffset, buffer)
					snippets = append(snippets, newSnippet)
					buffer.Reset()
				}

				canSetOffset = true // after whitespace, we can always add a snippet index
				incrementPosition(&position, segmentBytes)
			} else {

				writeTextToBufferAndUpdateOffset(&canSetOffset, &snippetOffset, &position, &segmentBytes, buffer)
				newSnippet := createSnippet(snippet, &position, buffer)
				snippets = append(snippets, newSnippet)

				buffer.Reset()
				//writeTextToBufferAndUpdateOffset(&canSetOffset, &snippetOffset, &position, &segmentBytes, buffer)
				incrementPosition(&position, segmentBytes)
				//}
			}
		default:
			writeTextToBufferAndUpdateOffset(&canSetOffset, &snippetOffset, &position, &segmentBytes, buffer)

			newSnippet := createSnippet(snippet, &position, buffer)
			snippets = append(snippets, newSnippet)
			incrementPosition(&position, segmentBytes)
			buffer.Reset()
		}
	}

	// if we have something in the buffer once the segmenter has finished, make a new snippet
	if buffer.Len() > 0 { // if we have something at the buffer make a new newSnippet
		newSnippet := createSnippet(snippet, &snippetOffset, buffer)
		snippets = append(snippets, newSnippet)
		buffer.Reset()
	}

	return snippets
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

	fmt.Println(string(textBytes), "bumping pos by", utf8.RuneCountInString(string(textBytes)), "to", *position)

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

	fmt.Println("current token len: ", len(currentToken), "current token: ", string(currentToken))

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
