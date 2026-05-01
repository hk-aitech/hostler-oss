// Package ports — TaskStore port.
//
// The first single-responsibility port. Gathers Task-domain CRUD, state
// transitions, and reads in one place to enable dependency inversion
// (CLI → port → adapter).
//
// Today `GraphStore` already includes every Task method, so any adapter
// implementing GraphStore automatically satisfies TaskStore (SQLiteStore /
// InMemoryStore). This port is an extracted declaration that establishes the
// pattern for subsequent Phase C Tasks to follow.
//
// Verification pattern (bottom of `internal/adapters/sqlite/sqlite_store.go`):
//
//	var _ ports.TaskStore = (*SQLiteStore)(nil)
//
// A successful compile-time check proves SQLiteStore satisfies the TaskStore contract.
package ports

// TaskListFilter is the filter for Task list queries.
//
// nil / empty values disable the corresponding condition. `Since` /
// `Until` apply to the `tasks.updated_at` column. When `Last` > 0 the
// result is sorted `updated_at DESC` and truncated to the top N. When
// `Limit` > 0 the result is truncated. When both `Last` and `Limit` are
// set, `Last` wins.
//
// Semantic SSOT: docs/08-references/standards/cli-filter-schema.md.
type TaskListFilter struct {
	Sprint   *string
	Status   *string
	Priority *string // p0|p1|p2|p3
	Type     *string // feature|bugfix|...
	Since    *string // ISO-8601, updated_at ≥
	Until    *string // ISO-8601, updated_at ≤
	Last     int     // 0 = unlimited, time-desc
	Limit    int     // 0 = unlimited, sort-agnostic truncation
}

// TaskStore — port dedicated to the Task domain (CRUD + state transitions + reads).
// Extracted from GraphStore as a single-responsibility port.
// adapters/fs is intended to become the primary implementation, but today the
// SQLite adapter handles everything.
type TaskStore interface {
	// ── Create / Upsert ─────────────────────────────────────────────────
	CreateTask(task *TaskRecord) error
	InsertTaskAndSyncCounter(task *TaskRecord, maxID int) error
	UpsertTaskInsert(task *TaskRecord, taskNum int) error

	// ── Read ─────────────────────────────────────────────────────────────
	GetTaskFilePath(taskID string) (string, error)
	GetTaskStatus(taskID string) (string, error)
	GetTaskType(taskID string) (string, error)
	GetTaskTypeAndPath(taskID string) (taskType, filePath string, err error)
	GetTaskDetails(taskID string) (*TaskDetails, error)
	GetTaskFull(taskID string) (*TaskGetResult, error)
	GetTaskReopenInfo(taskID string) (*TaskReopenInfo, error)
	HasPlaceholderBody(taskID string) (bool, error)
	FindPlaceholderTasks(sprintFilter string) ([]string, error)

	// ── List / Query ─────────────────────────────────────────────────────
	ListTasks(sprintFilter, statusFilter *string) (*TaskListResult, error)
	// Extended filter — priority/type/since/until/last/limit.
	// Legacy ListTasks is preserved (caller compatibility); new code paths use this method.
	// Detailed semantics: docs/08-references/standards/cli-filter-schema.md.
	ListTasksFiltered(filter TaskListFilter) (*TaskListResult, error)
	NextTask() (*TaskNextResult, error)
	GetAllTasksSync() ([]TaskSyncRecord, error)
	GetNonArchivedTasks() ([]TaskSyncRecord, error)
	GetDoneTasksWithPaths() ([]TaskSyncRecord, error)
	GetBacklogTasks(includeInProgress bool) ([]TaskInfo, error)

	// ── Update ───────────────────────────────────────────────────────────
	UpdateTaskFilePath(taskID, newPath string) error
	UpdateTaskStatus(taskID, status string) error
	UpdateTaskStatusDirect(taskID, status string) error
	UpdateTaskSprint(taskID string, sprint *string) error
	UpdateTaskDirect(taskID string, fields TaskUpdateFields) error

	// ── State Transition ────────────────────────────────────────────────
	TransitionTaskStatus(taskID, expectedStatus, newStatus string) error
	TransitionTaskAtomic(taskID, expectedStatus, newStatus string, fileOp func() (newFilePath string, err error)) error

	// ── Harness coupling (tightly bound to Task state) ──────────────────
	ResetTaskHarness(taskID string) error

	// ── Delete / Archive ────────────────────────────────────────────────
	DeleteTask(taskID string) error
	ArchiveTasks(taskIDs []string) error
}
