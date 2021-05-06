package lib

import (
	"bytes"
	"github.com/blevesearch/segment"
	"gitlab.mdcatapult.io/informatics/software-engineering/entity-recognition/gen/pb"
	"io"
)

func Tokenize(snippet *pb.Snippet, onToken func(*pb.Snippet) error) error {
	segmenter := segment.NewWordSegmenterDirect(snippet.GetData())
	buf := bytes.NewBuffer([]byte{})
	var currentToken []byte
	var position uint32 = 0
	for segmenter.Segment() {
		tokenBytes := segmenter.Bytes()
		tokenType := segmenter.Type()

		switch tokenType {
		case 0:
			if tokenBytes[0] > 32 {
				if _, err := buf.Write(tokenBytes); err != nil {
					return err
				}
				break
			}
			var err error
			currentToken, err = buf.ReadBytes(0)
			if err != nil && err != io.EOF {
				return err
			}

		default:
			if _, err := buf.Write(tokenBytes); err != nil {
				return err
			}
		}

		if len(currentToken) > 0 {
			pbEntity := &pb.Snippet{
				Data:   currentToken,
				Offset: snippet.GetOffset() + position,
			}
			err := onToken(pbEntity)
			if err != nil {
				return err
			}
			position += uint32(len(tokenBytes) + len(currentToken))
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
			Data:   currentToken,
			Offset: snippet.GetOffset() + position,
		}
		err := onToken(pbEntity)
		if err != nil {
			return err
		}
	}
	return nil
}
