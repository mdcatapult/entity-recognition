package text

import (
	"bytes"
	"unicode"
)

// StripLeft returns a byte slice equal to the given byte slice with all leading whitespace characters removed,
// and an integer indicating how many characters were stripped.
func StripLeft(b []byte) (strippedBytes []byte, nStripped int) {
	left := 0
	started := false
	return bytes.Map(func(r rune) rune {
		if !started && unicode.IsSpace(r) {
			left++
			return -1
		}
		started = true
		return r
	}, b), left
}
