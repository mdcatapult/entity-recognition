package blacklist

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

var testBlacklist = blacklist{
	BlacklistedEntities: map[string]bool{
		"aspirin": true,
	},
}

func TestMain(m *testing.M) {
	bl = &testBlacklist
	m.Run()
}

func TestBlacklist(t *testing.T) {
	blacklistedText := "aspirin"
	allowedText := "an-entity"

	isBlacklisted, err := Ok(blacklistedText)
	assert.NoError(t, err)
	assert.False(t, isBlacklisted)

	isBlacklisted, err = Ok(allowedText)
	assert.NoError(t, err)
	assert.True(t, isBlacklisted)
}
