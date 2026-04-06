package storage

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/Gabriel-Schiestl/pprof-analyzer/internal/domain"
)

// MetadataStore persists CollectionRun state as JSON files under runs/{endpointID}/.
type MetadataStore struct {
	dataDir string
}

// NewMetadataStore creates a MetadataStore rooted at dataDir.
func NewMetadataStore(dataDir string) *MetadataStore {
	return &MetadataStore{dataDir: dataDir}
}

func (s *MetadataStore) SaveRun(run domain.CollectionRun) error {
	dir := s.runDir(run.EndpointID)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("create run dir: %w", err)
	}
	path := filepath.Join(dir, run.ID+".json")
	return atomicWriteJSON(path, run)
}

func (s *MetadataStore) GetLastRun(endpointID string) (*domain.CollectionRun, error) {
	runs, err := s.ListRuns(endpointID, 1)
	if err != nil {
		return nil, err
	}
	if len(runs) == 0 {
		return nil, nil
	}
	return &runs[0], nil
}

func (s *MetadataStore) ListRuns(endpointID string, limit int) ([]domain.CollectionRun, error) {
	dir := s.runDir(endpointID)

	entries, err := os.ReadDir(dir)
	if os.IsNotExist(err) {
		return []domain.CollectionRun{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read runs dir: %w", err)
	}

	// Sort by filename descending (filenames contain timestamps via UUID or StartedAt)
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Name() > entries[j].Name()
	})

	var runs []domain.CollectionRun
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}

		data, err := os.ReadFile(filepath.Join(dir, entry.Name()))
		if err != nil {
			continue // skip unreadable files
		}

		var run domain.CollectionRun
		if err := json.Unmarshal(data, &run); err != nil {
			continue
		}
		runs = append(runs, run)

		if limit > 0 && len(runs) >= limit {
			break
		}
	}

	// Sort by StartedAt descending
	sort.Slice(runs, func(i, j int) bool {
		return runs[i].StartedAt.After(runs[j].StartedAt)
	})

	return runs, nil
}

func (s *MetadataStore) runDir(endpointID string) string {
	return filepath.Join(s.dataDir, "runs", endpointID)
}
