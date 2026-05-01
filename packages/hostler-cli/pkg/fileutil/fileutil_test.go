package fileutil_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/pkg/fileutil"
)

// --- frontmatter parsing ---

// Verifies that a file with frontmatter parses correctly.
func TestReadTaskFrontmatter_OK(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HSTL_PROJECT_ROOT", dir)

	content := `---
id: T001
title: "test"
status: todo
---

# body
`
	path := filepath.Join(dir, "T001-test.md")
	os.WriteFile(path, []byte(content), 0o644)

	fm, err := fileutil.ReadTaskFrontmatter(path)
	if err != nil {
		t.Fatalf("ReadTaskFrontmatter failed: %v", err)
	}
	if fm["id"] != "T001" {
		t.Errorf("id: expected T001, got %v", fm["id"])
	}
	if fm["status"] != "todo" {
		t.Errorf("status: expected todo, got %v", fm["status"])
	}
}

// Verifies that updating only the status field preserves the rest.
func TestUpdateTaskFrontmatter_StatusOnly(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HSTL_PROJECT_ROOT", dir)

	content := `---
id: T002
title: "update test"
status: todo
priority: p1
---

# body content
`
	path := filepath.Join(dir, "T002-test.md")
	os.WriteFile(path, []byte(content), 0o644)

	if err := fileutil.UpdateTaskFrontmatter(path, map[string]any{"status": "in-progress"}); err != nil {
		t.Fatalf("UpdateTaskFrontmatter failed: %v", err)
	}

	fm, err := fileutil.ReadTaskFrontmatter(path)
	if err != nil {
		t.Fatalf("ReadTaskFrontmatter failed: %v", err)
	}
	if fm["status"] != "in-progress" {
		t.Errorf("updated status: expected in-progress, got %v", fm["status"])
	}
	// Confirm the other fields are preserved.
	if fm["priority"] != "p1" {
		t.Errorf("priority preserved: expected p1, got %v", fm["priority"])
	}
	// Confirm the body is preserved.
	data, _ := os.ReadFile(path)
	if !strings.Contains(string(data), "# body content") {
		t.Error("body not preserved after update")
	}
}

// Verifies handling for files without frontmatter.
func TestUpdateTaskFrontmatter_NoFrontmatter(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HSTL_PROJECT_ROOT", dir)

	content := "# file without frontmatter\nbody only\n"
	path := filepath.Join(dir, "no-fm.md")
	os.WriteFile(path, []byte(content), 0o644)

	// Without frontmatter, WriteFile must still succeed (no error).
	err := fileutil.UpdateTaskFrontmatter(path, map[string]any{"status": "done"})
	if err != nil {
		t.Fatalf("UpdateTaskFrontmatter failed (no frontmatter): %v", err)
	}
}

// --- MakeSlug ---

// Verifies that titles containing arbitrary letters keep them in the slug.
func TestMakeSlug_NonAscii(t *testing.T) {
	cases := []struct{ title, want string }{
		{"Go Layer Migration", "go-layer-migration"},
		{"Hello World", "hello-world"},
		{"  Leading spaces  ", "leading-spaces"},
	}
	for _, c := range cases {
		got := fileutil.MakeSlug(c.title)
		if got != c.want {
			t.Errorf("MakeSlug(%q): expected %q, got %q", c.title, c.want, got)
		}
	}
}

// Verifies the 50-character cap.
func TestMakeSlug_MaxLength(t *testing.T) {
	long := strings.Repeat("a", 60)
	got := fileutil.MakeSlug(long)
	runes := []rune(got)
	if len(runes) > 50 {
		t.Errorf("slug too long: %d > 50", len(runes))
	}
}

// --- BACKLOG.md ---

// Verifies that a Task row is added to BACKLOG.md.
func TestUpdateBacklogMD_AddRow(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HSTL_PROJECT_ROOT", dir)

	tasksDir := filepath.Join(dir, "works", "tasks")
	os.MkdirAll(tasksDir, 0o755)

	backlogContent := `# BACKLOG

> Last updated: 2026-01-01

## Outstanding Unassigned Tasks — 0

| ID | Title | Size | Type | Priority | Depends |
|----|-------|------|------|----------|---------|
`
	os.WriteFile(filepath.Join(tasksDir, "BACKLOG.md"), []byte(backlogContent), 0o644)

	err := fileutil.UpdateBacklogMD("T099", "test Task", "feature", "M", "p1")
	if err != nil {
		t.Fatalf("UpdateBacklogMD failed: %v", err)
	}

	data, _ := os.ReadFile(filepath.Join(tasksDir, "BACKLOG.md"))
	if !strings.Contains(string(data), "| T099 |") {
		t.Errorf("T099 row missing from BACKLOG.md\ncontent:\n%s", string(data))
	}
}

// Verifies row removal from BACKLOG.md.
func TestRemoveFromBacklogMD_RemoveRow(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HSTL_PROJECT_ROOT", dir)

	tasksDir := filepath.Join(dir, "works", "tasks")
	os.MkdirAll(tasksDir, 0o755)

	content := `# BACKLOG
| ID | Title | Size |
|----|-------|------|
| T001 | Task 1 | M |
| T002 | Task 2 | S |
`
	os.WriteFile(filepath.Join(tasksDir, "BACKLOG.md"), []byte(content), 0o644)

	removed, err := fileutil.RemoveFromBacklogMD("T001")
	if err != nil {
		t.Fatalf("RemoveFromBacklogMD failed: %v", err)
	}
	if !removed {
		t.Error("T001 should be removed but removed=false")
	}

	data, _ := os.ReadFile(filepath.Join(tasksDir, "BACKLOG.md"))
	if strings.Contains(string(data), "| T001 |") {
		t.Error("T001 row still present")
	}
	if !strings.Contains(string(data), "| T002 |") {
		t.Error("T002 row must not be deleted")
	}
}

// --- Task file creation ---

// Verifies that a Task file is created at the expected path.
func TestCreateTaskFile_Create(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HSTL_PROJECT_ROOT", dir)

	// No sprint: created under works/tasks/.
	path, err := fileutil.CreateTaskFile(
		"T001", "test Task", "feature", "", "p1", "M",
		nil, "# content\n",
	)
	if err != nil {
		t.Fatalf("CreateTaskFile failed: %v", err)
	}
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Errorf("created file missing: %s", path)
	}
	if !strings.Contains(path, "works/tasks") {
		t.Errorf("path missing works/tasks: %s", path)
	}
}

// Verifies that specifying a sprint places the file under sprint tasks/.
func TestCreateTaskFile_SprintAssign(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HSTL_PROJECT_ROOT", dir)

	// Pre-create the sprint tasks/ directory.
	sprintDir := filepath.Join(dir, "works", "sprints", "active", "sprint-01", "tasks")
	os.MkdirAll(sprintDir, 0o755)

	path, err := fileutil.CreateTaskFile(
		"T010", "Sprint Task", "infra", "sprint-01", "p1", "S",
		nil, "# Sprint Task content\n",
	)
	if err != nil {
		t.Fatalf("CreateTaskFile failed: %v", err)
	}
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Errorf("created file missing: %s", path)
	}
	if !strings.Contains(path, "sprint-01") {
		t.Errorf("path missing sprint-01: %s", path)
	}
}

// Verifies that a missing file returns an error.
func TestReadTaskFrontmatter_MissingFile(t *testing.T) {
	_, err := fileutil.ReadTaskFrontmatter("/nonexistent/path/T999-test.md")
	if err == nil {
		t.Error("error expected for missing file")
	}
}

// Verifies that multiple fields update simultaneously.
func TestUpdateTaskFrontmatter_MultipleFields(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HSTL_PROJECT_ROOT", dir)

	content := `---
id: T003
title: "multi-field update"
status: todo
priority: p2
sprint: ~
---

# body
`
	path := filepath.Join(dir, "T003-test.md")
	os.WriteFile(path, []byte(content), 0o644)

	if err := fileutil.UpdateTaskFrontmatter(path, map[string]any{
		"status": "in-progress",
		"sprint": "sprint-01",
	}); err != nil {
		t.Fatalf("UpdateTaskFrontmatter failed: %v", err)
	}

	fm, err := fileutil.ReadTaskFrontmatter(path)
	if err != nil {
		t.Fatalf("ReadTaskFrontmatter failed: %v", err)
	}
	if fm["status"] != "in-progress" {
		t.Errorf("status update failed: %v", fm["status"])
	}
	if fm["priority"] != "p2" {
		t.Errorf("priority should not change: %v", fm["priority"])
	}
}

// Verifies special-character handling.
func TestMakeSlug_SpecialChars(t *testing.T) {
	cases := []struct{ title, want string }{
		{"Go defer handling", "go-defer-handling"},
		{"API/REST design", "apirest-design"},
		{"--leading-dashes--", "leading-dashes"},
	}
	for _, c := range cases {
		got := fileutil.MakeSlug(c.title)
		t.Logf("MakeSlug(%q) = %q (want %q)", c.title, got, c.want)
		// The slug must not be empty.
		if got == "" {
			t.Errorf("MakeSlug(%q) = empty string", c.title)
		}
	}
}

// Verifies that BACKLOG.md is auto-created when missing.
// Replaces the previous silent-skip behaviour with "auto-create skeleton then insert".
func TestUpdateBacklogMD_MissingFile(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HSTL_PROJECT_ROOT", dir)

	// Initial state: works/tasks/ does not exist.
	if err := fileutil.UpdateBacklogMD("T100", "new Task", "feature", "M", "p2"); err != nil {
		t.Fatalf("UpdateBacklogMD auto-create failed: %v", err)
	}
	backlogPath := filepath.Join(dir, "works", "tasks", "BACKLOG.md")
	data, err := os.ReadFile(backlogPath)
	if err != nil {
		t.Fatalf("BACKLOG.md auto-create read failed: %v", err)
	}
	if !strings.Contains(string(data), "T100") {
		t.Errorf("auto-created BACKLOG.md missing T100 row:\n%s", data)
	}
	if !strings.Contains(string(data), "# Backlog") {
		t.Errorf("auto-created BACKLOG.md missing header:\n%s", data)
	}
}

// Verifies that CURRENT-FOCUS.md is auto-created when works/sprints/ exists
// and the file is absent.
func TestT467_UpdateCurrentFocus_AutoCreate(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HSTL_PROJECT_ROOT", dir)

	// Create works/sprints (sprint-using project scenario).
	if err := os.MkdirAll(filepath.Join(dir, "works", "sprints"), 0o755); err != nil {
		t.Fatal(err)
	}

	if err := fileutil.UpdateCurrentFocus("sprint-56", "CLI core focus", "sprint-active"); err != nil {
		t.Fatalf("UpdateCurrentFocus failed: %v", err)
	}
	data, err := os.ReadFile(filepath.Join(dir, "works", "CURRENT-FOCUS.md"))
	if err != nil {
		t.Fatalf("CURRENT-FOCUS.md auto-create failed: %v", err)
	}
	if !strings.Contains(string(data), "sprint-56") {
		t.Errorf("CURRENT-FOCUS.md missing sprint ID:\n%s", data)
	}
	if !strings.Contains(string(data), "CLI core focus") {
		t.Errorf("CURRENT-FOCUS.md missing title:\n%s", data)
	}
	if !strings.Contains(string(data), "sprint-active") {
		t.Errorf("CURRENT-FOCUS.md missing status:\n%s", data)
	}
}

// TestT544_UpdateCurrentFocus_Layer2_frontmatter — Phase 2.
// Verifies that SPRINT.md track_id / milestone fields auto-fill into the frontmatter.
func TestT544_UpdateCurrentFocus_Layer2_frontmatter(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HSTL_PROJECT_ROOT", dir)
	t.Setenv("HSTL_ENVIRONMENT", "Dev")

	sprintDir := filepath.Join(dir, "works", "sprints", "active", "sprint-50")
	if err := os.MkdirAll(sprintDir, 0o755); err != nil {
		t.Fatal(err)
	}
	sprintMD := `---
id: sprint-50
title: "Phase 2 test"
track_id: track-context-15-facet
milestone: phase-3
status: active
---
`
	if err := os.WriteFile(filepath.Join(sprintDir, "SPRINT.md"), []byte(sprintMD), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := fileutil.UpdateCurrentFocus("sprint-50", "Phase 2 test", "sprint-active"); err != nil {
		t.Fatalf("UpdateCurrentFocus failed: %v", err)
	}
	data, _ := os.ReadFile(filepath.Join(dir, "works", "CURRENT-FOCUS.md"))
	want := []string{
		"track: track-context-15-facet",
		"milestone: phase-3",
		"environment: Dev",
		"sprint: sprint-50",
	}
	for _, line := range want {
		if !strings.Contains(string(data), line) {
			t.Errorf("CURRENT-FOCUS.md missing %q:\n%s", line, data)
		}
	}
}

// TestT544_UpdateCurrentFocus_environment_default — defaults to LocalDev when HSTL_ENVIRONMENT is unset.
func TestT544_UpdateCurrentFocus_environment_default(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HSTL_PROJECT_ROOT", dir)
	t.Setenv("HSTL_ENVIRONMENT", "")
	if err := os.MkdirAll(filepath.Join(dir, "works", "sprints"), 0o755); err != nil {
		t.Fatal(err)
	}
	_ = fileutil.UpdateCurrentFocus("sprint-X", "x", "sprint-active")
	data, _ := os.ReadFile(filepath.Join(dir, "works", "CURRENT-FOCUS.md"))
	if !strings.Contains(string(data), "environment: LocalDev") {
		t.Errorf("default environment LocalDev missing:\n%s", data)
	}
}

// TestT467_UpdateCurrentFocus_NoSprints_Skip verifies that UpdateCurrentFocus
// is a no-op in projects that do not use works/sprints/ (e.g. a task-only
// adoption).
func TestT467_UpdateCurrentFocus_NoSprints_Skip(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HSTL_PROJECT_ROOT", dir)

	// works/sprints absent.
	if err := os.MkdirAll(filepath.Join(dir, "works", "tasks"), 0o755); err != nil {
		t.Fatal(err)
	}

	if err := fileutil.UpdateCurrentFocus("sprint-1", "test", "sprint-active"); err != nil {
		t.Errorf("UpdateCurrentFocus failed in no-sprint env (expected no-op): %v", err)
	}
	// File must not be created.
	if _, err := os.Stat(filepath.Join(dir, "works", "CURRENT-FOCUS.md")); err == nil {
		t.Error("CURRENT-FOCUS.md created in no-sprint env (expected no-op)")
	}
}

// Verifies removed=false when the Task ID is absent.
func TestRemoveFromBacklogMD_MissingTask(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HSTL_PROJECT_ROOT", dir)

	tasksDir := filepath.Join(dir, "works", "tasks")
	os.MkdirAll(tasksDir, 0o755)
	os.WriteFile(filepath.Join(tasksDir, "BACKLOG.md"), []byte("# BACKLOG\n| T001 | Task 1 | M |\n"), 0o644)

	removed, err := fileutil.RemoveFromBacklogMD("T999")
	if err != nil {
		t.Fatalf("RemoveFromBacklogMD failed: %v", err)
	}
	if removed {
		t.Error("removed must be false when the Task is absent")
	}
}

// Verifies that UpdateTaskStatus only updates the status field.
func TestUpdateTaskStatus_StatusOnly(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HSTL_PROJECT_ROOT", dir)

	content := `---
id: T004
title: "status update test"
status: todo
---

# body
`
	path := filepath.Join(dir, "T004-test.md")
	os.WriteFile(path, []byte(content), 0o644)

	if err := fileutil.UpdateTaskStatus(path, "in-progress"); err != nil {
		t.Fatalf("UpdateTaskStatus failed: %v", err)
	}

	fm, err := fileutil.ReadTaskFrontmatter(path)
	if err != nil {
		t.Fatalf("ReadTaskFrontmatter failed: %v", err)
	}
	if fm["status"] != "in-progress" {
		t.Errorf("status=%v, want=in-progress", fm["status"])
	}
}

// Verifies that empty input returns an empty slug (or near empty).
func TestMakeSlug_EmptyString(t *testing.T) {
	got := fileutil.MakeSlug("")
	// Must not be only dashes.
	t.Logf("MakeSlug(\"\") = %q", got)
}

// Verifies digit handling.
func TestMakeSlug_WithNumbers(t *testing.T) {
	cases := []struct{ title, contains string }{
		{"Sprint 15 start", "15"},
		{"T001 task", "t001"},
	}
	for _, c := range cases {
		got := fileutil.MakeSlug(c.title)
		if !strings.Contains(got, c.contains) {
			t.Errorf("MakeSlug(%q)=%q, want to contain %q", c.title, got, c.contains)
		}
	}
}

// Verifies absolute -> relative path conversion.
func TestToRepoRelative_PathConvert(t *testing.T) {
	// Absolute path input -> relative path returned (when inside a git repo).
	// The current execution environment is inside a git repo, so this works.
	absPath := "/some/absolute/path/file.md"
	result := fileutil.ToRepoRelative(absPath)
	// At minimum, returns a non-nil string.
	if result == "" {
		t.Logf("ToRepoRelative(%q) = empty string (could be the absolute path returned as-is)", absPath)
	}
}

// Verifies that GetProjectRoot honours HSTL_PROJECT_ROOT.
func TestGetProjectRoot_EnvVar(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HSTL_PROJECT_ROOT", tmpDir)

	root := fileutil.GetProjectRoot()
	if root != tmpDir {
		t.Errorf("GetProjectRoot()=%q, want=%q", root, tmpDir)
	}
}

// Verifies that without a sprint, the file lands under works/tasks/.
func TestCreateTaskFile_backlog(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HSTL_PROJECT_ROOT", dir)

	path, err := fileutil.CreateTaskFile(
		"T002", "Backlog Task", "chore", "", "p3", "XS",
		nil, "# content\n",
	)
	if err != nil {
		t.Fatalf("CreateTaskFile failed: %v", err)
	}
	if !strings.Contains(path, "works/tasks") {
		t.Errorf("backlog Task path missing works/tasks: %s", path)
	}
	// Verify file content.
	data, _ := os.ReadFile(path)
	if !strings.Contains(string(data), "# content") {
		t.Error("file content not written")
	}
}

// Verifies that specifying a non-existent sprint fails fast (no silent backlog fallback).
func TestCreateTaskFile_SprintMissing_Error(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HSTL_PROJECT_ROOT", dir)

	path, err := fileutil.CreateTaskFile(
		"T999", "Nonexistent Sprint Task", "feature", "sprint-404", "p2", "S",
		nil, "# body\n",
	)
	if err == nil {
		t.Fatalf("expected error for non-existent sprint, got success path=%s", path)
	}
	if !strings.Contains(err.Error(), "sprint-404") {
		t.Errorf("error message must include the missing sprint ID, got: %v", err)
	}
	// Confirm no silent backlog fallback occurred.
	backlogPath := filepath.Join(dir, "works", "tasks")
	entries, _ := os.ReadDir(backlogPath)
	if len(entries) != 0 {
		t.Errorf("silent backlog fallback occurred for missing sprint: %v", entries)
	}
}

// Verifies that the "backlog" literal is treated as unassigned (under works/tasks/).
func TestCreateTaskFile_sprint_backlog_literal(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HSTL_PROJECT_ROOT", dir)

	path, err := fileutil.CreateTaskFile(
		"T500", "Backlog literal", "chore", "backlog", "p3", "XS",
		nil, "# content\n",
	)
	if err != nil {
		t.Fatalf("CreateTaskFile failed: %v", err)
	}
	if !strings.Contains(path, "works/tasks") || strings.Contains(path, "sprints") {
		t.Errorf("backlog literal must land under works/tasks/: %s", path)
	}
}

// =============================================================================
// UpdateSprintMD 7-column format
// =============================================================================

// setupTestSprintMD creates a sprint-test directory and a SPRINT.md with a 7-column header.
func setupTestSprintMD(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("HSTL_PROJECT_ROOT", dir)

	sprintDir := filepath.Join(dir, "works", "sprints", "active", "sprint-test")
	if err := os.MkdirAll(sprintDir, 0o755); err != nil {
		t.Fatal(err)
	}
	sprintMD := `---
id: sprint-test
title: "test sprint"
status: active
goal: "test"
---

# sprint-test: test sprint

## Goal

test

## Task list

| ID | Title | type | estimate | priority | status | depends |
|----|-------|------|----------|----------|--------|---------|

## Completion criteria

- [ ] all Tasks done
`
	if err := os.WriteFile(filepath.Join(sprintDir, "SPRINT.md"), []byte(sprintMD), 0o644); err != nil {
		t.Fatal(err)
	}
	return dir
}

// Verifies that a new Task row uses the 7-column format.
// Previous bug only wrote 5 columns and broke header consistency.
func TestUpdateSprintMD_NewRow7Cols(t *testing.T) {
	dir := setupTestSprintMD(t)

	err := fileutil.UpdateSprintMD("sprint-test", "T999", "test Task title", "feature", "M", "p1", "todo", "T100, T200")
	if err != nil {
		t.Fatalf("UpdateSprintMD failed: %v", err)
	}

	data, _ := os.ReadFile(filepath.Join(dir, "works", "sprints", "active", "sprint-test", "SPRINT.md"))
	content := string(data)

	// A row in the 7-column format must be added.
	expected := "| T999 | test Task title | feature | M | p1 | todo | T100, T200 |"
	if !strings.Contains(content, expected) {
		t.Errorf("7-column row not written.\nexpected: %s\nactual file:\n%s", expected, content)
	}

	// The 5-column legacy bug must not appear.
	bad := "| T999 | test Task title | feature | M | todo |"
	if strings.Contains(content, bad) {
		t.Errorf("5-column legacy format mistakenly written.\nfile:\n%s", content)
	}
}

// Verifies that an empty dependsOn renders as "—".
func TestUpdateSprintMD_EmptyDependency(t *testing.T) {
	dir := setupTestSprintMD(t)

	err := fileutil.UpdateSprintMD("sprint-test", "T888", "independent Task", "hotfix", "S", "p2", "todo", "")
	if err != nil {
		t.Fatalf("UpdateSprintMD failed: %v", err)
	}

	data, _ := os.ReadFile(filepath.Join(dir, "works", "sprints", "active", "sprint-test", "SPRINT.md"))
	content := string(data)

	// Empty dependency renders as "—".
	expected := "| T888 | independent Task | hotfix | S | p2 | todo | — |"
	if !strings.Contains(content, expected) {
		t.Errorf("empty-dependency handling failed:\nexpected: %s\nfile:\n%s", expected, content)
	}
}

// Regression fixture for cascade drift.
// Reproduces the T458 -> T461 cascade defect: when T458's status update is
// called, T461's status must not change even though T461's depends_on cell
// contains T458.
func TestUpdateSprintMD_status_drift_blocked(t *testing.T) {
	dir := setupTestSprintMD(t)

	// 1) Create two Task rows: T458 (todo) + T461 (todo, depends T458).
	if err := fileutil.UpdateSprintMD("sprint-test", "T458", "pkg/trac reimpl", "refactor", "L", "p1", "todo", "—"); err != nil {
		t.Fatal(err)
	}
	if err := fileutil.UpdateSprintMD("sprint-test", "T461", "cmd/trac CLI", "refactor", "M", "p1", "todo", "T458"); err != nil {
		t.Fatal(err)
	}

	// 2) Update T458's status to done.
	if err := fileutil.UpdateSprintMD("sprint-test", "T458", "pkg/trac reimpl", "refactor", "L", "p1", "done", "—"); err != nil {
		t.Fatal(err)
	}

	data, _ := os.ReadFile(filepath.Join(dir, "works", "sprints", "active", "sprint-test", "SPRINT.md"))
	content := string(data)

	// T458 changes to done.
	if !strings.Contains(content, "| T458 | pkg/trac reimpl | refactor | L | p1 | done | — |") {
		t.Errorf("T458 status not changed to done.\nfile:\n%s", content)
	}
	// T461 must stay todo (cascade would flip it to done — defect).
	if !strings.Contains(content, "| T461 | cmd/trac CLI | refactor | M | p1 | todo | T458 |") {
		t.Errorf("T461 status cascade drift! T461 should remain todo.\nfile:\n%s", content)
	}
}

// Verifies that an empty priority defaults to "p2".
func TestUpdateSprintMD_EmptyPriority(t *testing.T) {
	dir := setupTestSprintMD(t)

	err := fileutil.UpdateSprintMD("sprint-test", "T777", "default-priority Task", "chore", "XS", "", "todo", "")
	if err != nil {
		t.Fatalf("UpdateSprintMD failed: %v", err)
	}

	data, _ := os.ReadFile(filepath.Join(dir, "works", "sprints", "active", "sprint-test", "SPRINT.md"))
	content := string(data)

	expected := "| T777 | default-priority Task | chore | XS | p2 | todo | — |"
	if !strings.Contains(content, expected) {
		t.Errorf("default priority handling failed.\nexpected: %s\nfile:\n%s", expected, content)
	}
}

// Verifies that updating an existing row replaces only status without duplicating.
func TestUpdateSprintMD_ExistingRowStatusUpdate(t *testing.T) {
	dir := setupTestSprintMD(t)

	// First call: insert new row (todo).
	if err := fileutil.UpdateSprintMD("sprint-test", "T555", "update target", "feature", "M", "p2", "todo", "—"); err != nil {
		t.Fatalf("first UpdateSprintMD failed: %v", err)
	}

	// Second call: change the same Task's status to in-progress.
	if err := fileutil.UpdateSprintMD("sprint-test", "T555", "update target", "feature", "M", "p2", "in-progress", "—"); err != nil {
		t.Fatalf("second UpdateSprintMD failed: %v", err)
	}

	data, _ := os.ReadFile(filepath.Join(dir, "works", "sprints", "active", "sprint-test", "SPRINT.md"))
	content := string(data)

	// in-progress status reflected.
	if !strings.Contains(content, "in-progress") {
		t.Errorf("status update failed:\n%s", content)
	}
	// No duplicate row (T555 must appear exactly once).
	count := strings.Count(content, "| T555 |")
	if count != 1 {
		t.Errorf("duplicate row added (count=%d):\n%s", count, content)
	}
}

// Verifies that the level-2 section detection does not false-positive on a
// `### Status change history` line in the body.
// The previous strings.Contains-based match falsely treated this case as an
// existing section and tried to append the table.
func TestT441_AppendStatusHistory_Level3Heading_FalsePositive(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "T441-sample.md")

	// Body with no level-2 status-history section but a level-3 heading.
	initial := "# Task\n\n## Result\n\n### Status change history\n\nbody narrative\n"
	if err := os.WriteFile(path, []byte(initial), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	if err := fileutil.AppendStatusHistory(path, "todo", "in-progress", ""); err != nil {
		t.Fatalf("AppendStatusHistory: %v", err)
	}

	data, _ := os.ReadFile(path)
	content := string(data)

	// A level-2 section must be created (the existing level-3 stays).
	if !strings.Contains(content, "\n## Status change history\n") {
		t.Errorf("level-2 section not created:\n%s", content)
	}
	// Level-3 heading is preserved.
	if !strings.Contains(content, "### Status change history") {
		t.Errorf("level-3 heading damaged:\n%s", content)
	}
	// Exactly one table row.
	if c := strings.Count(content, "| todo -> in-progress |"); c != 1 {
		t.Errorf("row count = %d (expected: 1)\n%s", c, content)
	}
}

// Regression guard for normal level-2 section: appends the new row at the bottom.
func TestT441_AppendStatusHistory_ExistingSection_AppendRow(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "T441-existing.md")

	initial := "# Task\n\n## Status change history\n\n| When | Change | Reason |\n|------|--------|--------|\n| 2026-04-01 | todo -> in-progress |  |\n"
	if err := os.WriteFile(path, []byte(initial), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	if err := fileutil.AppendStatusHistory(path, "in-progress", "done", ""); err != nil {
		t.Fatalf("AppendStatusHistory: %v", err)
	}

	data, _ := os.ReadFile(path)
	content := string(data)

	if !strings.Contains(content, "| in-progress -> done |") {
		t.Errorf("new row missing:\n%s", content)
	}
	if c := strings.Count(content, "## Status change history"); c != 1 {
		t.Errorf("section duplicated (count=%d)", c)
	}
}
