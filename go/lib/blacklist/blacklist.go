package blacklist

import (
	"gopkg.in/yaml.v2"
	"io/ioutil"
)

const blacklistFileName = "../../../config/blacklist.yml"

type blacklist = struct {
	BlacklistedEntities map[string]bool `yaml:"blacklisted_entities"`
}

var bl *blacklist

// Ok returns false if snippet's text is blacklisted.
func Ok(snippetText string) (bool, error) {
	if bl == nil {
		var err error
		bl, err = loadBlacklist()
		if err != nil {
			return false, err
		}
	}

	if blacklisted, ok := bl.BlacklistedEntities[snippetText]; ok && blacklisted {
		return false, nil
	}
	return true, nil
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
