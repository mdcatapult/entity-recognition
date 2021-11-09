package leadmine

type Response struct {
	Created  string            `json:"created"`
	Entities []*LeadmineEntity `json:"entities"`
}

type Entity struct {
	Beg                   int             `json:"beg"`
	BegInNormalizedDoc    int             `json:"begInNormalizedDoc"`
	End                   int             `json:"end"`
	EndInNormalizedDoc    int             `json:"endInNormalizedDoc"`
	EntityText            string          `json:"entityText"`
	PossiblyCorrectedText string          `json:"possiblyCorrectedText"`
	RecognisingDict       RecognisingDict `json:"recognisingDict"`
	ResolvedEntity        string          `json:"resolvedEntity"`
	SectionType           string          `json:"sectionType"`
	EntityGroup           string          `json:"entityGroup"`
}

type Dict struct {
	EnforceBracketing            bool   `json:"enforceBracketing"`
	EntityType                   string `json:"entityType"`
	HtmlColor                    string `json:"htmlColor"`
	MaxCorrectionDistance        int    `json:"maxCorrectionDistance"`
	MinimumCorrectedEntityLength int    `json:"minimumCorrectedEntityLength"`
	MinimumEntityLength          int    `json:"minimumEntityLength"`
	Source                       string `json:"source"`
}

type Metadata struct {
	EntityGroup     string `json:"entityGroup"`
	RecognisingDict RecognisingDict
}
