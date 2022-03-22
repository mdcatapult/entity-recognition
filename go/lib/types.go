package lib

type APIEntity struct {
	Name        string            `json:"name"`
	Recogniser  string            `json:"recogniser"`
	Identifiers map[string]string `json:"identifiers"`
	Metadata    string            `json:"metadata"`
	Positions   []Position        `json:"positions"`
}

type Position struct {
	Xpath    string `json:"xpath"`
	Position uint32 `json:"position"`
}
