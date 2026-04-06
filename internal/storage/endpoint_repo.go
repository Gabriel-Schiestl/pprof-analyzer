package storage

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/Gabriel-Schiestl/pprof-analyzer/internal/domain"
)

// EndpointRepository persists endpoints as a JSON file with concurrent-safe access.
type EndpointRepository struct {
	mu   sync.RWMutex
	path string
}

// NewEndpointRepository creates a repository backed by the given JSON file path.
func NewEndpointRepository(dataDir string) *EndpointRepository {
	return &EndpointRepository{
		path: filepath.Join(dataDir, "endpoints.json"),
	}
}

func (r *EndpointRepository) List() ([]domain.Endpoint, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.readAll()
}

func (r *EndpointRepository) Get(id string) (*domain.Endpoint, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	all, err := r.readAll()
	if err != nil {
		return nil, err
	}

	for i := range all {
		if all[i].ID == id {
			return &all[i], nil
		}
	}
	return nil, fmt.Errorf("%w: %s", domain.ErrEndpointNotFound, id)
}

func (r *EndpointRepository) Save(e domain.Endpoint) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	all, err := r.readAll()
	if err != nil {
		return err
	}

	for i := range all {
		if all[i].ID == e.ID {
			all[i] = e
			return atomicWriteJSON(r.path, all)
		}
	}

	all = append(all, e)
	return atomicWriteJSON(r.path, all)
}

func (r *EndpointRepository) Delete(id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	all, err := r.readAll()
	if err != nil {
		return err
	}

	idx := -1
	for i := range all {
		if all[i].ID == id {
			idx = i
			break
		}
	}
	if idx == -1 {
		return fmt.Errorf("%w: %s", domain.ErrEndpointNotFound, id)
	}

	all = append(all[:idx], all[idx+1:]...)
	return atomicWriteJSON(r.path, all)
}

// readAll reads the JSON file; returns empty slice if the file does not exist yet.
func (r *EndpointRepository) readAll() ([]domain.Endpoint, error) {
	data, err := os.ReadFile(r.path)
	if os.IsNotExist(err) {
		return []domain.Endpoint{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read endpoints file: %w", err)
	}

	var endpoints []domain.Endpoint
	if err := json.Unmarshal(data, &endpoints); err != nil {
		return nil, fmt.Errorf("parse endpoints file: %w", err)
	}
	return endpoints, nil
}
