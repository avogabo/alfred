package memory

import (
	"encoding/json"
	"os"
	"path/filepath"
)

type Store struct {
	path string
	Data Data
}

type Data struct {
	UserPreferences Preferences        `json:"user_preferences"`
	Recipes         []Recipe           `json:"collection_recipes"`
	History         []CollectionRecord `json:"collection_history"`
	Clarifications  []Clarification    `json:"clarifications"`
	PendingDeletes  []PendingDelete    `json:"pending_deletes"`
}

type Preferences struct {
	DefaultMovieLibrary string `json:"default_movie_library"`
	DefaultShowLibrary  string `json:"default_show_library"`
	Locale              string `json:"locale"`
	OrderingDefault     string `json:"ordering_default"`
}

type Recipe struct {
	ID              string   `json:"id"`
	Name            string   `json:"name"`
	PromptAliases   []string `json:"prompt_aliases"`
	InclusionRules  []string `json:"inclusion_rules"`
	ExclusionRules  []string `json:"exclusion_rules"`
	OrderingRules   []string `json:"ordering_rules"`
	TemporaryByDefault bool  `json:"temporary_by_default"`
}

type CollectionRecord struct {
	Name         string `json:"name"`
	Library      string `json:"library"`
	SourcePrompt string `json:"source_prompt"`
	Temporary    bool   `json:"temporary"`
	ExpiresAt    string `json:"expires_at,omitempty"`
	CreatedAt    string `json:"created_at"`
	UpdatedAt    string `json:"updated_at"`
}

type Clarification struct {
	Topic          string `json:"topic"`
	Interpretation string `json:"interpretation"`
	UpdatedAt      string `json:"updated_at"`
}

type PendingDelete struct {
	Library     string `json:"library"`
	Name        string `json:"name"`
	RequestedAt string `json:"requested_at"`
}

func Open(dataDir string) (*Store, error) {
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return nil, err
	}
	path := filepath.Join(dataDir, "memory.json")
	s := &Store{path: path, Data: Data{}}
	blob, err := os.ReadFile(path)
	if err == nil && len(blob) > 0 {
		_ = json.Unmarshal(blob, &s.Data)
	}
	return s, nil
}

func (s *Store) Save() error {
	blob, err := json.MarshalIndent(s.Data, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.path, blob, 0644)
}
