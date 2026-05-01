// Package config — ConfigService ServiceAdapter.
//
// ADR-001 §4.2 Phase D. Routes the direct pkg/config calls in cmd/config.go,
// cmd/context.go, and cmd/validate_config.go (7 call sites / 5 unique
// functions) through app.ConfigService. Returns concrete types wrapped in
// any (preserves the KB A028 layering rule).
//
// Measured scope: 5 functions called directly from cmd —
// DetectConfigProblems, ApplyConfigFixes, RenderDiff, CollectEffective,
// GetProjectKey. The remaining ~90 pkg/config public APIs are internal
// helpers of these five or Effective-config helpers cmd does not call —
// out of scope for this Task.
package config

// ServiceAdapter is the default implementation of the ConfigService port.
// Acts as a pass-through for the five pkg/config functions invoked from cmd.
type ServiceAdapter struct{}

// NewServiceAdapter constructs the default ServiceAdapter.
func NewServiceAdapter() *ServiceAdapter { return &ServiceAdapter{} }

// DetectProblems is a pass-through for pkg/config.DetectConfigProblems.
// Returns *ProblemReport (cmd type-asserts).
func (ServiceAdapter) DetectProblems(yamlPath string) (any, error) {
	return DetectConfigProblems(yamlPath)
}

// ApplyFixes is a pass-through for pkg/config.ApplyConfigFixes. report is
// the *ProblemReport concrete type (created in cmd, passed in as any).
// Returns *FixResult.
func (ServiceAdapter) ApplyFixes(report any, dryRun bool) (any, error) {
	r, _ := report.(*ProblemReport)
	return ApplyConfigFixes(r, dryRun)
}

// RenderDiff is a pass-through for pkg/config.RenderDiff.
func (ServiceAdapter) RenderDiff(original, migrated []byte) string {
	return RenderDiff(original, migrated)
}

// CollectEffective is a pass-through for pkg/config.CollectEffective.
// Returns *EffectiveConfig.
func (ServiceAdapter) CollectEffective() any {
	return CollectEffective()
}

// GetProjectKey is a pass-through for pkg/config.GetProjectKey.
func (ServiceAdapter) GetProjectKey() string {
	return GetProjectKey()
}
