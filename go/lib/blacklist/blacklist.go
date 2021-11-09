package blacklist

import (
	"fmt"
	"github.com/rs/zerolog/log"
	"gitlab.mdcatapult.io/informatics/software-engineering/entity-recognition/go/gen/pb"
	"gitlab.mdcatapult.io/informatics/software-engineering/entity-recognition/go/lib/types/leadmine"
	"gopkg.in/yaml.v2"
	"io/ioutil"
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
		err = Load(blacklistFileName)
		if err != nil {
			log.Debug().Msg(fmt.Sprintf("could not load blacklist %v", err))
		}
	}
}

// FilterLeadmineEntities filters []leadmine.Entity based on the blacklist.
func FilterLeadmineEntities(entities []*leadmine.Entity) []*leadmine.Entity {

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

// SnippetAllowed returns true if the snippet's text does not exist in the blacklist
func SnippetAllowed(snippet *pb.Snippet) bool {

	fmt.Println(bl)
	if bl == nil {
		return true
	}

	_, isBlacklisted := bl.Entities[snippet.Text]

	return !isBlacklisted
}

func Load(path string) error {

	data, err := ioutil.ReadFile(path)
	if err != nil {
		return err
	}

	res := blacklist{}
	if err := yaml.Unmarshal(data, &res); err != nil {
		return err
	}

	log.Info().Msg(fmt.Sprintf("blacklist set from %v", path))
	bl = &res

	return nil
}
