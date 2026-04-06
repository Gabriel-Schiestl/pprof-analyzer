package ollama_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Gabriel-Schiestl/pprof-analyzer/internal/ai/ollama"
	"github.com/Gabriel-Schiestl/pprof-analyzer/internal/app"
	"github.com/Gabriel-Schiestl/pprof-analyzer/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func ollamaResponse(content string) map[string]any {
	return map[string]any{
		"model": "test-model",
		"message": map[string]any{
			"role":    "assistant",
			"content": content,
		},
		"done": true,
	}
}

func newTestServer(handler http.HandlerFunc) *httptest.Server {
	return httptest.NewServer(handler)
}

func TestOllamaClient_AnalyzeProfiles(t *testing.T) {
	findingJSON := `{"severity":"warning","summary":"High heap usage","details":"Memory usage elevated.","recommendations":[]}`
	consolidatedJSON := `{"overall_severity":"warning","executive_summary":"Elevated memory.","consolidated_analysis":"Heap is high.","recommendations":[]}`

	callCount := 0
	srv := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.Header().Set("Content-Type", "application/json")
		var content string
		if callCount <= 1 {
			content = findingJSON
		} else {
			content = consolidatedJSON
		}
		json.NewEncoder(w).Encode(ollamaResponse(content))
	})
	defer srv.Close()

	client, err := ollama.NewOllamaClient(srv.URL, "test-model", 10*time.Second)
	require.NoError(t, err)

	req := app.AnalysisRequest{
		Endpoint: domain.Endpoint{
			ID:          "ep-1",
			Name:        "test-app",
			Environment: domain.EnvDevelopment,
		},
		CollectedAt: time.Now(),
		Profiles: []domain.ProfileData{
			{
				Type:        domain.ProfileHeap,
				TextSummary: "heap data here",
			},
		},
		ToolVersion: "0.1.0",
	}

	result, err := client.AnalyzeProfiles(context.Background(), req)
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "test-app", result.EndpointName)
}

func TestOllamaClient_UnavailableProvider(t *testing.T) {
	client, err := ollama.NewOllamaClient("http://localhost:19999", "test-model", 1*time.Second)
	require.NoError(t, err)

	req := app.AnalysisRequest{
		Endpoint: domain.Endpoint{Name: "test"},
		Profiles: []domain.ProfileData{
			{Type: domain.ProfileHeap, TextSummary: "data"},
		},
	}

	_, err = client.AnalyzeProfiles(context.Background(), req)
	assert.Error(t, err)
}
