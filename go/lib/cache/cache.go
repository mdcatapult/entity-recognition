package cache

import "encoding/json"

// Lookup is the value we will store in the db.
type Lookup struct {
	Dictionary  string                 `json:"dictionary"`
	Identifiers map[string]interface{} `json:"identifiers,omitempty"`
	Metadata    json.RawMessage        `json:"metadata"`
}
type Type string
