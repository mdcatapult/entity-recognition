package text

import (
	"bytes"
	"io"

	"github.com/blevesearch/segment"
	"gitlab.mdcatapult.io/informatics/software-engineering/entity-recognition/go/gen/pb"
)

// Tokenize delimits text by whitespace and executes a callback for every token it
// finds.
func Tokenize(snippet *pb.Snippet, onToken func(*pb.Snippet) error, exactMatch bool) error {
	// segmenter is a utf8 word boundary segmenter. Instead of splitting on all
	// word boundaries, we check first that the boundary is whitespace.
	segmenter := segment.NewWordSegmenterDirect([]byte(snippet.GetText()))
	buf := bytes.NewBuffer([]byte{})
	var currentToken []byte
	var position uint32 = 0

	// token	exact_match		non-exact
	// 'some'	add				add, read
	// '-'		add				add, read
	// 'text	add				add, read
	//  end		read			add, read (doesn't return anything)


	// token	exact_match		non-exact
	// ' '		discard
	// 'some'	add
	// ' '		discard
	//  '-'		add
	// 'text'	add
	// end		read



	for segmenter.Segment() {
		tokenBytes := segmenter.Bytes()
		tokenType := segmenter.Type()



		switch tokenType {
		case 0:
			if exactMatch {
				// word boundary character
				if tokenBytes[0] > 32 {
					// not whitespace
					if _, err := buf.Write(tokenBytes); err != nil {
						return err
					}
					break
				} else { // is whitespace
					readBufferAndWrite(currentToken, buf, snippet, position, onToken, tokenBytes)
				}

			} else {
				if _, err := buf.Write(tokenBytes); err != nil {
					return err
				}
				break
			}

		default:
			// anything but a word boundary (i.e. '-', 'hello')
			// write to buffer
			if _, err := buf.Write(tokenBytes); err != nil {
				return err
			}
		}

		if !exactMatch {
			readBufferAndWrite(currentToken, buf, snippet, position, onToken, tokenBytes)
		}


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
	}
	return nil
}

func readBufferAndWrite(
	currentToken []byte,
	buf *bytes.Buffer,
	snippet *pb.Snippet,
	position uint32,
	onToken func(*pb.Snippet) error,
	tokenBytes []byte,
	) error {

	var err error
	currentToken, err = buf.ReadBytes(0)
	if err != nil && err != io.EOF {
		return err
	}

	// if currentToken has contents, create a snippet and execute the callback.
	if len(currentToken) > 0 {
		token := &pb.Snippet{
			Text:   string(currentToken),
			Offset: snippet.GetOffset() + position,
			Xpath:  snippet.Xpath,
		}
		err := onToken(token)
		if err != nil {
			return err
		}

		// increment the position
		position += uint32(len(tokenBytes)) // + len(currentToken))
		// reset the currentToken value
		currentToken = []byte{}
	}

	return nil
}

