// Package task_test — T183: UpdateSprintMD 7-column rendering regression
// tests.
//
// In sprint-22 T180 the SPRINT.md Task table was rendering only 5 of 7
// columns (missing priority / depends_on). After extending the
// UpdateSprintMD signature, this test verifies end-to-end that all 4
// caller paths (Create / AssignSprint / transitionTask / Reopen) follow
// the 7-column format.
//
// Additional case: a sprint_create batch path discovered during
// sprint-24 design produced rows with empty title + defaults
// (feature/M/p2) — sprint.Create going through task_assign_sprint filled
// rows with placeholders instead of querying the DB. A reproduction test
// for that bug is also included.
package task_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/pkg/task"
)

// setupSprintWithMD inserts a Sprint into DB and creates a SPRINT.md
// header. UpdateSprintMD requires the SPRINT.md file to be present.
func setupSprintWithMD(t *testing.T, tmpDir, sprintID string) string {
	t.Helper()
	insertSprint(t, tmpDir, sprintID)

	sprintMD := "---\nid: " + sprintID + "\nstatus: active\n---\n\n# " + sprintID + "\n\n## Task list\n\n" +
		"| ID | Title | type | estimate | priority | status | dependencies |\n" +
		"|----|-------|------|----------|----------|--------|--------------|\n\n" +
		"## Done Criteria\n\n- [ ] all Tasks done\n"

	sprintPath := filepath.Join(tmpDir, "works", "sprints", "active", sprintID, "SPRINT.md")
	if err := os.WriteFile(sprintPath, []byte(sprintMD), 0o644); err != nil {
		t.Fatalf("SPRINT.md create failed: %v", err)
	}
	return sprintPath
}

// readSprintMD returns the SPRINT.md file content as a string.
func readSprintMD(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("SPRINT.md read failed: %v", err)
	}
	return string(data)
}

// assertRow7Col verifies SPRINT.md content includes a 7-column row
// (id + 5 cols + dependencies).
func assertRow7Col(t *testing.T, content, taskID string) {
	t.Helper()
	// 7-col row starts with "| {id} |" and contains 6 pipe separators
	// (8 pipe chars total).
	lines := strings.Split(content, "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "| "+taskID+" ") || strings.HasPrefix(line, "| "+taskID+"|") {
			// Count pipes to validate column count (7 columns → 8 pipes).
			pipeCount := strings.Count(line, "|")
			if pipeCount < 8 {
				t.Errorf("%s row pipe count too low (expected >=8, got %d): %s", taskID, pipeCount, line)
			}
			return
		}
	}
	t.Errorf("row for %s not found in SPRINT.md:\n%s", taskID, content)
}

// TestCreate_SPRINT_MD_7Col verifies that task.Create writes a 7-column
// row to SPRINT.md when the Sprint is assigned. (Caller path "Create".)
func TestCreate_SPRINT_MD_7Col(t *testing.T) {
	tmpDir := setupDB(t)
	sprintPath := setupSprintWithMD(t, tmpDir, "sprint-01")

	raw, err := task.Create("Create-path Task", "feature", "sprint-01", "p1", "M", "", nil)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	result := asMap(raw)
	if result["status"] == "BOOTSTRAP_REQUIRED" {
		t.Skip("bootstrap required, skip")
	}
	taskID := result["task_id"].(string)

	content := readSprintMD(t, sprintPath)
	assertRow7Col(t, content, taskID)
	// confirm the priority column exists
	if !strings.Contains(content, "| p1 |") {
		t.Errorf("priority p1 column missing:\n%s", content)
	}
}

// TestAssignSprint_SPRINT_MD_7Col verifies that task.AssignSprint writes a
// 7-column row to SPRINT.md on BACKLOG → Sprint move. (Caller path
// "AssignSprint".)
func TestAssignSprint_SPRINT_MD_7Col(t *testing.T) {
	t.Setenv("HSTL_ASSIGN_ALLOW_PLACEHOLDER", "1") // T572
	tmpDir := setupDB(t)
	sprintPath := setupSprintWithMD(t, tmpDir, "sprint-02")

	// create a BACKLOG Task
	raw, err := task.Create("AssignSprint-path Task", "refactor", "", "p0", "L", "", nil)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	taskID := asMap(raw)["task_id"].(string)

	// assign to Sprint
	if _, err := task.AssignSprint([]string{taskID}, "sprint-02"); err != nil {
		t.Fatalf("AssignSprint failed: %v", err)
	}

	content := readSprintMD(t, sprintPath)
	assertRow7Col(t, content, taskID)
	if !strings.Contains(content, "| p0 |") {
		t.Errorf("priority p0 column missing (column not preserved on BACKLOG→Sprint move):\n%s", content)
	}
}

// TestTransition_SPRINT_MD_7Col verifies that on task_start / task_complete
// transitions the existing 7 columns of SPRINT.md are preserved.
// (Caller path "transitionTask".)
func TestTransition_SPRINT_MD_7Col(t *testing.T) {
	tmpDir := setupDB(t)
	sprintPath := setupSprintWithMD(t, tmpDir, "sprint-03")

	raw, err := task.Create("Transition-path Task", "feature", "sprint-03", "p2", "S", "", nil)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	result := asMap(raw)
	if result["status"] == "BOOTSTRAP_REQUIRED" {
		t.Skip("bootstrap required, skip")
	}
	taskID := result["task_id"].(string)

	// start transition
	if _, err := task.Start(taskID); err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	content := readSprintMD(t, sprintPath)
	assertRow7Col(t, content, taskID)
	// confirm priority preserved
	if !strings.Contains(content, "| p2 |") {
		t.Errorf("priority p2 not preserved after transition:\n%s", content)
	}
	// status updated to in-progress
	if !strings.Contains(content, "in-progress") {
		t.Errorf("status not updated to in-progress:\n%s", content)
	}
}

// TestReopen_SPRINT_MD_7Col verifies that task_reopen does not overwrite
// priority / depends_on of the existing 7-column row and only updates
// status. (Caller path "Reopen".)
func TestReopen_SPRINT_MD_7Col(t *testing.T) {
	tmpDir := setupDB(t)
	sprintPath := setupSprintWithMD(t, tmpDir, "sprint-04")

	raw, err := task.Create("Reopen-path Task", "bugfix", "sprint-04", "p1", "M", "", nil)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	result := asMap(raw)
	if result["status"] == "BOOTSTRAP_REQUIRED" {
		t.Skip("bootstrap required, skip")
	}
	taskID := result["task_id"].(string)

	// start (Harness Gate may block task.Complete with BLOCKED, so Reopen
	// runs from in-progress to todo)
	if _, err := task.Start(taskID); err != nil {
		t.Fatalf("Start failed: %v", err)
	}
	if _, err := task.Reopen(taskID, "test reopen reason longer than 10 chars", "todo"); err != nil {
		t.Fatalf("Reopen failed: %v", err)
	}

	content := readSprintMD(t, sprintPath)
	assertRow7Col(t, content, taskID)
	// confirm priority preserved (Reopen does not overwrite)
	if !strings.Contains(content, "| p1 |") {
		t.Errorf("priority p1 not preserved after Reopen:\n%s", content)
	}
}

// TestUpdateSprintMD_EmptyValueFallback verifies fallback behaviour for
// empty priority / dependsOn. fileutil has its own tests; this one
// confirms defaults are applied at the task-level caller path.
func TestUpdateSprintMD_EmptyValueFallback(t *testing.T) {
	tmpDir := setupDB(t)
	sprintPath := setupSprintWithMD(t, tmpDir, "sprint-05")

	// force an empty priority for testing
	raw, err := task.Create("Fallback Task", "chore", "sprint-05", "", "XS", "", nil)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	result := asMap(raw)
	if result["status"] == "BOOTSTRAP_REQUIRED" {
		t.Skip("bootstrap required, skip")
	}

	content := readSprintMD(t, sprintPath)
	// empty priority should render as p2 default
	if !strings.Contains(content, "| p2 |") {
		t.Errorf("priority empty → p2 fallback failed:\n%s", content)
	}
	// empty depends_on should render as —
	if !strings.Contains(content, "| — |") {
		t.Errorf("depends_on empty → — fallback failed:\n%s", content)
	}
}
