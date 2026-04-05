package collector

import (
	"bytes"
	"fmt"
	"sort"
	"strings"

	pprofProfile "github.com/google/pprof/profile"

	"github.com/gabri/pprof-analyzer/internal/domain"
)

const (
	maxTopFunctions    = 30
	maxGoroutineStacks = 10
	maxTokens          = 4000
)

// ParseToTextSummary converts raw pprof binary data into a human-readable
// text summary suitable for LLM consumption.
func ParseToTextSummary(rawData []byte, profileType domain.ProfileType) (string, error) {
	prof, err := pprofProfile.Parse(bytes.NewReader(rawData))
	if err != nil {
		return "", fmt.Errorf("parse pprof: %w", err)
	}

	var sb strings.Builder

	switch profileType {
	case domain.ProfileGoroutine:
		writeGoroutineSummary(&sb, prof)
	default:
		writeTopFunctions(&sb, prof, profileType)
	}

	result := sb.String()
	return truncate(result, maxTokens), nil
}

// writeTopFunctions writes a flat|flat%|cum|cum% table for the top-N samples.
func writeTopFunctions(sb *strings.Builder, prof *pprofProfile.Profile, profileType domain.ProfileType) {
	type row struct {
		name    string
		flat    int64
		cum     int64
		flatPct float64
		cumPct  float64
	}

	totals := make(map[string]*row)
	var totalFlat int64

	for _, s := range prof.Sample {
		if len(s.Value) == 0 {
			continue
		}
		flat := s.Value[0]
		totalFlat += flat

		for i, loc := range s.Location {
			for _, ln := range loc.Line {
				name := functionName(ln)
				r, ok := totals[name]
				if !ok {
					r = &row{name: name}
					totals[name] = r
				}
				r.cum += flat
				if i == 0 {
					r.flat += flat
				}
			}
		}
	}

	rows := make([]*row, 0, len(totals))
	for _, r := range totals {
		if totalFlat > 0 {
			r.flatPct = float64(r.flat) / float64(totalFlat) * 100
			r.cumPct = float64(r.cum) / float64(totalFlat) * 100
		}
		rows = append(rows, r)
	}

	sort.Slice(rows, func(i, j int) bool {
		return rows[i].flat > rows[j].flat
	})

	unit := unitLabel(profileType, prof)
	sb.WriteString(fmt.Sprintf("Profile: %s | Unit: %s | Total: %d\n\n", profileType, unit, totalFlat))
	sb.WriteString(fmt.Sprintf("%-12s %-8s %-12s %-8s %s\n", "flat", "flat%", "cum", "cum%", "function"))
	sb.WriteString(strings.Repeat("-", 80) + "\n")

	limit := maxTopFunctions
	if len(rows) < limit {
		limit = len(rows)
	}
	for _, r := range rows[:limit] {
		sb.WriteString(fmt.Sprintf("%-12d %-7.2f%% %-12d %-7.2f%% %s\n",
			r.flat, r.flatPct, r.cum, r.cumPct, r.name))
	}
}

// writeGoroutineSummary writes goroutine counts by state and top stacks.
func writeGoroutineSummary(sb *strings.Builder, prof *pprofProfile.Profile) {
	type stack struct {
		frames string
		count  int64
	}

	stacks := make(map[string]*stack)
	var total int64

	for _, s := range prof.Sample {
		if len(s.Value) == 0 {
			continue
		}
		count := s.Value[0]
		total += count

		var frames []string
		for _, loc := range s.Location {
			for _, ln := range loc.Line {
				frames = append(frames, functionName(ln))
			}
		}
		key := strings.Join(frames, " -> ")
		if _, ok := stacks[key]; !ok {
			stacks[key] = &stack{frames: key}
		}
		stacks[key].count += count
	}

	allStacks := make([]*stack, 0, len(stacks))
	for _, s := range stacks {
		allStacks = append(allStacks, s)
	}
	sort.Slice(allStacks, func(i, j int) bool {
		return allStacks[i].count > allStacks[j].count
	})

	sb.WriteString(fmt.Sprintf("Goroutine Profile | Total goroutines: %d\n\n", total))

	limit := maxGoroutineStacks
	if len(allStacks) < limit {
		limit = len(allStacks)
	}
	sb.WriteString(fmt.Sprintf("Top %d goroutine stacks:\n", limit))
	sb.WriteString(strings.Repeat("-", 80) + "\n")
	for i, s := range allStacks[:limit] {
		sb.WriteString(fmt.Sprintf("[%d] count=%d\n  %s\n\n", i+1, s.count, s.frames))
	}
}

func functionName(ln pprofProfile.Line) string {
	if ln.Function == nil {
		return "unknown"
	}
	if ln.Function.Name != "" {
		return ln.Function.Name
	}
	return ln.Function.SystemName
}

func unitLabel(profileType domain.ProfileType, prof *pprofProfile.Profile) string {
	switch profileType {
	case domain.ProfileHeap, domain.ProfileAllocs:
		return "bytes"
	case domain.ProfileCPU:
		return "nanoseconds"
	default:
		if len(prof.SampleType) > 0 {
			return prof.SampleType[0].Unit
		}
		return "count"
	}
}

// truncate limits text to approximately maxTokens (1 token ≈ 4 chars).
func truncate(s string, maxTok int) string {
	maxChars := maxTok * 4
	if len(s) <= maxChars {
		return s
	}
	return s[:maxChars] + "\n... [truncated]"
}
