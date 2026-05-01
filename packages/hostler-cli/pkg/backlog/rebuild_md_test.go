package backlog

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/pkg/db"
)

// TestT485_RebuildMD_BasicFlow verifies that BACKLOG.md is regenerated
// with all rows when there are unassigned Tasks in DB.
func TestT485_RebuildMD_BasicFlow(t *testing.T) {
	dir, cleanup := setupTestDir(t)
	defer cleanup()

	// seed 3 unassigned Tasks + 1 Sprint-assigned Task
	database := db.GetDB()
	for _, task := range []struct {
		id, title, typ, pri, est, sprint, status string
	}{
		{"T001", "first task", "feature", "p1", "S", "", "todo"},
		{"T002", "second task", "bugfix", "p2", "M", "", "todo"},
		{"T003", "third task", "docs", "p3", "XS", "", "todo"},
		{"T004", "assigned", "feature", "p2", "M", "sprint-01", "todo"},
	} {
		var sprintVal interface{}
		if task.sprint != "" {
			sprintVal = task.sprint
		}
		_, err := database.Exec(
			`INSERT INTO tasks (task_id, title, type, priority, estimate, sprint, status, file_path, created_at, updated_at)
			 VALUES (?, ?, ?, ?, ?, ?, ?, '', date('now'), date('now'))`,
			task.id, task.title, task.typ, task.pri, task.est, sprintVal, task.status,
		)
		if err != nil {
			t.Fatalf("seed %s failed: %v", task.id, err)
		}
	}

	result, err := RebuildMD(RebuildMDOptions{})
	if err != nil {
		t.Fatalf("RebuildMD failed: %v", err)
	}

	if result.Count != 3 {
		t.Errorf("count: got %d, want 3 (unassigned only)", result.Count)
	}

	data, err := os.ReadFile(filepath.Join(dir, "works", "tasks", "BACKLOG.md"))
	if err != nil {
		t.Fatalf("BACKLOG.md read failed: %v", err)
	}
	content := string(data)
	for _, expected := range []string{"T001", "T002", "T003", "first task"} {
		if !strings.Contains(content, expected) {
			t.Errorf("BACKLOG.md missing %q", expected)
		}
	}
	if strings.Contains(content, "T004") {
		t.Errorf("Sprint-assigned Task (T004) is included — should be excluded")
	}
}

// TestT485_RebuildMD_PrioritySort verifies p0~p3 sort order.
func TestT485_RebuildMD_PrioritySort(t *testing.T) {
	_, cleanup := setupTestDir(t)
	defer cleanup()

	database := db.GetDB()
	for _, task := range []struct{ id, pri string }{
		{"T010", "p3"},
		{"T011", "p1"},
		{"T012", "p2"},
		{"T013", "p0"},
	} {
		_, err := database.Exec(
			`INSERT INTO tasks (task_id, title, type, priority, estimate, status, file_path, created_at, updated_at)
			 VALUES (?, 'test', 'chore', ?, 'S', 'todo', '', date('now'), date('now'))`,
			task.id, task.pri,
		)
		if err != nil {
			t.Fatal(err)
		}
	}

	result, err := RebuildMD(RebuildMDOptions{})
	if err != nil {
		t.Fatalf("RebuildMD failed: %v", err)
	}

	data, _ := os.ReadFile(result.FilePath)
	content := string(data)
	// T013 (p0) must come before T011 (p1).
	idx13 := strings.Index(content, "T013")
	idx11 := strings.Index(content, "T011")
	idx12 := strings.Index(content, "T012")
	idx10 := strings.Index(content, "T010")
	if !(idx13 < idx11 && idx11 < idx12 && idx12 < idx10) {
		t.Errorf("sort error: expected T013<T011<T012<T010, got T013=%d T011=%d T012=%d T010=%d",
			idx13, idx11, idx12, idx10)
	}
}

// TestT491_RebuildMD_DryRun verifies that BACKLOG.md is not actually
// modified in dry-run mode.
func TestT491_RebuildMD_DryRun(t *testing.T) {
	dir, cleanup := setupTestDir(t)
	defer cleanup()

	// seed an unassigned Task
	database := db.GetDB()
	_, err := database.Exec(
		`INSERT INTO tasks (task_id, title, type, priority, estimate, status, file_path, created_at, updated_at)
		 VALUES ('T100', 'dry-run seed', 'chore', 'p2', 'S', 'todo', '', date('now'), date('now'))`,
	)
	if err != nil {
		t.Fatal(err)
	}

	// pre-create BACKLOG.md (then verify dry-run leaves it as-is)
	backlogDir := filepath.Join(dir, "works", "tasks")
	_ = os.MkdirAll(backlogDir, 0o755)
	backlogPath := filepath.Join(backlogDir, "BACKLOG.md")
	existing := "# existing content\nmust not change\n"
	_ = os.WriteFile(backlogPath, []byte(existing), 0o644)

	// T699: WouldBackup is set only when Backup: true
	result, err := RebuildMD(RebuildMDOptions{DryRun: true, Backup: true})
	if err != nil {
		t.Fatalf("dry-run failed: %v", err)
	}
	if !result.DryRun {
		t.Error("DryRun flag is false")
	}
	if result.Count != 1 {
		t.Errorf("count: got %d, want 1", result.Count)
	}
	if result.WouldBackup == "" {
		t.Error("file exists but WouldBackup is empty (Backup: true)")
	}

	// confirm no actual file change
	data, _ := os.ReadFile(backlogPath)
	if string(data) != existing {
		t.Errorf("dry-run modified BACKLOG.md:\n%s", data)
	}
	// .bak file must not be created either
	if _, err := os.Stat(backlogPath + ".bak"); err == nil {
		t.Error("dry-run created an actual .bak file")
	}
}

// TestT485_RebuildMD_Backup verifies that --backup opt-in produces a .bak
// of the existing file. T699 (Sprint-85): default flipped to false —
// Backup: true must be specified.
func TestT485_RebuildMD_Backup(t *testing.T) {
	dir, cleanup := setupTestDir(t)
	defer cleanup()

	// pre-create BACKLOG.md
	backlogDir := filepath.Join(dir, "works", "tasks")
	_ = os.MkdirAll(backlogDir, 0o755)
	backlogPath := filepath.Join(backlogDir, "BACKLOG.md")
	existing := "# existing content\nold BACKLOG\n"
	_ = os.WriteFile(backlogPath, []byte(existing), 0o644)

	result, err := RebuildMD(RebuildMDOptions{Backup: true})
	if err != nil {
		t.Fatalf("RebuildMD failed: %v", err)
	}
	if result.BackupPath == "" {
		t.Fatal("Backup=true but BackupPath is empty")
	}
	// .bak must contain the original content
	backupData, err := os.ReadFile(result.BackupPath)
	if err != nil {
		t.Fatalf(".bak read failed: %v", err)
	}
	if string(backupData) != existing {
		t.Errorf(".bak content mismatch: got %q, want %q", backupData, existing)
	}
}

// TestT699_RebuildMD_NoBackupByDefault verifies that running without
// --backup does not produce a .bak file (T699 default change).
func TestT699_RebuildMD_NoBackupByDefault(t *testing.T) {
	dir, cleanup := setupTestDir(t)
	defer cleanup()

	backlogDir := filepath.Join(dir, "works", "tasks")
	_ = os.MkdirAll(backlogDir, 0o755)
	backlogPath := filepath.Join(backlogDir, "BACKLOG.md")
	_ = os.WriteFile(backlogPath, []byte("old\n"), 0o644)

	result, err := RebuildMD(RebuildMDOptions{}) // Backup not set (default false)
	if err != nil {
		t.Fatalf("RebuildMD failed: %v", err)
	}
	if result.BackupPath != "" {
		t.Errorf("Backup=false but BackupPath is set: %q", result.BackupPath)
	}
	if _, err := os.Stat(backlogPath + ".bak"); err == nil {
		t.Error("Backup=false yet .bak was created")
	}
}
