package app

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"

	"github.com/gabri/pprof-analyzer/internal/domain"
)

// AnalysisService orchestrates the full collection → analysis → report cycle.
type AnalysisService struct {
	collector   ProfileCollector
	ai          AIProvider
	report      ReportWriter
	pprofStore  PprofFileStore
	metadata    MetadataStore
	toolVersion string
}

// NewAnalysisService creates the service with all required adapters.
func NewAnalysisService(
	collector ProfileCollector,
	ai AIProvider,
	report ReportWriter,
	pprofStore PprofFileStore,
	metadata MetadataStore,
	toolVersion string,
) *AnalysisService {
	return &AnalysisService{
		collector:   collector,
		ai:          ai,
		report:      report,
		pprofStore:  pprofStore,
		metadata:    metadata,
		toolVersion: toolVersion,
	}
}

// RunCycle executes one full collection cycle for the given endpoint.
// It always saves a CollectionRun to the MetadataStore, even on partial failure.
func (s *AnalysisService) RunCycle(ctx context.Context, endpoint domain.Endpoint) error {
	run := domain.CollectionRun{
		ID:         uuid.New().String(),
		EndpointID: endpoint.ID,
		StartedAt:  time.Now(),
		Status:     domain.RunStatusFailed,
	}

	defer func() {
		run.CompletedAt = time.Now()
		if err := s.metadata.SaveRun(run); err != nil {
			slog.Error("failed to save run metadata", "endpoint", endpoint.Name, "err", err)
		}
	}()

	// Step 1: Collect profiles
	profiles, err := s.collector.Collect(ctx, endpoint)
	if err != nil {
		run.FailureMsg = fmt.Sprintf("collection failed: %v", err)
		return fmt.Errorf("collect profiles: %w", err)
	}

	// Step 2: Persist raw pprof files
	for i := range profiles {
		path, saveErr := s.pprofStore.Save(endpoint.ID, profiles[i].Type, []byte(profiles[i].TextSummary))
		if saveErr != nil {
			slog.Warn("failed to save pprof file", "profile", profiles[i].Type, "err", saveErr)
			continue
		}
		profiles[i].RawPath = path

		// Apply retention policy per profile type
		if retErr := s.pprofStore.ApplyRetentionPolicy(endpoint.ID, profiles[i].Type); retErr != nil {
			slog.Warn("retention policy failed", "profile", profiles[i].Type, "err", retErr)
		}
	}

	run.Profiles = profiles
	run.Status = domain.RunStatusPartial

	// Step 3: AI analysis
	collectedAt := time.Now()
	analysisResult, err := s.ai.AnalyzeProfiles(ctx, AnalysisRequest{
		Endpoint:    endpoint,
		CollectedAt: collectedAt,
		Profiles:    profiles,
		ToolVersion: s.toolVersion,
	})
	if err != nil {
		slog.Error("AI analysis failed", "endpoint", endpoint.Name, "err", err)
		run.FailureMsg = fmt.Sprintf("AI analysis failed: %v", err)
		return fmt.Errorf("analyze profiles: %w", err)
	}

	// Step 4: Generate PDF report
	analysisResult.RunID = run.ID
	reportPath, err := s.report.Write(ctx, analysisResult)
	if err != nil {
		slog.Error("report generation failed", "endpoint", endpoint.Name, "err", err)
		run.FailureMsg = fmt.Sprintf("report generation failed: %v", err)
		return fmt.Errorf("write report: %w", err)
	}

	run.ReportPath = reportPath
	run.Status = domain.RunStatusSuccess

	slog.Info("collection cycle completed",
		"endpoint", endpoint.Name,
		"profiles", len(profiles),
		"report", reportPath,
	)
	return nil
}
