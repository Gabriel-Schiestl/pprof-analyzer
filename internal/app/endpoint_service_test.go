package app_test

import (
	"errors"
	"testing"
	"time"

	"github.com/gabri/pprof-analyzer/internal/app"
	"github.com/gabri/pprof-analyzer/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockEndpointRepo implements app.EndpointRepository in memory.
type mockEndpointRepo struct {
	data map[string]domain.Endpoint
}

func newMockRepo() *mockEndpointRepo {
	return &mockEndpointRepo{data: make(map[string]domain.Endpoint)}
}

func (m *mockEndpointRepo) List() ([]domain.Endpoint, error) {
	list := make([]domain.Endpoint, 0, len(m.data))
	for _, e := range m.data {
		list = append(list, e)
	}
	return list, nil
}

func (m *mockEndpointRepo) Get(id string) (*domain.Endpoint, error) {
	e, ok := m.data[id]
	if !ok {
		return nil, domain.ErrEndpointNotFound
	}
	return &e, nil
}

func (m *mockEndpointRepo) Save(e domain.Endpoint) error {
	m.data[e.ID] = e
	return nil
}

func (m *mockEndpointRepo) Delete(id string) error {
	if _, ok := m.data[id]; !ok {
		return domain.ErrEndpointNotFound
	}
	delete(m.data, id)
	return nil
}

func TestEndpointService_Add(t *testing.T) {
	svc := app.NewEndpointService(newMockRepo())

	ep, err := svc.Add(domain.Endpoint{
		Name:            "api",
		BaseURL:         "http://localhost:6060",
		Environment:     domain.EnvDevelopment,
		CollectInterval: 300 * time.Second,
	})

	require.NoError(t, err)
	assert.NotEmpty(t, ep.ID)
	assert.False(t, ep.CreatedAt.IsZero())
}

func TestEndpointService_Add_RequiredFields(t *testing.T) {
	svc := app.NewEndpointService(newMockRepo())

	_, err := svc.Add(domain.Endpoint{BaseURL: "http://localhost:6060"})
	assert.Error(t, err) // missing name

	_, err = svc.Add(domain.Endpoint{Name: "api"})
	assert.Error(t, err) // missing URL
}

func TestEndpointService_Delete_NotFound(t *testing.T) {
	svc := app.NewEndpointService(newMockRepo())
	err := svc.Delete("nonexistent")
	assert.True(t, errors.Is(err, domain.ErrEndpointNotFound))
}

func TestEndpointService_List(t *testing.T) {
	svc := app.NewEndpointService(newMockRepo())

	for i := 0; i < 3; i++ {
		_, err := svc.Add(domain.Endpoint{
			Name:    "app",
			BaseURL: "http://localhost:6060",
		})
		require.NoError(t, err)
	}

	list, err := svc.List()
	require.NoError(t, err)
	assert.Len(t, list, 3)
}
