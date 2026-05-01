// Package ports — CeremonyCollector port.
//
// hstl-specific port. Collects Sprint / Task lifecycle ceremony data
// (briefing, readiness, harness_gate, reminders). Today
// `cli/pkg/ceremony/` performs the file / DB queries together with the
// rendering-payload calculation.
//
// Verification pattern:
//
//	var _ ports.CeremonyCollector = (*CeremonyCollectorAdapter)(nil)
//
// The actual return types (SprintStartCeremony, etc.) keep their domain shape
// and are returned as `any` from the port — this preserves domain neutrality
// while letting adapters expose rich structures.
package ports

// CeremonyKind enumerates the kinds of ceremonies that can be collected.
type CeremonyKind string

const (
	CeremonySprintStart    CeremonyKind = "sprint_start"
	CeremonySprintComplete CeremonyKind = "sprint_complete"
	CeremonyTaskStart      CeremonyKind = "task_start"
	CeremonyTaskComplete   CeremonyKind = "task_complete"
)

// CeremonyCollector collects ceremony payloads.
// Currently implemented by `pkg/ceremony`.
//
// The return type is the raw domain structure (`any`) — adapters keep the
// freedom to expose rich types. The CLI layer recovers the concrete type
// through assertion.
type CeremonyCollector interface {
	// Collect gathers the ceremony payload identified by kind.
	// targetID: sprintID for Sprint kinds, taskID for Task kinds.
	Collect(kind CeremonyKind, targetID string) (any, error)
}
