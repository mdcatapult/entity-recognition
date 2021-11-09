package blacklist

import (
	"github.com/stretchr/testify/assert"
	"gitlab.mdcatapult.io/informatics/software-engineering/entity-recognition/go/lib/types/leadmine"
	"os"
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
	entityWithAllowedText = &leadmine.Entity{
		EntityText: "non-blacklisted-text",
	}
	entityWithBlacklistedText = &leadmine.Entity{
		EntityText:            blacklistedText,
	}
	entityWithBlacklistedGroup = &leadmine.Entity{
		EntityText: "non-blacklisted-text",
		EntityGroup: blacklistedEntityGroup,
	}
	entityWithGeneNameNotBlacklisted = &leadmine.Entity {
		EntityText: "SOME GENE NAME", // gene names are capitalised
		EntityGroup: geneOrProtein,
	}
	entityWithAbbreviationNotGeneName = &leadmine.Entity{
		EntityText:  blacklistedAbbreviation,
		EntityGroup: geneOrProtein,
	}
)

func TestMain(m *testing.M) {
	bl = &testBlacklist
	os.Exit(m.Run())
}

func TestBlacklistLeadmineEntities(t *testing.T) {
	leadmineEntities := []*leadmine.Entity{
		entityWithBlacklistedText,
		entityWithAbbreviationNotGeneName,
		entityWithBlacklistedGroup,
		entityWithGeneNameNotBlacklisted, // this should not be blacklisted
		entityWithAllowedText, // this should not be blacklisted
	}

	res := FilterLeadmineEntities(leadmineEntities)

	assert.Equal(t, 2, len(res))
	assert.True(t, containsLeadmineEntity(res, entityWithGeneNameNotBlacklisted))
	assert.True(t, containsLeadmineEntity(res, entityWithAllowedText))
}

func TestSnippetAllowed(t *testing.T) {
	assert.False(t, SnippetAllowed(blacklistedText))
	assert.True(t, SnippetAllowed("not blacklisted"))
}

func containsLeadmineEntity(
	haystack []*leadmine.Entity,
	needle *leadmine.Entity) bool {
	for _, v := range haystack {
		if v == needle {
			return true
		}
	}
	return false
}
