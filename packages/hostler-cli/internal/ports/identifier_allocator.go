// Package ports — IdentifierAllocator port.
//
// Isolates issuance of T### sequential numbers behind an interface.
//
// Today the global functions in `cli/pkg/id/id.go` access GraphStore's
// transactional counter directly. This port is an extracted declaration
// that the Service layer can accept in its constructor.
//
// Verification pattern:
//
//	var _ ports.IdentifierAllocator = (*IDAllocator)(nil)
//
// A successful compile-time check proves the pkg/id wrapper satisfies the
// IdentifierAllocator contract.
package ports

// IdentifierAllocator issues IDs based on monotonic counters.
// Currently implemented by `pkg/id`.
//
// Implementation contract:
//   - NextTaskID: "T001" format. The counter itself is monotonic in the DB
//     and may have gaps (counter values are not reused on rollback).
//   - NextCounter: general integer counter — shared base for mailbox / other sequences beyond Task.
type IdentifierAllocator interface {
	// NextTaskID returns the next Task ID ("T###").
	// Atomicity is guaranteed by the DB counter — concurrent calls never collide.
	NextTaskID() (string, error)

	// NextCounter increments the named counter by 1 and returns the new value.
	// Shared by Task plus mailbox / other sequences.
	NextCounter(counterKey string) (int64, error)
}
