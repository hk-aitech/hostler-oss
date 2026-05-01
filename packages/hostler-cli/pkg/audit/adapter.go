// Package audit — AuditFileOps adapter.
//
// File-based audit restore is outside the pure append/query/summarize
// contract of ports.AuditLog, so it is exposed via the
// internal/app.AuditFileOps convenience interface. Mirrors the
// FrontmatterFileOps pattern.
//
// May later be folded into adapters/fs/ as a struct method.
package audit

// FSRestoreAdapter is a stateless wrapper that exposes the global
// pkg/audit.Restore function as an app.AuditFileOps implementation.
//
// trac: TRC-CM004
type FSRestoreAdapter struct{}

// NewFSRestoreAdapter constructs the default fs-based audit restore adapter.
func NewFSRestoreAdapter() *FSRestoreAdapter { return &FSRestoreAdapter{} }

// Restore delegates to the package-level pkg/audit.Restore function.
// The any return follows the app.AuditFileOps contract — the caller
// (cmd/audit.go) re-interprets the value as the concrete type
// (*RestoreResult).
func (FSRestoreAdapter) Restore(dryRun bool) (any, error) {
	return Restore(dryRun)
}
