package blacklist

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBlacklist(t *testing.T) {
	var testBlacklist = Blacklist{
		CaseSensitive: map[string]bool{
			"caseSensitive": true,
		},
		CaseInsensitive: map[string]bool{
			"caseinsensitive": true,
		},
	}

	assert.False(t, testBlacklist.Allowed("caseInsensitive"))
	assert.False(t, testBlacklist.Allowed("CASEINSENSITIVE"))

	assert.False(t, testBlacklist.Allowed("caseSensitive"))
	assert.True(t, testBlacklist.Allowed("CASESENSITIVE"))

	assert.True(t, testBlacklist.Allowed("non-blacklisted-term"))
}
