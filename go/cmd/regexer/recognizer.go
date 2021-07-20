package main

import (
	"gitlab.mdcatapult.io/informatics/software-engineering/entity-recognition/go/lib/text"
	"io"
	"io/ioutil"
	"regexp"

	"gitlab.mdcatapult.io/informatics/software-engineering/entity-recognition/go/gen/pb"
	"gopkg.in/yaml.v2"
)

type recogniser struct {
	pb.UnimplementedRecognizerServer
	regexps map[string]*regexp.Regexp
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
		text.Normalize(token)

		// For every regexp try to match the token and send the recognised entity if there is a match.
		for name, re := range r.regexps {
			if re.MatchString(token.GetToken()) {
				err := stream.Send(&pb.RecognizedEntity{
					Entity:   token.GetToken(),
					Position: token.GetOffset(),
					Type:     name,
				})
				if err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func getRegexps() (map[string]*regexp.Regexp, error) {
	b, err := ioutil.ReadFile(config.RegexFile)
	if err != nil {
		return nil, err
	}
	var regexpStringMap map[string]string
	if err := yaml.Unmarshal(b, &regexpStringMap); err != nil {
		return nil, err
	}

	regexps := make(map[string]*regexp.Regexp)
	for name, uncompiledRegexp := range regexpStringMap {
		regexps[name], err = regexp.Compile(uncompiledRegexp)
		if err != nil {
			return nil, err
		}
	}
	return regexps, nil
}
