package text

import (
	"bufio"
	"bytes"
	"io"

	"github.com/blevesearch/segment"
	"gitlab.mdcatapult.io/informatics/software-engineering/entity-recognition/go/gen/pb"
)

// Tokenize delimits text by whitespace and executes a callback for every token it
// finds.
func Tokenize(snippet *pb.Snippet, onToken func(*pb.Snippet) error) error {
	// segmenter is a utf8 word boundary segmenter. Instead of splitting on all
	// word boundaries, we check first that the boundary is whitespace.
	segmenter := segment.NewWordSegmenterDirect([]byte(snippet.GetToken()))
	buf := bytes.NewBuffer([]byte{})
	var currentToken []byte
	var position uint32 = 0
	for segmenter.Segment() {
		tokenBytes := segmenter.Bytes()
		tokenType := segmenter.Type()

		switch tokenType {
		case 0:
			// word boundary character
			if tokenBytes[0] > 32 {
				// not whitespace
				if _, err := buf.Write(tokenBytes); err != nil {
					return err
				}
				break
			}
			// whitespace: read the contents of the buffer into currentToken
			var err error
			currentToken, err = buf.ReadBytes(0)
			if err != nil && err != io.EOF {
				return err
			}

		default:
			// anything but a word boundary (i.e. '-', 'hello')
			// write to buffer
			if _, err := buf.Write(tokenBytes); err != nil {
				return err
			}
		}

		// if currentToken has contents, create a snippet and execute the callback.
		if len(currentToken) > 0 {
			token := &pb.Snippet{
				Token:  string(currentToken),
				Offset: snippet.GetOffset() + position,
				Xpath:  snippet.Xpath,
			}
			err := onToken(token)
			if err != nil {
				return err
			}

			// increment the position
			position += uint32(len(tokenBytes) + len(currentToken))
			// reset the currentToken value
			currentToken = []byte{}
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
			Token:  string(currentToken),
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

// TokenizeWordBoundary tokenizes on word boundary instead of whitespace and doesn't
// give any offset information.
func TokenizeWordBoundary(snippet *pb.Snippet, onToken func(*pb.Snippet) error) error {
	r := bytes.NewReader([]byte(snippet.GetToken()))
	scanner := bufio.NewScanner(r)
	scanner.Split(segment.SplitWords)
	for scanner.Scan() {
		token := scanner.Bytes()
		err := onToken(&pb.Snippet{
			Token:  string(token),
			Offset: 0,
		})
		if err != nil {
			return err
		}
	}
	return nil
}
