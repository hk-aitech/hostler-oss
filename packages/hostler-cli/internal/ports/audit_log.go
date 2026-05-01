// Package ports — AuditLog port.
//
// Port dedicated to append-only audit events and the HMAC hash chain.
// Cooperates with GraphStore: GraphStore owns indexes / storage,
// AuditLog owns the audit log. The two ports share the same DB in the
// SQLite adapter, but their domain responsibilities are separated.
//
// Reuses the TaskStore / SprintStore / HarnessStore extraction pattern —
// three adapters compile-checked in parallel.
package ports

// AuditLog is the port for appending, querying, and aggregating audit events.
type AuditLog interface {
	// ── Append (append-only) ────────────────────────────────────────────
	AppendAuditEvent(event AuditEvent) error

	// ── Query ────────────────────────────────────────────────────────────
	QueryAuditEvents(filter AuditQueryFilter) ([]AuditEvent, error)

	// ── Aggregate ────────────────────────────────────────────────────────
	SummarizeAuditEvents(since, until string) (map[string]int, error)
}
