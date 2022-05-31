/*
 * Copyright 2022 Medicines Discovery Catapult
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *     http://www.apache.org/licenses/LICENSE-2.0
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

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
