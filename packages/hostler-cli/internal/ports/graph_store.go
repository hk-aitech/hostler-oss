// Package ports defines the port interfaces for the hstl domain.
// ADR-0001 Phase B: GraphStore isolates all SQL/storage operations from domain logic.
//
// Phase B+ (current): pkg/* packages migrated to use GraphStore/IDStore methods.
// pkg/db is only used for DB initialisation (InitDB, GetDB) at the entry point.
//
// Gate B criteria:
//   - `grep "db\.GetDB\|sql\." cli/pkg/` hits only inside pkg/db/
//   - All pkg/* use store.Get() (ports.GraphStore) or store.GetIDStore() (ports.IDStore)
package ports

// GraphStore centralises every domain-level storage operation.
// The SQLite adapter lives in internal/adapters/sqlite/.
type GraphStore interface {
	// ── Audit (audit log) ───────────────────────────────────────────────
	AppendAuditEvent(event AuditEvent) error
	QueryAuditEvents(filter AuditQueryFilter) ([]AuditEvent, error)
	SummarizeAuditEvents(since, until string) (map[string]int, error)

	// ── Sprint ──────────────────────────────────────────────────────────
	SaveSprint(sprint *SprintRecord) error
	UpdateSprintStatus(sprintID, status string, opts *SprintUpdateOpts) error
	GetSprint(sprintID string) (*SprintRecord, error)
	ListSprints(status string) ([]SprintRecord, error)
	AggregateSprintProgress(sprintID string) (*ProgressResult, error)
	GetSprintTaskFilePaths(sprintID string) ([]string, error)
	UpdateTaskPathsForSprint(sprintID, fromLocation, toLocation string) error
	UpdateSprintMetadata(sprintID, title, folderPath, status string) error
	InsertSprintIfMissing(sprint *SprintRecord) error
	GetAllSprintsSync() ([]SprintSyncRecord, error)
	// GetAllSprintsForSync returns the full Sprint metadata used by Sprint sync.
	GetAllSprintsForSync() ([]SprintFullSyncRecord, error)
	// GetSprintTaskCounts returns the Task count per sprint.
	GetSprintTaskCounts() (map[string]int, error)
	// UpsertSprintInsert performs INSERT OR REPLACE on the sprints table.
	UpsertSprintInsert(sprint *SprintRecord) error

	// ── Task ────────────────────────────────────────────────────────────
	CreateTask(task *TaskRecord) error
	GetTaskFilePath(taskID string) (string, error)
	GetTaskStatus(taskID string) (string, error)
	GetTaskType(taskID string) (string, error)
	GetTaskTypeAndPath(taskID string) (taskType, filePath string, err error)
	GetTaskDetails(taskID string) (*TaskDetails, error)
	GetTaskFull(taskID string) (*TaskGetResult, error)
	UpdateTaskFilePath(taskID, newPath string) error
	UpdateTaskStatus(taskID, status string) error
	UpdateTaskSprint(taskID string, sprint *string) error
	TransitionTaskStatus(taskID, expectedStatus, newStatus string) error
	ResetTaskHarness(taskID string) error
	ListTasks(sprintFilter, statusFilter *string) (*TaskListResult, error)
	// Extended filter — priority/type/since/until/last/limit.
	ListTasksFiltered(filter TaskListFilter) (*TaskListResult, error)
	NextTask() (*TaskNextResult, error)
	HasPlaceholderBody(taskID string) (bool, error)
	FindPlaceholderTasks(sprintFilter string) ([]string, error)
	GetAllTasksSync() ([]TaskSyncRecord, error)
	InsertTaskAndSyncCounter(task *TaskRecord, maxID int) error
	// GetTaskReopenInfo returns the composite Task data needed by the Reopen operation.
	GetTaskReopenInfo(taskID string) (*TaskReopenInfo, error)
	// UpdateTaskStatusDirect updates the Task status directly, skipping the expected-status check.
	UpdateTaskStatusDirect(taskID, status string) error
	// TransitionTaskAtomic wraps the DB status transition and file operations in a single transaction.
	// Flow:
	//   1) BEGIN; SELECT status → confirm expectedStatus matches
	//   2) UPDATE tasks.status = newStatus
	//   3) call fileOp() — the caller mutates frontmatter / moves the file and returns the new path
	//   4) when fileOp returns newFilePath != "", UPDATE tasks.file_path
	//   5) COMMIT (ROLLBACK on fileOp failure or COMMIT failure)
	// The fileOp implementer is responsible for filesystem rollback (reverting frontmatter, undoing the move).
	// This eliminates drift between file moves, frontmatter, DB status, and DB file_path.
	TransitionTaskAtomic(taskID, expectedStatus, newStatus string, fileOp func() (newFilePath string, err error)) error
	// DeleteHarnessItemsForEntity removes every harness_item attached to the entity.
	DeleteHarnessItemsForEntity(entityType, entityID string) error
	// UpsertTaskInsert performs INSERT OR IGNORE on tasks and syncs the counter.
	UpsertTaskInsert(task *TaskRecord, taskNum int) error
	// SetCounterMax upserts the counter to MAX(current, val).
	SetCounterMax(counterKey string, val int64) error
	// GetNonArchivedTasks returns every (task_id, status) pair that is not archived or superseded.
	GetNonArchivedTasks() ([]TaskSyncRecord, error)
	// ArchiveTasks sets the status of the listed taskIDs to archived.
	ArchiveTasks(taskIDs []string) error
	// UpdateTaskDirect updates the supplied fields unconditionally.
	UpdateTaskDirect(taskID string, fields TaskUpdateFields) error
	// DeleteTask removes the Task from the tasks table.
	DeleteTask(taskID string) error
	// GetDoneTasksWithPaths returns the (task_id, file_path) pairs of every done Task.
	GetDoneTasksWithPaths() ([]TaskSyncRecord, error)

	// ── Backlog ─────────────────────────────────────────────────────────
	GetBacklogTasks(includeInProgress bool) ([]TaskInfo, error)

	// ── Harness ─────────────────────────────────────────────────────────
	CountHarnessItems(entityType, entityID string) (int, error)
	EnsureHarnessItems(entityType, entityID string, items []HarnessItemTemplate) error
	GetHarnessItems(entityType, entityID string) ([]HarnessItem, error)
	GetHarnessTemplate(templateKey string) ([]map[string]any, error)
	GetHarnessItemID(entityType, entityID, itemID string) (int64, error)
	CheckHarnessItem(entityType, entityID, itemID, evidence, actor string) error
	CheckSprintProduction(sprintID string) (bool, error)

	// ── Project Meta ────────────────────────────────────────────────────
	EnsureProjectSecret() ([]byte, error)
	// RotateProjectSecret replaces project_meta.task_hmac_secret with newHexSecret.
	RotateProjectSecret(newHexSecret string) error

	// ── Work Ticket / Context Acknowledgment ────────────────────────────
	GetTaskWorkTicket(taskID string) (string, error)
	SetTaskWorkTicket(taskID, ticket string) error
	InsertContextAck(ticket, hash string, size int) error
	HasContextAck(ticket, hash string) (bool, error)
	// HasContextAckFresh — when ttlMinutes > 0, verifies that the ack
	// was created within the last ttlMinutes minutes.
	// When ttlMinutes <= 0, behaves identically to HasContextAck (no expiry).
	HasContextAckFresh(ticket, hash string, ttlMinutes int) (bool, error)
}

// IDStore manages atomic counter/ID generation.
// Separated from GraphStore because ID generation requires serialised
// transactions that are best isolated from bulk query operations.
type IDStore interface {
	HealAndGetNextTaskID() (string, error)
	NextCounter(counterKey string) (int64, error)
	// BumpTaskCounter raises the task counter to at least minValue.
	// Called before HealAndGetNextTaskID when SPRINT.md / BACKLOG.md scan
	// finds placeholder IDs higher than DB MAX. Idempotent: values <=
	// current counter are no-ops. Used for placeholder collision
	// prevention on the mailbox accept + task create paths.
	BumpTaskCounter(minValue int64) error
}

// ── Value types ─────────────────────────────────────────────────────────────

// AuditEvent represents a single audit log entry.
// The ID field exposes the SQLite `audit_events.id` PK so
// `hstl audit -o json` reports a real id. JSON tags serialize in the
// same snake_case as `pkg/audit.Event` to preserve the
// `hstl audit -o json` output schema.
type AuditEvent struct {
	ID         int64  `json:"id"`        // SQLite rowid / PK (populated by QueryAuditEvents; 0 for legacy rows)
	Timestamp  string `json:"timestamp"` // ISO 8601 (populated by QueryAuditEvents)
	EventType  string `json:"event_type"`
	EntityType string `json:"entity_type"`
	EntityID   string `json:"entity_id"`
	ActorID    string `json:"actor"`
	Details    any    `json:"details,omitempty"`
	SessionID  string `json:"session_id,omitempty"`
}

// AuditQueryFilter filters audit event queries.
type AuditQueryFilter struct {
	EntityType string
	EntityID   string
	EventType  string
	Actor      string
	Since      string
	Until      string
	Limit      int
}

// SprintRecord mirrors the sprints DB row.
type SprintRecord struct {
	SprintID    string
	Title       string
	Status      string
	Goal        string
	FolderPath  string
	StartedAt   string
	CompletedAt string
	CreatedAt   string
	UpdatedAt   string
}

// SprintUpdateOpts carries optional fields for UpdateSprintStatus.
type SprintUpdateOpts struct {
	FolderPath  string
	StartedAt   string
	CompletedAt string
}

// BurndownEntry holds a single day's completed task count.
type BurndownEntry struct {
	Date string
	Done int
}

// ProgressResult holds sprint burndown aggregates.
type ProgressResult struct {
	SprintID   string
	Total      int
	Done       int
	InProgress int
	Todo       int
	Percent    int
	Progress   int // deprecated alias for Percent; kept for backward compat
	Burndown   []BurndownEntry
}

// SprintSyncRecord is a lightweight Sprint row used during backlog sync.
type SprintSyncRecord struct {
	SprintID string
	Status   string
}

// SprintFullSyncRecord holds full Sprint metadata for sync/consistency checks.
type SprintFullSyncRecord struct {
	SprintID    string
	Title       string
	Status      string
	StartedAt   string
	CompletedAt string
	FolderPath  string
}

// TaskRecord mirrors the tasks DB row for inserts.
type TaskRecord struct {
	TaskID    string
	Title     string
	Type      string
	Status    string
	Priority  string
	Estimate  string
	Sprint    string
	FilePath  string
	DependsOn string
	CreatedAt string
	// Record the work_ticket frontmatter alongside the INSERT.
	// An empty string is stored as SQL NULL. Keeps the backlog sync
	// restore path consistent with frontmatter → DB propagation.
	WorkTicket string
}

// TaskDetails holds commonly queried task fields.
type TaskDetails struct {
	TaskID    string
	Title     string
	Type      string
	Status    string
	Priority  string
	Estimate  string
	Sprint    string
	FilePath  string
	DependsOn string
	CreatedAt string
	UpdatedAt string
}

// TaskGetResult is the full task row used by task get.
// Embeds TaskDetails which already includes CreatedAt, UpdatedAt.
type TaskGetResult struct {
	TaskDetails
}

// TaskListResult holds the result of a task list query.
type TaskListResult struct {
	Tasks                 []TaskDetails
	OutdatedTemplateCount int
}

// TaskNextResult holds the result of the next-task query.
type TaskNextResult struct {
	TaskID string
	Title  string
}

// TaskSyncRecord is a lightweight task row used during backlog sync.
type TaskSyncRecord struct {
	TaskID   string
	Status   string
	Sprint   string
	FilePath string
}

// TaskReopenInfo holds all fields needed by the Reopen operation.
type TaskReopenInfo struct {
	TaskID        string
	Status        string
	FilePath      string
	SprintID      string
	Title         string
	Type          string
	Estimate      string
	Priority      string
	DependsOnJSON string
}

// TaskUpdateFields holds optional fields for UpdateTaskDirect.
// Non-nil pointer fields are applied; nil pointers are skipped.
type TaskUpdateFields struct {
	Status    *string
	Sprint    *string
	FilePath  *string
	Title     *string
	Type      *string
	Priority  *string
	Estimate  *string
	DependsOn *string // JSON-encoded string
}

// TaskInfo is used by GetBacklogTasks.
type TaskInfo struct {
	TaskID   string
	Title    string
	Type     string
	Priority string
	Estimate string
	Sprint   string
	Status   string
}

// HarnessItemTemplate describes a harness check item template.
type HarnessItemTemplate struct {
	ID          string
	Description string
	Required    bool
}

// HarnessItem represents a single harness gate check item.
type HarnessItem struct {
	ID          string
	Description string
	Required    bool
	Done        bool
	Evidence    string
	CheckedAt   string
	CheckedBy   string
}
