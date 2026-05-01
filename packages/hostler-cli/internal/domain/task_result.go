// Package domain — bundle of return DTOs for the Task service.
//
// ADR-001 §9 E2 — promoted the Result DTOs scattered across
// pkg/task/types.go into internal/domain. pkg/task keeps backward
// compatibility through type aliases; cmd asserts directly against
// *domain.X.
//
// Original JSON tags must be preserved to avoid breaking the HMAC canonical round trip.
package domain

// CreateResult — return type of a successful task.Create.
type CreateResult struct {
	TaskID    string   `json:"task_id"`
	FilePath  string   `json:"file_path"`
	CreatedAt string   `json:"created_at"`
	Status    string   `json:"status"`
	Warnings  []string `json:"warnings,omitempty"`
}

// TransitionResult — return type of a successful state transition (Start / Complete / Reopen / etc.).
type TransitionResult struct {
	TaskID               string              `json:"task_id"`
	PreviousStatus       string              `json:"previous_status"`
	NewStatus            string              `json:"new_status"`
	Title                string              `json:"title"`
	Type                 string              `json:"type"`
	Estimate             string              `json:"estimate"`
	Priority             string              `json:"priority"`
	Sprint               any                 `json:"sprint"`
	DependsOn            []string            `json:"depends_on"`
	CommitPrefix         string              `json:"commit_prefix"`
	FileUpdated          bool                `json:"file_updated"`
	Warning              string              `json:"warning,omitempty"`
	Reminders            []string            `json:"reminders,omitempty"`
	SuggestedNextAction  string              `json:"suggested_next_action,omitempty"`
	ArtifactWarnings     []string            `json:"artifact_warnings,omitempty"`
	FollowupWarnings     []string            `json:"followup_warnings,omitempty"`
	PostActions          []string            `json:"post_actions,omitempty"`
	VerificationResult   *VerificationResult `json:"verification_result,omitempty"`
	WorkTicket           string              `json:"work_ticket,omitempty"`
	ResultSectionStubbed bool                `json:"result_section_stubbed,omitempty"`
	// ClaimDriftWarnings — warning messages produced when re-verification of
	// bash-executable claims (grep/wc/ls/rg/find) inside the body detects drift.
	ClaimDriftWarnings []string `json:"claim_drift_warnings,omitempty"`
}

// VerificationResult — detailed result of task_complete result-section verification.
//
// Policy: applied policy (off/warn/strict).
// SectionTitlesUsed: the list of result-section titles that were recognised.
// CheckedFiles: the list of file paths extracted from the result section.
// MissingFiles: files missing from the git diff or with mismatched paths
// (populated even at warn level).
type VerificationResult struct {
	Policy            string        `json:"policy"`
	SectionTitlesUsed []string      `json:"section_titles_used"`
	CheckedFiles      []string      `json:"checked_files"`
	MissingFiles      []MissingFile `json:"missing_files"`
}

// MissingFile — file that failed verification.
type MissingFile struct {
	Path   string `json:"path"`
	Reason string `json:"reason"`
}

// ListResult — return type of task.List.
type ListResult struct {
	Tasks []TaskSummary `json:"tasks"`
	Count int           `json:"count"`

	// OutdatedTemplateCount — count of Tasks with TemplateOutdated=true.
	OutdatedTemplateCount int `json:"outdated_template_count"`
}

// GetResult — return type of task.Get.
type GetResult struct {
	TaskID      string   `json:"task_id"`
	Title       string   `json:"title"`
	Type        string   `json:"type"`
	Sprint      any      `json:"sprint"`
	Status      string   `json:"status"`
	Priority    string   `json:"priority"`
	Estimate    string   `json:"estimate"`
	FilePath    string   `json:"file_path"`
	Summary     string   `json:"summary"` // first line of the ## Summary section
	DependsOn   []string `json:"depends_on"`
	CreatedAt   string   `json:"created_at"`
	UpdatedAt   string   `json:"updated_at"`
	Frontmatter any      `json:"frontmatter"`
	FileExists  *bool    `json:"file_exists,omitempty"`
}

// NextMessageResult — message returned when Next has no recommendation.
type NextMessageResult struct {
	Message string `json:"message"`
}

// UpdateResult — return type of task.Update.
type UpdateResult struct {
	TaskID        string         `json:"task_id"`
	UpdatedFields []string       `json:"updated_fields"`
	OldValues     map[string]any `json:"old_values"`
	NewValues     map[string]any `json:"new_values"`
	Warnings      []string       `json:"warnings,omitempty"`
}

// ReopenResult — return type of Reopen.
type ReopenResult struct {
	TaskID          string `json:"task_id"`
	PreviousStatus  string `json:"previous_status"`
	NewStatus       string `json:"new_status"`
	Reason          string `json:"reason"`
	HistoryAppended bool   `json:"history_appended"`
	HarnessReset    bool   `json:"harness_reset"`
}

// DeleteResult — return type of Delete.
type DeleteResult struct {
	TaskID string `json:"task_id"`
	Status string `json:"status"`
	Reason string `json:"reason"`
}

// FailedItem — failed entry from a batch operation.
type FailedItem struct {
	TaskID      string          `json:"task_id"`
	Reason      string          `json:"reason"`
	Rollback    string          `json:"rollback,omitempty"`
	Checkpoints map[string]bool `json:"checkpoints,omitempty"`
}

// SkippedItem — entry skipped by a batch operation.
type SkippedItem struct {
	TaskID string `json:"task_id"`
	Reason string `json:"reason"`
}

// AssignSprintResult — return type of AssignSprint.
type AssignSprintResult struct {
	SprintID       string        `json:"sprint_id"`
	Assigned       []string      `json:"assigned"`
	Skipped        []SkippedItem `json:"skipped"`
	Failed         []FailedItem  `json:"failed"`
	BacklogSummary any           `json:"backlog_summary"`
}

// UnassignSprintResult — return type of UnassignSprint.
type UnassignSprintResult struct {
	Unassigned []string      `json:"unassigned"`
	Skipped    []SkippedItem `json:"skipped"`
	Failed     []FailedItem  `json:"failed"`
	Warnings   []string      `json:"warnings"`
}

// CheckpointResult — return type of the task checkpoint CLI.
type CheckpointResult struct {
	TaskID    string `json:"task_id"`
	OldTicket string `json:"old_ticket"`
	NewTicket string `json:"new_ticket"`
	Reason    string `json:"reason,omitempty"`
}

// RotateResult — outcome of running rotate-hmac-secret.
type RotateResult struct {
	OldSecretPrefix string             `json:"old_secret_prefix"` // first 4 hex of the previous secret
	NewSecretPrefix string             `json:"new_secret_prefix"` // first 4 hex of the new secret
	TotalDone       int                `json:"total_done"`        // total number of done Tasks
	Resigned        int                `json:"resigned"`          // tasks actually re-signed
	Skipped         int                `json:"skipped"`           // tasks skipped
	Failed          int                `json:"failed"`            // tasks that failed
	DryRun          bool               `json:"dry_run"`
	Candidates      []RotateCandidate  `json:"candidates"` // affected Tasks during dry-run
	FailedTasks     []RotateFailedItem `json:"failed_tasks,omitempty"`
}

// RotateCandidate — Task targeted for re-signing.
type RotateCandidate struct {
	TaskID   string `json:"task_id"`
	FilePath string `json:"file_path"`
	HasHMAC  bool   `json:"has_hmac"` // false = legacy (no completion_hmac)
}

// RotateFailedItem — entry that failed re-signing.
type RotateFailedItem struct {
	TaskID   string `json:"task_id"`
	FilePath string `json:"file_path"`
	Reason   string `json:"reason"`
}
