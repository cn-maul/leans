package subject

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync"
)

// Summary is the lightweight list item returned by Store.List.
type Summary struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// Store loads and caches lecture subjects from a directory of markdown files.
// It is safe for concurrent use.
type Store struct {
	mu       sync.RWMutex
	dir      string
	subjects map[string]*Subject
	names    []string // sorted subject ids for stable ordering
}

// NewStore scans dir for *.md files and parses them. A missing directory is
// not an error — the store just stays empty so the app can still boot.
func NewStore(dir string) (*Store, error) {
	s := &Store{
		dir:      dir,
		subjects: make(map[string]*Subject),
	}
	if dir == "" {
		return s, nil
	}
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return s, nil
	}
	if err := s.reload(); err != nil {
		return nil, err
	}
	return s, nil
}

// reload (re)scans the directory, replacing all subjects.
func (s *Store) reload() error {
	entries, err := os.ReadDir(s.dir)
	if err != nil {
		return fmt.Errorf("read subjects dir: %w", err)
	}

	subjects := make(map[string]*Subject)
	var names []string
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".md" {
			continue
		}
		sub, err := LoadSubject(filepath.Join(s.dir, entry.Name()))
		if err != nil {
			continue
		}
		subjects[sub.ID] = sub
		names = append(names, sub.ID)
	}

	sort.Strings(names)
	s.mu.Lock()
	s.subjects = subjects
	s.names = names
	s.mu.Unlock()
	return nil
}

// List returns subject summaries in stable (sorted) order.
func (s *Store) List() []Summary {
	s.mu.RLock()
	defer s.mu.RUnlock()

	out := make([]Summary, 0, len(s.names))
	for _, id := range s.names {
		if sub, ok := s.subjects[id]; ok {
			out = append(out, Summary{ID: sub.ID, Name: sub.Name})
		}
	}
	return out
}

// Get returns a subject by id.
func (s *Store) Get(id string) (*Subject, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	sub, ok := s.subjects[id]
	if !ok {
		return nil, fmt.Errorf("subject not found: %s", id)
	}
	return sub, nil
}
