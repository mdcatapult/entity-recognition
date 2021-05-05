package main

import (
	"bytes"
	"fmt"
	"github.com/blevesearch/segment"
	"gitlab.mdcatapult.io/informatics/software-engineering/entity-recognition/gen/pb"
	"google.golang.org/grpc"
	"io"
	"net"
)

type tokenizer struct {
	pb.UnimplementedTokenizerServer
}

func (t tokenizer) Tokenize(stream pb.Tokenizer_TokenizeServer) error {
	for {
		snippet, err := stream.Recv()
		if err == io.EOF {
			break
		} else if err != nil {
			return err
		}

		err = tokenize(snippet, func(snippet *pb.Snippet) error {
			return stream.Send(snippet)
		})
		if err != nil {
			return err
		}
	}
	return nil
}

func tokenize(snippet *pb.Snippet, onToken func(*pb.Snippet) error) error {
	segmenter := segment.NewWordSegmenterDirect([]byte(snippet.GetData()))
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
				Data:   string(currentToken),
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
			Data:   string(currentToken),
			Offset: snippet.GetOffset() + position,
		}
		err := onToken(pbEntity)
		if err != nil {
			return err
		}
	}
	return nil
}

func main() {

	fmt.Println("Serving...")
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", 50051))
	if err != nil {
		panic(err)
	}
	var opts []grpc.ServerOption
	grpcServer := grpc.NewServer(opts...)
	pb.RegisterTokenizerServer(grpcServer, tokenizer{})
	err = grpcServer.Serve(lis)
	if err != nil {
		panic(err)
	}
}