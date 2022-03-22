package lib

type APIEntity struct {
	Name        string
	Recogniser  string
	Identifiers map[string]string
	Metadata    string
	Positions   []Position
}

type Position struct {
	Xpath    string
	Position uint32
}
