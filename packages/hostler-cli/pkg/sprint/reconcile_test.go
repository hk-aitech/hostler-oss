package sprint_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/internal/ports"
	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/pkg/sprint"
	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/pkg/store"
)

// T607 — Reconcile uses files as the SSOT to repair the DB whenever they
// diverge from each other.
// Scenario: main worktree records sprint-70 "A" in DB → another worktree
// creates the same id as "B" and overwrites the DB → return to main and
// reconcile → DB recovers.

func writeSprintMD(t *testing.T, root, location, id, title, goal string) string {
	t.Helper()
	dir := filepath.Join(root, "works", "sprints", location, id)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	content := "---\nid: " + id + "\ntitle: \"" + title + "\"\ngoal: \"" + goal + "\"\nstatus: " + location + "\n---\n\n# " + id + "\n"
	p := filepath.Join(dir, "SPRINT.md")
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	return p
}

func TestT607_Reconcile_TitleRecovery(t *testing.T) {
	root := setupDB(t)
	id := "sprint-70"

	// 1) Filesystem: write backlog/sprint-70/SPRINT.md with the canonical "A".
	writeSprintMD(t, root, "backlog", id, "canonical A", "goal-A")

	// 2) Save a drifted record ("wrong B") directly into DB.
	if err := sprint.SaveToDB(id, "wrong B", "goal-B", "backlog",
		"works/sprints/backlog/"+id, nil, nil); err != nil {
		t.Fatalf("SaveToDB: %v", err)
	}

	// 3) dry-run mode: detect diff but do not change DB.
	result, err := sprint.Reconcile(id, true)
	if err != nil {
		t.Fatalf("Reconcile dry-run: %v", err)
	}
	if len(result.Diffs) == 0 {
		t.Fatalf("dry-run: expected at least one diff")
	}
	if result.Applied {
		t.Errorf("dry-run but Applied=true")
	}
	rec, _ := sprint.GetFromDB(id)
	if rec.Title != "wrong B" {
		t.Errorf("dry-run yet DB changed: title=%q", rec.Title)
	}

	// 4) apply for real
	result2, err := sprint.Reconcile(id, false)
	if err != nil {
		t.Fatalf("Reconcile apply: %v", err)
	}
	if !result2.Applied {
		t.Errorf("apply: Applied=false")
	}
	rec2, _ := sprint.GetFromDB(id)
	if rec2.Title != "canonical A" {
		t.Errorf("recovery failed: got=%q want=%q", rec2.Title, "canonical A")
	}
	if rec2.Goal != "goal-A" {
		t.Errorf("goal recovery failed: got=%q", rec2.Goal)
	}
}

func TestT607_CheckCreateGuard_TitleMismatch(t *testing.T) {
	_ = setupDB(t)
	id := "sprint-99"

	// existing DB record present
	if err := sprint.SaveToDB(id, "existing title", "", "backlog",
		"works/sprints/backlog/"+id, nil, nil); err != nil {
		t.Fatalf("SaveToDB: %v", err)
	}

	// same title → pass
	if err := sprint.CheckCreateGuard(id, "existing title"); err != nil {
		t.Errorf("guard fired with same title: %v", err)
	}

	// different title → reject
	if err := sprint.CheckCreateGuard(id, "different title"); err == nil {
		t.Errorf("guard accepted a different title")
	}
}

// T437 — Reconcile detects and fixes Task DB drift.
func TestT437_Reconcile_TaskDriftDetection(t *testing.T) {
	root := setupDB(t)
	sprintID := "sprint-drift-test"
	writeSprintMD(t, root, "backlog", sprintID, "Task drift verification", "goal")
	if err := sprint.SaveToDB(sprintID, "Task drift verification", "goal", "backlog",
		"works/sprints/backlog/"+sprintID, nil, nil); err != nil {
		t.Fatalf("SaveToDB: %v", err)
	}

	// Register a Task in DB — file_path is under the sprint folder.
	gs := store.Get()
	if gs == nil {
		t.Fatal("store nil")
	}
	taskID := "T8881"
	dbPath := filepath.ToSlash(filepath.Join("works", "sprints", "backlog", sprintID, "tasks", taskID+"-test.md"))
	rec := &ports.TaskRecord{
		TaskID:   taskID,
		Title:    "drift test Task",
		Type:     "chore",
		Status:   "todo",
		Priority: "p3",
		Estimate: "XS",
		Sprint:   sprintID,
		FilePath: dbPath,
	}
	if err := gs.CreateTask(rec); err != nil {
		t.Fatalf("CreateTask: %v", err)
	}

	// File exists at a different location (simulating manual move into
	// works/tasks/).
	actualDir := filepath.Join(root, "works", "tasks")
	if err := os.MkdirAll(actualDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	actualFile := filepath.Join(actualDir, taskID+"-test.md")
	if err := os.WriteFile(actualFile, []byte("# "+taskID), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	// dry-run: only detect drift.
	result, err := sprint.Reconcile(sprintID, true)
	if err != nil {
		t.Fatalf("Reconcile dry-run: %v", err)
	}
	if len(result.TaskDrifts) != 1 {
		t.Fatalf("expected TaskDrifts count 1, got %d", len(result.TaskDrifts))
	}
	drift := result.TaskDrifts[0]
	if drift.TaskID != taskID {
		t.Errorf("drift.TaskID expected %s, got %s", taskID, drift.TaskID)
	}
	if drift.ActualPath == "" {
		t.Errorf("drift.ActualPath should be populated (actual file found)")
	}
	// dry-run, so DB is not yet updated.
	filePath, _ := gs.GetTaskFilePath(taskID)
	if filePath != dbPath {
		t.Errorf("dry-run yet DB file_path changed: %q", filePath)
	}

	// real apply (dryRun=false).
	result2, err := sprint.Reconcile(sprintID, false)
	if err != nil {
		t.Fatalf("Reconcile apply: %v", err)
	}
	if len(result2.TaskDrifts) != 1 {
		t.Fatalf("apply: expected TaskDrifts count 1, got %d", len(result2.TaskDrifts))
	}
	// confirm DB file_path was updated to the actual location.
	updatedPath, _ := gs.GetTaskFilePath(taskID)
	expectedPath := filepath.ToSlash(filepath.Join("works", "tasks", taskID+"-test.md"))
	if updatedPath != expectedPath {
		t.Errorf("DB file_path update expected %q, got %q", expectedPath, updatedPath)
	}
}

func TestT607_UpdateFields_PartialUpdate(t *testing.T) {
	_ = setupDB(t)
	id := "sprint-42"
	if err := sprint.SaveToDB(id, "original", "goal-orig", "backlog",
		"works/sprints/backlog/"+id, nil, nil); err != nil {
		t.Fatalf("SaveToDB: %v", err)
	}

	newTitle := "updated"
	if err := sprint.UpdateFields(id, &newTitle, nil, nil, nil); err != nil {
		t.Fatalf("UpdateFields: %v", err)
	}
	rec, _ := sprint.GetFromDB(id)
	if rec.Title != "updated" {
		t.Errorf("title not updated: %q", rec.Title)
	}
	if rec.Goal != "goal-orig" {
		t.Errorf("goal was modified: %q", rec.Goal)
	}
}
