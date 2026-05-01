// Package ports — BacklogGenerator port.
//
// hstl-specific port. Regenerates `works/tasks/BACKLOG.md` and verifies
// DB/file drift. Today `cli/pkg/backlog/` performs Sync / Rebuild
// directly against the filesystem.
//
// Verification pattern:
//
//	var _ ports.BacklogGenerator = (*BacklogFSGenerator)(nil)
//
// Sync semantics:
//   - dryRun=true: simulates BACKLOG.md regeneration and reports the expected changes.
//   - dryRun=false: performs the regeneration and persists the changes.
package ports

// BacklogSyncResult summarises a BacklogGenerator.Sync call.
//
// Even when dryRun is true the expected change counts are populated — only the
// actual write step is skipped.
type BacklogSyncResult struct {
	DryRun        bool
	TasksScanned  int
	AddedRows     int
	UpdatedRows   int
	RemovedRows   int
	FilePath      string // BACKLOG.md path (repo-relative)
	WriteOccurred bool   // false when dryRun=true or when nothing changed
	DriftDetected bool   // whether DB / file drift was detected
}

// BacklogSyncOptions is the port extension that provides finer-grained
// control of Sync. The zero value matches the legacy Sync(dryRun)
// behaviour.
type BacklogSyncOptions struct {
	DryRun bool
	// AllowStatusRegression — permit regression (e.g. done → todo). Default
	// is false. Opt in by setting true when intentionally reopening a sprint
	// or task. When allowed, an audit event
	// `backlog.sync.status_regression_allowed` is recorded.
	AllowStatusRegression bool
}

// BacklogGenerator manages the Task summary index (BACKLOG.md).
// Currently implemented by `pkg/backlog`.
type BacklogGenerator interface {
	// Sync regenerates BACKLOG.md and verifies DB / file drift.
	// When dryRun=true the call only reports findings and leaves files alone.
	Sync(dryRun bool) (*BacklogSyncResult, error)

	// SyncWithOptions adds option granularity over Sync. The legacy
	// Sync(dryRun) is a convenience wrapper around this method.
	SyncWithOptions(opts BacklogSyncOptions) (*BacklogSyncResult, error)
}
