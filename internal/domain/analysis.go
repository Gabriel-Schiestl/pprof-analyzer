package domain

import "time"

// Severity classifies the urgency of a finding or overall analysis result.
type Severity string

const (
	SeverityCritical Severity = "critical"
	SeverityWarning  Severity = "warning"
	SeverityNormal   Severity = "normal"
)

// ProfileFinding holds the AI analysis result for a single pprof profile.
type ProfileFinding struct {
	ProfileType ProfileType `json:"profile_type"`
	Severity    Severity    `json:"severity"`
	Summary     string      `json:"summary"`
	Details     string      `json:"details"`
}

// Recommendation is a prioritized action suggested by the AI.
type Recommendation struct {
	Priority       int    `json:"priority"`
	Title          string `json:"title"`
	Description    string `json:"description"`
	CodeSuggestion string `json:"code_suggestion,omitempty"`
}

// AnalysisResult is the complete output of one collection cycle analysis.
type AnalysisResult struct {
	RunID                string           `json:"run_id"`
	EndpointName         string           `json:"endpoint_name"`
	Environment          Environment      `json:"environment"`
	CollectedAt          time.Time        `json:"collected_at"`
	OverallSeverity      Severity         `json:"overall_severity"`
	ExecutiveSummary     string           `json:"executive_summary"`
	PerProfileFindings   []ProfileFinding `json:"per_profile_findings"`
	ConsolidatedAnalysis string           `json:"consolidated_analysis"`
	Recommendations      []Recommendation `json:"recommendations"`
	ModelUsed            string           `json:"model_used"`
	ToolVersion          string           `json:"tool_version"`
}
