// Package domain — DTOs returned by the Sprint service.
//
// ADR-001 §9 E2.
package domain

// SprintCreateResult — return type of sprint.Create.
type SprintCreateResult struct {
	SprintID   string `json:"sprint_id"`
	Title      string `json:"title"`
	Goal       string `json:"goal"`
	Status     string `json:"status"`
	FolderPath string `json:"folder_path"`
}

// SprintStartResult — return type of sprint.Start.
type SprintStartResult struct {
	SprintID   string `json:"sprint_id"`
	Status     string `json:"status"`
	StartedAt  string `json:"started_at"`
	FolderPath string `json:"folder_path"`
}

// SprintCompleteResult — return type of sprint.Complete.
//
// SideEffects: list of files modified during sprint.Complete (repo-relative
// paths). Includes CURRENT-FOCUS.md / SPRINT.md / Task files moved across
// folders — anything that escapes the atomicity boundary. Callers can
// pinpoint the next commit boundary without ambiguity. Resolves the root
// cause of issue ISS-20260419-003.
type SprintCompleteResult struct {
	SprintID    string             `json:"sprint_id"`
	Status      string             `json:"status"`
	CompletedAt string             `json:"completed_at"`
	FolderPath  string             `json:"folder_path"`
	SideEffects []SprintSideEffect `json:"side_effects,omitempty"`
}

// SprintSideEffect — describes a single file that sprint.Complete modified.
type SprintSideEffect struct {
	File   string `json:"file"`   // path relative to the repo
	Change string `json:"change"` // "moved" | "updated" | "stamped"
}

// SprintDiscardResult — return type of sprint.Discard.
type SprintDiscardResult struct {
	SprintID       string   `json:"sprint_id"`
	PreviousStatus string   `json:"previous_status"`
	Status         string   `json:"status"`
	Reason         string   `json:"reason"`
	FolderPath     string   `json:"folder_path,omitempty"`
	ReturnedTasks  []string `json:"returned_tasks,omitempty"`
	DBOnly         bool     `json:"db_only,omitempty"` // when true (--db-only), folders are not moved and only the DB is updated
}

// ProgressResult — return type of AggregateProgress.
type ProgressResult struct {
	SprintID   string          `json:"sprint_id"`
	Done       int             `json:"done"`
	InProgress int             `json:"in_progress"`
	Todo       int             `json:"todo"`
	Total      int             `json:"total"`
	Percent    int             `json:"percent"`
	Burndown   []BurndownEntry `json:"burndown"`
}

// ReconcileDiff — difference between the file-based expectation and the current DB value.
type ReconcileDiff struct {
	Field   string `json:"field"`
	DBValue string `json:"db_value"`
	File    string `json:"file_value"`
}

// TaskDrift — mismatch between the Task's DB file_path and its actual file location.
type TaskDrift struct {
	TaskID     string `json:"task_id"`
	DBFilePath string `json:"db_file_path"`
	ActualPath string `json:"actual_path,omitempty"` // populated when located, empty when not found
}

// ReconcileResult — return type of Reconcile.
type ReconcileResult struct {
	SprintID   string          `json:"sprint_id"`
	DryRun     bool            `json:"dry_run"`
	Applied    bool            `json:"applied"`
	Diffs      []ReconcileDiff `json:"diffs"`
	TaskDrifts []TaskDrift     `json:"task_drifts,omitempty"`
	FolderPath string          `json:"folder_path"`
	Status     string          `json:"status"`
	Title      string          `json:"title"`
}
