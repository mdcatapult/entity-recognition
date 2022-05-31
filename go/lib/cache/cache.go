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

package cache

import "encoding/json"

// Lookup is the value we will store in the db.
type Lookup struct {
	Dictionary  string                 `json:"dictionary"`
	Identifiers map[string]interface{} `json:"identifiers,omitempty"`
	Metadata    json.RawMessage        `json:"metadata"`
}
type Type string
