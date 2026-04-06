package storage_test

import (
	"testing"
	"time"

	"github.com/Gabriel-Schiestl/pprof-analyzer/internal/domain"
	"github.com/Gabriel-Schiestl/pprof-analyzer/internal/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newRun(id, endpointID string, started time.Time) domain.CollectionRun {
	return domain.CollectionRun{
		ID:          id,
		EndpointID:  endpointID,
		StartedAt:   started,
		CompletedAt: started.Add(5 * time.Second),
		Status:      domain.RunStatusSuccess,
	}
}

func TestMetadataStore_SaveAndGetLast(t *testing.T) {
	store := storage.NewMetadataStore(t.TempDir())

	now := time.Now()
	run1 := newRun("run-1", "ep-1", now.Add(-10*time.Second))
	run2 := newRun("run-2", "ep-1", now)

	require.NoError(t, store.SaveRun(run1))
	require.NoError(t, store.SaveRun(run2))

	last, err := store.GetLastRun("ep-1")
	require.NoError(t, err)
	require.NotNil(t, last)
	assert.Equal(t, "run-2", last.ID)
}

func TestMetadataStore_GetLastRun_NoRuns(t *testing.T) {
	store := storage.NewMetadataStore(t.TempDir())
	last, err := store.GetLastRun("ep-x")
	require.NoError(t, err)
	assert.Nil(t, last)
}

func TestMetadataStore_ListRuns(t *testing.T) {
	store := storage.NewMetadataStore(t.TempDir())
	now := time.Now()

	for i := 0; i < 5; i++ {
		run := newRun("run-"+string(rune('0'+i)), "ep-2", now.Add(time.Duration(i)*time.Second))
		require.NoError(t, store.SaveRun(run))
	}

	runs, err := store.ListRuns("ep-2", 3)
	require.NoError(t, err)
	assert.Len(t, runs, 3)

	// Most recent first
	assert.True(t, runs[0].StartedAt.After(runs[1].StartedAt))
}
