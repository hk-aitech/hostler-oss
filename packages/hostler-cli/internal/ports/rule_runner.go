// Package ports — RuleRunner port.
//
// hstl-specific port. Registers, queries, and executes the Rule engine
// that runs during git hooks / harness checks. Today `cli/pkg/rules/`
// implements it via a Registry + Rule interface.
//
// Verification pattern:
//
//	var _ ports.RuleRunner = (*RulesRegistryAdapter)(nil)
//
// The Rule itself (the `rules.Rule` interface) is hard to keep domain-neutral,
// so the port layer exposes only identifiers and status. Execution
// (r.Check / r.Apply) stays inside the adapter.
package ports

// RuleDescriptor holds the metadata for a single Rule.
// The actual execution interface is not exposed (preserves domain neutrality).
type RuleDescriptor struct {
	ID       string // "precommit.sprint.hmac" / "commit.files.no_secrets" etc.
	Category string // "precommit" / "commit" / "harness" / "sprintphase"
	Severity string // "block" | "warn" | "info"
}

// RuleExecutionResult is the result of one RuleRunner.Execute call.
type RuleExecutionResult struct {
	RuleID   string
	Status   string // "pass" | "fail" | "skip"
	Message  string
	Evidence []string
}

// RuleRunner — port for querying and executing the Rule engine.
// ADR-001 §4.2 — currently implemented by `pkg/rules` on top of a Registry.
type RuleRunner interface {
	// List returns every registered Rule descriptor.
	List() []RuleDescriptor

	// Get returns a specific Rule descriptor. The second return is false when not found.
	Get(ruleID string) (RuleDescriptor, bool)

	// Execute runs a single Rule.
	// Full-phase execution is composed at the CLI layer (`hstl rules run`)
	// by combining List with sequential Execute calls.
	Execute(ruleID string) (*RuleExecutionResult, error)
}
