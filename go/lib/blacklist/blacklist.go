package blacklist

import (
	"fmt"
	"github.com/rs/zerolog/log"
	"gitlab.mdcatapult.io/informatics/software-engineering/entity-recognition/go/lib/types/leadmine"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"strings"
)

var blacklistFileName = "./config/blacklist.yml"
const geneOrProtein = "Gene or Protein"

type blacklist = struct {
	Entities map[string]bool
	EntityGroups map[string]bool
	Abbreviations map[string]bool // known abbreviations
}

var bl *blacklist

func init() {
	if err := Load(blacklistFileName); err != nil {
		log.Debug().Msg(fmt.Sprintf("could not load blacklist %v", err))
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

// SnippetAllowed returns true if the text does not exist in the blacklist
func SnippetAllowed(text string) bool {
	if bl == nil {
		return true
	}

	_, isBlacklisted := bl.Entities[text]

	return !isBlacklisted
}

func Load(path string) error {

	bytes, err := ioutil.ReadFile(path)
	if err != nil {
		return err
	}

	type yamlBlacklist = struct {
		Entities []string `yaml:"entities"`
		EntityGroups []string `yaml:"entity_groups"`
		Abbreviations []string `yaml:"abbreviations"`
	}

	yamlBl := yamlBlacklist{}
	if err := yaml.Unmarshal(bytes, &yamlBl); err != nil {
		return err
	}

	res := blacklist{
		Entities: map[string]bool{},
		EntityGroups: map[string]bool{},
		Abbreviations: map[string]bool{},
	}

	for _, v := range yamlBl.Entities {
		res.Entities[v] = true
	}
	for _, v := range yamlBl.EntityGroups {
		res.EntityGroups[v] = true
	}
	for _, v := range yamlBl.Abbreviations {
		res.EntityGroups[v] = true
	}

	log.Info().Msg(fmt.Sprintf("blacklist set from %v", path))
	bl = &res

	return nil
}
