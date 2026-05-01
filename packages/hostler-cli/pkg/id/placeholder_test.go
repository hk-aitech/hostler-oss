package id

import (
	"os"
	"path/filepath"
	"testing"
)

// ScanPlaceholderTaskIDs regression tests.
// Body scan retired — only filename-based max is extracted.

func TestScan_EmptyDirectory(t *testing.T) {
	tmp := t.TempDir()
	max, err := ScanPlaceholderTaskIDs(tmp)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if max != 0 {
		t.Errorf("empty directory max=%d, want 0", max)
	}
}

func TestScan_FilenameMax(t *testing.T) {
	tmp := t.TempDir()
	tasksDir := filepath.Join(tmp, "works", "tasks")
	if err := os.MkdirAll(tasksDir, 0o755); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"T050-x.md", "T200-y.md", "T777-z.md"} {
		if err := os.WriteFile(filepath.Join(tasksDir, name), nil, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	max, err := ScanPlaceholderTaskIDs(tmp)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if max != 777 {
		t.Errorf("max=%d, want 777", max)
	}
}

func TestScan_CompletedFilenameExcluded(t *testing.T) {
	tmp := t.TempDir()
	completedTasks := filepath.Join(tmp, "works", "sprints", "completed", "sprint-14", "tasks")
	activeTasks := filepath.Join(tmp, "works", "sprints", "active", "sprint-15", "tasks")
	if err := os.MkdirAll(completedTasks, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(activeTasks, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(completedTasks, "T10001-x.md"), nil, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(activeTasks, "T500-y.md"), nil, 0o644); err != nil {
		t.Fatal(err)
	}
	max, err := ScanPlaceholderTaskIDs(tmp)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if max != 500 {
		t.Errorf("max=%d, want 500 (T10001 in completed must be excluded)", max)
	}
}

func TestScan_WorksMissing(t *testing.T) {
	tmp := t.TempDir()
	max, err := ScanPlaceholderTaskIDs(tmp)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if max != 0 {
		t.Errorf("works/ missing max=%d, want 0", max)
	}
}

func TestParseLeadingTaskID(t *testing.T) {
	cases := []struct {
		name string
		want int64
	}{
		{"T219-foo.md", 219},
		{"T9999-large.md", 9999},
		{"T1.md", 1},
		{"non-task.md", 0},
		{"", 0},
	}
	for _, c := range cases {
		got := parseLeadingTaskID(c.name)
		if got != c.want {
			t.Errorf("parseLeadingTaskID(%q) = %d, want %d", c.name, got, c.want)
		}
	}
}

// Guards the body-scan-retired behaviour. Even when SPRINT.md mentions
// T99999 in the body, max must be 0.
func TestScan_BodyIgnored_Regression(t *testing.T) {
	tmp := t.TempDir()
	sprintDir := filepath.Join(tmp, "works", "sprints", "active", "sprint-NN")
	if err := os.MkdirAll(sprintDir, 0o755); err != nil {
		t.Fatal(err)
	}
	// SPRINT.md body cites T99999 — must be excluded from the scan.
	if err := os.WriteFile(filepath.Join(sprintDir, "SPRINT.md"),
		[]byte("| T99999 | KPT retro citation |\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	max, err := ScanPlaceholderTaskIDs(tmp)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if max != 0 {
		t.Errorf("T99999 citation in body picked up as max=%d — body-scan retirement regressed", max)
	}
}
