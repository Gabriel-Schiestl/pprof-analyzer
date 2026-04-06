package ollama

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
	"time"

	ollamaAPI "github.com/ollama/ollama/api"

	"github.com/Gabriel-Schiestl/pprof-analyzer/internal/app"
	"github.com/Gabriel-Schiestl/pprof-analyzer/internal/domain"
)

// OllamaClient implements app.AIProvider using a local Ollama instance.
type OllamaClient struct {
	client  *ollamaAPI.Client
	model   string
	timeout time.Duration
}

// NewOllamaClient creates a client connecting to the given Ollama API URL.
func NewOllamaClient(apiURL, model string, timeout time.Duration) (*OllamaClient, error) {
	u, err := url.Parse(apiURL)
	if err != nil {
		return nil, fmt.Errorf("invalid ollama API URL: %w", err)
	}

	client := ollamaAPI.NewClient(u, nil)
	return &OllamaClient{
		client:  client,
		model:   model,
		timeout: timeout,
	}, nil
}

// AnalyzeProfiles runs a two-phase analysis:
// Phase 1 — individual analysis per profile.
// Phase 2 — consolidated cross-profile analysis.
func (c *OllamaClient) AnalyzeProfiles(ctx context.Context, req app.AnalysisRequest) (*domain.AnalysisResult, error) {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	findings, err := c.analyzeIndividualProfiles(ctx, req.Profiles)
	if err != nil {
		return nil, fmt.Errorf("individual analysis: %w", err)
	}

	consolidated, err := c.analyzeConsolidated(ctx, findings, req)
	if err != nil {
		return nil, fmt.Errorf("consolidated analysis: %w", err)
	}

	result := &domain.AnalysisResult{
		RunID:                req.Endpoint.ID + "_" + req.CollectedAt.Format("20060102_150405"),
		EndpointName:         req.Endpoint.Name,
		Environment:          req.Endpoint.Environment,
		CollectedAt:          req.CollectedAt,
		OverallSeverity:      domain.Severity(consolidated.OverallSeverity),
		ExecutiveSummary:     consolidated.ExecutiveSummary,
		PerProfileFindings:   findings,
		ConsolidatedAnalysis: consolidated.ConsolidatedAnalysis,
		Recommendations:      consolidated.Recommendations,
		ModelUsed:            c.model,
		ToolVersion:          req.ToolVersion,
	}
	return result, nil
}

// analyzeIndividualProfiles sends one prompt per profile and collects findings.
func (c *OllamaClient) analyzeIndividualProfiles(ctx context.Context, profiles []domain.ProfileData) ([]domain.ProfileFinding, error) {
	findings := make([]domain.ProfileFinding, 0, len(profiles))

	for _, p := range profiles {
		if p.TextSummary == "" {
			continue
		}

		finding, err := c.analyzeProfile(ctx, p)
		if err != nil {
			// Non-fatal: record warning finding and continue
			findings = append(findings, domain.ProfileFinding{
				ProfileType: p.Type,
				Severity:    domain.SeverityWarning,
				Summary:     "Analysis failed",
				Details:     err.Error(),
			})
			continue
		}
		findings = append(findings, *finding)
	}

	return findings, nil
}

type profileFindingResponse struct {
	Severity        string                    `json:"severity"`
	Summary         string                    `json:"summary"`
	Details         string                    `json:"details"`
	Recommendations []domain.Recommendation   `json:"recommendations"`
}

func (c *OllamaClient) analyzeProfile(ctx context.Context, p domain.ProfileData) (*domain.ProfileFinding, error) {
	sysPrompt, ok := systemPrompts[p.Type]
	if !ok {
		sysPrompt = systemPrompts[domain.ProfileHeap]
	}

	userMsg := fmt.Sprintf("Profile data:\n\n%s\n\nReturn JSON matching this schema:\n%s",
		p.TextSummary, profileFindingSchema)

	responseText, err := c.chat(ctx, sysPrompt, userMsg)
	if err != nil {
		return nil, err
	}

	var resp profileFindingResponse
	if err := parseJSON(responseText, &resp); err != nil {
		return nil, fmt.Errorf("parse finding response: %w", err)
	}

	return &domain.ProfileFinding{
		ProfileType: p.Type,
		Severity:    normalizeSeverity(resp.Severity),
		Summary:     resp.Summary,
		Details:     resp.Details,
	}, nil
}

type consolidatedResponse struct {
	OverallSeverity      string                  `json:"overall_severity"`
	ExecutiveSummary     string                  `json:"executive_summary"`
	ConsolidatedAnalysis string                  `json:"consolidated_analysis"`
	Recommendations      []domain.Recommendation `json:"recommendations"`
}

func (c *OllamaClient) analyzeConsolidated(ctx context.Context, findings []domain.ProfileFinding, req app.AnalysisRequest) (*consolidatedResponse, error) {
	findingsJSON, _ := json.MarshalIndent(findings, "", "  ")

	userMsg := fmt.Sprintf(
		"Application: %s (%s)\nCollected at: %s\n\nIndividual profile findings:\n%s\n\nReturn JSON matching this schema:\n%s",
		req.Endpoint.Name,
		req.Endpoint.Environment,
		req.CollectedAt.Format(time.RFC3339),
		string(findingsJSON),
		consolidatedSchema,
	)

	responseText, err := c.chat(ctx, consolidatedSystemPrompt, userMsg)
	if err != nil {
		return nil, err
	}

	var resp consolidatedResponse
	if err := parseJSON(responseText, &resp); err != nil {
		// Return a fallback if JSON parsing fails
		return &consolidatedResponse{
			OverallSeverity:      string(domain.SeverityWarning),
			ExecutiveSummary:     "Analysis completed but response parsing failed.",
			ConsolidatedAnalysis: responseText,
			Recommendations:      nil,
		}, nil
	}

	resp.OverallSeverity = string(normalizeSeverity(resp.OverallSeverity))
	return &resp, nil
}

// chat sends a single chat message to Ollama and returns the full response text.
func (c *OllamaClient) chat(ctx context.Context, systemPrompt, userMessage string) (string, error) {
	req := &ollamaAPI.ChatRequest{
		Model: c.model,
		Messages: []ollamaAPI.Message{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: userMessage},
		},
		Stream: boolPtr(false),
	}

	var sb strings.Builder
	err := c.client.Chat(ctx, req, func(resp ollamaAPI.ChatResponse) error {
		sb.WriteString(resp.Message.Content)
		return nil
	})
	if err != nil {
		return "", fmt.Errorf("%w: %v", domain.ErrAIProviderUnavailable, err)
	}

	return sb.String(), nil
}

// parseJSON extracts JSON from the model response, handling markdown code fences.
func parseJSON(text string, v any) error {
	text = strings.TrimSpace(text)

	// Strip markdown code fences if present
	if strings.HasPrefix(text, "```") {
		lines := strings.Split(text, "\n")
		if len(lines) > 2 {
			lines = lines[1 : len(lines)-1]
		}
		text = strings.Join(lines, "\n")
	}

	// Find the JSON object boundaries
	start := strings.IndexByte(text, '{')
	end := strings.LastIndexByte(text, '}')
	if start >= 0 && end > start {
		text = text[start : end+1]
	}

	return json.Unmarshal([]byte(text), v)
}

func normalizeSeverity(s string) domain.Severity {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "critical":
		return domain.SeverityCritical
	case "warning", "warn":
		return domain.SeverityWarning
	default:
		return domain.SeverityNormal
	}
}

func boolPtr(b bool) *bool { return &b }
