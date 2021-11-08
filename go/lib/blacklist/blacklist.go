package blacklist

import (
	"gitlab.mdcatapult.io/informatics/software-engineering/entity-recognition/go/gen/pb"
	"gopkg.in/yaml.v2"
	"io/ioutil"
)

const blacklistFileName = "../../../config/blacklist.yml"

type blacklist = map[string]bool
var bl blacklist

// Ok returns false if snippet's text is blacklisted.
func Ok(snippet *pb.Snippet) (bool, error) {
	if bl == nil {
		var err error
		bl, err = loadBlacklist()
		if err != nil {
			return false, err
		}
	}
	// should we check Text or NormalisedText?
	if blacklisted, ok := bl[snippet.Text]; ok && blacklisted {
		return false, nil
	}
	return true, nil
}

func loadBlacklist() (blacklist, error) {
	data, err := ioutil.ReadFile(blacklistFileName)
	if err != nil {
		return nil, err
	}

	bl := blacklist{}
	if err := yaml.Unmarshal(data, &bl); err != nil {
		return nil, err
	}

	return bl, nil
}
