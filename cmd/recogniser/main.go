package main

import (
	"fmt"
	"gitlab.mdcatapult.io/informatics/software-engineering/entity-recognition/gen/pb"
	"google.golang.org/grpc"
	"io"
	"net"
)

type recogniser struct {
	pb.UnimplementedRecognizerServer
}

func (r recogniser) Recognize(stream pb.Recognizer_RecognizeServer) error {
	for {
		token, err := stream.Recv()
		if err == io.EOF {
			break
		} else if err != nil {
			return err
		}

		entity := &pb.RecognizedEntity{
			Entity:     token.Data,
			Position:   token.Offset,
			Type:       "unimplemented",
			ResolvedTo: "unimplemented",
		}
		if err := stream.Send(entity); err != nil {
			return err
		}
	}
	return nil
}

func main() {

	fmt.Println("Serving...")
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", 50052))
	if err != nil {
		panic(err)
	}
	var opts []grpc.ServerOption
	grpcServer := grpc.NewServer(opts...)
	pb.RegisterRecognizerServer(grpcServer, recogniser{})
	err = grpcServer.Serve(lis)
	if err != nil {
		panic(err)
	}
}