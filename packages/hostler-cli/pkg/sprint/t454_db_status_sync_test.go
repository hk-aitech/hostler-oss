// T454 (Sprint-40): regression test that on sprint complete the Task DB
// status is force-aligned. Verifies reconcileTaskDBStatusAfterMove syncs
// the DB to the file SSOT.
package sprint

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/internal/adapters/sqlite"
	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/internal/ports"
	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/pkg/db"
	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/pkg/store"
)

// setupT454TestEnv prepares an isolated DB + project root for T454 tests.
func setupT454TestEnv(t *testing.T) (root string) {
	t.Helper()
	tmpDir := t.TempDir()
	t.Setenv("HSTL_DB_PATH", filepath.Join(tmpDir, "hstl.db"))
	t.Setenv("HSTL_PROJECT_ROOT", tmpDir)
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

// writeTaskFile creates a Task file (frontmatter + minimal body) for
// testing.
func writeTaskFile(t *testing.T, root, relPath, taskID, status string) string {
	t.Helper()
	absPath := filepath.Join(root, relPath)
	if err := os.MkdirAll(filepath.Dir(absPath), 0o755); err != nil {
		t.Fatalf("directory create failed: %v", err)
	}
	content := fmt.Sprintf(`---
id: %s
title: T454 test task
status: %s
sprint: sprint-t454
type: chore
priority: p2
estimate: S
---

# %s

body.
`, taskID, status, taskID)
	if err := os.WriteFile(absPath, []byte(content), 0o644); err != nil {
		t.Fatalf("file write failed: %v", err)
	}
	return absPath
}

// TestT454_ReconcileTaskDBStatusAfterMove_FileDoneDBTodo_ForceAlign verifies
// that when file=done but DB=todo (drift), the sprint-complete path
// force-aligns DB to done.
func TestT454_ReconcileTaskDBStatusAfterMove_FileDoneDBTodo_ForceAlign(t *testing.T) {
	root := setupT454TestEnv(t)

	relPath := "works/sprints/completed/sprint-t454/tasks/T0001-test.md"
	writeTaskFile(t, root, relPath, "T0001", "done")

	gs := store.Get()
	if gs == nil {
		t.Fatalf("store nil")
	}
	if err := gs.CreateTask(&ports.TaskRecord{
		TaskID:    "T0001",
		Title:     "T454 test",
		Type:      "chore",
		Status:    "todo",
		Priority:  "p2",
		Estimate:  "S",
		Sprint:    "sprint-t454",
		FilePath:  relPath,
		CreatedAt: "2026-04-24",
	}); err != nil {
		t.Fatalf("CreateTask failed: %v", err)
	}

	forced := reconcileTaskDBStatusAfterMove("sprint-t454")
	if forced != 1 {
		t.Errorf("force updated = %d, want 1", forced)
	}

	dbStatus, err := gs.GetTaskStatus("T0001")
	if err != nil {
		t.Fatalf("GetTaskStatus failed: %v", err)
	}
	if dbStatus != "done" {
		t.Errorf("DB status = %q, want done", dbStatus)
	}
}

// TestT454_ReconcileTaskDBStatusAfterMove_FileDoneDBDoneNoop verifies that
// when both file and DB are done, the call is a no-op (idempotency).
func TestT454_ReconcileTaskDBStatusAfterMove_FileDoneDBDoneNoop(t *testing.T) {
	root := setupT454TestEnv(t)

	relPath := "works/sprints/completed/sprint-t454/tasks/T0002-test.md"
	writeTaskFile(t, root, relPath, "T0002", "done")

	gs := store.Get()
	if err := gs.CreateTask(&ports.TaskRecord{
		TaskID:    "T0002",
		Title:     "T454 noop",
		Type:      "chore",
		Status:    "done",
		Priority:  "p2",
		Estimate:  "S",
		Sprint:    "sprint-t454",
		FilePath:  relPath,
		CreatedAt: "2026-04-24",
	}); err != nil {
		t.Fatalf("CreateTask failed: %v", err)
	}

	forced := reconcileTaskDBStatusAfterMove("sprint-t454")
	if forced != 0 {
		t.Errorf("force updated = %d, want 0 (no-op)", forced)
	}
}

// TestT454_ReconcileTaskDBStatusAfterMove_FileTodo_DoNotTouch verifies
// that when the file is todo (unfinished Task) the DB must not be touched.
// Sprint complete itself should already be blocked by the ceremony gate,
// so reconcile must not erroneously force-transition.
func TestT454_ReconcileTaskDBStatusAfterMove_FileTodo_DoNotTouch(t *testing.T) {
	root := setupT454TestEnv(t)

	relPath := "works/sprints/completed/sprint-t454/tasks/T0003-test.md"
	writeTaskFile(t, root, relPath, "T0003", "todo")

	gs := store.Get()
	if err := gs.CreateTask(&ports.TaskRecord{
		TaskID:    "T0003",
		Title:     "T454 file todo",
		Type:      "chore",
		Status:    "todo",
		Priority:  "p2",
		Estimate:  "S",
		Sprint:    "sprint-t454",
		FilePath:  relPath,
		CreatedAt: "2026-04-24",
	}); err != nil {
		t.Fatalf("CreateTask failed: %v", err)
	}

	forced := reconcileTaskDBStatusAfterMove("sprint-t454")
	if forced != 0 {
		t.Errorf("force updated = %d, want 0 (file todo — must not touch)", forced)
	}

	dbStatus, _ := gs.GetTaskStatus("T0003")
	if dbStatus != "todo" {
		t.Errorf("DB status = %q, want todo (must not change)", dbStatus)
	}
}
