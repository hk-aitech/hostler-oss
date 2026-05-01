package sprint_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/pkg/sprint"
)

// T245 (Sprint-17, ISS-20260421-002) — full Sprint lifecycle regression.
//
// Background: a downstream report observed that sprint-209 existed in both
// `works/sprints/backlog/sprint-209/` (SPRINT.md + 9 tasks) and
// `works/sprints/completed/sprint-209/` simultaneously, raising 13
// duplicate_task_id hits. hostler's MoveSprint (pkg/sprint/sprint.go
// L388-393) calls RemoveAll when residue is detected at the source path,
// so structurally this should not reproduce.
//
// This test exercises the full lifecycle (backlog → active → completed)
// and asserts that **the previous path is fully gone** after each move.
// If that invariant ever breaks the test fails immediately.

func TestT245_Lifecycle_NoResidueAfterEachMove(t *testing.T) {
	tmpRoot := setupDB(t)
	sprintID := "sprint-t245"

	// Step 1: create folder under backlog.
	backlogDir, err := sprint.CreateFolder(sprintID, "backlog")
	if err != nil {
		t.Fatalf("CreateFolder failed: %v", err)
	}
	// Add content — SPRINT.md + several task files (mirroring the
	// downstream pattern).
	if err := os.WriteFile(filepath.Join(backlogDir, "SPRINT.md"),
		[]byte("---\nid: "+sprintID+"\n---\n"), 0o644); err != nil {
		t.Fatalf("write SPRINT.md failed: %v", err)
	}
	tasksDir := filepath.Join(backlogDir, "tasks")
	for _, tid := range []string{"T001", "T002", "T003"} {
		if err := os.WriteFile(filepath.Join(tasksDir, tid+"-sample.md"),
			[]byte("---\nid: "+tid+"\n---\n"), 0o644); err != nil {
			t.Fatalf("write Task file failed: %v", err)
		}
	}

	expectBacklog := filepath.Join(tmpRoot, "works", "sprints", "backlog", sprintID)
	expectActive := filepath.Join(tmpRoot, "works", "sprints", "active", sprintID)
	expectCompleted := filepath.Join(tmpRoot, "works", "sprints", "completed", sprintID)

	// Step 2: backlog → active.
	if _, err := sprint.MoveSprint(sprintID, "backlog", "active"); err != nil {
		t.Fatalf("MoveSprint backlog→active failed: %v", err)
	}
	assertPathAbsent(t, "backlog residue (after backlog→active)", expectBacklog)
	assertPathExists(t, "active move result", expectActive)

	// Step 3: active → completed.
	if _, err := sprint.MoveSprint(sprintID, "active", "completed"); err != nil {
		t.Fatalf("MoveSprint active→completed failed: %v", err)
	}
	assertPathAbsent(t, "backlog residue (after active→completed)", expectBacklog)
	assertPathAbsent(t, "active residue (after active→completed)", expectActive)
	assertPathExists(t, "completed final result", expectCompleted)

	// Verify task files moved with the sprint — guards against the
	// duplicate_task_id root cause from the original incident.
	for _, tid := range []string{"T001", "T002", "T003"} {
		path := filepath.Join(expectCompleted, "tasks", tid+"-sample.md")
		assertPathExists(t, "task file in completed", path)
	}
}

// TestT245_MoveSprint_ExternalResidueNotDestroyed — when an external cause
// (manual user edit, etc.) creates a separate folder under backlog,
// MoveSprint must leave it alone. In that case the backlog sync regression
// guard (T246) classifies the "duplicate / residue" as AutoFixable=false
// to surface manual intervention.
func TestT245_MoveSprint_ExternalResidueNotDestroyed(t *testing.T) {
	tmpRoot := setupDB(t)
	sprintID := "sprint-t245-ext"

	// Create a folder under active.
	activeDir := filepath.Join(tmpRoot, "works", "sprints", "active", sprintID)
	if err := os.MkdirAll(filepath.Join(activeDir, "tasks"), 0o755); err != nil {
		t.Fatalf("create active folder failed: %v", err)
	}
	if err := os.WriteFile(filepath.Join(activeDir, "SPRINT.md"),
		[]byte("---\nid: "+sprintID+"\n---\n"), 0o644); err != nil {
		t.Fatalf("write active SPRINT.md failed: %v", err)
	}

	// Simulate an externally created backlog folder (manual cp, etc.).
	backlogDir := filepath.Join(tmpRoot, "works", "sprints", "backlog", sprintID)
	if err := os.MkdirAll(backlogDir, 0o755); err != nil {
		t.Fatalf("create external backlog folder failed: %v", err)
	}
	externalMarker := filepath.Join(backlogDir, "external-user-file.txt")
	if err := os.WriteFile(externalMarker, []byte("user data"), 0o644); err != nil {
		t.Fatalf("write external marker failed: %v", err)
	}

	// active → completed move (MoveSprint only touches active).
	if _, err := sprint.MoveSprint(sprintID, "active", "completed"); err != nil {
		t.Fatalf("MoveSprint failed: %v", err)
	}

	// External files must be preserved — MoveSprint never touches
	// unrelated paths.
	assertPathExists(t, "external backlog marker preserved", externalMarker)

	// At this point both directories coexist (the situation ISS-002
	// describes). sync detects this as a regression issue and surfaces it
	// to the user (T246 guard). This test only verifies the invariant
	// that MoveSprint does not delete external files.
}

func assertPathExists(t *testing.T, label, path string) {
	t.Helper()
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Errorf("%s: path should exist but is missing (%s)", label, path)
	}
}

func assertPathAbsent(t *testing.T, label, path string) {
	t.Helper()
	if _, err := os.Stat(path); err == nil {
		t.Errorf("%s: path should be absent but exists (%s)", label, path)
	}
}
