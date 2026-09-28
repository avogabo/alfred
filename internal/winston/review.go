package winston

type ItemState string

const (
	StateDetected    ItemState = "detected"
	StateNeedsReview ItemState = "needs_review"
	StateApproved    ItemState = "approved"
	StateImporting   ItemState = "importing"
	StateImported    ItemState = "imported"
	StateFailed      ItemState = "failed"
	StateCorrected   ItemState = "corrected"
)

type MatchConfidence string

const (
	ConfidenceHigh   MatchConfidence = "high"
	ConfidenceMedium MatchConfidence = "medium"
	ConfidenceLow    MatchConfidence = "low"
)

type CandidateMatch struct {
	Label  string `json:"label"`
	Kind   string `json:"kind"`
	TMDBID int    `json:"tmdb_id"`
	TVDBID int    `json:"tvdb_id"`
	IMDBID string `json:"imdb_id"`
	Year   int    `json:"year"`
	Reason string `json:"reason"`
	Score  int    `json:"score"`
}

type ItemPreview struct {
	SourceNZBPath string           `json:"source_nzb_path"`
	State         ItemState        `json:"state"`
	Confidence    MatchConfidence  `json:"confidence"`
	Kind          string           `json:"kind"`
	Metadata      ItemMetadata     `json:"metadata"`
	ProposedPath  string           `json:"proposed_path"`
	Reason        string           `json:"reason"`
	Candidates    []CandidateMatch `json:"candidates"`
	ResolverMethod string          `json:"resolver_method,omitempty"`
	ResolverError  string          `json:"resolver_error,omitempty"`
}
