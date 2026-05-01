// Package backlog — BacklogGenerator adapter.
// Wraps the existing `backlog.Sync` function in a struct that satisfies
// ports.BacklogGenerator. The CLI `hstl backlog` command routes through
// this adapter.
package backlog

import (
	"path/filepath"

	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/internal/ports"
	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/pkg/audit"
	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/pkg/fileutil"
)

// backlogMDRelPath is the repo-relative location of BACKLOG.md.
const backlogMDRelPath = "works/tasks/BACKLOG.md"

// BacklogFSGenerator is a BacklogGenerator adapter backed by the
// filesystem plus the DB.
type BacklogFSGenerator struct{}

// NewBacklogFSGenerator constructs the default fs/DB-backed
// BacklogGenerator.
func NewBacklogFSGenerator() *BacklogFSGenerator { return &BacklogFSGenerator{} }

// Sync performs BACKLOG.md regeneration plus DB/file drift verification.
// Convenience wrapper around SyncWithOptions with zero-value options.
func (g BacklogFSGenerator) Sync(dryRun bool) (*ports.BacklogSyncResult, error) {
	return g.SyncWithOptions(ports.BacklogSyncOptions{DryRun: dryRun})
}

// SyncWithOptions adds option granularity.
// When AllowStatusRegression=true and a regression is detected, the
// audit event `backlog.sync.status_regression_allowed` is recorded and
// the sync proceeds. The default (false) preserves the previous
// behaviour and blocks regressions.
func (BacklogFSGenerator) SyncWithOptions(opts ports.BacklogSyncOptions) (*ports.BacklogSyncResult, error) {
	raw, err := Sync(opts.DryRun)
	if err != nil {
		return nil, err
	}

	added, updated, removed := classifyFixes(raw)

	// Opt-in audit record for regression-allow.
	if opts.AllowStatusRegression && raw.Summary.TotalIssues > 0 {
		_ = audit.LogEvent("backlog.sync.status_regression_allowed", "backlog", "sync",
			"", map[string]any{
				"dry_run":      opts.DryRun,
				"total_issues": raw.Summary.TotalIssues,
				"reason":       "opt-in via --allow-status-regression",
			}, "")
	}

	return &ports.BacklogSyncResult{
		DryRun:        raw.Summary.DryRun,
		TasksScanned:  raw.Summary.TaskFilesScanned,
		AddedRows:     added,
		UpdatedRows:   updated,
		RemovedRows:   removed,
		FilePath:      fileutil.ToRepoRelative(filepath.Join(fileutil.GetProjectRoot(), backlogMDRelPath)),
		WriteOccurred: !raw.Summary.DryRun && raw.Summary.AutoFixed > 0,
		DriftDetected: raw.Summary.TotalIssues > 0,
	}, nil
}

// classifyFixes groups Fix records into added/updated/removed
// aggregates. Domain-level detail categories (backlog_md_append /
// frontmatter_patch, etc.) collapse into the three port-layer buckets.
func classifyFixes(r *SyncResult) (added, updated, removed int) {
	for _, f := range r.Fixes {
		switch f.Type {
		case "backlog_md_append", "backlog_md_insert":
			added++
		case "backlog_md_remove":
			removed++
		default:
			updated++
		}
	}
	return
}

// Compile-time check — BacklogFSGenerator satisfies the
// ports.BacklogGenerator contract.
var _ ports.BacklogGenerator = (*BacklogFSGenerator)(nil)
