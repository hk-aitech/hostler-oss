// Package inmemory provides an in-memory implementation of ports.GraphStore
// and ports.IDStore for integration tests.
//
// Behaviour:
//   - All data is held in process memory (concurrency-protected with
//     sync.RWMutex).
//   - **Semantically equivalent** to the SQLite adapter — the same call
//     sequence yields the same result.
//   - ID generation, atomic state transitions, hashed secrets, etc. honour
//     the same contract as SQLite.
//
// Scope:
//   - Used in place of SQLite during integration tests, removing the
//     filesystem dependency.
//   - Must not be used as production storage (data is lost on process exit).
//
// Design choices:
//   - Single file — the real logic is simple and a 1:1 mapping with the
//     SQLite adapter is easier to maintain.
//   - Deep copies — callers can mutate returned pointers without affecting
//     internal state.
//   - Sorted slices — list returns are deterministic (e.g. ascending TaskID),
//     preventing flaky tests.
package inmemory

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/internal/ports"
)

// Compile-time interface assertions.

var _ ports.GraphStore = (*InMemoryStore)(nil)
var _ ports.IDStore = (*InMemoryStore)(nil)
var _ ports.TaskStore = (*InMemoryStore)(nil)
var _ ports.SprintStore = (*InMemoryStore)(nil)
var _ ports.HarnessStore = (*InMemoryStore)(nil)
var _ ports.AuditLog = (*InMemoryStore)(nil)

// InMemoryStore is the in-memory implementation of ports.GraphStore +
// ports.IDStore.
type InMemoryStore struct {
	mu sync.RWMutex

	// Core entities.
	tasks   map[string]*taskEntry
	sprints map[string]*ports.SprintRecord

	// Audit log — order matters, so it stays a slice.
	auditEvents []ports.AuditEvent

	// harness — key: entityType|entityID|itemID
	harnessItems map[string]*ports.HarnessItem

	// harness template — key: templateKey (e.g. task:bugfix)
	harnessTemplates map[string][]map[string]any

	// Counters — used for ID generation.
	counters map[string]int64

	// Project secret — used for HMAC.
	projectSecret []byte

	// Per-task work ticket.
	workTickets map[string]string

	// Context ack — key: ticket|hash, value: creation time (UTC).
	// Stored as time.Time so TTL validation can be performed.
	contextAcks map[string]time.Time

	// Task body content (used to determine placeholder state).
	taskBodies map[string]string
}

// taskEntry is an in-memory-only Task wrapper (file body + frontmatter +
// status).
type taskEntry struct {
	rec ports.TaskDetails
}

// New returns an empty InMemoryStore.
func New() *InMemoryStore {
	return &InMemoryStore{
		tasks:            make(map[string]*taskEntry),
		sprints:          make(map[string]*ports.SprintRecord),
		harnessItems:     make(map[string]*ports.HarnessItem),
		harnessTemplates: make(map[string][]map[string]any),
		counters:         make(map[string]int64),
		workTickets:      make(map[string]string),
		contextAcks:      make(map[string]time.Time),
		taskBodies:       make(map[string]string),
	}
}

// NewForTest is a convenience factory that builds a store instance quickly
// for use in tests. It seeds default harness templates so common
// task/sprint scenarios work out of the box.
func NewForTest() *InMemoryStore {
	s := New()
	// Common harness template defaults (a mini version of the SQLite
	// adapter's harness_defaults.json).
	s.harnessTemplates["task:feature"] = []map[string]any{
		{"id": "criteria_checked", "name": "Criteria checked", "required": true},
		{"id": "build_passed", "name": "Build", "required": true},
		{"id": "tests_passed", "name": "Tests", "required": true},
	}
	s.harnessTemplates["task:chore"] = []map[string]any{
		{"id": "criteria_checked", "name": "Criteria checked", "required": true},
	}
	s.harnessTemplates["sprint:default"] = []map[string]any{
		{"id": "phase5_retro", "name": "Phase 5 retro", "required": true},
		{"id": "phase6_learned", "name": "Phase 6 learned", "required": true},
	}
	return s
}

// Compile-time interface assertions.
var (
	_ ports.GraphStore = (*InMemoryStore)(nil)
	_ ports.IDStore    = (*InMemoryStore)(nil)
)

// harnessKey is the format for harness map keys.
func harnessKey(entityType, entityID, itemID string) string {
	return entityType + "|" + entityID + "|" + itemID
}

func nowISO() string {
	return time.Now().UTC().Format(time.RFC3339)
}

// --- Audit ------------------------------------------------------------------

func (s *InMemoryStore) AppendAuditEvent(event ports.AuditEvent) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	ev := event
	if ev.Timestamp == "" {
		ev.Timestamp = nowISO()
	}
	s.auditEvents = append(s.auditEvents, ev)
	return nil
}

func (s *InMemoryStore) QueryAuditEvents(filter ports.AuditQueryFilter) ([]ports.AuditEvent, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	limit := filter.Limit
	if limit <= 0 {
		limit = 100
	}

	matches := func(ev ports.AuditEvent) bool {
		if filter.EntityType != "" && ev.EntityType != filter.EntityType {
			return false
		}
		if filter.EntityID != "" && ev.EntityID != filter.EntityID {
			return false
		}
		if filter.EventType != "" && ev.EventType != filter.EventType {
			return false
		}
		if filter.Actor != "" && ev.ActorID != filter.Actor {
			return false
		}
		if filter.Since != "" && ev.Timestamp < filter.Since {
			return false
		}
		if filter.Until != "" && ev.Timestamp > filter.Until {
			return false
		}
		return true
	}

	var out []ports.AuditEvent
	// Reverse order (newest first).
	for i := len(s.auditEvents) - 1; i >= 0 && len(out) < limit; i-- {
		if matches(s.auditEvents[i]) {
			out = append(out, s.auditEvents[i])
		}
	}
	return out, nil
}

func (s *InMemoryStore) SummarizeAuditEvents(since, until string) (map[string]int, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make(map[string]int)
	for _, ev := range s.auditEvents {
		if since != "" && ev.Timestamp < since {
			continue
		}
		if until != "" && ev.Timestamp > until {
			continue
		}
		out[ev.EventType]++
	}
	return out, nil
}

// --- Sprint -----------------------------------------------------------------

func (s *InMemoryStore) SaveSprint(sprint *ports.SprintRecord) error {
	if sprint == nil {
		return fmt.Errorf("sprint is nil")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	r := *sprint
	if r.CreatedAt == "" {
		r.CreatedAt = nowISO()
	}
	r.UpdatedAt = nowISO()
	s.sprints[r.SprintID] = &r
	return nil
}

func (s *InMemoryStore) UpdateSprintStatus(sprintID, status string, opts *ports.SprintUpdateOpts) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	sp, ok := s.sprints[sprintID]
	if !ok {
		return fmt.Errorf("sprint not found: %s", sprintID)
	}
	sp.Status = status
	if opts != nil {
		if opts.FolderPath != "" {
			sp.FolderPath = opts.FolderPath
		}
		if opts.StartedAt != "" {
			sp.StartedAt = opts.StartedAt
		}
		if opts.CompletedAt != "" {
			sp.CompletedAt = opts.CompletedAt
		}
	}
	sp.UpdatedAt = nowISO()
	return nil
}

func (s *InMemoryStore) GetSprint(sprintID string) (*ports.SprintRecord, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	sp, ok := s.sprints[sprintID]
	if !ok {
		return nil, nil
	}
	r := *sp
	return &r, nil
}

func (s *InMemoryStore) ListSprints(status string) ([]ports.SprintRecord, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]ports.SprintRecord, 0, len(s.sprints))
	for _, sp := range s.sprints {
		if status != "" && sp.Status != status {
			continue
		}
		out = append(out, *sp)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].SprintID < out[j].SprintID })
	return out, nil
}

func (s *InMemoryStore) AggregateSprintProgress(sprintID string) (*ports.ProgressResult, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := &ports.ProgressResult{SprintID: sprintID, Burndown: []ports.BurndownEntry{}}
	for _, t := range s.tasks {
		if t.rec.Sprint != sprintID {
			continue
		}
		result.Total++
		switch t.rec.Status {
		case "done":
			result.Done++
		case "in-progress":
			result.InProgress++
		case "todo":
			result.Todo++
		}
	}
	if result.Total > 0 {
		result.Percent = (result.Done * 100) / result.Total
		result.Progress = result.Percent
	}
	return result, nil
}

func (s *InMemoryStore) GetSprintTaskFilePaths(sprintID string) ([]string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var out []string
	for _, t := range s.tasks {
		if t.rec.Sprint == sprintID && t.rec.FilePath != "" {
			out = append(out, t.rec.FilePath)
		}
	}
	sort.Strings(out)
	return out, nil
}

func (s *InMemoryStore) UpdateTaskPathsForSprint(sprintID, fromLocation, toLocation string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, t := range s.tasks {
		if t.rec.Sprint != sprintID {
			continue
		}
		t.rec.FilePath = strings.Replace(t.rec.FilePath, fromLocation, toLocation, 1)
	}
	return nil
}

func (s *InMemoryStore) UpdateSprintMetadata(sprintID, title, folderPath, status string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	sp, ok := s.sprints[sprintID]
	if !ok {
		return fmt.Errorf("sprint not found: %s", sprintID)
	}
	if title != "" {
		sp.Title = title
	}
	if folderPath != "" {
		sp.FolderPath = folderPath
	}
	if status != "" {
		sp.Status = status
	}
	sp.UpdatedAt = nowISO()
	return nil
}

func (s *InMemoryStore) InsertSprintIfMissing(sprint *ports.SprintRecord) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.sprints[sprint.SprintID]; ok {
		return nil
	}
	r := *sprint
	if r.CreatedAt == "" {
		r.CreatedAt = nowISO()
	}
	r.UpdatedAt = nowISO()
	s.sprints[r.SprintID] = &r
	return nil
}

func (s *InMemoryStore) GetAllSprintsSync() ([]ports.SprintSyncRecord, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]ports.SprintSyncRecord, 0, len(s.sprints))
	for _, sp := range s.sprints {
		out = append(out, ports.SprintSyncRecord{SprintID: sp.SprintID, Status: sp.Status})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].SprintID < out[j].SprintID })
	return out, nil
}

func (s *InMemoryStore) GetAllSprintsForSync() ([]ports.SprintFullSyncRecord, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]ports.SprintFullSyncRecord, 0, len(s.sprints))
	for _, sp := range s.sprints {
		out = append(out, ports.SprintFullSyncRecord{
			SprintID:    sp.SprintID,
			Title:       sp.Title,
			Status:      sp.Status,
			StartedAt:   sp.StartedAt,
			CompletedAt: sp.CompletedAt,
			FolderPath:  sp.FolderPath,
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].SprintID < out[j].SprintID })
	return out, nil
}

func (s *InMemoryStore) GetSprintTaskCounts() (map[string]int, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make(map[string]int)
	for _, t := range s.tasks {
		if t.rec.Sprint != "" {
			out[t.rec.Sprint]++
		}
	}
	return out, nil
}

func (s *InMemoryStore) UpsertSprintInsert(sprint *ports.SprintRecord) error {
	return s.SaveSprint(sprint)
}

// --- Task -------------------------------------------------------------------

func (s *InMemoryStore) CreateTask(task *ports.TaskRecord) error {
	if task == nil {
		return fmt.Errorf("task is nil")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.tasks[task.TaskID]; exists {
		return fmt.Errorf("task already exists: %s", task.TaskID)
	}
	now := nowISO()
	entry := &taskEntry{
		rec: ports.TaskDetails{
			TaskID: task.TaskID, Title: task.Title, Type: task.Type,
			Status: task.Status, Priority: task.Priority, Estimate: task.Estimate,
			Sprint: task.Sprint, FilePath: task.FilePath,
			DependsOn: task.DependsOn,
			CreatedAt: task.CreatedAt, UpdatedAt: now,
		},
	}
	if entry.rec.CreatedAt == "" {
		entry.rec.CreatedAt = now
	}
	s.tasks[task.TaskID] = entry
	return nil
}

func (s *InMemoryStore) GetTaskFilePath(taskID string) (string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	t, ok := s.tasks[taskID]
	if !ok {
		return "", fmt.Errorf("task not found: %s", taskID)
	}
	return t.rec.FilePath, nil
}

func (s *InMemoryStore) GetTaskStatus(taskID string) (string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	t, ok := s.tasks[taskID]
	if !ok {
		return "", fmt.Errorf("task not found: %s", taskID)
	}
	return t.rec.Status, nil
}

func (s *InMemoryStore) GetTaskType(taskID string) (string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	t, ok := s.tasks[taskID]
	if !ok {
		return "", fmt.Errorf("task not found: %s", taskID)
	}
	return t.rec.Type, nil
}

func (s *InMemoryStore) GetTaskTypeAndPath(taskID string) (string, string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	t, ok := s.tasks[taskID]
	if !ok {
		return "", "", fmt.Errorf("task not found: %s", taskID)
	}
	return t.rec.Type, t.rec.FilePath, nil
}

func (s *InMemoryStore) GetTaskDetails(taskID string) (*ports.TaskDetails, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	t, ok := s.tasks[taskID]
	if !ok {
		return nil, nil
	}
	rec := t.rec
	return &rec, nil
}

func (s *InMemoryStore) GetTaskFull(taskID string) (*ports.TaskGetResult, error) {
	d, err := s.GetTaskDetails(taskID)
	if err != nil || d == nil {
		return nil, err
	}
	return &ports.TaskGetResult{TaskDetails: *d}, nil
}

func (s *InMemoryStore) UpdateTaskFilePath(taskID, newPath string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	t, ok := s.tasks[taskID]
	if !ok {
		return fmt.Errorf("task not found: %s", taskID)
	}
	t.rec.FilePath = newPath
	t.rec.UpdatedAt = nowISO()
	return nil
}

func (s *InMemoryStore) UpdateTaskStatus(taskID, status string) error {
	return s.UpdateTaskStatusDirect(taskID, status)
}

func (s *InMemoryStore) UpdateTaskSprint(taskID string, sprint *string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	t, ok := s.tasks[taskID]
	if !ok {
		return fmt.Errorf("task not found: %s", taskID)
	}
	if sprint == nil {
		t.rec.Sprint = ""
	} else {
		t.rec.Sprint = *sprint
	}
	t.rec.UpdatedAt = nowISO()
	return nil
}

func (s *InMemoryStore) TransitionTaskStatus(taskID, expectedStatus, newStatus string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	t, ok := s.tasks[taskID]
	if !ok {
		return fmt.Errorf("task not found: %s", taskID)
	}
	if t.rec.Status != expectedStatus {
		return fmt.Errorf("expected=%s current=%s", expectedStatus, t.rec.Status)
	}
	t.rec.Status = newStatus
	t.rec.UpdatedAt = nowISO()
	return nil
}

func (s *InMemoryStore) ResetTaskHarness(taskID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	prefix := "task|" + taskID + "|"
	for k := range s.harnessItems {
		if strings.HasPrefix(k, prefix) {
			delete(s.harnessItems, k)
		}
	}
	return nil
}

func (s *InMemoryStore) ListTasks(sprintFilter, statusFilter *string) (*ports.TaskListResult, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	out := &ports.TaskListResult{Tasks: []ports.TaskDetails{}}
	for _, t := range s.tasks {
		if sprintFilter != nil && t.rec.Sprint != *sprintFilter {
			continue
		}
		if statusFilter != nil && t.rec.Status != *statusFilter {
			continue
		}
		out.Tasks = append(out.Tasks, t.rec)
	}
	sort.Slice(out.Tasks, func(i, j int) bool { return out.Tasks[i].TaskID < out.Tasks[j].TaskID })
	return out, nil
}

// SetTaskTimestampsForTest is a test-only helper. CreateTask freezes
// UpdatedAt to the internal "now", so tests covering since/until filters
// use this to set fixed timestamps. Not for production use.
func (s *InMemoryStore) SetTaskTimestampsForTest(taskID, createdAt, updatedAt string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	t, ok := s.tasks[taskID]
	if !ok {
		return fmt.Errorf("task not found: %s", taskID)
	}
	if createdAt != "" {
		t.rec.CreatedAt = createdAt
	}
	if updatedAt != "" {
		t.rec.UpdatedAt = updatedAt
	}
	return nil
}

// ListTasksFiltered is the in-memory implementation, semantically
// equivalent to the SQLite implementation. Test use only.
func (s *InMemoryStore) ListTasksFiltered(filter ports.TaskListFilter) (*ports.TaskListResult, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	out := &ports.TaskListResult{Tasks: []ports.TaskDetails{}}
	for _, t := range s.tasks {
		if filter.Sprint != nil && t.rec.Sprint != *filter.Sprint {
			continue
		}
		if filter.Status != nil && t.rec.Status != *filter.Status {
			continue
		}
		if filter.Priority != nil && t.rec.Priority != *filter.Priority {
			continue
		}
		if filter.Type != nil && t.rec.Type != *filter.Type {
			continue
		}
		if filter.Since != nil {
			cmp := t.rec.UpdatedAt
			if cmp == "" {
				cmp = t.rec.CreatedAt
			}
			if cmp < *filter.Since {
				continue
			}
		}
		if filter.Until != nil {
			cmp := t.rec.UpdatedAt
			if cmp == "" {
				cmp = t.rec.CreatedAt
			}
			if cmp > *filter.Until {
				continue
			}
		}
		out.Tasks = append(out.Tasks, t.rec)
	}

	// --last forces time-desc + truncation. --limit sorts by ID + truncates.
	if filter.Last > 0 {
		sort.Slice(out.Tasks, func(i, j int) bool {
			a := out.Tasks[i].UpdatedAt
			if a == "" {
				a = out.Tasks[i].CreatedAt
			}
			b := out.Tasks[j].UpdatedAt
			if b == "" {
				b = out.Tasks[j].CreatedAt
			}
			return a > b
		})
		if len(out.Tasks) > filter.Last {
			out.Tasks = out.Tasks[:filter.Last]
		}
	} else {
		sort.Slice(out.Tasks, func(i, j int) bool { return out.Tasks[i].TaskID < out.Tasks[j].TaskID })
		if filter.Limit > 0 && len(out.Tasks) > filter.Limit {
			out.Tasks = out.Tasks[:filter.Limit]
		}
	}
	return out, nil
}

func (s *InMemoryStore) NextTask() (*ports.TaskNextResult, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	// Returns the smallest-ID todo Task — equivalent to the SQLite
	// implementation.
	candidates := make([]*taskEntry, 0)
	for _, t := range s.tasks {
		if t.rec.Status == "todo" {
			candidates = append(candidates, t)
		}
	}
	if len(candidates) == 0 {
		return nil, nil
	}
	sort.Slice(candidates, func(i, j int) bool { return candidates[i].rec.TaskID < candidates[j].rec.TaskID })
	return &ports.TaskNextResult{TaskID: candidates[0].rec.TaskID, Title: candidates[0].rec.Title}, nil
}

func (s *InMemoryStore) HasPlaceholderBody(taskID string) (bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	body, ok := s.taskBodies[taskID]
	if !ok {
		return true, nil // an unregistered body counts as a placeholder
	}
	return strings.Contains(body, "{The problem this Task") || strings.Contains(body, "{Requirement "), nil
}

// SetTaskBody is a test convenience for injecting body content directly.
func (s *InMemoryStore) SetTaskBody(taskID, body string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.taskBodies[taskID] = body
}

func (s *InMemoryStore) FindPlaceholderTasks(sprintFilter string) ([]string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var out []string
	for id, body := range s.taskBodies {
		if strings.Contains(body, "{The problem this Task") {
			if sprintFilter != "" {
				if t, ok := s.tasks[id]; !ok || t.rec.Sprint != sprintFilter {
					continue
				}
			}
			out = append(out, id)
		}
	}
	sort.Strings(out)
	return out, nil
}

func (s *InMemoryStore) GetAllTasksSync() ([]ports.TaskSyncRecord, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]ports.TaskSyncRecord, 0, len(s.tasks))
	for _, t := range s.tasks {
		out = append(out, ports.TaskSyncRecord{
			TaskID: t.rec.TaskID, Status: t.rec.Status, Sprint: t.rec.Sprint, FilePath: t.rec.FilePath,
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].TaskID < out[j].TaskID })
	return out, nil
}

func (s *InMemoryStore) InsertTaskAndSyncCounter(task *ports.TaskRecord, maxID int) error {
	if err := s.CreateTask(task); err != nil {
		return err
	}
	s.mu.Lock()
	if n := int64(maxID); n > s.counters["task"] {
		s.counters["task"] = n
	}
	s.mu.Unlock()
	return nil
}

func (s *InMemoryStore) GetTaskReopenInfo(taskID string) (*ports.TaskReopenInfo, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	t, ok := s.tasks[taskID]
	if !ok {
		return nil, fmt.Errorf("task not found: %s", taskID)
	}
	return &ports.TaskReopenInfo{
		TaskID: t.rec.TaskID, Status: t.rec.Status, FilePath: t.rec.FilePath,
		SprintID: t.rec.Sprint, Title: t.rec.Title, Type: t.rec.Type,
		Estimate: t.rec.Estimate, Priority: t.rec.Priority, DependsOnJSON: t.rec.DependsOn,
	}, nil
}

func (s *InMemoryStore) UpdateTaskStatusDirect(taskID, status string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	t, ok := s.tasks[taskID]
	if !ok {
		return fmt.Errorf("task not found: %s", taskID)
	}
	t.rec.Status = status
	t.rec.UpdatedAt = nowISO()
	return nil
}

// TransitionTaskAtomic — atomicity is enforced by a single in-process lock.
// On fileOp failure the original status is restored; when fileOp returns a
// new path, file_path is updated.
func (s *InMemoryStore) TransitionTaskAtomic(taskID, expectedStatus, newStatus string, fileOp func() (string, error)) error {
	if fileOp == nil {
		return fmt.Errorf("fileOp is nil")
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	t, ok := s.tasks[taskID]
	if !ok {
		return fmt.Errorf("task not found: %s", taskID)
	}
	if t.rec.Status != expectedStatus {
		return fmt.Errorf("expected=%s current=%s", expectedStatus, t.rec.Status)
	}
	origStatus := t.rec.Status
	origPath := t.rec.FilePath
	t.rec.Status = newStatus

	newFilePath, err := fileOp()
	if err != nil {
		// rollback
		t.rec.Status = origStatus
		t.rec.FilePath = origPath
		return fmt.Errorf("fileOp: %w", err)
	}
	if newFilePath != "" {
		t.rec.FilePath = newFilePath
	}
	t.rec.UpdatedAt = nowISO()
	return nil
}

func (s *InMemoryStore) DeleteHarnessItemsForEntity(entityType, entityID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	prefix := entityType + "|" + entityID + "|"
	for k := range s.harnessItems {
		if strings.HasPrefix(k, prefix) {
			delete(s.harnessItems, k)
		}
	}
	return nil
}

func (s *InMemoryStore) UpsertTaskInsert(task *ports.TaskRecord, taskNum int) error {
	s.mu.Lock()
	if _, exists := s.tasks[task.TaskID]; !exists {
		now := nowISO()
		s.tasks[task.TaskID] = &taskEntry{
			rec: ports.TaskDetails{
				TaskID: task.TaskID, Title: task.Title, Type: task.Type,
				Status: task.Status, Priority: task.Priority, Estimate: task.Estimate,
				Sprint: task.Sprint, FilePath: task.FilePath,
				DependsOn: task.DependsOn,
				CreatedAt: task.CreatedAt, UpdatedAt: now,
			},
		}
		if s.tasks[task.TaskID].rec.CreatedAt == "" {
			s.tasks[task.TaskID].rec.CreatedAt = now
		}
	}
	if n := int64(taskNum); n > s.counters["task"] {
		s.counters["task"] = n
	}
	s.mu.Unlock()
	return nil
}

func (s *InMemoryStore) SetCounterMax(counterKey string, val int64) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if val > s.counters[counterKey] {
		s.counters[counterKey] = val
	}
	return nil
}

func (s *InMemoryStore) GetNonArchivedTasks() ([]ports.TaskSyncRecord, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var out []ports.TaskSyncRecord
	for _, t := range s.tasks {
		if t.rec.Status == "archived" || t.rec.Status == "superseded" {
			continue
		}
		out = append(out, ports.TaskSyncRecord{
			TaskID: t.rec.TaskID, Status: t.rec.Status, Sprint: t.rec.Sprint, FilePath: t.rec.FilePath,
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].TaskID < out[j].TaskID })
	return out, nil
}

func (s *InMemoryStore) ArchiveTasks(taskIDs []string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, id := range taskIDs {
		if t, ok := s.tasks[id]; ok {
			t.rec.Status = "archived"
			t.rec.UpdatedAt = nowISO()
		}
	}
	return nil
}

func (s *InMemoryStore) UpdateTaskDirect(taskID string, fields ports.TaskUpdateFields) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	t, ok := s.tasks[taskID]
	if !ok {
		return fmt.Errorf("task not found: %s", taskID)
	}
	if fields.Status != nil {
		t.rec.Status = *fields.Status
	}
	if fields.Sprint != nil {
		t.rec.Sprint = *fields.Sprint
	}
	if fields.FilePath != nil {
		t.rec.FilePath = *fields.FilePath
	}
	if fields.Title != nil {
		t.rec.Title = *fields.Title
	}
	if fields.Type != nil {
		t.rec.Type = *fields.Type
	}
	if fields.Priority != nil {
		t.rec.Priority = *fields.Priority
	}
	if fields.Estimate != nil {
		t.rec.Estimate = *fields.Estimate
	}
	if fields.DependsOn != nil {
		t.rec.DependsOn = *fields.DependsOn
	}
	t.rec.UpdatedAt = nowISO()
	return nil
}

func (s *InMemoryStore) DeleteTask(taskID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.tasks, taskID)
	// cascade: harness items
	prefix := "task|" + taskID + "|"
	for k := range s.harnessItems {
		if strings.HasPrefix(k, prefix) {
			delete(s.harnessItems, k)
		}
	}
	delete(s.workTickets, taskID)
	delete(s.taskBodies, taskID)
	return nil
}

func (s *InMemoryStore) GetDoneTasksWithPaths() ([]ports.TaskSyncRecord, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var out []ports.TaskSyncRecord
	for _, t := range s.tasks {
		if t.rec.Status != "done" {
			continue
		}
		out = append(out, ports.TaskSyncRecord{
			TaskID: t.rec.TaskID, Status: t.rec.Status, Sprint: t.rec.Sprint, FilePath: t.rec.FilePath,
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].TaskID < out[j].TaskID })
	return out, nil
}

// --- Backlog ----------------------------------------------------------------

func (s *InMemoryStore) GetBacklogTasks(includeInProgress bool) ([]ports.TaskInfo, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var out []ports.TaskInfo
	for _, t := range s.tasks {
		if t.rec.Sprint != "" {
			continue
		}
		if t.rec.Status != "todo" && !(includeInProgress && t.rec.Status == "in-progress") {
			continue
		}
		out = append(out, ports.TaskInfo{
			TaskID: t.rec.TaskID, Title: t.rec.Title, Type: t.rec.Type,
			Priority: t.rec.Priority, Estimate: t.rec.Estimate,
			Sprint: t.rec.Sprint, Status: t.rec.Status,
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].TaskID < out[j].TaskID })
	return out, nil
}

// --- Harness ----------------------------------------------------------------

func (s *InMemoryStore) CountHarnessItems(entityType, entityID string) (int, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	prefix := entityType + "|" + entityID + "|"
	n := 0
	for k := range s.harnessItems {
		if strings.HasPrefix(k, prefix) {
			n++
		}
	}
	return n, nil
}

func (s *InMemoryStore) EnsureHarnessItems(entityType, entityID string, items []ports.HarnessItemTemplate) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, tmpl := range items {
		key := harnessKey(entityType, entityID, tmpl.ID)
		if _, exists := s.harnessItems[key]; !exists {
			s.harnessItems[key] = &ports.HarnessItem{
				ID: tmpl.ID, Description: tmpl.Description, Required: tmpl.Required,
			}
		}
	}
	return nil
}

func (s *InMemoryStore) GetHarnessItems(entityType, entityID string) ([]ports.HarnessItem, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	prefix := entityType + "|" + entityID + "|"
	var out []ports.HarnessItem
	for k, item := range s.harnessItems {
		if strings.HasPrefix(k, prefix) {
			out = append(out, *item)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out, nil
}

func (s *InMemoryStore) GetHarnessTemplate(templateKey string) ([]map[string]any, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.harnessTemplates[templateKey], nil
}

func (s *InMemoryStore) GetHarnessItemID(entityType, entityID, itemID string) (int64, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	key := harnessKey(entityType, entityID, itemID)
	if _, ok := s.harnessItems[key]; !ok {
		return 0, fmt.Errorf("harness item not found: %s", key)
	}
	// In memory we only confirm existence — the actual ID is meaningless.
	return 1, nil
}

func (s *InMemoryStore) CheckHarnessItem(entityType, entityID, itemID, evidence, actor string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	key := harnessKey(entityType, entityID, itemID)
	item, ok := s.harnessItems[key]
	if !ok {
		return fmt.Errorf("harness item not found: %s", key)
	}
	item.Done = true
	item.Evidence = evidence
	item.CheckedAt = nowISO()
	item.CheckedBy = actor
	return nil
}

func (s *InMemoryStore) CheckSprintProduction(sprintID string) (bool, error) {
	// In-memory is not used in real production — always false.
	return false, nil
}

// --- Project Meta -----------------------------------------------------------

func (s *InMemoryStore) EnsureProjectSecret() ([]byte, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.projectSecret == nil {
		// 32-byte deterministic secret (test reproducibility).
		s.projectSecret = make([]byte, 32)
		for i := range s.projectSecret {
			s.projectSecret[i] = byte(i)
		}
	}
	return s.projectSecret, nil
}

func (s *InMemoryStore) RotateProjectSecret(newHexSecret string) error {
	b, err := hex.DecodeString(newHexSecret)
	if err != nil {
		return fmt.Errorf("invalid hex: %w", err)
	}
	s.mu.Lock()
	s.projectSecret = b
	s.mu.Unlock()
	return nil
}

// --- Work Ticket / Context Ack ----------------------------------------------

func (s *InMemoryStore) GetTaskWorkTicket(taskID string) (string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.workTickets[taskID], nil
}

func (s *InMemoryStore) SetTaskWorkTicket(taskID, ticket string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.workTickets[taskID] = ticket
	return nil
}

func (s *InMemoryStore) InsertContextAck(ticket, hash string, size int) error {
	_ = size // In memory we do not store size (statistics only).
	s.mu.Lock()
	defer s.mu.Unlock()
	key := ticket + "|" + hash
	// Idempotent: if it already exists, leave the timestamp alone (matches
	// SQLite INSERT OR IGNORE).
	if _, exists := s.contextAcks[key]; !exists {
		s.contextAcks[key] = time.Now().UTC()
	}
	return nil
}

func (s *InMemoryStore) HasContextAck(ticket, hash string) (bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	_, ok := s.contextAcks[ticket+"|"+hash]
	return ok, nil
}

// HasContextAckFresh validates the TTL. ttlMinutes <= 0 behaves the same as
// HasContextAck.
func (s *InMemoryStore) HasContextAckFresh(ticket, hash string, ttlMinutes int) (bool, error) {
	if ttlMinutes <= 0 {
		return s.HasContextAck(ticket, hash)
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	createdAt, ok := s.contextAcks[ticket+"|"+hash]
	if !ok {
		return false, nil
	}
	cutoff := time.Now().UTC().Add(-time.Duration(ttlMinutes) * time.Minute)
	return !createdAt.Before(cutoff), nil
}

// --- IDStore ----------------------------------------------------------------

func (s *InMemoryStore) HealAndGetNextTaskID() (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	// Find the largest Task ID.
	max := s.counters["task"]
	for id := range s.tasks {
		if !strings.HasPrefix(id, "T") {
			continue
		}
		var n int64
		if _, err := fmt.Sscanf(id, "T%d", &n); err == nil && n > max {
			max = n
		}
	}
	max++
	s.counters["task"] = max
	return fmt.Sprintf("T%d", max), nil
}

func (s *InMemoryStore) NextCounter(counterKey string) (int64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.counters[counterKey]++
	return s.counters[counterKey], nil
}

// BumpTaskCounter raises the task counter to at least minValue.
// If the current value is already minValue or above, it is a no-op.
// Used by the SPRINT.md / BACKLOG.md placeholder reflection path.
func (s *InMemoryStore) BumpTaskCounter(minValue int64) error {
	if minValue <= 0 {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.counters["task"] < minValue {
		s.counters["task"] = minValue
	}
	return nil
}

// Interface hygiene — no-op so that json.Marshal on any types does not
// trigger linter warnings.
var _ = json.Marshal
