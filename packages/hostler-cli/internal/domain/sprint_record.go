// Package domain — Sprint record / metadata DTOs.
//
// ADR-001 §9 E2 — promoted the record / lookup DTOs from pkg/sprint into
// internal/domain.
package domain

// SprintRecord is a single sprint row.
type SprintRecord struct {
	SprintID    string `json:"sprint_id"`
	Title       string `json:"title"`
	Status      string `json:"status"`
	FolderPath  string `json:"folder_path"`
	StartedAt   string `json:"started_at"`
	CompletedAt string `json:"completed_at"`
	Goal        string `json:"goal"`
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`
}

// LocatedSprint — location / metadata for a Sprint found on disk.
type LocatedSprint struct {
	SprintID    string         // e.g. sprint-70
	Location    string         // active|backlog|completed (per folder)
	FolderPath  string         // path relative to the repo root
	SprintMD    string         // absolute path to SPRINT.md
	Frontmatter map[string]any // parsed frontmatter (nil when absent)
}

// BurndownEntry — count of completed Tasks per date.
type BurndownEntry struct {
	Date string `json:"date"`
	Done int    `json:"done"`
}
