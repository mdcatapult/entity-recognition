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

package dict

import (
	"bufio"
	"os"
	"strings"
)

func NewLeadmineReader() Reader {
	return leadmineReader{}
}

type leadmineReader struct{}

func (l leadmineReader) Read(dict *os.File) (chan Entry, chan error) {
	entries := make(chan Entry)
	errors := make(chan error)
	go l.read(dict, entries, errors)
	return entries, errors
}

func (l leadmineReader) read(dict *os.File, entries chan Entry, errors chan error) {

	// Instantiate variables we need to keep track of across lines.
	scn := bufio.NewScanner(dict)

	for scn.Scan() {
		line := scn.Text()

		// skip empty lines and commented out lines.
		if len(line) == 0 || line[0] == '#' {
			continue
		}

		row := strings.Split(line, "\t")

		// The identifier is the last entry, other entries are synonyms.
		identifier := row[len(row)-1]
		synonyms := row[:len(row)-1]

		// Create a redis lookup for each synonym.
		entries <- &NerEntry{
			Synonyms:    synonyms,
			Identifiers: map[string]string{identifier: ""},
		}
	}
	errors <- nil
}
