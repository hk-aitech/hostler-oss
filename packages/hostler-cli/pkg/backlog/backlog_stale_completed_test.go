// Package backlog — T393 (Sprint-32) unit tests.
//
// Verify that done Tasks moved to works/tasks/completed/ but still
// remaining in BACKLOG.md (drift) are classified as auto-removable
// (`backlog_md_stale_completed`, AutoFixable=true) and that the apply
// path (dryRun=false) actually removes the row.
// Regression guard for ISS-20260422-013.
package backlog

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestT393_StaleCompleted_Detected — Sync(dryRun=true) classifies the
// completed/ + DB done combination as `backlog_md_stale_completed`.
func TestT393_StaleCompleted_Detected(t *testing.T) {
	dir, cleanup := setupTestDir(t)
	defer cleanup()

	// T1858 row remaining in BACKLOG.md
	writeTaskFile(t, filepath.Join(dir, "works", "tasks", "BACKLOG.md"), `# Backlog

## Unassigned Tasks

| T1858 | old work | chore | S | p2 |
`)

	// file lives under works/tasks/completed/
	completedDir := filepath.Join(dir, "works", "tasks", "completed")
	if err := os.MkdirAll(completedDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	writeTaskFile(t, filepath.Join(completedDir, "T1858-old-task.md"), `---
id: T1858
title: old work
type: chore
status: done
created: "2026-04-22"
---

# T1858
`)

	// insert DB row with status=done
	insertDBTask(t, "T1858", "done", "", "works/tasks/completed/T1858-old-task.md")

	result, err := Sync(true)
	if err != nil {
		t.Fatalf("Sync: %v", err)
	}

	var found *Issue
	for i := range result.Issues {
		if result.Issues[i].TaskID == "T1858" && result.Issues[i].Type == "backlog_md_stale_completed" {
			found = &result.Issues[i]
			break
		}
	}
	if found == nil {
		t.Fatalf("backlog_md_stale_completed detection failed: issues=%+v", result.Issues)
	}
	if !found.AutoFixable {
		t.Errorf("expected AutoFixable=true, got false")
	}

	// dryRun=true so BACKLOG.md still contains T1858
	data, _ := os.ReadFile(filepath.Join(dir, "works", "tasks", "BACKLOG.md"))
	if !strings.Contains(string(data), "T1858") {
		t.Error("BACKLOG.md modified under dryRun=true")
	}
}

// TestT393_StaleCompleted_ApplyFixes_RemovesRow — Sync(dryRun=false)
// actually removes the row from BACKLOG.md.
func TestT393_StaleCompleted_ApplyFixes_RemovesRow(t *testing.T) {
	dir, cleanup := setupTestDir(t)
	defer cleanup()

	writeTaskFile(t, filepath.Join(dir, "works", "tasks", "BACKLOG.md"), `# Backlog

## Unassigned Tasks

| T1858 | old work | chore | S | p2 |
| T1859 | must keep | chore | S | p2 |
`)

	completedDir := filepath.Join(dir, "works", "tasks", "completed")
	if err := os.MkdirAll(completedDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	writeTaskFile(t, filepath.Join(completedDir, "T1858-old-task.md"), `---
id: T1858
title: old work
type: chore
status: done
---

# T1858
`)

	// T1858: completed + DB done → stale
	insertDBTask(t, "T1858", "done", "", "works/tasks/completed/T1858-old-task.md")
	// T1859: file is missing in backlog and there is no DB/completed entry
	// either — it goes through the existing missing_file path. (Out of scope
	// for this test; if drift co-occurs only T1858 must be removed.)

	result, err := Sync(false)
	if err != nil {
		t.Fatalf("Sync: %v", err)
	}

	// the Fix list should include backlog_md_row_removed
	var removed bool
	for _, fx := range result.Fixes {
		if fx.Type == "backlog_md_row_removed" && fx.TaskID == "T1858" {
			removed = true
			break
		}
	}
	if !removed {
		t.Fatalf("backlog_md_row_removed fix missing: fixes=%+v", result.Fixes)
	}

	// BACKLOG.md must lose T1858 but keep T1859
	data, _ := os.ReadFile(filepath.Join(dir, "works", "tasks", "BACKLOG.md"))
	body := string(data)
	if strings.Contains(body, "| T1858 |") {
		t.Errorf("T1858 row not removed:\n%s", body)
	}
	if !strings.Contains(body, "| T1859 |") {
		t.Errorf("T1859 row wrongly removed:\n%s", body)
	}
}

// TestT399_SprintCompleted_Detected (Sprint-33) — when a sprint-assigned
// Task has been moved to works/sprints/completed/sprint-NN/tasks/ but
// remains in BACKLOG.md, classify the drift as backlog_md_stale_completed
// automatically.
func TestT399_SprintCompleted_Detected(t *testing.T) {
	dir, cleanup := setupTestDir(t)
	defer cleanup()

	writeTaskFile(t, filepath.Join(dir, "works", "tasks", "BACKLOG.md"), `# Backlog

## Unassigned Tasks

| T1900 | sprint-assigned leftover | feature | M | p2 |
`)

	// place file at sprint-completed location
	sprintDir := filepath.Join(dir, "works", "sprints", "completed", "sprint-28", "tasks")
	if err := os.MkdirAll(sprintDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	writeTaskFile(t, filepath.Join(sprintDir, "T1900-sprint-task.md"), `---
id: T1900
title: sprint-assigned leftover
type: feature
status: done
sprint: sprint-28
created: "2026-04-22"
---

# T1900
`)

	insertDBTask(t, "T1900", "done", "sprint-28", "works/sprints/completed/sprint-28/tasks/T1900-sprint-task.md")

	result, err := Sync(true)
	if err != nil {
		t.Fatalf("Sync: %v", err)
	}

	var found *Issue
	for i := range result.Issues {
		if result.Issues[i].TaskID == "T1900" && result.Issues[i].Type == "backlog_md_stale_completed" {
			found = &result.Issues[i]
			break
		}
	}
	if found == nil {
		t.Fatalf("sprint-completed drift not detected: issues=%+v", result.Issues)
	}
	if !found.AutoFixable {
		t.Errorf("expected AutoFixable=true")
	}
	if !strings.Contains(found.File, "sprints/completed/") {
		t.Errorf("iss.File should be on a sprint-completed path: %s", found.File)
	}

	// dryRun=true so BACKLOG.md is unchanged
	data, _ := os.ReadFile(filepath.Join(dir, "works", "tasks", "BACKLOG.md"))
	if !strings.Contains(string(data), "T1900") {
		t.Error("BACKLOG.md modified under dryRun=true")
	}
}

// TestT399_SprintCompleted_ApplyFixes_RemovesRow (Sprint-33) —
// Sync(dryRun=false) actually removes the BACKLOG.md row for a
// sprint-completed drift.
func TestT399_SprintCompleted_ApplyFixes_RemovesRow(t *testing.T) {
	dir, cleanup := setupTestDir(t)
	defer cleanup()

	writeTaskFile(t, filepath.Join(dir, "works", "tasks", "BACKLOG.md"), `# Backlog

## Unassigned Tasks

| T1901 | sprint-assigned leftover | feature | M | p2 |
| T1902 | must keep | chore | S | p3 |
`)

	sprintDir := filepath.Join(dir, "works", "sprints", "completed", "sprint-30", "tasks")
	if err := os.MkdirAll(sprintDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	writeTaskFile(t, filepath.Join(sprintDir, "T1901-task.md"), `---
id: T1901
title: sprint-assigned leftover
type: feature
status: done
sprint: sprint-30
---

# T1901
`)
	insertDBTask(t, "T1901", "done", "sprint-30", "works/sprints/completed/sprint-30/tasks/T1901-task.md")

	result, err := Sync(false)
	if err != nil {
		t.Fatalf("Sync: %v", err)
	}

	var removed bool
	for _, fx := range result.Fixes {
		if fx.Type == "backlog_md_row_removed" && fx.TaskID == "T1901" {
			removed = true
			break
		}
	}
	if !removed {
		t.Fatalf("T1901 backlog_md_row_removed fix missing: fixes=%+v", result.Fixes)
	}

	data, _ := os.ReadFile(filepath.Join(dir, "works", "tasks", "BACKLOG.md"))
	body := string(data)
	if strings.Contains(body, "| T1901 |") {
		t.Errorf("T1901 row not removed:\n%s", body)
	}
	if !strings.Contains(body, "| T1902 |") {
		t.Errorf("T1902 row wrongly removed:\n%s", body)
	}
}

// TestT399_SprintActive_NotStale (Sprint-33) — files on the sprint-active
// path (in progress) are NOT subject to the stale_completed branch
// (sprintStatus == "active").
func TestT399_SprintActive_NotStale(t *testing.T) {
	dir, cleanup := setupTestDir(t)
	defer cleanup()

	writeTaskFile(t, filepath.Join(dir, "works", "tasks", "BACKLOG.md"), `# Backlog

## Unassigned Tasks

| T1910 | in-progress Task | feature | M | p2 |
`)

	sprintDir := filepath.Join(dir, "works", "sprints", "active", "sprint-33", "tasks")
	if err := os.MkdirAll(sprintDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	writeTaskFile(t, filepath.Join(sprintDir, "T1910-active.md"), `---
id: T1910
title: in-progress Task
type: feature
status: in-progress
sprint: sprint-33
---

# T1910
`)
	insertDBTask(t, "T1910", "in-progress", "sprint-33", "works/sprints/active/sprint-33/tasks/T1910-active.md")

	result, err := Sync(true)
	if err != nil {
		t.Fatalf("Sync: %v", err)
	}
	for _, iss := range result.Issues {
		if iss.TaskID == "T1910" && iss.Type == "backlog_md_stale_completed" {
			t.Errorf("active-sprint Task classified as stale_completed: %+v", iss)
		}
	}
}

// TestT393_StaleCompleted_DBNotDone_NoAutoFix — when DB status is not done,
// keep the existing missing_file path (AutoFixable=false) conservatively.
func TestT393_StaleCompleted_DBNotDone_NoAutoFix(t *testing.T) {
	dir, cleanup := setupTestDir(t)
	defer cleanup()

	writeTaskFile(t, filepath.Join(dir, "works", "tasks", "BACKLOG.md"), `# Backlog

## Unassigned Tasks

| T1860 | ambiguous | chore | S | p2 |
`)

	completedDir := filepath.Join(dir, "works", "tasks", "completed")
	if err := os.MkdirAll(completedDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	writeTaskFile(t, filepath.Join(completedDir, "T1860-task.md"), `---
id: T1860
title: ambiguous
type: chore
status: todo
---

# T1860
`)
	// DB has status=todo
	insertDBTask(t, "T1860", "todo", "", "works/tasks/completed/T1860-task.md")

	result, err := Sync(true)
	if err != nil {
		t.Fatalf("Sync: %v", err)
	}
	for _, iss := range result.Issues {
		if iss.TaskID != "T1860" {
			continue
		}
		if iss.Type == "backlog_md_stale_completed" {
			t.Errorf("DB=todo but classified as stale_completed: %+v", iss)
		}
	}
}
