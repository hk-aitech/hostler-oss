// Package sqlite_test — TransitionTaskAtomic regression tests.
//
// Verifies the atomicity contract:
//   - Happy path: DB status + file_path commit in a single transaction.
//   - Wrong expectedStatus: rollback, DB state unchanged.
//   - fileOp error: rollback, DB status reverts.
//   - fileOp returns newPath=="": status updated, file_path untouched.
//
// Root purpose: prevent the 3-way drift (DB ↔ file location ↔
// frontmatter) that previously occurred when task start/complete
// failed mid-way during file move + DB update.
package sqlite_test

import (
	"errors"
	"testing"

	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/internal/adapters/sqlite"
	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/internal/ports"
)

// seedTransitionTask inserts a Task into the DB with the given
// status/file_path.
func seedTransitionTask(t *testing.T, store ports.GraphStore, taskID, status, filePath string) {
	t.Helper()
	rec := &ports.TaskRecord{
		TaskID:    taskID,
		Title:     "transition seed",
		Type:      "test",
		Status:    status,
		Priority:  "p3",
		Estimate:  "S",
		FilePath:  filePath,
		DependsOn: "[]",
		CreatedAt: "2026-04-17T00:00:00Z",
	}
	if err := store.CreateTask(rec); err != nil {
		t.Fatalf("seed CreateTask: %v", err)
	}
}

func readTaskStatusAndPath(t *testing.T, store ports.GraphStore, taskID string) (status, path string) {
	t.Helper()
	got, err := store.GetTaskFull(taskID)
	if err != nil || got == nil {
		t.Fatalf("GetTaskFull(%s): %v (got=%v)", taskID, err, got)
	}
	return got.Status, got.FilePath
}

// TestT687_TransitionTaskAtomic_Commits_StatusAndPath verifies that on
// a successful transition both status and file_path commit in the same
// transaction.
func TestT687_TransitionTaskAtomic_Commits_StatusAndPath(t *testing.T) {
	store := sqlite.New(openTestDB(t))
	seedTransitionTask(t, store, "T901", "todo", "works/tasks/T901.md")

	called := false
	err := store.TransitionTaskAtomic("T901", "todo", "in-progress",
		func() (string, error) {
			called = true
			return "works/tasks/in-progress/T901.md", nil
		},
	)
	if err != nil {
		t.Fatalf("TransitionTaskAtomic: %v", err)
	}
	if !called {
		t.Fatal("fileOp callback was not invoked")
	}

	gotStatus, gotPath := readTaskStatusAndPath(t, store, "T901")
	if gotStatus != "in-progress" {
		t.Errorf("status = %q, want in-progress", gotStatus)
	}
	if gotPath != "works/tasks/in-progress/T901.md" {
		t.Errorf("file_path = %q, want works/tasks/in-progress/T901.md", gotPath)
	}
}

// TestT687_TransitionTaskAtomic_RollsBackOnFileOpError verifies that
// when fileOp returns an error the status is rolled back too.
func TestT687_TransitionTaskAtomic_RollsBackOnFileOpError(t *testing.T) {
	store := sqlite.New(openTestDB(t))
	seedTransitionTask(t, store, "T902", "todo", "works/tasks/T902.md")

	sentinel := errors.New("file move failed")
	err := store.TransitionTaskAtomic("T902", "todo", "done",
		func() (string, error) {
			return "", sentinel
		},
	)
	if err == nil {
		t.Fatal("fileOp error was not propagated")
	}
	if !errors.Is(err, sentinel) {
		// errors.Is requires %w wrapping — implementation does wrap, so this should pass.
		t.Errorf("error unwrap failed: %v", err)
	}

	gotStatus, gotPath := readTaskStatusAndPath(t, store, "T902")
	if gotStatus != "todo" {
		t.Errorf("status = %q, want todo (rollback)", gotStatus)
	}
	if gotPath != "works/tasks/T902.md" {
		t.Errorf("file_path = %q, want original (rollback)", gotPath)
	}
}

// TestT687_TransitionTaskAtomic_RejectsWrongExpectedStatus verifies
// that the UPDATE does not run when the expected state does not match.
func TestT687_TransitionTaskAtomic_RejectsWrongExpectedStatus(t *testing.T) {
	store := sqlite.New(openTestDB(t))
	seedTransitionTask(t, store, "T903", "in-progress", "works/tasks/T903.md")

	called := false
	err := store.TransitionTaskAtomic("T903", "todo", "done",
		func() (string, error) {
			called = true
			return "", nil
		},
	)
	if err == nil {
		t.Fatal("expected-status mismatch did not produce an error")
	}
	if called {
		t.Error("fileOp must not be called when the expected status does not match")
	}
	gotStatus, _ := readTaskStatusAndPath(t, store, "T903")
	if gotStatus != "in-progress" {
		t.Errorf("status = %q, want in-progress (unchanged)", gotStatus)
	}
}

// TestT687_TransitionTaskAtomic_EmptyPath_KeepsFilePath verifies that
// when fileOp returns an empty path, the file_path column is untouched
// while only the status is updated.
func TestT687_TransitionTaskAtomic_EmptyPath_KeepsFilePath(t *testing.T) {
	store := sqlite.New(openTestDB(t))
	seedTransitionTask(t, store, "T904", "todo", "works/sprints/active/sprint-81/tasks/T904.md")

	err := store.TransitionTaskAtomic("T904", "todo", "in-progress",
		func() (string, error) {
			return "", nil
		},
	)
	if err != nil {
		t.Fatalf("TransitionTaskAtomic: %v", err)
	}

	gotStatus, gotPath := readTaskStatusAndPath(t, store, "T904")
	if gotStatus != "in-progress" {
		t.Errorf("status = %q, want in-progress", gotStatus)
	}
	if gotPath != "works/sprints/active/sprint-81/tasks/T904.md" {
		t.Errorf("file_path = %q, want original (Sprint-assigned Task keeps its path)", gotPath)
	}
}

// TestT687_TransitionTaskAtomic_NilFileOp verifies the nil-fileOp guard.
func TestT687_TransitionTaskAtomic_NilFileOp(t *testing.T) {
	store := sqlite.New(openTestDB(t))
	seedTransitionTask(t, store, "T905", "todo", "works/tasks/T905.md")

	err := store.TransitionTaskAtomic("T905", "todo", "in-progress", nil)
	if err == nil {
		t.Fatal("nil fileOp did not return an error")
	}
	gotStatus, _ := readTaskStatusAndPath(t, store, "T905")
	if gotStatus != "todo" {
		t.Errorf("status = %q, want todo (must not change when nil-guard fires)", gotStatus)
	}
}
