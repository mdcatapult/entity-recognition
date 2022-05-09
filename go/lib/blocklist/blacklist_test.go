package blocklist

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBlocklist(t *testing.T) {
	var testBlocklist = Blocklist{
		CaseSensitive: map[string]bool{
			"caseSensitive": true,
		},
		CaseInsensitive: map[string]bool{
			"caseinsensitive": true,
		},
	}

	assert.False(t, testBlocklist.Allowed("caseInsensitive"))
	assert.False(t, testBlocklist.Allowed("CASEINSENSITIVE"))

	assert.False(t, testBlocklist.Allowed("caseSensitive"))
	assert.True(t, testBlocklist.Allowed("CASESENSITIVE"))

	assert.True(t, testBlocklist.Allowed("non-blocklisted-term"))
}
