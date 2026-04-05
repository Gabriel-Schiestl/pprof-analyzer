package domain

import "time"

// ProfileType identifies which pprof profile was collected.
type ProfileType string

const (
	ProfileHeap         ProfileType = "heap"
	ProfileAllocs       ProfileType = "allocs"
	ProfileGoroutine    ProfileType = "goroutine"
	ProfileCPU          ProfileType = "profile"
	ProfileBlock        ProfileType = "block"
	ProfileMutex        ProfileType = "mutex"
	ProfileThreadCreate ProfileType = "threadcreate"
)

// AllProfileTypes lists every profile type collected each cycle.
// CPU is intentionally last — it requires a sampling window.
var AllProfileTypes = []ProfileType{
	ProfileHeap, ProfileAllocs, ProfileGoroutine,
	ProfileBlock, ProfileMutex, ProfileThreadCreate,
	ProfileCPU,
}

// RunStatus represents the outcome of a collection cycle.
type RunStatus string

const (
	RunStatusSuccess RunStatus = "success"
	RunStatusPartial RunStatus = "partial"
	RunStatusFailed  RunStatus = "failed"
)

// ProfileData holds the result of collecting a single pprof profile.
type ProfileData struct {
	Type        ProfileType `json:"type"`
	RawPath     string      `json:"raw_path"`
	TextSummary string      `json:"text_summary"`
	CollectedAt time.Time   `json:"collected_at"`
	SizeBytes   int64       `json:"size_bytes"`
}

// CollectionRun represents one complete collection cycle for an endpoint.
type CollectionRun struct {
	ID          string        `json:"id"`
	EndpointID  string        `json:"endpoint_id"`
	StartedAt   time.Time     `json:"started_at"`
	CompletedAt time.Time     `json:"completed_at"`
	Profiles    []ProfileData `json:"profiles"`
	Status      RunStatus     `json:"status"`
	FailureMsg  string        `json:"failure_msg,omitempty"`
	ReportPath  string        `json:"report_path,omitempty"`
}
