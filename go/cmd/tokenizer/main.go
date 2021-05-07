package main

import (
	"fmt"
	"gitlab.mdcatapult.io/informatics/software-engineering/entity-recognition/go/gen/pb"
	"gitlab.mdcatapult.io/informatics/software-engineering/entity-recognition/go/lib"
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

		err = lib.Tokenize(snippet, func(snippet *pb.Snippet) error {
			return stream.Send(snippet)
		})
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