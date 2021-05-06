package main

import (
	"fmt"
	"github.com/go-redis/redis"
	"gitlab.mdcatapult.io/informatics/software-engineering/entity-recognition/gen/pb"
	"gitlab.mdcatapult.io/informatics/software-engineering/entity-recognition/lib"
	"google.golang.org/grpc"
	"io"
	"io/ioutil"
	"net"
	"regexp"

	"gopkg.in/yaml.v2"
)

type recogniser struct {
	pb.UnimplementedRecognizerServer
	redisClient *redis.Client
}

func (r recogniser) Recognize(stream pb.Recognizer_RecognizeServer) error {
	for {
		token, err := stream.Recv()
		if err == io.EOF {
			break
		} else if err != nil {
			return err
		}
		for name, re := range regexps {
			lib.Normalize(token)
			if re.Match(token.GetData()) {
				err := stream.Send(&pb.RecognizedEntity{
					Entity:     string(token.GetData()),
					Position:   token.GetOffset(),
					Type:       name,
				})
				if err != nil {
					return err
				}
			}
		}
	}
	return nil
}

var regexps map[string]*regexp.Regexp
var regexpStringMap = make(map[string]string)

func init() {
	b, err := ioutil.ReadFile("cmd/regexer/regexps.yml")
	if err != nil {
		panic(err)
	}

	if err := yaml.Unmarshal(b, &regexpStringMap); err != nil {
		panic(err)
	}

	regexps = make(map[string]*regexp.Regexp)
	for name, uncompiledRegexp := range regexpStringMap {
		regexps[name] = regexp.MustCompile(uncompiledRegexp)
	}
}

func main() {

	fmt.Println("Serving...")
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", 50053))
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