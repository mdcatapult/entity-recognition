/*
 * Copyright 2022 Medicines Discovery Catapult
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *     http://www.apache.org/licenses/LICENSE-2.0
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

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
