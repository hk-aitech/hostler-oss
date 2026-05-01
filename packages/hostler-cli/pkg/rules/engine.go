package rules

import (
	"sort"
	"sync"
	"time"
)

// Rule is the atomic unit of validation. Each rule implements ID(),
// Category(), Description(), DefaultSeverity(), and Check(). Rules are
// registered in the Registry and consumed by the Engine.
type Rule interface {
	ID() string                // e.g. "task.result_section.files_exist"
	Category() string          // e.g. "task" | "sprint" | "commit"
	Description() string       // 1-line description (for CLI output / explain)
	DefaultSeverity() Severity // default when no preset specifies a severity
	Check(ctx *RuleContext) *RuleResult
}

// RuleStatus is the result of a single Rule execution.
type RuleStatus int

const (
	// StatusOK means the check passed.
	StatusOK RuleStatus = iota
	// StatusViolated means the rule was violated.
	StatusViolated
	// StatusSkipped means the rule was not applicable (policy/condition mismatch).
	StatusSkipped
	// StatusError means an internal error occurred inside Check() (treated as skip).
	StatusError
)

// RuleResult is the value returned by Rule.Check.
type RuleResult struct {
	RuleID   string        // rule ID
	Status   RuleStatus    // OK | Violated | Skipped | Error
	Severity Severity      // severity actually applied (cascade result or default)
	Message  string        // user-facing message on violation
	Evidence []string      // evidence (file paths / lines, etc.)
	Duration time.Duration // execution time
	Err      error         // root cause when StatusError
}

// registry is the process-wide Rule store.
var registry = struct {
	sync.RWMutex
	rules map[string]Rule
}{rules: make(map[string]Rule)}

// Register adds a new Rule. Duplicate IDs panic (caught at init time).
// Tests may re-register after calling Clear().
func Register(r Rule) {
	registry.Lock()
	defer registry.Unlock()
	id := r.ID()
	if _, exists := registry.rules[id]; exists {
		panic("rules: duplicate Register: " + id)
	}
	registry.rules[id] = r
}

// Get looks up a Rule by ID.
func Get(id string) (Rule, bool) {
	registry.RLock()
	defer registry.RUnlock()
	r, ok := registry.rules[id]
	return r, ok
}

// List returns all registered Rules sorted by ID.
func List() []Rule {
	registry.RLock()
	defer registry.RUnlock()
	out := make([]Rule, 0, len(registry.rules))
	for _, r := range registry.rules {
		out = append(out, r)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID() < out[j].ID() })
	return out
}

// Clear empties the Registry (test-only).
func Clear() {
	registry.Lock()
	defer registry.Unlock()
	registry.rules = make(map[string]Rule)
}

// FilterEvidence is a shared helper that removes ctx.SelfFilePath from the
// evidence array. Per KB mistakes/validators.md M001 ("self-reference
// exclusion"), this prevents the Task file itself from showing up as a false
// positive when validating result-section files.
func FilterEvidence(evidence []string, selfPath string) []string {
	if selfPath == "" {
		return evidence
	}
	out := make([]string, 0, len(evidence))
	for _, e := range evidence {
		if e == selfPath {
			continue
		}
		out = append(out, e)
	}
	return out
}
