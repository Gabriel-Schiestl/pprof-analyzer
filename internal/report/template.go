package report

import "github.com/Gabriel-Schiestl/pprof-analyzer/internal/domain"

// Layout constants for the PDF template.
const (
	PageMarginLeft   = 15.0
	PageMarginRight  = 15.0
	PageMarginTop    = 20.0
	PageMarginBottom = 20.0

	FontSizeTitle    = 18.0
	FontSizeSection  = 14.0
	FontSizeBody     = 10.0
	FontSizeSmall    = 8.0
	FontSizeCode     = 9.0

	ColWidthLabel = 60.0
	ColWidthValue = 120.0
)

// Severity color palette (hex RGB).
const (
	ColorCritical = "#DC2626" // red
	ColorWarning  = "#D97706" // amber
	ColorNormal   = "#16A34A" // green
	ColorDark     = "#1F2937" // near-black for headings
	ColorGray     = "#6B7280" // secondary text
	ColorLightBg  = "#F9FAFB" // section background
)

// SeverityColor returns the hex color for a given severity.
func SeverityColor(s domain.Severity) string {
	switch s {
	case domain.SeverityCritical:
		return ColorCritical
	case domain.SeverityWarning:
		return ColorWarning
	default:
		return ColorNormal
	}
}

// SeverityLabel returns a human-readable label for the severity.
func SeverityLabel(s domain.Severity) string {
	switch s {
	case domain.SeverityCritical:
		return "CRITICAL"
	case domain.SeverityWarning:
		return "WARNING"
	default:
		return "NORMAL"
	}
}

// ProfileTypeLabel returns a friendly label for a profile type.
func ProfileTypeLabel(pt domain.ProfileType) string {
	labels := map[domain.ProfileType]string{
		domain.ProfileHeap:         "Heap Memory",
		domain.ProfileAllocs:       "Allocations",
		domain.ProfileGoroutine:    "Goroutines",
		domain.ProfileCPU:          "CPU Usage",
		domain.ProfileBlock:        "Block Profile",
		domain.ProfileMutex:        "Mutex Contention",
		domain.ProfileThreadCreate: "Thread Creation",
	}
	if label, ok := labels[pt]; ok {
		return label
	}
	return string(pt)
}
