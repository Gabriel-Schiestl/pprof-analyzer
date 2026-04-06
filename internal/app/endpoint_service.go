package app

import (
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/Gabriel-Schiestl/pprof-analyzer/internal/domain"
)

// EndpointService manages CRUD operations for registered endpoints.
type EndpointService struct {
	repo EndpointRepository
}

// NewEndpointService creates an EndpointService backed by the given repository.
func NewEndpointService(repo EndpointRepository) *EndpointService {
	return &EndpointService{repo: repo}
}

// Add validates and persists a new endpoint, generating its ID and timestamps.
func (s *EndpointService) Add(e domain.Endpoint) (domain.Endpoint, error) {
	if e.Name == "" {
		return domain.Endpoint{}, fmt.Errorf("endpoint name is required")
	}
	if e.BaseURL == "" {
		return domain.Endpoint{}, fmt.Errorf("endpoint base URL is required")
	}

	now := time.Now()
	e.ID = uuid.New().String()
	e.CreatedAt = now
	e.UpdatedAt = now

	if err := s.repo.Save(e); err != nil {
		return domain.Endpoint{}, fmt.Errorf("save endpoint: %w", err)
	}
	return e, nil
}

// List returns all registered endpoints.
func (s *EndpointService) List() ([]domain.Endpoint, error) {
	return s.repo.List()
}

// Get returns a single endpoint by ID.
func (s *EndpointService) Get(id string) (*domain.Endpoint, error) {
	return s.repo.Get(id)
}

// Update persists changes to an existing endpoint, updating its timestamp.
func (s *EndpointService) Update(e domain.Endpoint) error {
	// Verify it exists first
	if _, err := s.repo.Get(e.ID); err != nil {
		return err
	}
	e.UpdatedAt = time.Now()
	return s.repo.Save(e)
}

// Delete removes an endpoint by ID.
func (s *EndpointService) Delete(id string) error {
	return s.repo.Delete(id)
}
