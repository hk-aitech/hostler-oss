// Package sprint — scope review heuristic.
// Prevents Sprint goals with overly broad scope (integration with other
// systems, introducing new binaries, etc.) from being entered into the
// backlog without explicit user approval. Returns a list of signals
// derived from a deterministic keyword analysis of the goal string.
// Design choices:
// No LLM calls — deterministic word matching. Returns identical
// results in CI / automation.
// The keyword lists are intentionally generic — not tied to any
// specific external system.
// "Corresponding Task size" is undetermined at sprint-creation
// time and is therefore out of scope for this module.
package sprint

import "strings"

// ScopeSignal is one scope-risk signal detected in the goal.
type ScopeSignal struct {
	Category string `json:"category"` // "scope_boundary" | "external_integration" | "new_binary"
	Keyword  string `json:"keyword"`  // matched keyword (after lowercase normalisation)
	Hint     string `json:"hint"`     // explanation shown to the user
}

// scopeBoundaryKeywords — work that crosses the project boundary.
var scopeBoundaryKeywords = []string{
	"other system", "external domain", "separate project", "upper layer",
	"scope boundary", "out of scope", "other persona",
}

// externalIntegrationKeywords — external-system integration signals.
var externalIntegrationKeywords = []string{
	"traefik",
	"gitlab", "github",
	"external api",
	"mcp", "c# core", "dotnet",
	"k8s", "kubernetes", "rancher",
}

// newBinaryKeywords — signals indicating a new binary / sub-command.
var newBinaryKeywords = []string{
	"new binary",
	"split cli", "separate cli",
}

// AnalyzeScope extracts scope-risk signals from the Sprint goal string.
// Case-insensitive, returns sorted results.
func AnalyzeScope(goal string) []ScopeSignal {
	lowered := strings.ToLower(goal)
	seen := map[string]bool{}
	signals := []ScopeSignal{}

	check := func(category, hint string, keywords []string) {
		for _, kw := range keywords {
			if strings.Contains(lowered, kw) {
				key := category + ":" + kw
				if seen[key] {
					continue
				}
				seen[key] = true
				signals = append(signals, ScopeSignal{
					Category: category,
					Keyword:  kw,
					Hint:     hint,
				})
			}
		}
	}

	check("scope_boundary",
		"work crosses the project boundary — Sprints scoped to other systems / separate projects require explicit user approval",
		scopeBoundaryKeywords)
	check("external_integration",
		"external-system integration — infrastructure/service couplings risk Sprint scope creep",
		externalIntegrationKeywords)
	check("new_binary",
		"new binary or CLI split — affects the release pipeline; explicit user approval required",
		newBinaryKeywords)

	return signals
}

// ScopeReviewBlocked builds the user-facing block message from a
// signal list. Called when --require-review is true and at least one
// signal has been detected.
func ScopeReviewBlocked(signals []ScopeSignal) string {
	if len(signals) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString("Sprint goal requires up-front scope review — the following signals were detected:\n")
	for _, s := range signals {
		b.WriteString("  • [")
		b.WriteString(s.Category)
		b.WriteString("] keyword '")
		b.WriteString(s.Keyword)
		b.WriteString("' — ")
		b.WriteString(s.Hint)
		b.WriteString("\n")
	}
	b.WriteString("\nObtain explicit user approval and use --force to override, or narrow the goal and retry.\n")
	return b.String()
}
