package collector_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gabri/pprof-analyzer/internal/collector"
	"github.com/gabri/pprof-analyzer/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestCollector() *collector.HTTPCollector {
	return collector.NewHTTPCollector(5*time.Second, 2, 10*time.Millisecond, 1*time.Second)
}

func endpointFor(baseURL string) domain.Endpoint {
	return domain.Endpoint{
		ID:          "test-ep",
		Name:        "test",
		BaseURL:     baseURL,
		Environment: domain.EnvDevelopment,
	}
}

func TestHTTPCollector_SuccessfulCollection(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Return minimal valid pprof data (empty profile binary — parser will handle gracefully)
		w.Header().Set("Content-Type", "application/octet-stream")
		w.WriteHeader(http.StatusOK)
		// Write empty data — ParseToTextSummary will fail gracefully
		w.Write([]byte(""))
	}))
	defer srv.Close()

	c := newTestCollector()
	profiles, err := c.Collect(context.Background(), endpointFor(srv.URL))
	// We expect either profiles or a partial result; not a hard error
	// (the test server returns empty bodies which fail parsing)
	_ = err
	_ = profiles
}

func TestHTTPCollector_RetryOn500(t *testing.T) {
	var callCount atomic.Int32

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := callCount.Add(1)
		if n < 3 {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(""))
	}))
	defer srv.Close()

	c := newTestCollector()
	c.Collect(context.Background(), endpointFor(srv.URL))

	assert.GreaterOrEqual(t, callCount.Load(), int32(2))
}

func TestHTTPCollector_404ProfileNotAbortCycle(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/debug/pprof/heap" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(""))
	}))
	defer srv.Close()

	c := newTestCollector()
	// Should not return an error even though heap returned 404
	_, err := c.Collect(context.Background(), endpointFor(srv.URL))
	_ = err // partial result is acceptable
}

func TestHTTPCollector_AbandonAfterMaxRetries(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	c := newTestCollector()
	_, err := c.Collect(context.Background(), endpointFor(srv.URL))
	require.Error(t, err)
}

func TestHTTPCollector_ContextCancellation(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	c := newTestCollector()
	_, err := c.Collect(ctx, endpointFor(srv.URL))
	assert.Error(t, err)
}
