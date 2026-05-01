// Package ports — ConsistencyChecker port.
//
// hstl-specific port. Quickly assembles the consistency warnings
// (placeholder Task bodies left behind, DB/file drift, etc.) that MCP
// tool responses attach. Today `cli/pkg/consistency/` combines a TTL
// cache with a Task placeholder scan.
//
// Verification pattern:
//
//	var _ ports.ConsistencyChecker = (*ConsistencyCheckerAdapter)(nil)
package ports

// ConsistencyChecker collects consistency warnings.
// Currently implemented by `pkg/consistency`.
type ConsistencyChecker interface {
	// Check returns the current consistency warnings.
	// The internal TTL-cache policy is an adapter decision — the port
	// layer does not require "guaranteed freshness", only "data from a
	// reasonable point in time".
	Check() ([]string, error)

	// InvalidateCache evicts the internal cache (for tests / forced rechecks).
	InvalidateCache()
}
