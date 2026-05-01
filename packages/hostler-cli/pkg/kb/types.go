// Package kb return-type definitions.
// Provides compile-time type safety instead of map[string]any.
package kb

// RecountResult is the return type of RecountIndex.
type RecountResult struct {
	TotalCards     int      `json:"total_cards"`
	FilesScanned   int      `json:"files_scanned"`
	FilesUpdated   []string `json:"files_updated"`
	MissingInIndex []string `json:"missing_in_index"`
	AddedToIndex   []string `json:"added_to_index"`
}
