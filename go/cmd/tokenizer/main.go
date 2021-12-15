package main

import (
	"fmt"
	"io"
	"net"

	"gitlab.mdcatapult.io/informatics/software-engineering/entity-recognition/go/gen/pb"
	"gitlab.mdcatapult.io/informatics/software-engineering/entity-recognition/go/lib/text"
	"google.golang.org/grpc"
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

		err = text.Tokenize(snippet, func(snippet *pb.Snippet) error {
			return stream.Send(snippet)
		}, false)
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
