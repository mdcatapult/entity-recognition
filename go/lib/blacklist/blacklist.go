package blacklist

import (
	httpRecogniser "gitlab.mdcatapult.io/informatics/software-engineering/entity-recognition/go/lib/recogniser/http-recogniser"

	"gopkg.in/yaml.v2"
	"io/ioutil"
	"strings"
)

const blacklistFileName = "../../../config/blacklist.yml"

type blacklist = struct {
	BlacklistedEntities map[string]bool `yaml:"blacklisted_entities"`
	BlacklistedEntityGroups map[string]bool `yaml:"blacklisted_entity_groups"`
	Abbreviations map[string]bool `yaml:"abbreviations"` // known abbreviations
}

var bl *blacklist

func init() {
	if bl == nil {
		var err error
		bl, err = loadBlacklist()
		if err != nil {
			panic("Could not load blacklist")
		}
	}
}

func Leadmine(entities []*httpRecogniser.LeadmineEntity) []*httpRecogniser.LeadmineEntity {

	res := make([]*httpRecogniser.LeadmineEntity, len(entities))
	for _, entity := range entities {
		// skip if entity group is in group blacklist
		if blacklisted, ok := bl.BlacklistedEntityGroups[strings.ToLower(entity.EntityGroup)]; blacklisted && ok {
			continue
		}

		// if text is all uppercase, skip if it's a known abbreviation rather than a gene
		if entity.EntityGroup == "Gene or Protein" && strings.ToUpper(entity.EntityText) == entity.EntityText {
			if blacklisted, ok := bl.Abbreviations[entity.EntityText]; blacklisted && ok {
				continue
			}
		}

		//  skip if entity text is blacklisted
		if blacklisted, ok := bl.BlacklistedEntities[entity.EntityText]; ok && blacklisted {
			continue
		}
	}
	return res
}


func loadBlacklist() (*blacklist, error) {
	data, err := ioutil.ReadFile(blacklistFileName)
	if err != nil {
		return nil, err
	}

	bl := blacklist{}
	if err := yaml.Unmarshal(data, &bl); err != nil {
		return nil, err
	}

	return &bl, nil
}
