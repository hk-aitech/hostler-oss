package sqlite

import (
	"database/sql"
	"os"
	"path/filepath"
	"testing"

	_ "modernc.org/sqlite"

	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/internal/ports"
)

// UpdateTaskPathsForSprint regression test.
//
// On sprint complete, `file_path LIKE 'active/sprint-NN/%'` did not
// match the actual DB value `works/sprints/active/sprint-NN/...` so
// every UPDATE affected zero rows (downstream produced four
// db_file_path_mismatch reports). This regression test guards the fix.

func setupT233DB(t *testing.T) *SQLiteStore {
	t.Helper()
	tmp := t.TempDir()
	dbPath := filepath.Join(tmp, "test.db")
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatal(err)
	}
	// schema initialisation
	schema := `
CREATE TABLE IF NOT EXISTS tasks (
    task_id TEXT PRIMARY KEY,
    title TEXT,
    type TEXT,
    sprint TEXT,
    status TEXT,
    priority TEXT,
    estimate TEXT,
    file_path TEXT,
    depends_on TEXT,
    created_at TEXT,
    updated_at TEXT,
    work_ticket TEXT
);`
	if _, err := db.Exec(schema); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = db.Close()
		_ = os.RemoveAll(tmp)
	})
	return New(db)
}

func TestT233_UpdateTaskPathsForSprint_ActiveToCompleted(t *testing.T) {
	s := setupT233DB(t)
	_ = s.CreateTask(&ports.TaskRecord{
		TaskID:   "T001",
		Title:    "A",
		Sprint:   "sprint-99",
		Status:   "done",
		FilePath: "works/sprints/active/sprint-99/tasks/T001.md",
	})

	err := s.UpdateTaskPathsForSprint("sprint-99", "active/sprint-99/", "completed/sprint-99/")
	if err != nil {
		t.Fatal(err)
	}

	path, _ := s.GetTaskFilePath("T001")
	want := "works/sprints/completed/sprint-99/tasks/T001.md"
	if path != want {
		t.Errorf("file_path = %q, want %q (before the fix the prefix mismatch left the row untouched)", path, want)
	}
}

func TestT233_UpdateTaskPathsForSprint_BacklogToActive(t *testing.T) {
	s := setupT233DB(t)
	_ = s.CreateTask(&ports.TaskRecord{
		TaskID:   "T002",
		Sprint:   "sprint-99",
		FilePath: "works/sprints/backlog/sprint-99/tasks/T002.md",
	})

	err := s.UpdateTaskPathsForSprint("sprint-99", "backlog/sprint-99/", "active/sprint-99/")
	if err != nil {
		t.Fatal(err)
	}

	path, _ := s.GetTaskFilePath("T002")
	want := "works/sprints/active/sprint-99/tasks/T002.md"
	if path != want {
		t.Errorf("file_path = %q, want %q", path, want)
	}
}

func TestT233_UpdateTaskPathsForSprint_NoMatch_NoOp(t *testing.T) {
	s := setupT233DB(t)
	// A Task in a different sprint must not be affected.
	_ = s.CreateTask(&ports.TaskRecord{
		TaskID:   "T003",
		Sprint:   "sprint-other",
		FilePath: "works/sprints/active/sprint-other/tasks/T003.md",
	})

	err := s.UpdateTaskPathsForSprint("sprint-99", "active/sprint-99/", "completed/sprint-99/")
	if err != nil {
		t.Fatal(err)
	}

	path, _ := s.GetTaskFilePath("T003")
	want := "works/sprints/active/sprint-other/tasks/T003.md"
	if path != want {
		t.Errorf("a Task from a different sprint was affected: %q (want %q)", path, want)
	}
}
