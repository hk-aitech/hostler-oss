// Package domain — DTOs for Sprint stamp (HMAC signature).
//
// ADR-001 §9 E2.
package domain

// SprintHeuristicPolicy — heuristic strictness applied during Sprint HMAC signing.
//
// Promoted from pkg/sprint's HeuristicPolicy (int) into the domain layer.
// The Sprint prefix distinguishes it from TaskHeuristicPreset (string).
type SprintHeuristicPolicy int

const (
	// SprintHeuristicStrict (default): every item required — section must
	// exist and contain substantive content.
	SprintHeuristicStrict SprintHeuristicPolicy = iota
	// SprintHeuristicModerate: section must exist; explicit "0 entries"
	// statements are tolerated.
	SprintHeuristicModerate
	// SprintHeuristicLenient: only the presence of required sections is checked.
	SprintHeuristicLenient
	// SprintHeuristicOff: skip (used as a fallback).
	SprintHeuristicOff
)

// SprintHeuristicResult — heuristic verdict result.
type SprintHeuristicResult struct {
	OK     bool
	Policy SprintHeuristicPolicy
	Issues []string
}

// StampResult — return type of StampWithValidation.
type StampResult struct {
	Hash            string // signed sprint_hmac value (empty when unsigned)
	HeuristicPolicy SprintHeuristicPolicy
	HeuristicOK     bool
	SubagentCalled  bool
	SubagentVerdict string // "ok" | "reject" | "" (when not invoked)
	SubagentReasons []string
	Warnings        []string // not failures, but messages worth surfacing to operators
}

// StampDurationStats — statistics for duration_ms.
type StampDurationStats struct {
	Min int64 `json:"min"`
	Max int64 `json:"max"`
	Avg int64 `json:"avg"`
}

// SprintStampStats — aggregated log statistics.
type SprintStampStats struct {
	Count          int                `json:"count"`
	Window         int                `json:"window"`         // most recent N entries actually applied
	VerdictDist    map[string]int     `json:"verdict_dist"`   // ok/reject/error distribution
	SprintIDDist   map[string]int     `json:"sprint_id_dist"` // meta_type distribution (top N)
	ErrorClassDist map[string]int     `json:"error_class_dist"`
	DurationMs     StampDurationStats `json:"duration_ms"`
	ScoreAvg       float64            `json:"score_avg"`
	ResumeRatio    float64            `json:"resume_ratio"`
	Alerts         []string           `json:"alerts"`
}

// StampStatsOptions — options for AggregateStampLog.
type StampStatsOptions struct {
	Window int    // aggregate only the most recent N entries (0 uses the default window)
	Path   string // log file path (uses the default path when empty)
}
