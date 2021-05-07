package main

import (
	"fmt"
	"github.com/go-redis/redis"
	"github.com/spf13/viper"
	"gitlab.mdcatapult.io/informatics/software-engineering/entity-recognition/gen/pb"
	"gitlab.mdcatapult.io/informatics/software-engineering/entity-recognition/lib"
	"google.golang.org/grpc"
	"gopkg.in/yaml.v2"
	"io"
	"io/ioutil"
	"net"
	"path"
	"path/filepath"
	"regexp"
	"runtime"
)

type conf struct {
	LogLevel string `mapstructure:"log_level"`
	Server struct{
		GrpcPort int `mapstructure:"grpc_port"`
	}
}

var config conf
var regexps map[string]*regexp.Regexp
var regexpStringMap = make(map[string]string)

func init() {
	err := lib.InitializeConfig(map[string]interface{}{
		"log_level": "info",
		"server": map[string]interface{}{
			"grpc_port": 50053,
		},
	})
	if err != nil {
		panic(err)
	}

	err = viper.Unmarshal(&config)
	if err != nil {
		panic(err)
	}
}

func main() {

	compileRegexps()

	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", config.Server.GrpcPort))
	if err != nil {
		panic(err)
	}
	var opts []grpc.ServerOption
	grpcServer := grpc.NewServer(opts...)
	pb.RegisterRecognizerServer(grpcServer, recogniser{})
	fmt.Println("Serving...")
	err = grpcServer.Serve(lis)
	if err != nil {
		panic(err)
	}

}

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

func compileRegexps() {
	_, thisFile, _, _ := runtime.Caller(0)
	thisDirectory := path.Dir(thisFile)
	b, err := ioutil.ReadFile(filepath.Join(thisDirectory, "regexps.yml"))
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