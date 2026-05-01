// Package noop provides a no-op implementation of ports.GraphStore and
// ports.IDStore. Every method returns a zero value and a nil error, making it
// suitable for use in tests, dry-run modes, and build configurations that do
// not require persistent storage.
package noop

import "github.com/hk-aitech/hostler-oss/packages/hostler-cli/internal/ports"

// Compile-time interface assertions.

var _ ports.GraphStore = (*NoopStore)(nil)
var _ ports.IDStore = (*NoopStore)(nil)
var _ ports.TaskStore = (*NoopStore)(nil)
var _ ports.SprintStore = (*NoopStore)(nil)
var _ ports.HarnessStore = (*NoopStore)(nil)
var _ ports.AuditLog = (*NoopStore)(nil)

// NoopStore is a store whose methods all return a zero value and a nil error.
type NoopStore struct{}

// New returns a fresh NoopStore instance.
func New() *NoopStore { return &NoopStore{} }

// Audit.

func (n *NoopStore) AppendAuditEvent(_ ports.AuditEvent) error { return nil }

func (n *NoopStore) QueryAuditEvents(_ ports.AuditQueryFilter) ([]ports.AuditEvent, error) {
	return nil, nil
}

func (n *NoopStore) SummarizeAuditEvents(_, _ string) (map[string]int, error) {
	return map[string]int{}, nil
}

// Sprint.

func (n *NoopStore) SaveSprint(_ *ports.SprintRecord) error { return nil }

func (n *NoopStore) UpdateSprintStatus(_, _ string, _ *ports.SprintUpdateOpts) error { return nil }

func (n *NoopStore) GetSprint(_ string) (*ports.SprintRecord, error) { return nil, nil }

func (n *NoopStore) ListSprints(_ string) ([]ports.SprintRecord, error) { return nil, nil }

func (n *NoopStore) AggregateSprintProgress(_ string) (*ports.ProgressResult, error) {
	return &ports.ProgressResult{Burndown: []ports.BurndownEntry{}}, nil
}

func (n *NoopStore) GetSprintTaskFilePaths(_ string) ([]string, error) { return nil, nil }

func (n *NoopStore) UpdateTaskPathsForSprint(_, _, _ string) error { return nil }

func (n *NoopStore) UpdateSprintMetadata(_, _, _, _ string) error { return nil }

func (n *NoopStore) InsertSprintIfMissing(_ *ports.SprintRecord) error { return nil }

func (n *NoopStore) GetAllSprintsSync() ([]ports.SprintSyncRecord, error) { return nil, nil }

// Task.

func (n *NoopStore) CreateTask(_ *ports.TaskRecord) error { return nil }

func (n *NoopStore) GetTaskFilePath(_ string) (string, error) { return "", nil }

func (n *NoopStore) GetTaskStatus(_ string) (string, error) { return "", nil }

func (n *NoopStore) GetTaskType(_ string) (string, error) { return "", nil }

func (n *NoopStore) GetTaskTypeAndPath(_ string) (string, string, error) { return "", "", nil }

func (n *NoopStore) GetTaskDetails(_ string) (*ports.TaskDetails, error) { return nil, nil }

func (n *NoopStore) GetTaskFull(_ string) (*ports.TaskGetResult, error) { return nil, nil }

func (n *NoopStore) UpdateTaskFilePath(_, _ string) error { return nil }

func (n *NoopStore) UpdateTaskStatus(_, _ string) error { return nil }

func (n *NoopStore) UpdateTaskSprint(_ string, _ *string) error { return nil }

func (n *NoopStore) TransitionTaskStatus(_, _, _ string) error { return nil }

func (n *NoopStore) ResetTaskHarness(_ string) error { return nil }

func (n *NoopStore) ListTasks(_, _ *string) (*ports.TaskListResult, error) {
	return &ports.TaskListResult{}, nil
}

// ListTasksFiltered noop stub.
func (n *NoopStore) ListTasksFiltered(_ ports.TaskListFilter) (*ports.TaskListResult, error) {
	return &ports.TaskListResult{}, nil
}

func (n *NoopStore) NextTask() (*ports.TaskNextResult, error) { return nil, nil }

func (n *NoopStore) HasPlaceholderBody(_ string) (bool, error) { return false, nil }

func (n *NoopStore) FindPlaceholderTasks(_ string) ([]string, error) { return nil, nil }

func (n *NoopStore) GetAllTasksSync() ([]ports.TaskSyncRecord, error) { return nil, nil }

func (n *NoopStore) InsertTaskAndSyncCounter(_ *ports.TaskRecord, _ int) error { return nil }

func (n *NoopStore) GetTaskReopenInfo(_ string) (*ports.TaskReopenInfo, error) { return nil, nil }

func (n *NoopStore) UpdateTaskStatusDirect(_, _ string) error { return nil }

func (n *NoopStore) TransitionTaskAtomic(
	_, _, _ string,
	fileOp func() (string, error),
) error {
	if fileOp == nil {
		return nil
	}
	_, err := fileOp()
	return err
}

func (n *NoopStore) DeleteHarnessItemsForEntity(_, _ string) error { return nil }

func (n *NoopStore) UpsertTaskInsert(_ *ports.TaskRecord, _ int) error { return nil }

func (n *NoopStore) UpsertSprintInsert(_ *ports.SprintRecord) error { return nil }

func (n *NoopStore) SetCounterMax(_ string, _ int64) error { return nil }

func (n *NoopStore) GetNonArchivedTasks() ([]ports.TaskSyncRecord, error) { return nil, nil }

func (n *NoopStore) ArchiveTasks(_ []string) error { return nil }

func (n *NoopStore) GetAllSprintsForSync() ([]ports.SprintFullSyncRecord, error) { return nil, nil }

func (n *NoopStore) GetSprintTaskCounts() (map[string]int, error) { return map[string]int{}, nil }

func (n *NoopStore) UpdateTaskDirect(_ string, _ ports.TaskUpdateFields) error { return nil }

func (n *NoopStore) DeleteTask(_ string) error { return nil }

func (n *NoopStore) GetDoneTasksWithPaths() ([]ports.TaskSyncRecord, error) { return nil, nil }

// Backlog.

func (n *NoopStore) GetBacklogTasks(_ bool) ([]ports.TaskInfo, error) { return nil, nil }

// Harness.

func (n *NoopStore) CountHarnessItems(_, _ string) (int, error) { return 0, nil }

func (n *NoopStore) EnsureHarnessItems(_, _ string, _ []ports.HarnessItemTemplate) error {
	return nil
}

func (n *NoopStore) GetHarnessItems(_, _ string) ([]ports.HarnessItem, error) { return nil, nil }

func (n *NoopStore) GetHarnessTemplate(_ string) ([]map[string]any, error) { return nil, nil }

func (n *NoopStore) GetHarnessItemID(_, _, _ string) (int64, error) { return 0, nil }

func (n *NoopStore) CheckHarnessItem(_, _, _, _, _ string) error { return nil }

func (n *NoopStore) CheckSprintProduction(_ string) (bool, error) { return false, nil }

func (n *NoopStore) EnsureProjectSecret() ([]byte, error) { return nil, nil }

func (n *NoopStore) RotateProjectSecret(_ string) error { return nil }

func (n *NoopStore) GetTaskWorkTicket(_ string) (string, error) { return "", nil }

func (n *NoopStore) SetTaskWorkTicket(_, _ string) error { return nil }

func (n *NoopStore) InsertContextAck(_, _ string, _ int) error { return nil }

func (n *NoopStore) HasContextAck(_, _ string) (bool, error) { return false, nil }

func (n *NoopStore) HasContextAckFresh(_, _ string, _ int) (bool, error) { return false, nil }

// IDStore.

func (n *NoopStore) HealAndGetNextTaskID() (string, error) { return "", nil }

func (n *NoopStore) NextCounter(_ string) (int64, error) { return 0, nil }

// BumpTaskCounter — the noop store does not maintain a counter, so this
// is always a no-op success.
func (n *NoopStore) BumpTaskCounter(_ int64) error { return nil }
