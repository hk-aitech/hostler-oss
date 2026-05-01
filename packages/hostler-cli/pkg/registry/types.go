// Return-type definitions for package registry.
// Provides compile-time type safety in place of map[string]any.
package registry

// AllocateResult is returned on Allocate success.
type AllocateResult struct {
	Status    string     `json:"status"`
	Resource  string     `json:"resource"`
	Owner     string     `json:"owner"`
	Allocated Allocation `json:"allocated"`
}

// AllocateConflictResult is returned when Allocate detects a conflict.
type AllocateConflictResult struct {
	Status    string       `json:"status"`
	Conflicts []Allocation `json:"conflicts"`
	Message   string       `json:"message"`
}

// CheckResult is returned on Check success.
type CheckResult struct {
	Status     string       `json:"status"`
	Conflicts  []Allocation `json:"conflicts"`
	Message    string       `json:"message,omitempty"`
	Suggestion any          `json:"suggestion,omitempty"`
}

// ResourceSummary is the per-resource summary embedded in a
// ListResources response.
type ResourceSummary struct {
	Name            string       `json:"name"`
	Type            string       `json:"type"`
	Description     string       `json:"description"`
	AllocationCount int          `json:"allocation_count"`
	Allocations     []Allocation `json:"allocations"`
}

// ListResult is returned on ListResources success.
type ListResult struct {
	Resources []ResourceSummary `json:"resources"`
	Project   string            `json:"project"`
	UpdatedAt string            `json:"updated_at"`
}

// SyncResult is returned on SyncToProject success.
type SyncResult struct {
	Synced    bool   `json:"synced"`
	DiffCount int    `json:"diff_count"`
	DocsPath  string `json:"docs_path"`
	Error     string `json:"error,omitempty"`
}
