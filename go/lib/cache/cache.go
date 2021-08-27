package cache

// Lookup is the value we will store in the db.
type Lookup struct {
	Dictionary       string   `json:"dictionary"`
	ResolvedEntities []string `json:"resolvedEntities,omitempty"`
}

type KeyAndLookup struct {
	Key string `json:"key"`
	Lookup
}

type Type string

const (
	Local         Type = "local"
	Redis         Type = "redis"
	Elasticsearch Type = "elasticsearch"
)
