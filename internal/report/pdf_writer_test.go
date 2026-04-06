package report_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/Gabriel-Schiestl/pprof-analyzer/internal/domain"
	"github.com/Gabriel-Schiestl/pprof-analyzer/internal/report"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func fixtureResult() *domain.AnalysisResult {
	return &domain.AnalysisResult{
		RunID:           "run-test-001",
		EndpointName:    "api-gateway",
		Environment:     domain.EnvProduction,
		CollectedAt:     time.Date(2026, 4, 1, 10, 0, 0, 0, time.UTC),
		OverallSeverity: domain.SeverityWarning,
		ExecutiveSummary: "The application shows elevated memory usage with signs of a potential memory leak " +
			"in the HTTP handler pool. CPU usage is within normal range.",
		PerProfileFindings: []domain.ProfileFinding{
			{
				ProfileType: domain.ProfileHeap,
				Severity:    domain.SeverityWarning,
				Summary:     "High heap retention in HTTP handler",
				Details:     "The net/http handler pool retains approximately 450MB of heap memory. This exceeds expected baseline by 3x.",
			},
			{
				ProfileType: domain.ProfileGoroutine,
				Severity:    domain.SeverityNormal,
				Summary:     "Goroutine count within expected range",
				Details:     "128 goroutines active, consistent with load pattern.",
			},
		},
		ConsolidatedAnalysis: "The heap growth correlates with incoming request rate, suggesting connection pool or buffer accumulation.",
		Recommendations: []domain.Recommendation{
			{Priority: 1, Title: "Review HTTP client pool size", Description: "Reduce max idle connections from 100 to 20.", CodeSuggestion: "transport.MaxIdleConns = 20"},
			{Priority: 2, Title: "Enable GC pressure monitoring", Description: "Add runtime.ReadMemStats to metrics endpoint."},
		},
		ModelUsed:   "llama3.3:70b",
		ToolVersion: "0.1.0",
	}
}

func TestPDFWriter_GeneratesFile(t *testing.T) {
	dir := t.TempDir()
	writer := report.NewPDFWriter(dir)

	path, err := writer.Write(context.Background(), fixtureResult())
	require.NoError(t, err)
	assert.NotEmpty(t, path)

	info, err := os.Stat(path)
	require.NoError(t, err)
	assert.Greater(t, info.Size(), int64(0))
}

func TestPDFWriter_OutputPath(t *testing.T) {
	dir := t.TempDir()
	writer := report.NewPDFWriter(dir)

	path, err := writer.Write(context.Background(), fixtureResult())
	require.NoError(t, err)

	assert.Contains(t, path, "api-gateway")
	assert.Contains(t, path, "production")
	assert.Contains(t, path, ".pdf")
}
