// Package fileutil return-type definitions.
// Provides compile-time type safety instead of using map[string]any.
package fileutil

// BacklogSummaryResult is the return type of RebuildBacklogSummary.
type BacklogSummaryResult struct {
	Updated   bool   `json:"updated"`
	Timestamp string `json:"timestamp,omitempty"`
	TodoCount int    `json:"todo_count,omitempty"`
	Reason    string `json:"reason,omitempty"`
}
