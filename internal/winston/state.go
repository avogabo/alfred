package winston

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
)

type StateStore struct {
	path string
	mu   sync.Mutex
	Data StateData
}

type StateData struct {
	Imported map[string]ImportedRecord `json:"imported"`
}

type ImportedRecord struct {
	QueueID      int             `json:"queue_id"`
	RelativePath string          `json:"relative_path"`
	Status       string          `json:"status"`
	State        ItemState       `json:"state"`
	Confidence   MatchConfidence `json:"confidence"`
	Metadata     ItemMetadata    `json:"metadata"`
	Preview      *ItemPreview    `json:"preview,omitempty"`
	Reason       string          `json:"reason,omitempty"`
}

func NewStateStore(root string) (*StateStore, error) {
	path := filepath.Join(root, ".winston-state.json")
	s := &StateStore{path: path, Data: StateData{Imported: map[string]ImportedRecord{}}}
	b, err := os.ReadFile(path)
	if err == nil && len(b) > 0 {
		_ = json.Unmarshal(b, &s.Data)
	}
	return s, nil
}

func (s *StateStore) Has(path string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, ok := s.Data.Imported[path]
	return ok
}

func (s *StateStore) Put(path string, rec ImportedRecord) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Data.Imported[path] = rec
	b, err := json.MarshalIndent(s.Data, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.path, b, 0644)
}

func (s *StateStore) Delete(path string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.Data.Imported, path)
	b, err := json.MarshalIndent(s.Data, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.path, b, 0644)
}

func (s *StateStore) Save() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	b, err := json.MarshalIndent(s.Data, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.path, b, 0644)
}

func (s *StateStore) ClearAll() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Data.Imported = map[string]ImportedRecord{}
	b, err := json.MarshalIndent(s.Data, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.path, b, 0644)
}

func (s *StateStore) PruneMissing() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	pruned := 0
	for path := range s.Data.Imported {
		if _, err := os.Stat(path); err != nil && os.IsNotExist(err) {
			delete(s.Data.Imported, path)
			pruned++
		}
	}
	if pruned > 0 {
		b, err := json.MarshalIndent(s.Data, "", "  ")
		if err == nil {
			_ = os.WriteFile(s.path, b, 0644)
		}
	}
	return pruned
}
