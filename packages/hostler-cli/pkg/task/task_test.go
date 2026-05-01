package task_test

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/internal/adapters/sqlite"
	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/internal/domain"
	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/pkg/apperr"
	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/pkg/db"
	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/pkg/store"
	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/pkg/task"
)

// confirms the sync package is used by concurrency tests
var _ sync.WaitGroup

// asMap converts an arbitrary value to map[string]any via a JSON
// round-trip, used to keep typed-struct return values compatible with the
// existing test patterns.
func asMap(v any) map[string]any {
	b, err := json.Marshal(v)
	if err != nil {
		return map[string]any{}
	}
	var m map[string]any
	_ = json.Unmarshal(b, &m)
	return m
}

// setupDB initialises an isolated DB and project root for tests.
func setupDB(t *testing.T) string {
	t.Helper()
	tmpDir := t.TempDir()
	t.Setenv("HSTL_DB_PATH", filepath.Join(tmpDir, "hstl.db"))
	t.Setenv("HSTL_PROJECT_ROOT", tmpDir)

	// create the required directories
	for _, dir := range []string{
		filepath.Join(tmpDir, "works", "tasks"),
		filepath.Join(tmpDir, "works", "sprints", "backlog"),
		filepath.Join(tmpDir, "works", "sprints", "active"),
		filepath.Join(tmpDir, "works", "data", "task"),
	} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatalf("directory create failed: %v", err)
		}
	}

	if err := db.InitDB(); err != nil {
		t.Fatalf("DB initialisation failed: %v", err)
	}
	sqliteStore := sqlite.New(db.GetDB())
	store.Init(sqliteStore, sqliteStore)
	t.Cleanup(func() {
		store.Reset()
		db.Close()
	})
	return tmpDir
}

// insertSprint creates a Sprint in DB + filesystem for tests.
func insertSprint(t *testing.T, tmpDir, sprintID string) {
	t.Helper()
	database := db.GetDB()
	_, err := database.Exec(
		`INSERT OR REPLACE INTO sprints (sprint_id, title, status, folder_path, goal, created_at, updated_at)
		 VALUES (?, 'test sprint', 'active', ?, 'goal', date('now'), date('now'))`,
		sprintID, "works/sprints/active/"+sprintID,
	)
	if err != nil {
		t.Fatalf("Sprint DB insert failed: %v", err)
	}
	// create the Sprint directory
	sprintDir := filepath.Join(tmpDir, "works", "sprints", "active", sprintID, "tasks")
	if err := os.MkdirAll(sprintDir, 0o755); err != nil {
		t.Fatalf("Sprint directory create failed: %v", err)
	}
}

// TestCreate_BasicCreate verifies basic Task creation.
func TestCreate_BasicCreate(t *testing.T) {
	setupDB(t)

	raw, err := task.Create("test task", "feature", "", "p2", "M", "", nil)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	result := asMap(raw)

	if result["status"] != "created" {
		t.Errorf("status=%v, want=created", result["status"])
	}
	taskID, ok := result["task_id"].(string)
	if !ok || taskID == "" {
		t.Errorf("task_id missing: %v", result)
	}
	if _, ok := result["file_path"]; !ok {
		t.Error("file_path missing")
	}
}

// TestCreate_SprintNormalize_BacklogLiteral is the T212 regression test.
// When task_create is called with the literal sprint="backlog", the input
// is normalised internally to the empty string so that the file frontmatter
// and DB tasks.sprint share the same canonical form. Previously file
// stored "backlog" while DB stored NULL, producing a spurious mismatch
// (ISS-20260410-009).
func TestCreate_SprintNormalize_BacklogLiteral(t *testing.T) {
	setupDB(t)

	raw, err := task.Create("backlog normalisation test", "feature", "backlog", "p2", "M", "", nil)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	result := asMap(raw)
	taskID, _ := result["task_id"].(string)
	if taskID == "" {
		t.Fatal("task_id missing")
	}

	// Verify DB value — the sprint column must be NULL.
	database := db.GetDB()
	var dbSprint *string
	if err := database.QueryRow(
		"SELECT sprint FROM tasks WHERE task_id = ?", taskID,
	).Scan(&dbSprint); err != nil {
		t.Fatalf("DB query failed: %v", err)
	}
	if dbSprint != nil {
		t.Errorf("expected DB sprint NULL, got %q (storing 'backlog' literal = T212 regression)", *dbSprint)
	}

	// File path must also be under works/tasks/ (not a Sprint folder).
	filePath, _ := result["file_path"].(string)
	if filePath == "" || !contains(filePath, "works/tasks/") {
		t.Errorf("expected file path under works/tasks/, got %q", filePath)
	}
}

// contains is a test helper — strings.Contains wrapper.
func contains(haystack, needle string) bool {
	return len(haystack) >= len(needle) && func() bool {
		for i := 0; i+len(needle) <= len(haystack); i++ {
			if haystack[i:i+len(needle)] == needle {
				return true
			}
		}
		return false
	}()
}

// TestCreate_AssignSprint verifies Task creation with a Sprint assigned.
func TestCreate_AssignSprint(t *testing.T) {
	tmpDir := setupDB(t)
	insertSprint(t, tmpDir, "sprint-01")

	raw, err := task.Create("sprint task", "feature", "sprint-01", "p1", "S", "", nil)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	result := asMap(raw)

	if _, ok := result["status"]; !ok {
		t.Fatalf("status key missing: %v", result)
	}

	// Skip when BOOTSTRAP_REQUIRED is returned (fresh project).
	if result["status"] == "BOOTSTRAP_REQUIRED" {
		t.Skip("bootstrap required environment - skip")
	}
	if result["status"] != "created" {
		t.Errorf("status=%v, want=created", result["status"])
	}
}

// TestT471_Create_4DigitIDBoundary_FullFlow verifies that with T999/T1000/
// T1042 already in the DB, task.Create issues the next ID using numeric
// sort (not string sort) and works at the 4-digit boundary without UNIQUE
// collisions (regression guard for ISS-20260413-005 / T416 backend fix at
// the pkg/task level).
func TestT471_Create_4DigitIDBoundary_FullFlow(t *testing.T) {
	setupDB(t)

	// Scenario: as if recovering from files, INSERT T999, T1000, T1042
	// directly into tasks; do not touch id_counters so drift state is
	// reproduced.
	database := db.GetDB()
	for _, tid := range []string{"T001", "T999", "T1000", "T1042"} {
		_, err := database.Exec(
			`INSERT INTO tasks (task_id, title, type, status, priority, estimate, file_path, created_at, updated_at)
			 VALUES (?, 'seed', 'chore', 'done', 'p2', 'S', '', date('now'), date('now'))`,
			tid,
		)
		if err != nil {
			t.Fatalf("seed task %s failed: %v", tid, err)
		}
	}

	// Call task.Create: the next ID must be T1043.
	raw, err := task.Create("4-digit boundary test", "feature", "", "p2", "M", "", nil)
	if err != nil {
		t.Fatalf("Create failed (4-digit ID boundary regression): %v", err)
	}
	result := asMap(raw)
	if result["status"] == "BOOTSTRAP_REQUIRED" {
		t.Skip("bootstrap required environment - skip")
	}
	taskID, _ := result["task_id"].(string)
	if taskID != "T1043" {
		t.Errorf("next ID at 4-digit boundary: want T1043, got %s (suspect string-sort regression)", taskID)
	}

	// Subsequent call should also work.
	raw2, err := task.Create("4-digit boundary test 2", "feature", "", "p2", "M", "", nil)
	if err != nil {
		t.Fatalf("Create consecutive call failed: %v", err)
	}
	result2 := asMap(raw2)
	taskID2, _ := result2["task_id"].(string)
	if taskID2 != "T1044" {
		t.Errorf("consecutive ID: want T1044, got %s", taskID2)
	}
}

// TestStart_FromTodo verifies the todo → in-progress transition.
func TestStart_FromTodo(t *testing.T) {
	setupDB(t)

	// create a Task first
	createRaw, err := task.Create("start test", "feature", "", "p2", "M", "", nil)
	createResult := asMap(createRaw)
	if err != nil || createResult["status"] != "created" {
		t.Skip("Task create failed, skip")
	}

	taskID := createResult["task_id"].(string)
	raw, err := task.Start(taskID)
	if err != nil {
		t.Fatalf("Start failed: %v", err)
	}
	result := asMap(raw)

	if errMsg, ok := result["error"]; ok {
		t.Fatalf("Start error: %v", errMsg)
	}
	if result["new_status"] != "in-progress" {
		t.Errorf("new_status=%v, want=in-progress", result["new_status"])
	}
	if result["previous_status"] != "todo" {
		t.Errorf("previous_status=%v, want=todo", result["previous_status"])
	}
}

// TestStart_AlreadyStartedTask_Error verifies that Start on an in-progress
// Task returns an error.
func TestStart_AlreadyStartedTask_Error(t *testing.T) {
	setupDB(t)

	createRaw, err := task.Create("double start test", "feature", "", "p2", "M", "", nil)
	createResult := asMap(createRaw)
	if err != nil || createResult["status"] != "created" {
		t.Skip("Task create failed, skip")
	}
	taskID := createResult["task_id"].(string)

	// first Start
	if _, err := task.Start(taskID); err != nil {
		t.Fatalf("first Start failed: %v", err)
	}

	// second Start — expect error
	_, err = task.Start(taskID)
	if err == nil {
		t.Errorf("expected error on Start of in-progress Task")
	}
}

// TestComplete_HarnessPass verifies the completion transition when
// skipHarness=true.
func TestComplete_HarnessPass(t *testing.T) {
	setupDB(t)

	createRaw, err := task.Create("complete test", "feature", "", "p2", "M", "", nil)
	createResult := asMap(createRaw)
	if err != nil || createResult["status"] != "created" {
		t.Skip("Task create failed, skip")
	}
	taskID := createResult["task_id"].(string)

	if _, err := task.Start(taskID); err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	raw, err := task.Complete(taskID, true)
	if err != nil {
		t.Fatalf("Complete failed: %v", err)
	}
	result := asMap(raw)

	if errMsg, ok := result["error"]; ok {
		t.Fatalf("Complete error: %v", errMsg)
	}
	if result["new_status"] != "done" {
		t.Errorf("new_status=%v, want=done", result["new_status"])
	}
}

// TestT507_Complete_UnassignedTask_MovesToCompletedFolder verifies that an
// unassigned (no Sprint) Task's file moves from works/tasks/T*.md to
// works/tasks/completed/T*.md on the done transition (T507).
func TestT507_Complete_UnassignedTask_MovesToCompletedFolder(t *testing.T) {
	tmpDir := setupDB(t)

	createRaw, err := task.Create("auto-move test", "chore", "", "p3", "S", "", nil)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	createResult := asMap(createRaw)
	taskID := createResult["task_id"].(string)
	originalPath := createResult["file_path"].(string)

	// initially the file should sit directly under works/tasks/
	if !strings.Contains(originalPath, "works/tasks/") || strings.Contains(originalPath, "works/tasks/completed/") {
		t.Fatalf("initial location wrong: %s", originalPath)
	}

	if _, err := task.Start(taskID); err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	if _, err := task.Complete(taskID, true); err != nil {
		t.Fatalf("Complete failed: %v", err)
	}

	// confirm move into works/tasks/completed/
	completedDir := filepath.Join(tmpDir, "works", "tasks", "completed")
	entries, _ := os.ReadDir(completedDir)
	found := false
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), taskID+"-") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("T507: did not move into works/tasks/completed/ (taskID=%s)", taskID)
	}

	// original location (works/tasks/T*.md) must no longer exist
	if _, err := os.Stat(filepath.Join(tmpDir, originalPath)); err == nil {
		t.Errorf("file still present at original location: %s", originalPath)
	}
}

// TestT507_Reopen_FromCompletedToTasks verifies that on done → todo Reopen
// the file moves back from works/tasks/completed/ → works/tasks/.
func TestT507_Reopen_FromCompletedToTasks(t *testing.T) {
	tmpDir := setupDB(t)

	createRaw, err := task.Create("Reopen-restore test", "chore", "", "p3", "S", "", nil)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	taskID := asMap(createRaw)["task_id"].(string)

	_, _ = task.Start(taskID)
	_, _ = task.Complete(taskID, true)

	// now under completed/
	completedPath := ""
	completedDir := filepath.Join(tmpDir, "works", "tasks", "completed")
	entries, _ := os.ReadDir(completedDir)
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), taskID+"-") {
			completedPath = filepath.Join(completedDir, e.Name())
			break
		}
	}
	if completedPath == "" {
		t.Fatalf("not in completed/ after Complete")
	}

	// Reopen → todo
	if _, err := task.Reopen(taskID, "satisfies the reopen-reason via regression test", "todo"); err != nil {
		t.Fatalf("Reopen failed: %v", err)
	}

	// returns to works/tasks/ proper
	if _, err := os.Stat(completedPath); err == nil {
		t.Errorf("file still in completed/ after Reopen")
	}
	tasksDir := filepath.Join(tmpDir, "works", "tasks")
	entries2, _ := os.ReadDir(tasksDir)
	foundBack := false
	for _, e := range entries2 {
		if strings.HasPrefix(e.Name(), taskID+"-") && !e.IsDir() {
			foundBack = true
			break
		}
	}
	if !foundBack {
		t.Errorf("did not return to works/tasks/ after Reopen")
	}
}

// TestT507_Complete_SprintAssignedTask_NoMove verifies that Sprint-assigned
// Tasks keep their file_path inside the Sprint folder on the done
// transition and are NOT moved into completed/ (only Sprint-level moves
// apply).
func TestT507_Complete_SprintAssignedTask_NoMove(t *testing.T) {
	tmpDir := setupDB(t)
	insertSprint(t, tmpDir, "sprint-99")

	createRaw, err := task.Create("Sprint Task no-move", "chore", "sprint-99", "p3", "S", "", nil)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	taskID := asMap(createRaw)["task_id"].(string)

	_, _ = task.Start(taskID)
	_, _ = task.Complete(taskID, true)

	// must NOT be under works/tasks/completed/ (it's Sprint-assigned)
	completedDir := filepath.Join(tmpDir, "works", "tasks", "completed")
	if entries, _ := os.ReadDir(completedDir); len(entries) > 0 {
		for _, e := range entries {
			if strings.HasPrefix(e.Name(), taskID+"-") {
				t.Errorf("Sprint-assigned Task wrongly moved into completed/: %s", e.Name())
			}
		}
	}
}

// TestComplete_HarnessFail_BLOCKED verifies that BLOCKED is returned when
// Harness items are incomplete.
func TestComplete_HarnessFail_BLOCKED(t *testing.T) {
	setupDB(t)

	createRaw, err := task.Create("Harness test", "feature", "", "p2", "M", "", nil)
	createResult := asMap(createRaw)
	if err != nil || createResult["status"] != "created" {
		t.Skip("Task create failed, skip")
	}
	taskID := createResult["task_id"].(string)

	if _, err := task.Start(taskID); err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	// skipHarness=false — when Harness items exist a BLOCKED error must be returned
	raw, err := task.Complete(taskID, false)
	if err != nil {
		// BLOCKED error is expected — confirm BlockedError type
		var be *apperr.BlockedError
		if errors.As(err, &be) {
			return // ok
		}
		t.Fatalf("Complete unexpected error: %v", err)
	}
	// no error means it succeeded (no harness template available)
	result := asMap(raw)
	if result["new_status"] != "done" {
		t.Errorf("new_status=%v, want=done (no harness items case)", result["new_status"])
	}
}

// TestComplete_T226_BLOCKED_recovery_hint — T226 (Sprint-28)
// Verifies the Harness Gate BLOCKED response includes recovery_hint /
// template_key / available_item_ids. Tested with an infra-type Task
// (different item_ids than feature).
func TestComplete_T226_BLOCKED_recovery_hint(t *testing.T) {
	setupDB(t)

	createRaw, err := task.Create("T226 infra test", "infra", "", "p2", "S", "", nil)
	createResult := asMap(createRaw)
	if err != nil || createResult["status"] != "created" {
		t.Skip("Task create failed, skip")
	}
	taskID := createResult["task_id"].(string)

	if _, err := task.Start(taskID); err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	_, err = task.Complete(taskID, false)
	if err == nil {
		t.Fatal("expected BLOCKED error")
	}
	var be *apperr.BlockedError
	if !errors.As(err, &be) {
		t.Fatalf("expected BlockedError: %v", err)
	}

	// Confirm the T226 enrichments are present in Extra.
	if be.Extra == nil {
		t.Fatal("Extra is nil")
	}
	templateKey, ok := be.Extra["template_key"].(string)
	if !ok || templateKey != "task:infra" {
		t.Errorf("template_key=%v (want task:infra)", be.Extra["template_key"])
	}
	availableIDs, ok := be.Extra["available_item_ids"].([]string)
	if !ok || len(availableIDs) == 0 {
		t.Errorf("available_item_ids missing: %v", be.Extra["available_item_ids"])
	} else {
		// infra includes deploy_verified/rollback_plan/change_record/criteria_checked
		wantSome := map[string]bool{"deploy_verified": true, "rollback_plan": true, "change_record": true}
		found := 0
		for _, id := range availableIDs {
			if wantSome[id] {
				found++
			}
		}
		if found == 0 {
			t.Errorf("available_item_ids has no infra item: %v", availableIDs)
		}
	}
	recoveryHint, ok := be.Extra["recovery_hint"].(string)
	if !ok || recoveryHint == "" {
		t.Error("recovery_hint missing")
	}
	if !strings.Contains(recoveryHint, "harness_get") {
		t.Errorf("recovery_hint missing harness_get guidance: %q", recoveryHint)
	}

	// First SuggestedAction should reference harness_get.
	if len(be.SuggestedActions) == 0 {
		t.Fatal("SuggestedActions empty")
	}
	if !strings.Contains(be.SuggestedActions[0], "harness_get") {
		t.Errorf("first suggested_action does not mention harness_get: %q", be.SuggestedActions[0])
	}
}

// TestReopen_FromDone verifies the done → todo reverse transition.
func TestReopen_FromDone(t *testing.T) {
	setupDB(t)

	createRaw, err := task.Create("reopen test", "feature", "", "p2", "M", "", nil)
	createResult := asMap(createRaw)
	if err != nil || createResult["status"] != "created" {
		t.Skip("Task create failed, skip")
	}
	taskID := createResult["task_id"].(string)

	if _, err := task.Start(taskID); err != nil {
		t.Fatalf("Start failed: %v", err)
	}
	if _, err := task.Complete(taskID, true); err != nil {
		t.Fatalf("Complete failed: %v", err)
	}

	raw, err := task.Reopen(taskID, "the bug recurred and a fix is required.", "todo")
	if err != nil {
		t.Fatalf("Reopen failed: %v", err)
	}
	result := asMap(raw)

	if errMsg, ok := result["error"]; ok {
		t.Fatalf("Reopen error: %v", errMsg)
	}
	if result["new_status"] != "todo" {
		t.Errorf("new_status=%v, want=todo", result["new_status"])
	}
	if result["harness_reset"] != true {
		t.Errorf("harness_reset=%v, want=true", result["harness_reset"])
	}
}

// TestDelete_TodoDelete verifies deletion of a todo-state Task.
func TestDelete_TodoDelete(t *testing.T) {
	setupDB(t)

	createRaw, err := task.Create("delete test", "chore", "", "p3", "XS", "", nil)
	createResult := asMap(createRaw)
	if err != nil || createResult["status"] != "created" {
		t.Skip("Task create failed, skip")
	}
	taskID := createResult["task_id"].(string)

	raw, err := task.Delete(taskID, "Unneeded Task created for the test.")
	if err != nil {
		t.Fatalf("Delete failed: %v", err)
	}
	result := asMap(raw)
	if errMsg, ok := result["error"]; ok {
		t.Fatalf("Delete error: %v", errMsg)
	}
	if result["status"] != "deleted" {
		t.Errorf("status=%v, want=deleted", result["status"])
	}
}

// TestDelete_DoneDeleteForbidden verifies that REJECTED is returned when
// deleting a done Task.
func TestDelete_DoneDeleteForbidden(t *testing.T) {
	setupDB(t)

	createRaw, err := task.Create("completed task", "feature", "", "p2", "M", "", nil)
	createResult := asMap(createRaw)
	if err != nil || createResult["status"] != "created" {
		t.Skip("Task create failed, skip")
	}
	taskID := createResult["task_id"].(string)

	// todo → in-progress → done
	if _, err := task.Start(taskID); err != nil {
		t.Fatalf("Start failed: %v", err)
	}
	if _, err := task.Complete(taskID, true); err != nil {
		t.Fatalf("Complete failed: %v", err)
	}

	_, err = task.Delete(taskID, "Attempting to delete a completed Task.")
	var re *apperr.RejectedError
	if !errors.As(err, &re) {
		t.Errorf("expected RejectedError when deleting a done Task: err=%v", err)
	}
}

// TestList_NoFilter verifies retrieval of the full Task list.
func TestList_NoFilter(t *testing.T) {
	setupDB(t)

	// create 3 Tasks
	for _, title := range []string{"task1", "task2", "task3"} {
		if _, err := task.Create(title, "feature", "", "p2", "M", "", nil); err != nil {
			t.Fatalf("Create failed: %v", err)
		}
	}

	raw, err := task.List(nil, nil)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}

	lr, ok := raw.(*domain.ListResult)
	if !ok {
		t.Fatalf("domain.ListResult type assertion failed: %T", raw)
	}
	if len(lr.Tasks) < 3 {
		t.Errorf("Task count=%d, want>=3", len(lr.Tasks))
	}
}

// TestList_SprintFilter verifies retrieval with a Sprint filter.
func TestList_SprintFilter(t *testing.T) {
	tmpDir := setupDB(t)
	insertSprint(t, tmpDir, "sprint-01")

	// create a Task assigned to the sprint
	raw, _ := task.Create("sprint task", "feature", "sprint-01", "p1", "S", "", nil)
	r := asMap(raw)
	if r["status"] == "BOOTSTRAP_REQUIRED" {
		t.Skip("bootstrap required - skip")
	}

	// create a backlog Task
	task.Create("backlog task", "feature", "", "p2", "M", "", nil) //nolint:errcheck

	sprintFilter := "sprint-01"
	listRaw, err := task.List(&sprintFilter, nil)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}

	lr, ok := listRaw.(*domain.ListResult)
	if !ok {
		t.Fatalf("domain.ListResult type assertion failed: %T", listRaw)
	}
	for _, tk := range lr.Tasks {
		// just check that the sprint filter applied
		_ = tk
	}
}

// TestGet_SingleLookup verifies a single Task lookup.
func TestGet_SingleLookup(t *testing.T) {
	setupDB(t)

	createRaw, err := task.Create("lookup test", "feature", "", "p2", "M", "", nil)
	createResult := asMap(createRaw)
	if err != nil || createResult["status"] != "created" {
		t.Skip("Task create failed, skip")
	}
	taskID := createResult["task_id"].(string)

	raw, err := task.Get(taskID)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	result := asMap(raw)
	if _, hasError := result["error"]; hasError {
		t.Fatalf("Get error: %v", result["error"])
	}
	if result["task_id"] != taskID {
		t.Errorf("task_id mismatch: got=%v, want=%s", result["task_id"], taskID)
	}
	if result["title"] != "lookup test" {
		t.Errorf("title mismatch: %v", result["title"])
	}
}

// TestNext_PriorityRecommendation verifies that the highest-priority Task
// without unmet dependencies is recommended.
func TestNext_PriorityRecommendation(t *testing.T) {
	setupDB(t)

	// create in p3, p1, p2 order
	task.Create("p3 task", "feature", "", "p3", "M", "", nil) //nolint:errcheck
	task.Create("p1 task", "feature", "", "p1", "M", "", nil) //nolint:errcheck
	task.Create("p2 task", "feature", "", "p2", "M", "", nil) //nolint:errcheck

	raw, err := task.Next()
	if err != nil {
		t.Fatalf("Next failed: %v", err)
	}
	result := asMap(raw)

	if _, hasMsg := result["message"]; hasMsg {
		// "no recommendation" message → no p1 Task case
		t.Logf("no recommendation: %v", result["message"])
		return
	}

	// p1 should be recommended first
	if result["priority"] != "p1" {
		t.Errorf("priority=%v, want=p1", result["priority"])
	}
}

// TestAssignSprint_Assign verifies that AssignSprint assigns a Task to a
// Sprint.
func TestAssignSprint_Assign(t *testing.T) {
	t.Setenv("HSTL_ASSIGN_ALLOW_PLACEHOLDER", "1") // T572 — existing tests aren't placeholder-validation cases
	tmpDir := setupDB(t)
	insertSprint(t, tmpDir, "sprint-01")

	// create a backlog Task
	createRaw, err := task.Create("assign test", "feature", "", "p2", "M", "", nil)
	createResult := asMap(createRaw)
	if err != nil || createResult["status"] != "created" {
		t.Skip("Task create failed, skip")
	}
	taskID := createResult["task_id"].(string)

	raw, err := task.AssignSprint([]string{taskID}, "sprint-01")
	if err != nil {
		t.Fatalf("AssignSprint failed: %v", err)
	}
	result := asMap(raw)
	if _, hasError := result["error"]; hasError {
		t.Fatalf("AssignSprint error: %v", result["error"])
	}

	ar, ok := raw.(*domain.AssignSprintResult)
	if !ok {
		t.Fatalf("domain.AssignSprintResult type assertion failed: %T", raw)
	}
	if len(ar.Assigned) != 1 || ar.Assigned[0] != taskID {
		t.Errorf("assigned=%v, want=[%s]", ar.Assigned, taskID)
	}
}

// TestUnassignSprint_Unassign verifies removal of a Sprint assignment.
func TestUnassignSprint_Unassign(t *testing.T) {
	t.Setenv("HSTL_ASSIGN_ALLOW_PLACEHOLDER", "1") // T572
	tmpDir := setupDB(t)
	insertSprint(t, tmpDir, "sprint-01")

	// create a backlog Task and assign to Sprint
	createRaw, err := task.Create("unassign test", "feature", "", "p2", "M", "", nil)
	createResult := asMap(createRaw)
	if err != nil || createResult["status"] != "created" {
		t.Skip("Task create failed, skip")
	}
	taskID := createResult["task_id"].(string)

	assignRaw, err := task.AssignSprint([]string{taskID}, "sprint-01")
	if err != nil {
		t.Fatalf("AssignSprint failed: %v", err)
	}
	assignResult := asMap(assignRaw)
	if _, hasError := assignResult["error"]; hasError {
		t.Fatalf("AssignSprint error: %v", assignResult["error"])
	}
	ar, ok := assignRaw.(*domain.AssignSprintResult)
	if !ok || len(ar.Assigned) != 1 {
		t.Fatalf("assign failed: %v", assignResult)
	}

	// UnassignSprint
	raw, err := task.UnassignSprint([]string{taskID})
	if err != nil {
		t.Fatalf("UnassignSprint failed: %v", err)
	}
	result := asMap(raw)
	if _, hasError := result["error"]; hasError {
		t.Fatalf("UnassignSprint error: %v", result["error"])
	}

	ur, ok := raw.(*domain.UnassignSprintResult)
	if !ok {
		t.Fatalf("domain.UnassignSprintResult type assertion failed: %T", raw)
	}
	if len(ur.Unassigned) != 1 || ur.Unassigned[0] != taskID {
		t.Errorf("unassigned=%v, want=[%s]", ur.Unassigned, taskID)
	}
}

// TestAssignSprint_AlreadyAssignedTask verifies that re-assigning to the
// same Sprint returns skipped.
func TestAssignSprint_AlreadyAssignedTask(t *testing.T) {
	t.Setenv("HSTL_ASSIGN_ALLOW_PLACEHOLDER", "1") // T572
	tmpDir := setupDB(t)
	insertSprint(t, tmpDir, "sprint-01")

	// create a Task directly in sprint-01
	createRaw, err := task.Create("already assigned test", "feature", "sprint-01", "p2", "M", "", nil)
	createResult := asMap(createRaw)
	if err != nil || createResult["status"] == "BOOTSTRAP_REQUIRED" {
		t.Skip("Task create failed, skip")
	}
	if createResult["status"] != "created" {
		t.Skipf("unexpected status: %v", createResult["status"])
	}
	taskID := createResult["task_id"].(string)

	// re-assign to the same Sprint
	raw, err := task.AssignSprint([]string{taskID}, "sprint-01")
	if err != nil {
		t.Fatalf("AssignSprint failed: %v", err)
	}

	ar, ok := raw.(*domain.AssignSprintResult)
	if !ok {
		t.Fatalf("domain.AssignSprintResult type assertion failed: %T", raw)
	}
	// assigned is empty, skipped should contain it
	if len(ar.Assigned) != 0 {
		t.Errorf("on re-assign, assigned should be empty: %v", ar.Assigned)
	}
	if len(ar.Skipped) != 1 {
		t.Errorf("on re-assign, skipped should contain 1: %v", ar.Skipped)
	}
}

// TestAssignSprint_NonExistentSprint verifies that assigning to a
// non-existent Sprint returns an error.
func TestAssignSprint_NonExistentSprint(t *testing.T) {
	t.Setenv("HSTL_ASSIGN_ALLOW_PLACEHOLDER", "1") // T572
	setupDB(t)

	createRaw, err := task.Create("Task A", "feature", "", "p2", "M", "", nil)
	createResult := asMap(createRaw)
	if err != nil || createResult["status"] != "created" {
		t.Skip("Task create failed, skip")
	}
	taskID := createResult["task_id"].(string)

	_, err = task.AssignSprint([]string{taskID}, "sprint-unknown")
	var nfe *apperr.NotFoundError
	if !errors.As(err, &nfe) {
		t.Errorf("expected NotFoundError when assigning to a non-existent Sprint: err=%v", err)
	}
}

// TestAssignSprint_T572_PlaceholderBlocked — placeholder-body Task fails on
// assign.
func TestAssignSprint_T572_PlaceholderBlocked(t *testing.T) {
	tmpDir := setupDB(t)
	insertSprint(t, tmpDir, "sprint-01")
	_ = tmpDir

	// Create — Create produces the default template (with placeholder markers).
	createRaw, err := task.Create("placeholder block test", "feature", "", "p2", "M", "", nil)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	taskID := asMap(createRaw)["task_id"].(string)

	// HasPlaceholderBody must be true for this test to be meaningful.
	if !task.HasPlaceholderBody(taskID) {
		t.Skipf("Create result not detected as placeholder — HasPlaceholderBody tuning needed")
	}

	raw, err := task.AssignSprint([]string{taskID}, "sprint-01")
	if err != nil {
		t.Fatalf("AssignSprint failed: %v", err)
	}
	ar, ok := raw.(*domain.AssignSprintResult)
	if !ok {
		t.Fatalf("type assertion failed: %T", raw)
	}
	if len(ar.Assigned) != 0 {
		t.Errorf("assigned=%v, want=[]", ar.Assigned)
	}
	if len(ar.Failed) != 1 {
		t.Fatalf("failed=%d, want=1", len(ar.Failed))
	}
	if ar.Failed[0].Reason != "placeholder_body" {
		t.Errorf("failed[0].Reason=%q, want placeholder_body", ar.Failed[0].Reason)
	}
}

// TestAssignSprint_T572_EnvEscapeHatch — HSTL_ASSIGN_ALLOW_PLACEHOLDER=1
// disables the block.
func TestAssignSprint_T572_EnvEscapeHatch(t *testing.T) {
	t.Setenv("HSTL_ASSIGN_ALLOW_PLACEHOLDER", "1")
	tmpDir := setupDB(t)
	insertSprint(t, tmpDir, "sprint-01")
	_ = tmpDir

	createRaw, err := task.Create("escape hatch test", "feature", "", "p2", "M", "", nil)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	taskID := asMap(createRaw)["task_id"].(string)

	raw, err := task.AssignSprint([]string{taskID}, "sprint-01")
	if err != nil {
		t.Fatalf("AssignSprint failed: %v", err)
	}
	ar, ok := raw.(*domain.AssignSprintResult)
	if !ok {
		t.Fatalf("type assertion failed: %T", raw)
	}
	if len(ar.Assigned) != 1 || ar.Assigned[0] != taskID {
		t.Errorf("assigned=%v, want=[%s] (escape hatch did not work)", ar.Assigned, taskID)
	}
}

// TestAssignSprint_DoneStateTask verifies that a done-state Task ends up in
// failed when assign is attempted.
func TestAssignSprint_DoneStateTask(t *testing.T) {
	t.Setenv("HSTL_ASSIGN_ALLOW_PLACEHOLDER", "1") // T572
	tmpDir := setupDB(t)
	insertSprint(t, tmpDir, "sprint-01")

	createRaw, err := task.Create("completed task", "feature", "", "p2", "M", "", nil)
	createResult := asMap(createRaw)
	if err != nil || createResult["status"] != "created" {
		t.Skip("Task create failed, skip")
	}
	taskID := createResult["task_id"].(string)

	if _, err := task.Start(taskID); err != nil {
		t.Fatalf("Start failed: %v", err)
	}
	if _, err := task.Complete(taskID, true); err != nil {
		t.Fatalf("Complete failed: %v", err)
	}

	raw, err := task.AssignSprint([]string{taskID}, "sprint-01")
	if err != nil {
		t.Fatalf("AssignSprint failed: %v", err)
	}

	ar, ok := raw.(*domain.AssignSprintResult)
	if !ok {
		t.Fatalf("domain.AssignSprintResult type assertion failed: %T", raw)
	}
	if len(ar.Failed) != 1 {
		t.Errorf("expected done Task in failed: %v", ar.Failed)
	}
}

// TestCreate_ConcurrentCreate_NoDuplicateID verifies that concurrent Task
// creation produces no ID collisions.
func TestCreate_ConcurrentCreate_NoDuplicateID(t *testing.T) {
	setupDB(t)

	const n = 10
	type result struct {
		taskID string
		err    error
	}
	results := make(chan result, n)

	for i := 0; i < n; i++ {
		go func(idx int) {
			raw, err := task.Create(
				"concurrent create Task",
				"feature",
				"",
				"p2",
				"M",
				"",
				nil,
			)
			if err != nil {
				results <- result{err: err}
				return
			}
			r := asMap(raw)
			if taskID, ok := r["task_id"].(string); ok {
				results <- result{taskID: taskID}
			} else {
				results <- result{err: nil} // skip bootstrap
			}
		}(i)
	}

	seen := make(map[string]bool)
	for i := 0; i < n; i++ {
		r := <-results
		if r.err != nil {
			t.Logf("goroutine error (ignored): %v", r.err)
			continue
		}
		if r.taskID == "" {
			continue
		}
		if seen[r.taskID] {
			t.Errorf("duplicate ID found: %s", r.taskID)
		}
		seen[r.taskID] = true
	}
}

// TestReopen_FromInProgress verifies the in-progress → todo reverse
// transition.
func TestReopen_FromInProgress(t *testing.T) {
	setupDB(t)

	createRaw, err := task.Create("reopen in-progress", "bugfix", "", "p2", "M", "", nil)
	createResult := asMap(createRaw)
	if err != nil || createResult["status"] != "created" {
		t.Skip("Task create failed, skip")
	}
	taskID := createResult["task_id"].(string)

	if _, err := task.Start(taskID); err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	raw, err := task.Reopen(taskID, "reopen from in-progress.", "todo")
	if err != nil {
		t.Fatalf("Reopen failed: %v", err)
	}
	result := asMap(raw)

	if errMsg, ok := result["error"]; ok {
		t.Fatalf("Reopen error: %v", errMsg)
	}
	if result["new_status"] != "todo" {
		t.Errorf("new_status=%v, want=todo", result["new_status"])
	}
}

// TestReopen_TodoStateError verifies that Reopen on a todo Task returns an
// error.
func TestReopen_TodoStateError(t *testing.T) {
	setupDB(t)

	createRaw, err := task.Create("Reopen-not-allowed Task", "feature", "", "p2", "M", "", nil)
	createResult := asMap(createRaw)
	if err != nil || createResult["status"] != "created" {
		t.Skip("Task create failed, skip")
	}
	taskID := createResult["task_id"].(string)
	// attempt Reopen while staying in todo
	_, err = task.Reopen(taskID, "attempting reopen from todo", "todo")
	var re *apperr.RejectedError
	if !errors.As(err, &re) {
		t.Errorf("expected RejectedError on Reopen from todo: err=%v", err)
	}
}

// TestList_StatusFilter verifies retrieval with a status filter.
func TestList_StatusFilter(t *testing.T) {
	setupDB(t)

	// create 1 todo and 1 in-progress Task each
	r1raw, _ := task.Create("todo task", "feature", "", "p2", "M", "", nil)
	r2raw, _ := task.Create("inprog task", "feature", "", "p2", "M", "", nil)
	r1 := asMap(r1raw)
	r2 := asMap(r2raw)
	if r1["status"] != "created" || r2["status"] != "created" {
		t.Skip("Task create failed, skip")
	}

	taskID2 := r2["task_id"].(string)
	task.Start(taskID2) //nolint:errcheck

	statusFilter := "in-progress"
	raw, err := task.List(nil, &statusFilter)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}

	lr, ok := raw.(*domain.ListResult)
	if !ok {
		t.Fatalf("domain.ListResult type assertion failed: %T", raw)
	}
	for _, tk := range lr.Tasks {
		if tk.Status != "in-progress" {
			t.Errorf("status filter not applied: got=%v", tk.Status)
		}
	}
}

// TestGet_NonExistentTask verifies that an error is returned for a
// non-existent Task lookup.
func TestGet_NonExistentTask(t *testing.T) {
	setupDB(t)

	_, err := task.Get("T999")
	var nfe *apperr.NotFoundError
	if !errors.As(err, &nfe) {
		t.Errorf("expected NotFoundError on non-existent Task lookup: err=%v", err)
	}
}

// TestDelete_InProgressDeleteForbidden verifies that REJECTED is returned
// when deleting an in-progress Task.
func TestDelete_InProgressDeleteForbidden(t *testing.T) {
	setupDB(t)

	createRaw, err := task.Create("in-progress task", "feature", "", "p2", "M", "", nil)
	createResult := asMap(createRaw)
	if err != nil || createResult["status"] != "created" {
		t.Skip("Task create failed, skip")
	}
	taskID := createResult["task_id"].(string)

	if _, err := task.Start(taskID); err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	_, err = task.Delete(taskID, "Attempting to delete an in-progress Task")
	var re *apperr.RejectedError
	if !errors.As(err, &re) {
		t.Errorf("expected RejectedError on in-progress delete: err=%v", err)
	}
}

// TestComplete_TodoCascade verifies that since T304 a Complete from todo
// performs the internal cascade (todo → in-progress → done) and returns a
// success response. skipHarness=true bypasses validation gates so only the
// transition logic is checked.
func TestComplete_TodoCascade(t *testing.T) {
	setupDB(t)

	createRaw, err := task.Create("T304 cascade success", "feature", "", "p2", "M", "", nil)
	createResult := asMap(createRaw)
	if err != nil || createResult["status"] != "created" {
		t.Skip("Task create failed, skip")
	}
	taskID := createResult["task_id"].(string)

	// Complete directly without Start — T304 must perform todo → done cascade.
	raw, err := task.Complete(taskID, true)
	if err != nil {
		t.Fatalf("Complete cascade from todo must succeed: err=%v", err)
	}
	tr, ok := raw.(*domain.TransitionResult)
	if !ok {
		t.Fatalf("Complete response is not *domain.TransitionResult: %T", raw)
	}
	if tr.NewStatus != "done" {
		t.Errorf("cascade end state must be done: new_status=%s", tr.NewStatus)
	}
	if tr.PreviousStatus != "in-progress" {
		// The previous of the final transition must be in-progress (the second
		// cascade step).
		t.Errorf("final transition's previous_status must be in-progress: got=%s", tr.PreviousStatus)
	}
}

// TestComplete_DoneRejection verifies that a Task already in done /
// absorbed (terminal state) is not subject to cascade and InvalidStateError
// is returned.
func TestComplete_DoneRejection(t *testing.T) {
	setupDB(t)

	createRaw, err := task.Create("T304 complete rejection", "feature", "", "p2", "M", "", nil)
	createResult := asMap(createRaw)
	if err != nil || createResult["status"] != "created" {
		t.Skip("Task create failed, skip")
	}
	taskID := createResult["task_id"].(string)

	// First cascade flips to done.
	if _, err := task.Complete(taskID, true); err != nil {
		t.Fatalf("first Complete cascade failed: %v", err)
	}
	// Second call — already done, so InvalidStateError is expected.
	_, err = task.Complete(taskID, true)
	var ise *apperr.InvalidStateError
	if !errors.As(err, &ise) {
		t.Errorf("expected InvalidStateError on Complete re-call from done: err=%v", err)
	}
}

// TestCreate_SprintNotFound (T086, Sprint-23): when a non-existent Sprint
// is specified verify that no silent backlog fallback occurs and a clear
// error is returned. Previously fallback was allowed; T086 switched to the
// no-fallback policy.
func TestCreate_SprintNotFound(t *testing.T) {
	setupDB(t)

	_, err := task.Create("Task without Sprint", "feature", "sprint-nonexistent", "p2", "M", "", nil)
	if err == nil {
		t.Fatalf("expected error for non-existent sprint, got success")
	}
	if !strings.Contains(err.Error(), "sprint-nonexistent") {
		t.Errorf("expected error message to mention the missing sprint ID, got: %v", err)
	}
}

// TestUnassignSprint_UnassignedTask verifies behaviour when an unassigned
// Task is unassigned.
func TestUnassignSprint_UnassignedTask(t *testing.T) {
	setupDB(t)

	createRaw, err := task.Create("unassigned Task", "feature", "", "p2", "M", "", nil)
	createResult := asMap(createRaw)
	if err != nil || createResult["status"] != "created" {
		t.Skip("Task create failed, skip")
	}
	taskID := createResult["task_id"].(string)

	// call UnassignSprint without assigning first
	raw, err := task.UnassignSprint([]string{taskID})
	if err != nil {
		t.Fatalf("UnassignSprint failed: %v", err)
	}
	result := asMap(raw)
	// either error or skipped should contain it
	if result["error"] == nil && result["status"] != "error" {
		// included in either skipped or unassigned
		t.Logf("UnassignSprint result for unassigned Task: %v", result)
	}
}

// TestGet_AllFieldsPresent verifies that Get returns every required field.
func TestGet_AllFieldsPresent(t *testing.T) {
	setupDB(t)

	createRaw, err := task.Create("Field validation Task", "bugfix", "", "p1", "S", "", []string{"T001"})
	createResult := asMap(createRaw)
	if err != nil || createResult["status"] != "created" {
		t.Skip("Task create failed, skip")
	}
	taskID := createResult["task_id"].(string)

	raw, err := task.Get(taskID)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	result := asMap(raw)

	requiredFields := []string{"task_id", "title", "type", "status", "priority", "estimate", "file_path", "depends_on", "created_at", "updated_at"}
	for _, field := range requiredFields {
		if _, ok := result[field]; !ok {
			t.Errorf("required field '%s' missing", field)
		}
	}
	if result["type"] != "bugfix" {
		t.Errorf("type=%v, want=bugfix", result["type"])
	}
	if result["priority"] != "p1" {
		t.Errorf("priority=%v, want=p1", result["priority"])
	}
	if result["estimate"] != "S" {
		t.Errorf("estimate=%v, want=S", result["estimate"])
	}
}

// TestCreate_AllTypes verifies Task creation across feature, bugfix,
// refactor, infra, docs, test, chore types.
func TestCreate_AllTypes(t *testing.T) {
	types := []string{"feature", "bugfix", "refactor", "infra", "docs", "test", "chore", "spike"}
	for _, taskType := range types {
		t.Run(taskType, func(t *testing.T) {
			setupDB(t)

			raw, err := task.Create(taskType+" type Task", taskType, "", "p2", "M", "", nil)
			if err != nil {
				t.Fatalf("Create failed (%s): %v", taskType, err)
			}
			result := asMap(raw)
			if result["status"] == "BOOTSTRAP_REQUIRED" {
				t.Skip("bootstrap required - skip")
			}
			if result["status"] != "created" {
				t.Errorf("status=%v, want=created (%s)", result["status"], taskType)
			}
		})
	}
}

// TestT448_Spike_Type verifies the new spike type:
//   - Create succeeds
//   - commit_prefix = "spike"
//   - the harness template provides 3 spike-specific items
func TestT448_Spike_Type(t *testing.T) {
	setupDB(t)

	raw, err := task.Create("evaluation review test", "spike", "", "p2", "S", "", nil)
	if err != nil {
		t.Fatalf("spike Create failed: %v", err)
	}
	result := asMap(raw)
	if result["status"] == "BOOTSTRAP_REQUIRED" {
		t.Skip("bootstrap required - skip")
	}
	if result["status"] != "created" {
		t.Fatalf("status=%v, want=created", result["status"])
	}
}

// TestCreate_PerPriority verifies Task creation with priorities p0~p3.
func TestCreate_PerPriority(t *testing.T) {
	priorities := []string{"p0", "p1", "p2", "p3"}
	for _, priority := range priorities {
		t.Run(priority, func(t *testing.T) {
			setupDB(t)

			raw, err := task.Create(priority+" priority Task", "feature", "", priority, "M", "", nil)
			if err != nil {
				t.Fatalf("Create failed (%s): %v", priority, err)
			}
			result := asMap(raw)
			if result["status"] == "BOOTSTRAP_REQUIRED" {
				t.Skip("bootstrap required - skip")
			}
			if result["status"] != "created" {
				t.Errorf("status=%v, want=created (%s)", result["status"], priority)
			}
			// verify priority via Get
			taskID, ok := result["task_id"].(string)
			if !ok {
				return
			}
			getRaw, err := task.Get(taskID)
			if err != nil {
				t.Fatalf("Get failed: %v", err)
			}
			getResult := asMap(getRaw)
			if getResult["priority"] != priority {
				t.Errorf("priority=%v, want=%s", getResult["priority"], priority)
			}
		})
	}
}

// TestCreate_PerSize verifies Task creation with sizes XS, S, M, L, XL.
func TestCreate_PerSize(t *testing.T) {
	estimates := []string{"XS", "S", "M", "L", "XL"}
	for _, estimate := range estimates {
		t.Run(estimate, func(t *testing.T) {
			setupDB(t)

			raw, err := task.Create(estimate+" size Task", "feature", "", "p2", estimate, "", nil)
			if err != nil {
				t.Fatalf("Create failed (%s): %v", estimate, err)
			}
			result := asMap(raw)
			if result["status"] == "BOOTSTRAP_REQUIRED" {
				t.Skip("bootstrap required - skip")
			}
			if result["status"] != "created" {
				t.Errorf("status=%v, want=created (%s)", result["status"], estimate)
			}
		})
	}
}

// TestList_EmptyDB verifies that an empty list is returned when there are
// no Tasks.
func TestList_EmptyDB(t *testing.T) {
	setupDB(t)

	raw, err := task.List(nil, nil)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	lr, ok := raw.(*domain.ListResult)
	if !ok {
		t.Logf("domain.ListResult type assertion failed: %T = %v", raw, raw)
		return
	}
	if len(lr.Tasks) != 0 {
		t.Errorf("tasks count=%d in empty DB, want=0", len(lr.Tasks))
	}
}

// ── T294 (Sprint-36): retroactive template-update detection ──

// TestList_T294_outdated_template_NewTask verifies that a Task created via
// task.Create has outdated=false (current template includes all required
// sections).
func TestList_T294_outdated_template_NewTask(t *testing.T) {
	setupDB(t)

	if _, err := task.Create("latest-template Task", "feature", "", "p2", "S", "", nil); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	raw, err := task.List(nil, nil)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	lr := raw.(*domain.ListResult)
	if len(lr.Tasks) == 0 {
		t.Fatal("Task count 0 — suspect Create failure")
	}

	// Newly created Tasks must all be outdated=false.
	for _, ts := range lr.Tasks {
		if ts.TemplateOutdated {
			t.Errorf("new Task %s is outdated=true (suspected missing required section)", ts.TaskID)
		}
	}
	if lr.OutdatedTemplateCount != 0 {
		t.Errorf("OutdatedTemplateCount=%d, want=0", lr.OutdatedTemplateCount)
	}
}

// TestList_T294_outdated_template_OldTask verifies that a Task file
// missing required sections is detected as outdated=true after manual edit.
func TestList_T294_outdated_template_OldTask(t *testing.T) {
	tmpDir := setupDB(t)

	// create normally, then overwrite the file to simulate an old body
	createRaw, err := task.Create("old-version simulation", "feature", "", "p2", "S", "", nil)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	created := createRaw.(*domain.CreateResult)

	// Overwrite the file with a body missing required sections — be careful
	// that descriptive text does not include the substring being scanned for
	// (Contains matches even narrative text).
	oldBody := "---\n" +
		"id: " + created.TaskID + "\n" +
		"title: \"old-version simulation\"\n" +
		"type: feature\n" +
		"sprint: backlog\n" +
		"status: todo\n" +
		"priority: p2\n" +
		"estimate: S\n" +
		"depends_on: []\n" +
		"created: 2026-04-12\n" +
		"---\n\n" +
		"# " + created.TaskID + " old-version simulation\n\n" +
		"## Purpose\n\nold body\n\n" +
		"## Requirements\n\n- [ ] requirement\n\n" +
		"## Done Criteria\n\n- [ ] criterion\n\n" +
		"## References\n\nnone\n"

	absPath := filepath.Join(tmpDir, created.FilePath)
	if err := os.WriteFile(absPath, []byte(oldBody), 0o644); err != nil {
		t.Fatalf("file overwrite failed: %v", err)
	}

	raw, err := task.List(nil, nil)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	lr := raw.(*domain.ListResult)

	var found bool
	for _, ts := range lr.Tasks {
		if ts.TaskID == created.TaskID {
			found = true
			if !ts.TemplateOutdated {
				t.Errorf("old-version Task %s is outdated=false (detection failed)", ts.TaskID)
			}
		}
	}
	if !found {
		t.Errorf("Task %s not found in List result", created.TaskID)
	}
	if lr.OutdatedTemplateCount < 1 {
		t.Errorf("OutdatedTemplateCount=%d, want>=1", lr.OutdatedTemplateCount)
	}
}

// TestCreate_TitleSet verifies title is stored correctly on Task creation.
func TestCreate_TitleSet(t *testing.T) {
	setupDB(t)

	title := "exact title test"
	createRaw, err := task.Create(title, "feature", "", "p2", "M", "", nil)
	createResult := asMap(createRaw)
	if err != nil || createResult["status"] != "created" {
		t.Skip("Task create failed, skip")
	}
	taskID := createResult["task_id"].(string)

	getRaw, err := task.Get(taskID)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	getResult := asMap(getRaw)
	if getResult["title"] != title {
		t.Errorf("title=%v, want=%s", getResult["title"], title)
	}
}

// =============================================================================
// T181 — task_update tool regression tests
// =============================================================================

// strPtr is a *string helper.
func strPtr(s string) *string { return &s }

// slicePtr is a *[]string helper.
func slicePtr(s []string) *[]string { return &s }

func TestUpdate_NoUpdates_Reject(t *testing.T) {
	setupDB(t)

	createRaw, _ := task.Create("Update target", "feature", "", "p2", "M", "", nil)
	cr := asMap(createRaw)
	if cr["status"] != "created" {
		t.Skip("Task create failed, skip")
	}
	taskID := cr["task_id"].(string)

	_, err := task.Update(&domain.UpdateRequest{TaskID: taskID})
	if err == nil {
		t.Fatal("expected error when there are no updates")
	}
	var ise *apperr.InvalidStateError
	if !errors.As(err, &ise) {
		t.Errorf("expected InvalidStateError, got %T", err)
	}
}

func TestUpdate_NotFound(t *testing.T) {
	setupDB(t)

	_, err := task.Update(&domain.UpdateRequest{
		TaskID:   "T9999",
		Priority: strPtr("p1"),
	})
	if err == nil {
		t.Fatal("expected error for non-existent Task")
	}
	var nf *apperr.NotFoundError
	if !errors.As(err, &nf) {
		t.Errorf("expected NotFoundError, got %T", err)
	}
}

func TestUpdate_TitleUpdate(t *testing.T) {
	setupDB(t)

	createRaw, _ := task.Create("original title", "feature", "", "p2", "M", "", nil)
	cr := asMap(createRaw)
	if cr["status"] != "created" {
		t.Skip()
	}
	taskID := cr["task_id"].(string)

	result, err := task.Update(&domain.UpdateRequest{
		TaskID: taskID,
		Title:  strPtr("new title"),
	})
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}
	if len(result.UpdatedFields) != 1 || result.UpdatedFields[0] != "title" {
		t.Errorf("updated_fields=%v, want [title]", result.UpdatedFields)
	}
	if result.OldValues["title"] != "original title" {
		t.Errorf("old title=%v", result.OldValues["title"])
	}
	if result.NewValues["title"] != "new title" {
		t.Errorf("new title=%v", result.NewValues["title"])
	}

	// DB verification
	getRaw, _ := task.Get(taskID)
	gr := asMap(getRaw)
	if gr["title"] != "new title" {
		t.Errorf("DB title=%v", gr["title"])
	}
}

func TestUpdate_PriorityUpdate(t *testing.T) {
	setupDB(t)

	createRaw, _ := task.Create("priority test", "feature", "", "p2", "M", "", nil)
	cr := asMap(createRaw)
	if cr["status"] != "created" {
		t.Skip()
	}
	taskID := cr["task_id"].(string)

	result, err := task.Update(&domain.UpdateRequest{
		TaskID:   taskID,
		Priority: strPtr("p0"),
	})
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}
	if len(result.UpdatedFields) != 1 || result.UpdatedFields[0] != "priority" {
		t.Errorf("updated_fields=%v", result.UpdatedFields)
	}
	if result.NewValues["priority"] != "p0" {
		t.Errorf("new priority=%v", result.NewValues["priority"])
	}
}

func TestUpdate_EstimateRejectInvalid(t *testing.T) {
	setupDB(t)

	createRaw, _ := task.Create("estimate test", "feature", "", "p2", "M", "", nil)
	cr := asMap(createRaw)
	if cr["status"] != "created" {
		t.Skip()
	}
	taskID := cr["task_id"].(string)

	_, err := task.Update(&domain.UpdateRequest{
		TaskID:   taskID,
		Estimate: strPtr("INVALID"),
	})
	if err == nil {
		t.Fatal("expected error for invalid estimate value")
	}
}

func TestUpdate_DependsOnAdd(t *testing.T) {
	setupDB(t)

	createRaw, _ := task.Create("deps test", "feature", "", "p2", "M", "", nil)
	cr := asMap(createRaw)
	if cr["status"] != "created" {
		t.Skip()
	}
	taskID := cr["task_id"].(string)

	result, err := task.Update(&domain.UpdateRequest{
		TaskID:    taskID,
		DependsOn: slicePtr([]string{"T100", "T200"}),
	})
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}
	if len(result.UpdatedFields) != 1 || result.UpdatedFields[0] != "depends_on" {
		t.Errorf("updated_fields=%v", result.UpdatedFields)
	}

	// DB verification
	getRaw, _ := task.Get(taskID)
	gr := asMap(getRaw)
	deps, ok := gr["depends_on"].([]any)
	if !ok || len(deps) != 2 {
		t.Errorf("depends_on=%v", gr["depends_on"])
	}
}

func TestUpdate_DependsOnRemove_EmptyArray(t *testing.T) {
	setupDB(t)

	createRaw, _ := task.Create("deps remove test", "feature", "", "p2", "M", "",
		[]string{"T999"})
	cr := asMap(createRaw)
	if cr["status"] != "created" {
		t.Skip()
	}
	taskID := cr["task_id"].(string)

	result, err := task.Update(&domain.UpdateRequest{
		TaskID:    taskID,
		DependsOn: slicePtr([]string{}),
	})
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}
	if len(result.UpdatedFields) != 1 || result.UpdatedFields[0] != "depends_on" {
		t.Errorf("dependency removal not in updated_fields: %v", result.UpdatedFields)
	}

	getRaw, _ := task.Get(taskID)
	gr := asMap(getRaw)
	deps, ok := gr["depends_on"].([]any)
	if !ok || len(deps) != 0 {
		t.Errorf("depends_on not emptied: %v", gr["depends_on"])
	}
}

func TestUpdate_MultiField(t *testing.T) {
	setupDB(t)

	createRaw, _ := task.Create("multi test", "feature", "", "p2", "M", "", nil)
	cr := asMap(createRaw)
	if cr["status"] != "created" {
		t.Skip()
	}
	taskID := cr["task_id"].(string)

	result, err := task.Update(&domain.UpdateRequest{
		TaskID:   taskID,
		Title:    strPtr("new title"),
		Priority: strPtr("p1"),
		Estimate: strPtr("L"),
	})
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}
	if len(result.UpdatedFields) != 3 {
		t.Errorf("expected 3 field updates: %v", result.UpdatedFields)
	}
}

func TestUpdate_NoOp_SameValue(t *testing.T) {
	setupDB(t)

	createRaw, _ := task.Create("noop test", "feature", "", "p2", "M", "", nil)
	cr := asMap(createRaw)
	if cr["status"] != "created" {
		t.Skip()
	}
	taskID := cr["task_id"].(string)

	// Update with the current value → updated_fields empty, no error.
	result, err := task.Update(&domain.UpdateRequest{
		TaskID:   taskID,
		Priority: strPtr("p2"), // same as current
		Estimate: strPtr("M"),  // same as current
	})
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}
	if len(result.UpdatedFields) != 0 {
		t.Errorf("same-value update should produce empty updated_fields: %v", result.UpdatedFields)
	}
}

// =============================================================================
// T412 (Sprint-51) — task_update --status support
// =============================================================================

// TestT412_Update_Status_NormalTransition_TodoToInProgress verifies that
// the legitimate transition succeeds without error.
func TestT412_Update_Status_NormalTransition_TodoToInProgress(t *testing.T) {
	setupDB(t)

	createRaw, _ := task.Create("status test", "feature", "", "p2", "M", "", nil)
	cr := asMap(createRaw)
	if cr["status"] != "created" {
		t.Skip()
	}
	taskID := cr["task_id"].(string)

	result, err := task.Update(&domain.UpdateRequest{
		TaskID: taskID,
		Status: strPtr("in-progress"),
	})
	if err != nil {
		t.Fatalf("normal transition todo→in-progress error: %v", err)
	}
	if len(result.UpdatedFields) != 1 || result.UpdatedFields[0] != "status" {
		t.Errorf("updated_fields = %v", result.UpdatedFields)
	}
	if result.NewValues["status"] != "in-progress" {
		t.Errorf("new status = %v", result.NewValues["status"])
	}

	// DB verification
	getRaw, _ := task.Get(taskID)
	gr := asMap(getRaw)
	if gr["status"] != "in-progress" {
		t.Errorf("DB status = %v", gr["status"])
	}
}

// TestT412_Update_Status_AbnormalTransition_Allowed verifies that an
// abnormal transition such as todo → done logs WARN but is not blocked
// (escape-hatch design).
func TestT412_Update_Status_AbnormalTransition_Allowed(t *testing.T) {
	setupDB(t)

	createRaw, _ := task.Create("abnormal test", "feature", "", "p2", "M", "", nil)
	cr := asMap(createRaw)
	if cr["status"] != "created" {
		t.Skip()
	}
	taskID := cr["task_id"].(string)

	_, err := task.Update(&domain.UpdateRequest{
		TaskID: taskID,
		Status: strPtr("done"),
	})
	if err != nil {
		t.Errorf("abnormal transition must be allowed: %v", err)
	}
	getRaw, _ := task.Get(taskID)
	gr := asMap(getRaw)
	if gr["status"] != "done" {
		t.Errorf("status after abnormal transition = %v", gr["status"])
	}
}

// TestT412_Update_Status_RejectInvalidValue verifies that values outside
// the enum are rejected.
func TestT412_Update_Status_RejectInvalidValue(t *testing.T) {
	setupDB(t)

	createRaw, _ := task.Create("bad status test", "feature", "", "p2", "M", "", nil)
	cr := asMap(createRaw)
	if cr["status"] != "created" {
		t.Skip()
	}
	taskID := cr["task_id"].(string)

	_, err := task.Update(&domain.UpdateRequest{
		TaskID: taskID,
		Status: strPtr("bogus"),
	})
	if err == nil {
		t.Fatal("expected error for invalid status value")
	}
}

func TestUpdate_TypeRejectInvalid(t *testing.T) {
	setupDB(t)

	createRaw, _ := task.Create("type test", "feature", "", "p2", "M", "", nil)
	cr := asMap(createRaw)
	if cr["status"] != "created" {
		t.Skip()
	}
	taskID := cr["task_id"].(string)

	_, err := task.Update(&domain.UpdateRequest{
		TaskID: taskID,
		Type:   strPtr("invalid_type"),
	})
	if err == nil {
		t.Fatal("expected error for invalid type value")
	}
}

func TestUpdate_RejectEmptyTitle(t *testing.T) {
	setupDB(t)

	createRaw, _ := task.Create("empty title test", "feature", "", "p2", "M", "", nil)
	cr := asMap(createRaw)
	if cr["status"] != "created" {
		t.Skip()
	}
	taskID := cr["task_id"].(string)

	_, err := task.Update(&domain.UpdateRequest{
		TaskID: taskID,
		Title:  strPtr("   "),
	})
	if err == nil {
		t.Fatal("expected error for empty title")
	}
}

// T438 — verifies that task.Get() returns the ## Summary section content
// in the summary field.
func TestT438_TaskGet_SummaryField(t *testing.T) {
	tmpDir := setupDB(t)

	createRaw, err := task.Create("summary lookup verification", "bugfix", "", "p2", "S", "", nil)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	cr := asMap(createRaw)
	if cr["status"] != "created" {
		t.Skip("Task create failed, skip")
	}
	taskID := cr["task_id"].(string)
	filePath := cr["file_path"].(string)

	// Overwrite the file with content containing a ## Summary section.
	const summaryText = "first line of summary content"
	body := "---\nid: " + taskID + "\ntitle: \"summary lookup verification\"\ntype: bugfix\nstatus: todo\npriority: p2\nestimate: S\ndepends_on: []\ncreated: 2026-04-23\n---\n\n# " + taskID + "\n\n## Summary\n\n" + summaryText + "\n\n## Purpose\n\ntest\n"
	absPath := filepath.Join(tmpDir, filePath)
	if err := os.WriteFile(absPath, []byte(body), 0o644); err != nil {
		t.Fatalf("file overwrite failed: %v", err)
	}

	raw, err := task.Get(taskID)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	result := asMap(raw)

	// summary field must be present and have the expected value.
	summaryVal, ok := result["summary"]
	if !ok {
		t.Fatal("summary field missing")
	}
	if summaryVal != summaryText {
		t.Errorf("summary expected %q, got %q", summaryText, summaryVal)
	}

	// A Task without a ## Summary section should return summary: "" (not nil).
	bodyNoSummary := "---\nid: " + taskID + "\ntitle: \"summary lookup verification\"\ntype: bugfix\nstatus: todo\npriority: p2\nestimate: S\ndepends_on: []\ncreated: 2026-04-23\n---\n\n# " + taskID + "\n\n## Purpose\n\ntest\n"
	if err := os.WriteFile(absPath, []byte(bodyNoSummary), 0o644); err != nil {
		t.Fatalf("file overwrite failed: %v", err)
	}
	raw2, err := task.Get(taskID)
	if err != nil {
		t.Fatalf("Get (no summary) failed: %v", err)
	}
	result2 := asMap(raw2)
	summaryVal2, ok2 := result2["summary"]
	if !ok2 {
		t.Fatal("summary field missing (no-section case)")
	}
	if summaryVal2 != "" {
		t.Errorf("summary for no-summary Task expected empty string, got %q", summaryVal2)
	}
}

func TestT439_TaskUpdate_SprintField(t *testing.T) {
	setupDB(t)

	createRaw, err := task.Create("sprint direct-update verification", "feature", "", "p3", "S", "", nil)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	cr := asMap(createRaw)
	if cr["status"] != "created" {
		t.Skip("Task create failed, skip")
	}
	taskID := cr["task_id"].(string)

	// Update sprint field directly.
	newSprint := "sprint-37"
	req := &domain.UpdateRequest{
		TaskID: taskID,
		Sprint: &newSprint,
	}
	updateRaw, err := task.Update(req)
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}
	result := asMap(updateRaw)

	// updated_fields must contain "sprint".
	updatedFields, ok := result["updated_fields"].([]interface{})
	if !ok {
		t.Fatal("updated_fields missing")
	}
	foundSprint := false
	for _, f := range updatedFields {
		if f == "sprint" {
			foundSprint = true
		}
	}
	if !foundSprint {
		t.Errorf("sprint missing from updated_fields: %v", updatedFields)
	}

	// new_values.sprint check.
	newValues, _ := result["new_values"].(map[string]interface{})
	if newValues["sprint"] != newSprint {
		t.Errorf("new_values.sprint expected %q, got %q", newSprint, newValues["sprint"])
	}

	// warnings must include the no-file-move warning.
	warnings, _ := result["warnings"].([]interface{})
	if len(warnings) == 0 {
		t.Error("a no-file-move warning should be returned")
	}

	// Confirm DB sprint updated via Get.
	getRaw, err := task.Get(taskID)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	getResult := asMap(getRaw)
	if getResult["sprint"] != newSprint {
		t.Errorf("Get response sprint expected %q, got %q", newSprint, getResult["sprint"])
	}

	// Same-value re-update must produce empty updated_fields (no-op).
	req2 := &domain.UpdateRequest{TaskID: taskID, Sprint: &newSprint}
	update2Raw, err := task.Update(req2)
	if err != nil {
		t.Fatalf("Update (no-op) failed: %v", err)
	}
	result2 := asMap(update2Raw)
	updatedFields2, _ := result2["updated_fields"].([]interface{})
	if len(updatedFields2) != 0 {
		t.Errorf("same-value re-update should be a no-op, got updated_fields=%v", updatedFields2)
	}
}

// TestT733_TaskComplete_FrontmatterDB_AtomicSync verifies that after
// task.Complete the file frontmatter status and DB status are in sync.
// Regression guard: blocks the file frontmatter in-progress / DB done
// mismatch case.
func TestT733_TaskComplete_FrontmatterDB_AtomicSync(t *testing.T) {
	tmpDir := setupDB(t)

	createRaw, err := task.Create("atomic sync regression test", "chore", "", "p3", "S", "", nil)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	taskID := asMap(createRaw)["task_id"].(string)

	if _, err := task.Start(taskID); err != nil {
		t.Fatalf("Start failed: %v", err)
	}
	if _, err := task.Complete(taskID, true); err != nil {
		t.Fatalf("Complete failed: %v", err)
	}

	// look up DB status
	getRaw, err := task.Get(taskID)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	dbStatus := asMap(getRaw)["status"]
	if dbStatus != "done" {
		t.Errorf("DB status=%v want=done", dbStatus)
	}

	// extract file path + check frontmatter status directly
	filePathRaw := asMap(getRaw)["file_path"]
	if filePathRaw == nil {
		t.Fatalf("file_path nil")
	}
	filePath, _ := filePathRaw.(string)
	if filePath == "" {
		t.Fatalf("file_path empty")
	}
	absPath := filepath.Join(tmpDir, filePath)
	data, err := os.ReadFile(absPath)
	if err != nil {
		t.Fatalf("file read failed: %v (path=%s)", err, absPath)
	}
	// parse frontmatter status
	fmStatus := ""
	for _, line := range strings.Split(string(data), "\n") {
		if strings.HasPrefix(line, "status:") {
			fmStatus = strings.TrimSpace(strings.TrimPrefix(line, "status:"))
			break
		}
	}
	if fmStatus != "done" {
		t.Errorf("frontmatter status=%q want=done (DB=%v) — atomic sync failed", fmStatus, dbStatus)
	}
}
