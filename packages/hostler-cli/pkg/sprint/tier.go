// Package sprint — Sprint Tier 4-level classification + capacity check.
// Sprint Tier is a sprint-level classification distinct from Task
// estimate (XS/S/M/L/XL). Task estimates are human time guesses
// (XS = 15 min ~ XL = 1 week) at the Task level; Sprint Tier
// categorises a Sprint by Task count + capacity (point sum) into four
// levels and is the practical knob for the human-vs-AI ~100x
// asymmetry policy (KB A020).
// The four Tier names use the analogy of simultaneous collaborators:
// solo: 1 AI working autonomously.
// standard: classic human Scrum.
// pair: human + 1 AI 1:1.
// squad: human + multiple AIs.
// SSOT: docs/08-references/standards/sprint-tier-spec.md
package sprint

import (
	"fmt"
	"strings"

	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/internal/domain"
)

// SprintTier classifies a Sprint by size.
type SprintTier string

const (
	// TierSolo — autonomous workflow with 1 AI (default).
	TierSolo SprintTier = "solo"
	// TierStandard — classic human Scrum.
	TierStandard SprintTier = "standard"
	// TierPair — human + AI 1:1 collaboration.
	TierPair SprintTier = "pair"
	// TierSquad — human + multiple AI team.
	TierSquad SprintTier = "squad"
)

// TierDefaults are the per-tier default thresholds.
// MaxTaskCount = 0 means unlimited (squad).
// MaxCapacity = 0 means unlimited (squad).
type TierDefaults struct {
	MinTaskCount int
	MaxTaskCount int
	MaxCapacity  int
}

// tierDefaultsTable is the default-thresholds table for the four
// tiers.
// Source: body table.
//	| Tier | Task count | capacity (pt) |
//	| solo | 1~3 | ~8 |
//	| standard | 3~7 | ~20 |
//	| pair | 7~15 | ~50 |
//	| squad | 15+ | 100+ |
var tierDefaultsTable = map[SprintTier]TierDefaults{
	TierSolo:     {MinTaskCount: 1, MaxTaskCount: 3, MaxCapacity: 8},
	TierStandard: {MinTaskCount: 3, MaxTaskCount: 7, MaxCapacity: 20},
	TierPair:     {MinTaskCount: 7, MaxTaskCount: 15, MaxCapacity: 50},
	TierSquad:    {MinTaskCount: 15, MaxTaskCount: 0, MaxCapacity: 0},
}

// ParseTier parses a string into a SprintTier.
// Empty / unsupported values fall back to TierSolo (the default).
// The second return value indicates explicit match (true = explicit,
// false = fallback).
func ParseTier(s string) (SprintTier, bool) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "solo":
		return TierSolo, true
	case "standard":
		return TierStandard, true
	case "pair":
		return TierPair, true
	case "squad":
		return TierSquad, true
	default:
		return TierSolo, false
	}
}

// DefaultsFor returns the per-tier default thresholds.
// Unsupported tiers fall back to the TierSolo defaults.
func DefaultsFor(tier SprintTier) TierDefaults {
	d, ok := tierDefaultsTable[tier]
	if !ok {
		return tierDefaultsTable[TierSolo]
	}
	return d
}

// EstimateToPoints maps a Task estimate to Fibonacci points.
//	XS=1, S=2, M=3, L=5, XL=8
// Unsupported estimates (including the empty string) return 0 (ignored
// during capacity summation).
func EstimateToPoints(est domain.Estimate) int {
	switch est {
	case domain.EstimateXS:
		return 1
	case domain.EstimateS:
		return 2
	case domain.EstimateM:
		return 3
	case domain.EstimateL:
		return 5
	case domain.EstimateXL:
		return 8
	default:
		return 0
	}
}

// SumPoints returns the point total for a list of estimates.
func SumPoints(estimates []domain.Estimate) int {
	total := 0
	for _, est := range estimates {
		total += EstimateToPoints(est)
	}
	return total
}

// TierCheckResult is the verification result for a Sprint Tier.
type TierCheckResult struct {
	Tier         SprintTier
	TaskCount    int
	Capacity     int
	MinTaskCount int
	MaxTaskCount int // 0 = unlimited
	MaxCapacity  int // 0 = unlimited
	Violations   []string
}

// IsBlocked reports whether a violation requires a BLOCK.
func (r *TierCheckResult) IsBlocked() bool {
	return len(r.Violations) > 0
}

// Reason is the BLOCK reason string (empty when no violations).
func (r *TierCheckResult) Reason() string {
	if len(r.Violations) == 0 {
		return ""
	}
	return strings.Join(r.Violations, "; ")
}

// CheckTier verifies a Task list against the tier + override
// thresholds.
// nil fields in the override use the tier defaults.
// minTaskCount/maxTaskCount/maxCapacity = 0 are interpreted as
// "unlimited" (e.g. squad defaults).
// When the returned Violations is empty the check passes; otherwise,
// BLOCK.
func CheckTier(tier SprintTier, estimates []domain.Estimate, override TierDefaults) TierCheckResult {
	defaults := DefaultsFor(tier)

	min := defaults.MinTaskCount
	if override.MinTaskCount > 0 {
		min = override.MinTaskCount
	}
	max := defaults.MaxTaskCount
	if override.MaxTaskCount > 0 {
		max = override.MaxTaskCount
	}
	maxCap := defaults.MaxCapacity
	if override.MaxCapacity > 0 {
		maxCap = override.MaxCapacity
	}

	taskCount := len(estimates)
	capacity := SumPoints(estimates)

	result := TierCheckResult{
		Tier:         tier,
		TaskCount:    taskCount,
		Capacity:     capacity,
		MinTaskCount: min,
		MaxTaskCount: max,
		MaxCapacity:  maxCap,
	}

	if min > 0 && taskCount < min {
		result.Violations = append(result.Violations,
			fmt.Sprintf("task count %d < min %d (tier=%s)", taskCount, min, tier))
	}
	if max > 0 && taskCount > max {
		result.Violations = append(result.Violations,
			fmt.Sprintf("task count %d > max %d (tier=%s)", taskCount, max, tier))
	}
	if maxCap > 0 && capacity > maxCap {
		result.Violations = append(result.Violations,
			fmt.Sprintf("capacity %dpt > max %dpt (tier=%s)", capacity, maxCap, tier))
	}

	return result
}
