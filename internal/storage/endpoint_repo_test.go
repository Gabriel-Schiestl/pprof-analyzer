package storage_test

import (
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/Gabriel-Schiestl/pprof-analyzer/internal/domain"
	"github.com/Gabriel-Schiestl/pprof-analyzer/internal/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newEndpoint(id, name string) domain.Endpoint {
	return domain.Endpoint{
		ID:              id,
		Name:            name,
		BaseURL:         "http://localhost:6060",
		Environment:     domain.EnvDevelopment,
		CollectInterval: 300 * time.Second,
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}
}

func TestEndpointRepository_CRUD(t *testing.T) {
	repo := storage.NewEndpointRepository(t.TempDir())

	// Empty list
	list, err := repo.List()
	require.NoError(t, err)
	assert.Empty(t, list)

	// Save
	ep := newEndpoint("id-1", "api-gateway")
	require.NoError(t, repo.Save(ep))

	// Get
	got, err := repo.Get("id-1")
	require.NoError(t, err)
	assert.Equal(t, ep.Name, got.Name)

	// List
	list, err = repo.List()
	require.NoError(t, err)
	assert.Len(t, list, 1)

	// Update
	ep.Name = "api-gateway-v2"
	require.NoError(t, repo.Save(ep))
	got, err = repo.Get("id-1")
	require.NoError(t, err)
	assert.Equal(t, "api-gateway-v2", got.Name)

	// Delete
	require.NoError(t, repo.Delete("id-1"))
	list, err = repo.List()
	require.NoError(t, err)
	assert.Empty(t, list)
}

func TestEndpointRepository_GetNotFound(t *testing.T) {
	repo := storage.NewEndpointRepository(t.TempDir())
	_, err := repo.Get("nonexistent")
	assert.True(t, errors.Is(err, domain.ErrEndpointNotFound))
}

func TestEndpointRepository_DeleteNotFound(t *testing.T) {
	repo := storage.NewEndpointRepository(t.TempDir())
	err := repo.Delete("nonexistent")
	assert.True(t, errors.Is(err, domain.ErrEndpointNotFound))
}

func TestEndpointRepository_Concurrency(t *testing.T) {
	repo := storage.NewEndpointRepository(t.TempDir())

	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		i := i
		wg.Add(1)
		go func() {
			defer wg.Done()
			ep := newEndpoint(string(rune('a'+i)), "app")
			_ = repo.Save(ep)
		}()
	}
	wg.Wait()

	list, err := repo.List()
	require.NoError(t, err)
	assert.Len(t, list, 10)
}
