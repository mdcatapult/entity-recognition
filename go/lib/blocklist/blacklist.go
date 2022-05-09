package blocklist

import (
	"fmt"
	"io/ioutil"
	"strings"

	"github.com/rs/zerolog/log"
	"gitlab.mdcatapult.io/informatics/software-engineering/entity-recognition/go/gen/pb"
	"gopkg.in/yaml.v2"
)

type Blocklist struct {
	CaseSensitive   map[string]bool
	CaseInsensitive map[string]bool
}

// Allowed returns true if entity is not blocklisted.
func (blocklist Blocklist) Allowed(entity string) bool {
	if _, ok := blocklist.CaseSensitive[entity]; ok {
		return false
	}

	if _, ok := blocklist.CaseInsensitive[strings.ToLower(entity)]; ok {
		return false
	}

	return true
}

// FilterEntities filters []*pb.Entity based on blocklist.
func (blocklist Blocklist) FilterEntities(entities []*pb.Entity) []*pb.Entity {
	res := make([]*pb.Entity, 0, len(entities))
	for _, entity := range entities {
		if blocklist.Allowed(entity.Name) {
			res = append(res, entity)
		}
	}
	return res
}

// Load returns an unmarshalled blocklist from a YAML file at the given path.
func Load(path string) (*Blocklist, error) {

	bytes, err := ioutil.ReadFile(path)
	if err != nil {
		log.Error().Msg(fmt.Sprintf("could not find blocklist at %v", path))
		return nil, err
	}

	type yamlBlocklist struct {
		CaseSensitive   []string `yaml:"case_sensitive"`
		CaseInsensitive []string `yaml:"case_insensitive"`
	}

	yamlBl := yamlBlocklist{}
	if err := yaml.Unmarshal(bytes, &yamlBl); err != nil {
		log.Error().Msg(fmt.Sprintf("could not load blocklist from %v", path))
		return nil, err
	}

	res := Blocklist{
		CaseSensitive:   map[string]bool{},
		CaseInsensitive: map[string]bool{},
	}

	for _, v := range yamlBl.CaseSensitive {
		res.CaseSensitive[v] = true
	}
	for _, v := range yamlBl.CaseInsensitive {
		res.CaseInsensitive[v] = true
	}

	log.Info().Msg(fmt.Sprintf("blocklist set from %v", path))

	return &res, nil
}
