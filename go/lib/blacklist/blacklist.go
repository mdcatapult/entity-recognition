package blacklist

import (
	"fmt"
	"github.com/rs/zerolog/log"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"strings"
)

type Blacklist struct {
	CaseSensitive map[string]bool
	CaseInsensitive map[string]bool
}

func (blacklist Blacklist) Allowed(entity string) bool {
	if _, ok := blacklist.CaseSensitive[entity]; ok {
		return false
	}

	if _, ok := blacklist.CaseInsensitive[strings.ToLower(entity)]; ok {
		return false
	}

	return true
}

func Load(path string) Blacklist {

	bytes, err := ioutil.ReadFile(path)
	if err != nil {
		log.Warn().Msg(fmt.Sprintf("could not find blacklist at %v", path))
	}

	type yamlBlacklist = struct {
		CaseSensitive []string `yaml:"case_sensitive"`
		CaseInsensitive []string `yaml:"case_insensitive"`
	}

	yamlBl := yamlBlacklist{}
	if err := yaml.Unmarshal(bytes, &yamlBl); err != nil {
		log.Warn().Msg(fmt.Sprintf("could not load blacklist"))
	}

	res := Blacklist{
		CaseSensitive: map[string]bool{},
		CaseInsensitive: map[string]bool{},
	}

	for _, v := range yamlBl.CaseSensitive {
		res.CaseSensitive[v] = true
	}
	for _, v := range yamlBl.CaseInsensitive {
		res.CaseInsensitive[v] = true
	}

	log.Info().Msg(fmt.Sprintf("blacklist set from %v", path))

	return res
}
