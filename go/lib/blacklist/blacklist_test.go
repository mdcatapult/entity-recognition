package blacklist

import (
	"github.com/stretchr/testify/assert"
	httpRecogniser "gitlab.mdcatapult.io/informatics/software-engineering/entity-recognition/go/lib/recogniser/http-recogniser"
	"testing"
)

const (
	blacklistedText = "blacklistedText"
	blacklistedEntityGroup = "blacklistedentitygroup"
	blacklistedAbbreviation = "ABCD"
)

var testBlacklist = blacklist{
	Entities: map[string]bool{
		blacklistedText: true,
	},
	EntityGroups: map[string]bool {
		blacklistedEntityGroup: true,
	},
	Abbreviations: map[string]bool {
		blacklistedAbbreviation: true,
	},
}

var (
	entityWithAllowedText = &httpRecogniser.LeadmineEntity{
		EntityText: "non-blacklisted-text",
	}
	entityWithBlacklistedText = &httpRecogniser.LeadmineEntity{
		EntityText:            blacklistedText,
	}
	entityWithBlacklistedGroup = &httpRecogniser.LeadmineEntity{
		EntityText: "non-blacklisted-text",
		EntityGroup: blacklistedEntityGroup,
	}
	entityWithGeneNameNotBlacklisted = &httpRecogniser.LeadmineEntity{
		EntityText: "SOME GENE NAME", // gene names are capitalised
		EntityGroup: geneOrProtein,
	}
	entityWithAbbreviationNotGeneName = &httpRecogniser.LeadmineEntity{
		EntityText:  blacklistedAbbreviation,
		EntityGroup: geneOrProtein,
	}
)

func TestMain(m *testing.M) {
	bl = &testBlacklist
	m.Run()
}

func TestLeadmineEntities(t *testing.T) {
	leadmineEntities := []*httpRecogniser.LeadmineEntity{
		entityWithBlacklistedText,
		entityWithAbbreviationNotGeneName,
		entityWithBlacklistedGroup,
		entityWithGeneNameNotBlacklisted, // this should not be blacklisted
		entityWithAllowedText, // this should not be blacklisted
	}

	res := Leadmine(leadmineEntities)

	assert.Equal(t, 2, len(res))
	assert.True(t, containsLeadmineEntity(res, entityWithGeneNameNotBlacklisted))
	assert.True(t, containsLeadmineEntity(res, entityWithAllowedText))
}

func containsLeadmineEntity(
	haystack []*httpRecogniser.LeadmineEntity,
	needle *httpRecogniser.LeadmineEntity) bool {
	for _, v := range haystack {
		if v == needle {
			return true
		}
	}
	return false
}
