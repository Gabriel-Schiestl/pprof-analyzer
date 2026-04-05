package app_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/gabri/pprof-analyzer/internal/app"
	"github.com/gabri/pprof-analyzer/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- Mocks ---

type mockCollector struct {
	profiles []domain.ProfileData
	err      error
}

func (m *mockCollector) Collect(_ context.Context, _ domain.Endpoint) ([]domain.ProfileData, error) {
	return m.profiles, m.err
}

type mockAI struct {
	result *domain.AnalysisResult
	err    error
}

func (m *mockAI) AnalyzeProfiles(_ context.Context, req app.AnalysisRequest) (*domain.AnalysisResult, error) {
	if m.result != nil {
		m.result.RunID = req.Endpoint.ID
	}
	return m.result, m.err
}

type mockReportWriter struct {
	path string
	err  error
}

func (m *mockReportWriter) Write(_ context.Context, _ *domain.AnalysisResult) (string, error) {
	return m.path, m.err
}

type mockPprofStore struct{}

func (m *mockPprofStore) Save(_ string, _ domain.ProfileType, _ []byte) (string, error) {
	return "/tmp/test.pb.gz", nil
}

func (m *mockPprofStore) ApplyRetentionPolicy(_ string, _ domain.ProfileType) error {
	return nil
}

type mockMetadataStore struct {
	saved []domain.CollectionRun
}

func (m *mockMetadataStore) SaveRun(run domain.CollectionRun) error {
	m.saved = append(m.saved, run)
	return nil
}

func (m *mockMetadataStore) GetLastRun(_ string) (*domain.CollectionRun, error) { return nil, nil }
func (m *mockMetadataStore) ListRuns(_ string, _ int) ([]domain.CollectionRun, error) {
	return nil, nil
}

// --- Tests ---

func newTestService(collector app.ProfileCollector, ai app.AIProvider, report app.ReportWriter, meta *mockMetadataStore) *app.AnalysisService {
	return app.NewAnalysisService(collector, ai, report, &mockPprofStore{}, meta, "0.1.0")
}

func testEndpoint() domain.Endpoint {
	return domain.Endpoint{
		ID:          "ep-test",
		Name:        "test-app",
		Environment: domain.EnvDevelopment,
	}
}

func TestAnalysisService_SuccessfulCycle(t *testing.T) {
	profiles := []domain.ProfileData{{Type: domain.ProfileHeap, TextSummary: "data"}}
	result := &domain.AnalysisResult{
		EndpointName:    "test-app",
		OverallSeverity: domain.SeverityNormal,
		CollectedAt:     time.Now(),
	}

	meta := &mockMetadataStore{}
	svc := newTestService(
		&mockCollector{profiles: profiles},
		&mockAI{result: result},
		&mockReportWriter{path: "/reports/test.pdf"},
		meta,
	)

	err := svc.RunCycle(context.Background(), testEndpoint())
	require.NoError(t, err)

	require.Len(t, meta.saved, 1)
	assert.Equal(t, domain.RunStatusSuccess, meta.saved[0].Status)
	assert.Equal(t, "/reports/test.pdf", meta.saved[0].ReportPath)
}

func TestAnalysisService_CollectorFailure(t *testing.T) {
	meta := &mockMetadataStore{}
	svc := newTestService(
		&mockCollector{err: errors.New("connection refused")},
		&mockAI{},
		&mockReportWriter{},
		meta,
	)

	err := svc.RunCycle(context.Background(), testEndpoint())
	assert.Error(t, err)

	require.Len(t, meta.saved, 1)
	assert.Equal(t, domain.RunStatusFailed, meta.saved[0].Status)
}

func TestAnalysisService_AIFailureSavesRun(t *testing.T) {
	profiles := []domain.ProfileData{{Type: domain.ProfileHeap, TextSummary: "data"}}

	meta := &mockMetadataStore{}
	svc := newTestService(
		&mockCollector{profiles: profiles},
		&mockAI{err: errors.New("ollama unavailable")},
		&mockReportWriter{path: "/reports/test.pdf"},
		meta,
	)

	err := svc.RunCycle(context.Background(), testEndpoint())
	assert.Error(t, err)

	require.Len(t, meta.saved, 1)
	// Run is saved even when AI fails
	assert.NotEmpty(t, meta.saved[0].ID)
}
