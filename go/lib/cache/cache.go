package cache

import "encoding/json"

// Lookup is the value we will store in the db.
type Lookup struct {
	Dictionary  string            `json:"dictionary"`
	Identifiers map[string]string `json:"identifiers,omitempty"`
	Metadata    json.RawMessage   `json:"metadata"`
}
type Type string

const (
	Local         Type = "local"
	Redis         Type = "redis"
	Elasticsearch Type = "elasticsearch"
)
