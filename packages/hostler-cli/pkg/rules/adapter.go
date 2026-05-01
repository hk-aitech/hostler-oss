// Package rules — RuleRunner adapter.
//
// Wraps the global Registry and Rule interface in an adapter struct that
// satisfies ports.RuleRunner.
//
// Execute runs a Rule in isolation with a minimal RuleContext (ProjectRoot
// + Phase). End-to-end execution (phase cascade, staged_diff injection) is
// the responsibility of the CLI layer.
package rules

import (
	"fmt"
	"os"

	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/internal/ports"
)

// RulesRegistryAdapter is a RuleRunner adapter backed by the global registry.
type RulesRegistryAdapter struct{}

// NewRulesRegistryAdapter constructs a RuleRunner backed by the global
// registry.
func NewRulesRegistryAdapter() *RulesRegistryAdapter { return &RulesRegistryAdapter{} }

// List returns the list of registered Rule descriptors.
func (RulesRegistryAdapter) List() []ports.RuleDescriptor {
	rs := List()
	out := make([]ports.RuleDescriptor, 0, len(rs))
	for _, r := range rs {
		out = append(out, describeRule(r))
	}
	return out
}

// Get returns a specific Rule descriptor.
func (RulesRegistryAdapter) Get(ruleID string) (ports.RuleDescriptor, bool) {
	r, ok := Get(ruleID)
	if !ok {
		return ports.RuleDescriptor{}, false
	}
	return describeRule(r), true
}

// Execute runs a specific Rule in isolation with a minimal RuleContext.
func (RulesRegistryAdapter) Execute(ruleID string) (*ports.RuleExecutionResult, error) {
	r, ok := Get(ruleID)
	if !ok {
		return nil, fmt.Errorf("rule %q not registered", ruleID)
	}
	cwd, _ := os.Getwd()
	ctx := &RuleContext{ProjectRoot: cwd}
	res := r.Check(ctx)
	return &ports.RuleExecutionResult{
		RuleID:   res.RuleID,
		Status:   statusToString(res.Status),
		Message:  res.Message,
		Evidence: res.Evidence,
	}, nil
}

// describeRule extracts a RuleDescriptor from a Rule.
func describeRule(r Rule) ports.RuleDescriptor {
	return ports.RuleDescriptor{
		ID:       r.ID(),
		Category: r.Category(),
		Severity: severityName(r.DefaultSeverity()),
	}
}

// statusToString converts RuleStatus to a port-layer string.
func statusToString(s RuleStatus) string {
	switch s {
	case StatusOK:
		return "pass"
	case StatusViolated:
		return "fail"
	case StatusSkipped, StatusError:
		return "skip"
	default:
		return "unknown"
	}
}

// severityName converts Severity to a port-layer string.
func severityName(s Severity) string {
	switch s {
	case SeverityHardBlock:
		return "hard-block"
	case SeverityBlock:
		return "block"
	case SeverityWarn:
		return "warn"
	case SeverityAdvisory:
		return "advisory"
	case SeverityOff:
		return "off"
	default:
		return "info"
	}
}

// Compile-time assertion that RulesRegistryAdapter satisfies ports.RuleRunner.
var _ ports.RuleRunner = (*RulesRegistryAdapter)(nil)
