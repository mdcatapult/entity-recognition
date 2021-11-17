package blacklist

import (
	"fmt"
	"io/ioutil"
	"strings"

	"github.com/rs/zerolog/log"
	"gitlab.mdcatapult.io/informatics/software-engineering/entity-recognition/go/gen/pb"
	"gopkg.in/yaml.v2"
)

type Blacklist struct {
	CaseSensitive   map[string]bool
	CaseInsensitive map[string]bool
}

// Allowed returns true if entity is not blacklisted.
func (blacklist Blacklist) Allowed(entity string) bool {
	if _, ok := blacklist.CaseSensitive[entity]; ok {
		return false
	}

	if _, ok := blacklist.CaseInsensitive[strings.ToLower(entity)]; ok {
		return false
	}

	return true
}

// FilterEntities filters []*pb.Entity based on blacklist.
func (blacklist Blacklist) FilterEntities(entities []*pb.Entity) []*pb.Entity {
	res := make([]*pb.Entity, 0, len(entities))
	for _, entity := range entities {
		if blacklist.Allowed(entity.Name) {
			res = append(res, entity)
		}
	}
	return res
}

// Load returns an unmarshalled blacklist from a YAML file at the given path.
func Load(path string) (*Blacklist, error) {

	bytes, err := ioutil.ReadFile(path)
	if err != nil {
		log.Error().Msg(fmt.Sprintf("could not find blacklist at %v", path))
		return nil ,err
	}

	type yamlBlacklist struct {
		CaseSensitive   []string `yaml:"case_sensitive"`
		CaseInsensitive []string `yaml:"case_insensitive"`
	}

	yamlBl := yamlBlacklist{}
	if err := yaml.Unmarshal(bytes, &yamlBl); err != nil {
		log.Error().Msg(fmt.Sprintf("could not load blacklist from %v", path))
		return nil, err
	}

	res := Blacklist{
		CaseSensitive:   map[string]bool{},
		CaseInsensitive: map[string]bool{},
	}

	for _, v := range yamlBl.CaseSensitive {
		res.CaseSensitive[v] = true
	}
	for _, v := range yamlBl.CaseInsensitive {
		res.CaseInsensitive[v] = true
	}

	log.Info().Msg(fmt.Sprintf("blacklist set from %v", path))

	return &res, nil
}
