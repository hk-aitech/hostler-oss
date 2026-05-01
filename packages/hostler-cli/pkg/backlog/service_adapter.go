// Package backlog — BacklogService ServiceAdapter.
// Pass-through adapter that lets cmd/backlog.go reach pkg/backlog's
// functions through app.BacklogService. Parameters and return values
// flow as `any` so cmd performs concrete-type assertions and
// internal/app does not need to import pkg/backlog (preserving the
// layering).
// The existing BacklogFSGenerator remains as the abbreviated mapping
// adapter for ports.BacklogGenerator — the cmd path and port path
// surface different return types, so this dedicated ServiceAdapter is
// kept separate.
package backlog

// ServiceAdapter is the default implementation of the BacklogService
// port. Stateless — every call passes through to a pkg/backlog
// top-level function.
type ServiceAdapter struct{}

// NewServiceAdapter constructs the default ServiceAdapter.
func NewServiceAdapter() *ServiceAdapter { return &ServiceAdapter{} }

// Sync is a pass-through to pkg/backlog.Sync.
// Returns *SyncResult (cmd performs the assertion).
func (ServiceAdapter) Sync(dryRun bool) (any, error) {
	return Sync(dryRun)
}

// SyncWithOptions is a pass-through with finer option granularity.
// Internally calls pkg/backlog.Sync and returns the raw result as
// *SyncResult so cmd can assert it. The audit event is recorded by
// this adapter.
func (ServiceAdapter) SyncWithOptions(dryRun, allowStatusRegression bool) (any, error) {
	raw, err := Sync(dryRun)
	if err != nil {
		return nil, err
	}
	if allowStatusRegression && raw.Summary.TotalIssues > 0 {
		// Audit record (mirrors adapter.go's SyncWithOptions event).
		_ = auditLog("backlog.sync.status_regression_allowed", map[string]any{
			"dry_run":      dryRun,
			"total_issues": raw.Summary.TotalIssues,
			"reason":       "opt-in via --allow-status-regression",
		})
	}
	return raw, nil
}

// RebuildMD is a pass-through to pkg/backlog.RebuildMD.
// opts is the concrete RebuildMDOptions type (created in cmd).
// Returns *RebuildMDResult.
func (ServiceAdapter) RebuildMD(opts any) (any, error) {
	rebuildOpts, _ := opts.(RebuildMDOptions)
	return RebuildMD(rebuildOpts)
}
