// Package id — IdentifierAllocator adapter.
//
// ADR-001 §4.1 Phase C. Wraps the existing globals (`TaskIDNext`,
// `NextCounter`) so they satisfy ports.IdentifierAllocator.
// In Phase D the service layer accepts the adapter via its constructor.
package id

import "github.com/hk-aitech/hostler-oss/packages/hostler-cli/internal/ports"

// IDAllocator exposes the global id functions as a port (stateless).
type IDAllocator struct{}

// NewIDAllocator constructs the default counter-based allocator.
func NewIDAllocator() *IDAllocator { return &IDAllocator{} }

// NextTaskID delegates to TaskIDNext.
func (IDAllocator) NextTaskID() (string, error) { return TaskIDNext() }

// NextCounter delegates to the NextCounter global.
func (IDAllocator) NextCounter(counterKey string) (int64, error) {
	return NextCounter(counterKey)
}

// Compile-time check — IDAllocator satisfies the ports.IdentifierAllocator contract.
var _ ports.IdentifierAllocator = (*IDAllocator)(nil)
