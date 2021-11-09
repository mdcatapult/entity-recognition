package blacklist

import (
	"fmt"
	"github.com/rs/zerolog/log"
	"gitlab.mdcatapult.io/informatics/software-engineering/entity-recognition/go/lib/types/leadmine"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"os"
	"strings"
)

var blacklistFileName = "./config/blacklist.yml"
const geneOrProtein = "Gene or Protein"

type blacklist = struct {
	Entities map[string]bool `yaml:"entities"`
	EntityGroups map[string]bool `yaml:"entity_groups"`
	Abbreviations map[string]bool `yaml:"abbreviations"` // known abbreviations
}

var bl *blacklist

func init() {
	if bl == nil {
		var err error
		bl, err = loadBlacklist()
		if err != nil {
			log.Debug().Msg(fmt.Sprintf("could not load blacklist %v", err))
		}
	}
}

// TODO: implement blacklist for grpc recognisers
func Leadmine(entities []*leadmine.Entity) []*leadmine.Entity {

	if bl == nil {
		log.Warn().Msg("blacklist is not set")
		return entities
	}

	res := make([]*leadmine.Entity, 0, len(entities))
	for _, entity := range entities {
		// skip if entity group is in group blacklist
		if blacklisted, ok := bl.EntityGroups[strings.ToLower(entity.EntityGroup)]; blacklisted && ok {
			continue
		}

		// if text is all uppercase, skip if it's a known abbreviation rather than a gene
		if entity.EntityGroup == geneOrProtein && strings.ToUpper(entity.EntityText) == entity.EntityText {
			if blacklisted, ok := bl.Abbreviations[entity.EntityText]; blacklisted && ok {
				continue
			}
		}

		//  skip if entity text is blacklisted
		if blacklisted, ok := bl.Entities[entity.EntityText]; ok && blacklisted {
			continue
		}

		res = append(res, entity)
	}

	return res
}

func loadBlacklist() (*blacklist, error) {
	data, err := ioutil.ReadFile(blacklistFileName)

	wd, _ := os.Getwd()
	fmt.Println("wd:", wd)
	fmt.Println("data:", data, err)
	if err != nil {
		return nil, err
	}

	bl := blacklist{}

	if err := yaml.Unmarshal(data, &bl); err != nil {
		return nil, err
	}

	return &bl, nil
}
