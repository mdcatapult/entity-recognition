package main

import (
	"fmt"
	"github.com/go-redis/redis"
	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
	"gitlab.mdcatapult.io/informatics/software-engineering/entity-recognition/go/gen/pb"
	"gitlab.mdcatapult.io/informatics/software-engineering/entity-recognition/go/lib"
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

// config structure
type conf struct {
	LogLevel string `mapstructure:"log_level"`
	Server struct{
		GrpcPort int `mapstructure:"grpc_port"`
	}
}

// global vars initialised on startup (should never be edited after that).
var config conf
var regexps map[string]*regexp.Regexp
var regexpStringMap = make(map[string]string)

func init() {
	// Initialize config with default values
	err := lib.InitializeConfig(map[string]interface{}{
		"log_level": "info",
		"server": map[string]interface{}{
			"grpc_port": 50051,
		},
	})
	if err != nil {
		panic(err)
	}

	// unmarshal viper contents into our struct
	err = viper.Unmarshal(&config)
	if err != nil {
		log.Fatal().Err(err).Send()
	}
}

func main() {

	// Reads the regexp.yml file and compiles them, populating the `regexp` map above.
	compileRegexps()

	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", config.Server.GrpcPort))
	if err != nil {
		log.Fatal().Err(err).Send()
	}
	var opts []grpc.ServerOption
	grpcServer := grpc.NewServer(opts...)
	pb.RegisterRecognizerServer(grpcServer, recogniser{})
	fmt.Println("Serving...")
	err = grpcServer.Serve(lis)
	if err != nil {
		log.Fatal().Err(err).Send()
	}

}

type recogniser struct {
	pb.UnimplementedRecognizerServer
	redisClient *redis.Client
}

func (r recogniser) Recognize(stream pb.Recognizer_RecognizeServer) error {
	// listen for tokens
	for {
		token, err := stream.Recv()
		if err == io.EOF {
			break
		} else if err != nil {
			return err
		}

		// normalize the token (removes punctuation and enforces NFKC encoding on the utf8 characters).
		lib.Normalize(token)

		// For every regexp try to match the token and send the recognised entity if there is a match.
		for name, re := range regexps {
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