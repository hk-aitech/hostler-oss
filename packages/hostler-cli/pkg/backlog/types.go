// Package backlog handles consistency verification and synchronisation
// across Task files, the DB, and BACKLOG.md.
package backlog

// Issue is a single detected consistency problem.
type Issue struct {
	Type        string `json:"type"`
	TaskID      string `json:"task_id,omitempty"`
	File        string `json:"file,omitempty"`
	Detail      string `json:"detail,omitempty"`
	AutoFixable bool   `json:"auto_fixable"`
	Message     string `json:"message"`

	// Type-specific extra fields.
	Files          []string `json:"files,omitempty"`
	FMSprint       string   `json:"fm_sprint,omitempty"`
	SprintID       string   `json:"sprint_id,omitempty"`
	DBStatus       string   `json:"db_status,omitempty"`
	FileStatus     string   `json:"file_status,omitempty"`
	DBSprint       string   `json:"db_sprint,omitempty"`
	FileSprint     string   `json:"file_sprint,omitempty"`
	DBFilePath     string   `json:"db_file_path,omitempty"`
	ActualFilePath string   `json:"actual_file_path,omitempty"`
	Error          string   `json:"error,omitempty"`

	// Internal repair data (stripped before response serialisation).
	fm         map[string]any
	dbSprints  []dbSprintRow
	sprintInfo *sprintFileInfo // for db_missing_sprint repair
}

// Fix is one completed auto-repair entry.
type Fix struct {
	Type        string `json:"type"`
	TaskID      string `json:"task_id,omitempty"`
	SprintID    string `json:"sprint_id,omitempty"`
	NewStatus   string `json:"new_status,omitempty"`
	OldSprint   string `json:"old_sprint,omitempty"` // location_frontmatter_sprint_cleared
	NewSprint   string `json:"new_sprint,omitempty"`
	NewFilePath string `json:"new_file_path,omitempty"`
	Timestamp   string `json:"timestamp,omitempty"`
	TodoCount   int    `json:"todo_count,omitempty"`
}

// Summary summarises a backlog_sync run.
type Summary struct {
	TotalIssues      int  `json:"total_issues"`
	AutoFixable      int  `json:"auto_fixable"`
	AutoFixed        int  `json:"auto_fixed"`
	ManualRequired   int  `json:"manual_required"`
	DryRun           bool `json:"dry_run"`
	TaskFilesScanned int  `json:"task_files_scanned"`
}

// SyncResult is the full backlog_sync result.
type SyncResult struct {
	Issues  []Issue `json:"issues"`
	Fixes   []Fix   `json:"fixes"`
	Summary Summary `json:"summary"`
}

// dbTaskRow is one row read from the DB tasks table.
type dbTaskRow struct {
	taskID   string
	status   string
	sprint   *string
	filePath string
}

// dbSprintRow is one row read from the DB sprints table.
type dbSprintRow struct {
	sprintID string
	title    string
	status   string
}

// dirLocation is the analysis result for a file-path location.
// `inTasksCompleted` distinguishes the `works/tasks/completed/T*.md`
// location (the auto-move destination for unassigned Tasks) from the
// other paths so that the auto-removal branch for BACKLOG.md drift can
// behave correctly.
type dirLocation struct {
	inBacklogTasks   bool
	inTasksCompleted bool
	inSprintTasks    bool
	sprintID         string
	sprintStatus     string
}
