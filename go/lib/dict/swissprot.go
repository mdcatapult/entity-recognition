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
	"encoding/json"
	"os"
)

func NewSwissProtReader() Reader {
	return swissProtReader{}
}

type swissProtReader struct{}

func (p swissProtReader) Read(file *os.File) (chan Entry, chan error) {
	entries := make(chan Entry)
	errors := make(chan error)
	go p.read(file, entries, errors)
	return entries, errors
}

func (p swissProtReader) read(dict *os.File, entries chan Entry, errors chan error) {
	scn := bufio.NewScanner(dict)
	for scn.Scan() {
		var e SwissProtEntry
		if err := json.Unmarshal(scn.Bytes(), &e); err != nil {
			errors <- err
			return
		}
		entries <- &e
	}
}
