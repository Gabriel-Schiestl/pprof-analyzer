package app

import (
	"context"
	"time"

	"github.com/gabri/pprof-analyzer/internal/domain"
)

// ProfileCollector collects pprof profiles from a live endpoint.
type ProfileCollector interface {
	Collect(ctx context.Context, endpoint domain.Endpoint) ([]domain.ProfileData, error)
}

// AIProvider analyzes pprof profiles and returns a structured diagnosis.
type AIProvider interface {
	AnalyzeProfiles(ctx context.Context, req AnalysisRequest) (*domain.AnalysisResult, error)
}

// ReportWriter generates a PDF report from an AnalysisResult.
type ReportWriter interface {
	Write(ctx context.Context, result *domain.AnalysisResult) (filePath string, err error)
}

// EndpointRepository persists and retrieves registered endpoints.
type EndpointRepository interface {
	List() ([]domain.Endpoint, error)
	Get(id string) (*domain.Endpoint, error)
	Save(e domain.Endpoint) error
	Delete(id string) error
}

// MetadataStore persists collection run state for the dashboard.
type MetadataStore interface {
	SaveRun(run domain.CollectionRun) error
	GetLastRun(endpointID string) (*domain.CollectionRun, error)
	ListRuns(endpointID string, limit int) ([]domain.CollectionRun, error)
}

// PprofFileStore manages storage and retention of raw pprof files on disk.
type PprofFileStore interface {
	Save(endpointID string, profileType domain.ProfileType, data []byte) (path string, err error)
	ApplyRetentionPolicy(endpointID string, profileType domain.ProfileType) error
}

// AnalysisRequest groups the input context sent to the AIProvider.
type AnalysisRequest struct {
	Endpoint    domain.Endpoint
	CollectedAt time.Time
	Profiles    []domain.ProfileData
	ToolVersion string
}
