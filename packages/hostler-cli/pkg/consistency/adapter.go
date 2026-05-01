// Package consistency — ConsistencyChecker adapter.
//
// ADR-001 §4.2 Phase C. Wraps the existing QuickCheck / InvalidateCache in
// a struct so the result satisfies ports.ConsistencyChecker.
//
// Current status: the adapter and compile-time verification marker exist,
// but internal/app.Services has not yet wired it in. There are zero
// cmd/internal consumers — once attaching consistency warnings to MCP tool
// responses goes live, the composition root will add Services.Consistency
// and expose the accessor. The port contract and adapter implementation
// are ready; only a consumer needs to be plugged in.
package consistency

import (
	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/internal/ports"
)

// ConsistencyCheckerAdapter is the consistency-check adapter backed by a TTL cache.
type ConsistencyCheckerAdapter struct{}

// NewConsistencyCheckerAdapter constructs the default consistency adapter.
func NewConsistencyCheckerAdapter() *ConsistencyCheckerAdapter {
	return &ConsistencyCheckerAdapter{}
}

// Check returns the current list of consistency warnings.
// The internal TTL cache policy (30s default) is decided by the adapter.
func (ConsistencyCheckerAdapter) Check() ([]string, error) {
	return QuickCheck(), nil
}

// InvalidateCache invalidates the internal cache.
func (ConsistencyCheckerAdapter) InvalidateCache() {
	InvalidateCache()
}

// Compile-time check — ConsistencyCheckerAdapter satisfies the ports.ConsistencyChecker contract.
var _ ports.ConsistencyChecker = (*ConsistencyCheckerAdapter)(nil)
