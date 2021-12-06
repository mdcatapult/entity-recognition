package text

import (
	"bytes"
	"fmt"
	"io"

	"github.com/blevesearch/segment"
	"gitlab.mdcatapult.io/informatics/software-engineering/entity-recognition/go/gen/pb"
)

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
		onExactMatch(snippet, onToken, segmenter, buf, &currentToken, &position)
	} else {
		onNonExactMatch(snippet, onToken, segmenter, buf, &currentToken, &position)
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

func onNonExactMatch(
	snippet *pb.Snippet,
	onToken func(*pb.Snippet) error,
	segmenter *segment.Segmenter,
	buffer *bytes.Buffer,
	currentToken *[]byte,
	position *uint32) error {

	for segmenter.Segment() {
		tokenBytes := segmenter.Bytes()
		tokenType := segmenter.Type()

		switch tokenType {
		case 0: // non alphanumeric
			fmt.Println("non alpha: ", tokenBytes, string(tokenBytes), "position: ", *position)

			if _, err := buffer.Write(tokenBytes); err != nil {
				return err
			}
			//fmt.Println("token:", string(tokenBytes), " position:", position)

			break

		default: // alphanumeric
			// anything but a word boundary (i.e. '-', 'hello')
			// write to buffer
			fmt.Println("alpha: ", tokenBytes, string(tokenBytes), "position: ", *position)

			if _, err := buffer.Write(tokenBytes); err != nil {
				return err
			}
			//position += uint32(len(string(tokenBytes)))
			//fmt.Println("token:", string(tokenBytes), " position:", position)

		}

		readBufferAndWriteToken(*currentToken, buffer, snippet, &position, onToken, tokenBytes)

	}

	return nil
}

func onExactMatch(
	snippet *pb.Snippet,
	onToken func(*pb.Snippet) error,
	segmenter *segment.Segmenter,
	buffer *bytes.Buffer,
	currentToken *[]byte,
	position *uint32) error {

	var lastSegmentWasWhitespace = false

	for segmenter.Segment() {
		tokenBytes := segmenter.Bytes()

		switch segmenter.Type() {
		case 0: // non alphanumeric
			fmt.Println("non alpha: ", tokenBytes, string(tokenBytes), "position: ", *position)
			// word boundary character

			if tokenBytes[0] > 32 {
				fmt.Println("in non whitespace")

				// not whitespace
				if _, err := buffer.Write(tokenBytes); err != nil {
					return err
				}

				break
			} else {
				fmt.Println("in whitespace")

				lastSegmentWasWhitespace = true
				//*position += 1
				//readBufferAndWriteToken(*currentToken, buffer, snippet, &position, onToken, tokenBytes)

			}

			//if lastSegmentWasWhitespace {
			readBufferAndWriteToken(*currentToken, buffer, snippet, &position, onToken, tokenBytes)

			//}

		default: // alphanumeric
			//fmt.Println("alpha: ", tokenBytes, string(tokenBytes), "position: ", *position)

			if _, err := buffer.Write(tokenBytes); err != nil {
				return err
			}

			if lastSegmentWasWhitespace {
				readBufferAndWriteToken(*currentToken, buffer, snippet, &position, onToken, tokenBytes)
				lastSegmentWasWhitespace = false
			}
		}
	}

	return nil
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

	fmt.Println("current token len: ", len(currentToken), "current token: ", string(currentToken) )

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
