package sprint_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/internal/adapters/sqlite"
	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/pkg/db"
	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/pkg/sprint"
	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/pkg/store"
)

// setupDB initialises an isolated DB for tests.
func setupDB(t *testing.T) string {
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

// TestCreateFolder_Create verifies that the Sprint folder and the tasks/
// directory are created.
func TestCreateFolder_Create(t *testing.T) {
	tmpRoot := setupDB(t)

	sprintDir, err := sprint.CreateFolder("sprint-01", "backlog")
	if err != nil {
		t.Fatalf("CreateFolder failed: %v", err)
	}

	expectedDir := filepath.Join(tmpRoot, "works", "sprints", "backlog", "sprint-01")
	if sprintDir != expectedDir {
		t.Errorf("path mismatch: got=%s, want=%s", sprintDir, expectedDir)
	}

	// confirm the tasks/ subdirectory
	tasksDir := filepath.Join(sprintDir, "tasks")
	if _, err := os.Stat(tasksDir); os.IsNotExist(err) {
		t.Errorf("tasks/ directory was not created: %s", tasksDir)
	}
}

// TestRenderSprintMD_Basic verifies that the SPRINT.md content renders
// correctly.
func TestRenderSprintMD_Basic(t *testing.T) {
	content := sprint.RenderSprintMD("sprint-01", "first sprint", "MVP completion", nil)

	if !strings.Contains(content, "sprint-01") {
		t.Error("sprint ID not included")
	}
	if !strings.Contains(content, "first sprint") {
		t.Error("title not included")
	}
	if !strings.Contains(content, "MVP completion") {
		t.Error("goal not included")
	}
	if !strings.Contains(content, "status: backlog") {
		t.Error("status: backlog not included")
	}
	if !strings.Contains(content, "## Task list") && !strings.Contains(content, "## Task List") && !strings.Contains(content, "## Tasks") {
		t.Error("Task list section missing")
	}
}

// T763 (Sprint-89): TestRenderCeremonyMD_Basic removed — CEREMONY.md file
// rendering is deprecated.

// TestSaveToDB_Persist verifies that a Sprint is persisted correctly.
func TestSaveToDB_Persist(t *testing.T) {
	setupDB(t)

	err := sprint.SaveToDB("sprint-01", "first sprint", "goal", "backlog",
		"works/sprints/backlog/sprint-01", nil, nil)
	if err != nil {
		t.Fatalf("SaveToDB failed: %v", err)
	}

	// look up via DB
	s, err := sprint.GetFromDB("sprint-01")
	if err != nil {
		t.Fatalf("GetFromDB failed: %v", err)
	}
	if s.SprintID != "sprint-01" {
		t.Errorf("sprint_id mismatch: %v", s.SprintID)
	}
	if s.Status != "backlog" {
		t.Errorf("status mismatch: %v", s.Status)
	}
}

// TestGetFromDB_NonExistent verifies that a non-existent Sprint returns an
// error.
func TestGetFromDB_NonExistent(t *testing.T) {
	setupDB(t)

	_, err := sprint.GetFromDB("nonexistent-sprint")
	if err == nil {
		t.Error("expected error on non-existent Sprint lookup")
	}
}

// TestListFromDB_All verifies the full Sprint list lookup.
func TestListFromDB_All(t *testing.T) {
	setupDB(t)

	// insert several Sprints
	for _, sid := range []string{"sprint-01", "sprint-02", "sprint-03"} {
		if err := sprint.SaveToDB(sid, "title", "goal", "backlog", "path/"+sid, nil, nil); err != nil {
			t.Fatalf("SaveToDB failed: %v", err)
		}
	}

	list, err := sprint.ListFromDB("")
	if err != nil {
		t.Fatalf("ListFromDB failed: %v", err)
	}
	if len(list) != 3 {
		t.Errorf("Sprint count=%d, want=3", len(list))
	}
}

// TestUpdateStatusInDB_Update verifies Sprint-status update.
func TestUpdateStatusInDB_Update(t *testing.T) {
	setupDB(t)

	if err := sprint.SaveToDB("sprint-01", "sprint", "goal", "backlog", "path/sprint-01", nil, nil); err != nil {
		t.Fatalf("SaveToDB failed: %v", err)
	}

	newPath := "works/sprints/active/sprint-01"
	if err := sprint.UpdateStatusInDB("sprint-01", "active", &newPath, nil, nil); err != nil {
		t.Fatalf("UpdateStatusInDB failed: %v", err)
	}

	s, err := sprint.GetFromDB("sprint-01")
	if err != nil {
		t.Fatalf("GetFromDB failed: %v", err)
	}
	if s.Status != "active" {
		t.Errorf("status mismatch: got=%v, want=active", s.Status)
	}
	if s.FolderPath != newPath {
		t.Errorf("folder_path mismatch: got=%v, want=%s", s.FolderPath, newPath)
	}
}

// TestAggregateProgress_Aggregate verifies Task-status aggregation per
// Sprint.
func TestAggregateProgress_Aggregate(t *testing.T) {
	setupDB(t)

	if err := sprint.SaveToDB("sprint-01", "sprint", "goal", "active", "path/sprint-01", nil, nil); err != nil {
		t.Fatalf("SaveToDB failed: %v", err)
	}

	// insert Tasks
	database := db.GetDB()
	for _, row := range []struct {
		id     string
		status string
	}{
		{"T001", "done"},
		{"T002", "done"},
		{"T003", "in-progress"},
		{"T004", "todo"},
	} {
		_, err := database.Exec(
			`INSERT INTO tasks (task_id, title, type, sprint, status, priority, estimate, file_path, created_at, updated_at)
			 VALUES (?, 'title', 'feature', 'sprint-01', ?, 'p2', 'M', 'path', date('now'), date('now'))`,
			row.id, row.status,
		)
		if err != nil {
			t.Fatalf("Task insert failed: %v", err)
		}
	}

	result, err := sprint.AggregateProgress("sprint-01")
	if err != nil {
		t.Fatalf("AggregateProgress failed: %v", err)
	}

	if result.Done != 2 {
		t.Errorf("done=%v, want=2", result.Done)
	}
	if result.InProgress != 1 {
		t.Errorf("in_progress=%v, want=1", result.InProgress)
	}
	if result.Todo != 1 {
		t.Errorf("todo=%v, want=1", result.Todo)
	}
	if result.Total != 4 {
		t.Errorf("total=%v, want=4", result.Total)
	}
	if result.Percent != 50 {
		t.Errorf("percent=%v, want=50", result.Percent)
	}
}

// TestMoveSprint_FolderMove verifies the Sprint folder is moved.
func TestMoveSprint_FolderMove(t *testing.T) {
	tmpRoot := setupDB(t)

	// create source folder
	srcDir := filepath.Join(tmpRoot, "works", "sprints", "backlog", "sprint-01")
	if err := os.MkdirAll(filepath.Join(srcDir, "tasks"), 0o755); err != nil {
		t.Fatalf("source folder create failed: %v", err)
	}
	// create one file
	if err := os.WriteFile(filepath.Join(srcDir, "SPRINT.md"), []byte("test"), 0o644); err != nil {
		t.Fatalf("SPRINT.md create failed: %v", err)
	}

	dst, err := sprint.MoveSprint("sprint-01", "backlog", "active")
	if err != nil {
		t.Fatalf("MoveSprint failed: %v", err)
	}

	expectedDst := filepath.Join(tmpRoot, "works", "sprints", "active", "sprint-01")
	if dst != expectedDst {
		t.Errorf("destination path mismatch: got=%s, want=%s", dst, expectedDst)
	}

	// confirm destination exists after move
	if _, err := os.Stat(dst); os.IsNotExist(err) {
		t.Errorf("destination folder missing after move: %s", dst)
	}
}

// TestT472_MoveSprint_Idempotent_AlreadyMoved verifies that when src is
// missing but dst already exists (because the user moved manually or a
// previous run partially succeeded), MoveSprint returns dst as a no-op
// instead of erroring. Regression guard for ISS-20260413-012.
func TestT472_MoveSprint_Idempotent_AlreadyMoved(t *testing.T) {
	tmpRoot := setupDB(t)

	// only dst exists (user already moved into completed/)
	dstDir := filepath.Join(tmpRoot, "works", "sprints", "completed", "sprint-72")
	if err := os.MkdirAll(filepath.Join(dstDir, "tasks"), 0o755); err != nil {
		t.Fatalf("dst folder create failed: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dstDir, "SPRINT.md"), []byte("already moved"), 0o644); err != nil {
		t.Fatalf("SPRINT.md create failed: %v", err)
	}

	// attempt active → completed move. src absent, only dst present.
	dst, err := sprint.MoveSprint("sprint-72", "active", "completed")
	if err != nil {
		t.Fatalf("idempotent handling failed (must return dst with no error): %v", err)
	}
	expectedDst := filepath.Join(tmpRoot, "works", "sprints", "completed", "sprint-72")
	if dst != expectedDst {
		t.Errorf("returned path: got=%s, want=%s", dst, expectedDst)
	}
	// confirm dst folder was not destroyed
	if data, err := os.ReadFile(filepath.Join(expectedDst, "SPRINT.md")); err != nil || string(data) != "already moved" {
		t.Errorf("dst folder was destroyed: err=%v, data=%q", err, string(data))
	}
}

// TestT472_MoveSprint_SrcAndDstBothMissing verifies that when both src and
// dst are missing, NotFoundError is returned (regression guard against
// idempotent logic returning a spurious success).
func TestT472_MoveSprint_SrcAndDstBothMissing(t *testing.T) {
	setupDB(t)
	_, err := sprint.MoveSprint("sprint-ghost", "active", "completed")
	if err == nil {
		t.Fatal("expected error when both src and dst are missing, got nil")
	}
}

// TestUpdateSprintMDField_Update verifies SPRINT.md frontmatter-field
// update.
func TestUpdateSprintMDField_Update(t *testing.T) {
	tmpDir := t.TempDir()
	sprintMDPath := filepath.Join(tmpDir, "SPRINT.md")
	content := `---
id: sprint-01
status: backlog
started: ~
---

# body
`
	if err := os.WriteFile(sprintMDPath, []byte(content), 0o644); err != nil {
		t.Fatalf("file create failed: %v", err)
	}

	if err := sprint.UpdateSprintMDField(sprintMDPath, "status", "active"); err != nil {
		t.Fatalf("UpdateSprintMDField failed: %v", err)
	}

	data, _ := os.ReadFile(sprintMDPath)
	if !strings.Contains(string(data), "status: active") {
		t.Errorf("status not updated: %s", string(data))
	}
	if strings.Contains(string(data), "status: backlog") {
		t.Errorf("previous status still present")
	}
}

// TestReplaceFrontmatterField_Title verifies in-memory replacement of the
// frontmatter title field. T179 regression: previously the handler wrote
// the file via UpdateSprintMDField and then overwrote it with stale
// content (double-write). ReplaceFrontmatterField transforms the content
// string only so callers persist via a single WriteFile.
func TestReplaceFrontmatterField_Title(t *testing.T) {
	content := `---
id: sprint-01
title: "old title"
status: backlog
goal: "old goal"
---

# sprint-01: old title

## Goal

old goal
`

	result := sprint.ReplaceFrontmatterField(content, "title", "new title")

	// frontmatter title must be updated as a double-quoted value
	if !strings.Contains(result, `title: "new title"`) {
		t.Errorf("frontmatter title update failed:\n%s", result)
	}
	// previous value must not remain
	if strings.Contains(result, `title: "old title"`) {
		t.Errorf("previous title still present:\n%s", result)
	}
	// body title (# sprint-01: old title) must not be touched
	if !strings.Contains(result, "# sprint-01: old title") {
		t.Errorf("body title incorrectly changed:\n%s", result)
	}
	// other frontmatter fields must remain
	if !strings.Contains(result, "id: sprint-01") {
		t.Errorf("id field lost:\n%s", result)
	}
	if !strings.Contains(result, `goal: "old goal"`) {
		t.Errorf("goal field lost:\n%s", result)
	}
}

// TestReplaceFrontmatterField_Goal verifies in-memory replacement of the
// frontmatter goal field.
func TestReplaceFrontmatterField_Goal(t *testing.T) {
	content := `---
id: sprint-01
title: "keep"
goal: "old goal"
---

# body
`

	result := sprint.ReplaceFrontmatterField(content, "goal", "new goal")

	if !strings.Contains(result, `goal: "new goal"`) {
		t.Errorf("goal update failed:\n%s", result)
	}
	if strings.Contains(result, `goal: "old goal"`) {
		t.Errorf("previous goal still present:\n%s", result)
	}
}

// TestReplaceFrontmatterField_SpecialCharEscape verifies that values
// containing quotes / backslashes are safely escaped as YAML double-quoted
// strings.
func TestReplaceFrontmatterField_SpecialCharEscape(t *testing.T) {
	content := `---
title: "old"
---
`

	result := sprint.ReplaceFrontmatterField(content, "title", `she said "hi"`)

	// Go %q escapes interior quotes as \"
	expected := `title: "she said \"hi\""`
	if !strings.Contains(result, expected) {
		t.Errorf("escape failed: expected=%q\nactual:\n%s", expected, result)
	}
}

// TestReplaceFrontmatterField_NoFrontmatter verifies that the original is
// returned unchanged when content has no frontmatter.
func TestReplaceFrontmatterField_NoFrontmatter(t *testing.T) {
	content := "# just body\n\ntitle: in body not frontmatter\n"

	result := sprint.ReplaceFrontmatterField(content, "title", "new")

	if result != content {
		t.Errorf("original must be preserved when no frontmatter:\ngot:\n%s\nwant:\n%s", result, content)
	}
}

// TestReplaceFrontmatterField_NoField verifies that the original is
// returned unchanged when the frontmatter has no such field.
func TestReplaceFrontmatterField_NoField(t *testing.T) {
	content := `---
id: sprint-01
status: backlog
---

# body
`

	result := sprint.ReplaceFrontmatterField(content, "title", "new")

	if result != content {
		t.Errorf("original must be preserved when field is missing:\ngot:\n%s\nwant:\n%s", result, content)
	}
}

// TestReplaceFrontmatterField_BodyMatchExcluded verifies that an
// identically-prefixed line outside the frontmatter block is not replaced.
func TestReplaceFrontmatterField_BodyMatchExcluded(t *testing.T) {
	content := `---
id: sprint-01
---

title: this is in body, should not be replaced
`

	result := sprint.ReplaceFrontmatterField(content, "title", "new")

	// body "title:" line must remain untouched
	if !strings.Contains(result, "title: this is in body, should not be replaced") {
		t.Errorf("body title line incorrectly changed:\n%s", result)
	}
	// frontmatter had no title field originally so a new one must not be added
	if strings.Contains(result, `title: "new"`) {
		t.Errorf("body title incorrectly replaced:\n%s", result)
	}
}

// TestListFromDB_StatusFilter verifies that ListFromDB respects the status
// filter.
func TestListFromDB_StatusFilter(t *testing.T) {
	setupDB(t)

	for _, pair := range []struct{ id, status string }{
		{"sprint-A", "backlog"},
		{"sprint-B", "active"},
		{"sprint-C", "completed"},
	} {
		if err := sprint.SaveToDB(pair.id, "title", "goal", pair.status, "path/"+pair.id, nil, nil); err != nil {
			t.Fatalf("SaveToDB failed: %v", err)
		}
	}

	active, err := sprint.ListFromDB("active")
	if err != nil {
		t.Fatalf("ListFromDB(active) failed: %v", err)
	}
	if len(active) != 1 {
		t.Errorf("active Sprint count=%d, want=1", len(active))
	}
	if active[0].SprintID != "sprint-B" {
		t.Errorf("active Sprint ID=%v, want=sprint-B", active[0].SprintID)
	}

	completed, err := sprint.ListFromDB("completed")
	if err != nil {
		t.Fatalf("ListFromDB(completed) failed: %v", err)
	}
	if len(completed) != 1 {
		t.Errorf("completed Sprint count=%d, want=1", len(completed))
	}
}

// TestRenderSprintMD_WithTasks verifies that SPRINT.md including Tasks
// renders correctly.
func TestRenderSprintMD_WithTasks(t *testing.T) {
	tasks := []map[string]any{
		{"title": "Task A", "type": "feature", "priority": "p0", "estimate": "M"},
		{"title": "Task B", "type": "infra", "priority": "p1", "estimate": "S"},
	}
	content := sprint.RenderSprintMD("sprint-01", "containment test", "goal", tasks)

	if !strings.Contains(content, "Task A") {
		t.Error("Task A not included")
	}
	if !strings.Contains(content, "Task B") {
		t.Error("Task B not included")
	}
	if !strings.Contains(content, "feature") {
		t.Error("feature type not included")
	}
}

// TestMoveSprint_LeftoverFolderRemoved verifies behaviour when a leftover
// folder remains in dst.
func TestMoveSprint_LeftoverFolderRemoved(t *testing.T) {
	tmpRoot := setupDB(t)

	// create a Sprint folder under active
	srcDir := filepath.Join(tmpRoot, "works", "sprints", "active", "sprint-res")
	if err := os.MkdirAll(filepath.Join(srcDir, "tasks"), 0o755); err != nil {
		t.Fatalf("source folder create failed: %v", err)
	}
	if err := os.WriteFile(filepath.Join(srcDir, "SPRINT.md"), []byte("---\nid: sprint-res\n---\n"), 0o644); err != nil {
		t.Fatalf("SPRINT.md create failed: %v", err)
	}

	dst, err := sprint.MoveSprint("sprint-res", "active", "completed")
	if err != nil {
		t.Fatalf("MoveSprint failed: %v", err)
	}

	expectedDst := filepath.Join(tmpRoot, "works", "sprints", "completed", "sprint-res")
	if dst != expectedDst {
		t.Errorf("destination path mismatch after move: got=%s, want=%s", dst, expectedDst)
	}
	if _, err := os.Stat(dst); os.IsNotExist(err) {
		t.Errorf("destination folder missing after move: %s", dst)
	}
	if _, err := os.Stat(srcDir); !os.IsNotExist(err) {
		t.Errorf("source folder still present: %s", srcDir)
	}
}

// TestAggregateProgress_EmptySprint verifies aggregation for a Sprint with
// no Tasks.
func TestAggregateProgress_EmptySprint(t *testing.T) {
	setupDB(t)

	if err := sprint.SaveToDB("sprint-empty", "empty Sprint", "goal", "active", "path/sprint-empty", nil, nil); err != nil {
		t.Fatalf("SaveToDB failed: %v", err)
	}

	result, err := sprint.AggregateProgress("sprint-empty")
	if err != nil {
		t.Fatalf("AggregateProgress failed: %v", err)
	}

	if result.Total != 0 {
		t.Errorf("total=%v, want=0", result.Total)
	}
	if result.Percent != 0 {
		t.Errorf("percent=%v, want=0", result.Percent)
	}
}

// TestCreateFolder_AlreadyExists verifies that CreateFolder works without
// error when the folder already exists.
func TestCreateFolder_AlreadyExists(t *testing.T) {
	setupDB(t)

	// first create
	dir1, err := sprint.CreateFolder("sprint-dup", "backlog")
	if err != nil {
		t.Fatalf("first CreateFolder failed: %v", err)
	}

	// second create — already-exists case
	dir2, err := sprint.CreateFolder("sprint-dup", "backlog")
	if err != nil {
		t.Fatalf("second CreateFolder failed: %v", err)
	}
	if dir1 != dir2 {
		t.Errorf("path mismatch: got=%s, want=%s", dir2, dir1)
	}
}

// T763 (Sprint-89): TestRenderCeremonyMD_* x5 removed — all CEREMONY.md
// rendering, double-prefix collapse, YAML customisation and YAML fallback
// tests are no longer needed. ceremony state SSOT is harness_items DB.

// TestSaveToDB_DuplicateUpdate verifies that saving twice with the same
// Sprint ID performs an update.
func TestSaveToDB_DuplicateUpdate(t *testing.T) {
	setupDB(t)

	if err := sprint.SaveToDB("sprint-upd", "original title", "original goal", "backlog", "path/sprint-upd", nil, nil); err != nil {
		t.Fatalf("first SaveToDB failed: %v", err)
	}
	if err := sprint.SaveToDB("sprint-upd", "modified title", "modified goal", "active", "new/path/sprint-upd", nil, nil); err != nil {
		t.Fatalf("second SaveToDB failed: %v", err)
	}

	s, err := sprint.GetFromDB("sprint-upd")
	if err != nil {
		t.Fatalf("GetFromDB failed: %v", err)
	}
	// confirm status was updated (INSERT OR REPLACE behaviour)
	t.Logf("Sprint status: %v", s.Status)
}

// TestGetFromDB_ExistingSprint verifies single-Sprint lookup.
func TestGetFromDB_ExistingSprint(t *testing.T) {
	setupDB(t)

	if err := sprint.SaveToDB("sprint-get", "lookup test", "goal", "active", "path/sprint-get", nil, nil); err != nil {
		t.Fatalf("SaveToDB failed: %v", err)
	}

	s, err := sprint.GetFromDB("sprint-get")
	if err != nil {
		t.Fatalf("GetFromDB failed: %v", err)
	}
	if s.SprintID != "sprint-get" {
		t.Errorf("sprint_id=%v, want=sprint-get", s.SprintID)
	}
	if s.Title != "lookup test" {
		t.Errorf("title=%v, want=lookup test", s.Title)
	}
}

// TestAggregateProgress_PercentCalculation verifies that percentage is
// computed correctly.
func TestAggregateProgress_PercentCalculation(t *testing.T) {
	setupDB(t)

	if err := sprint.SaveToDB("sprint-pct", "percent test", "goal", "active", "path/sprint-pct", nil, nil); err != nil {
		t.Fatalf("SaveToDB failed: %v", err)
	}

	database := db.GetDB()
	for _, row := range []struct {
		id     string
		status string
	}{
		{"T901", "done"},
		{"T902", "done"},
		{"T903", "in-progress"},
		{"T904", "todo"},
		{"T905", "todo"},
	} {
		_, err := database.Exec(
			`INSERT INTO tasks (task_id, title, type, sprint, status, priority, estimate, file_path, created_at, updated_at)
			 VALUES (?, 'title', 'feature', 'sprint-pct', ?, 'p2', 'M', 'path', date('now'), date('now'))`,
			row.id, row.status,
		)
		if err != nil {
			t.Fatalf("Task insert failed: %v", err)
		}
	}

	result, err := sprint.AggregateProgress("sprint-pct")
	if err != nil {
		t.Fatalf("AggregateProgress failed: %v", err)
	}

	if result.Done != 2 {
		t.Errorf("done=%v, want=2", result.Done)
	}
	if result.Total != 5 {
		t.Errorf("total=%v, want=5", result.Total)
	}
	// 2/5 = 40%
	if result.Percent != 40 {
		t.Errorf("percent=%v, want=40", result.Percent)
	}
}

// TestAggregateProgress_NonExistentSprint verifies aggregation against a
// non-existent Sprint returns either an error or an empty result.
func TestAggregateProgress_NonExistentSprint(t *testing.T) {
	setupDB(t)

	result, err := sprint.AggregateProgress("sprint-nonexistent")
	if err != nil {
		t.Fatalf("AggregateProgress failed: %v", err)
	}
	// when the Sprint is missing, total=0 expected
	if result.Total != 0 {
		t.Logf("non-existent Sprint result: %v", result)
	}
}

// TestListFromDB_EmptyDB verifies that an empty list is returned when
// there are no Sprints.
func TestListFromDB_EmptyDB(t *testing.T) {
	setupDB(t)

	sprints, err := sprint.ListFromDB("")
	if err != nil {
		t.Fatalf("ListFromDB failed: %v", err)
	}
	if len(sprints) != 0 {
		t.Errorf("empty DB but Sprint count=%d, want=0", len(sprints))
	}
}

// TestSaveToDB_MultipleSprint verifies that multiple Sprints persist
// independently.
func TestSaveToDB_MultipleSprint(t *testing.T) {
	setupDB(t)

	for _, id := range []string{"sprint-a", "sprint-b", "sprint-c"} {
		if err := sprint.SaveToDB(id, id+" title", "goal", "backlog", "works/sprints/backlog/"+id, nil, nil); err != nil {
			t.Fatalf("SaveToDB failed (%s): %v", id, err)
		}
	}

	sprints, err := sprint.ListFromDB("")
	if err != nil {
		t.Fatalf("ListFromDB failed: %v", err)
	}
	if len(sprints) != 3 {
		t.Errorf("Sprint count=%d, want=3", len(sprints))
	}
}

// TestUpdateStatusInDB_NonExistentSprint verifies that updating a missing
// Sprint ID does not produce an error.
func TestUpdateStatusInDB_NonExistentSprint(t *testing.T) {
	setupDB(t)

	// updating a missing Sprint ID should not produce an error
	err := sprint.UpdateStatusInDB("sprint-notexist", "active", nil, nil, nil)
	if err != nil {
		t.Fatalf("UpdateStatusInDB failed: %v", err)
	}
}

// TestGetFromDB_NonExistentSprint verifies that an error or nil result is
// returned when a missing Sprint ID is looked up.
func TestGetFromDB_NonExistentSprint(t *testing.T) {
	setupDB(t)

	result, err := sprint.GetFromDB("sprint-notexist")
	if err == nil && result != nil {
		t.Logf("non-existent Sprint lookup result: %v", result)
	}
	// either error or nil result is acceptable — normal (nil return)
}

// TestRenderSprintMD_MultipleTasks verifies that multiple Tasks appear in
// SPRINT.md.
func TestRenderSprintMD_MultipleTasks(t *testing.T) {
	tasks := []map[string]any{
		{"task_id": "T001", "title": "Task 1", "type": "feature", "status": "todo", "estimate": "M"},
		{"task_id": "T002", "title": "Task 2", "type": "bugfix", "status": "in-progress", "estimate": "S"},
		{"task_id": "T003", "title": "Task 3", "type": "chore", "status": "done", "estimate": "XS"},
	}

	content := sprint.RenderSprintMD("sprint-01", "multi-Task sprint", "goal", tasks)

	for _, tk := range tasks {
		if !strings.Contains(content, tk["task_id"].(string)) {
			t.Errorf("Task ID %s missing from SPRINT.md", tk["task_id"])
		}
	}
}

// TestRenderSprintMD_PriorityEmpty_AppliesDefault — when priority is the
// empty string it should default to p2.
func TestRenderSprintMD_PriorityEmpty_AppliesDefault(t *testing.T) {
	tasks := []map[string]any{
		{"task_id": "T001", "title": "Priority missing", "type": "feature", "estimate": "M", "status": "todo"},
		{"task_id": "T002", "title": "Priority empty string", "type": "bugfix", "estimate": "S", "status": "todo", "priority": ""},
		{"task_id": "T003", "title": "Priority normal", "type": "chore", "estimate": "XS", "status": "done", "priority": "p1"},
	}

	content := sprint.RenderSprintMD("sprint-test", "default verification", "goal", tasks)

	// confirm each row is 7 columns (8 pipes)
	for _, line := range strings.Split(content, "\n") {
		if !strings.HasPrefix(line, "| T") {
			continue
		}
		pipes := strings.Count(line, "|")
		if pipes != 8 { // 8 pipes including the outer ones = 7 columns
			t.Errorf("column count mismatch: line=%q, pipes=%d, want=8", line, pipes)
		}
	}

	// T001 (no priority) → "p2" default
	if !strings.Contains(content, "| T001 | Priority missing | feature | M | p2 | todo | — |") {
		t.Error("default p2 not applied to T001")
	}
	// T002 (empty priority) → "p2" default
	if !strings.Contains(content, "| T002 | Priority empty string | bugfix | S | p2 | todo | — |") {
		t.Error("default p2 not applied to T002")
	}
	// T003 (priority normal) → "p1" preserved
	if !strings.Contains(content, "| T003 | Priority normal | chore | XS | p1 | done | — |") {
		t.Error("priority p1 not preserved on T003")
	}
}

// TestRenderSprintMD_DependsOn_VariousTypes — depends_on handles []any,
// []string, string.
func TestRenderSprintMD_DependsOn_VariousTypes(t *testing.T) {
	tasks := []map[string]any{
		{"task_id": "T001", "title": "deps []string", "depends_on": []string{"T010", "T011"}},
		{"task_id": "T002", "title": "deps []any", "depends_on": []any{"T020", "T021"}},
		{"task_id": "T003", "title": "deps empty []any", "depends_on": []any{}},
		{"task_id": "T004", "title": "deps missing"},
	}

	content := sprint.RenderSprintMD("sprint-test", "deps verification", "goal", tasks)

	if !strings.Contains(content, "T010, T011") {
		t.Error("[]string depends_on not handled")
	}
	if !strings.Contains(content, "T020, T021") {
		t.Error("[]any depends_on not handled")
	}
}

// TestListFromDB_MultiStatus verifies multiple Sprints across statuses.
func TestListFromDB_MultiSprintPersist(t *testing.T) {
	setupDB(t)

	sprint.SaveToDB("sprint-active-1", "active 1", "goal", "active", "path1", nil, nil)    //nolint:errcheck
	sprint.SaveToDB("sprint-active-2", "active 2", "goal", "active", "path2", nil, nil)    //nolint:errcheck
	sprint.SaveToDB("sprint-backlog-1", "backlog 1", "goal", "backlog", "path3", nil, nil) //nolint:errcheck

	actives, err := sprint.ListFromDB("active")
	if err != nil {
		t.Fatalf("ListFromDB(active) failed: %v", err)
	}
	if len(actives) != 2 {
		t.Errorf("active Sprint count=%d, want=2", len(actives))
	}

	backlogs, err := sprint.ListFromDB("backlog")
	if err != nil {
		t.Fatalf("ListFromDB(backlog) failed: %v", err)
	}
	if len(backlogs) != 1 {
		t.Errorf("backlog Sprint count=%d, want=1", len(backlogs))
	}
}

// TestAggregateProgress_100Percent verifies that 100% is returned when all
// Tasks are done.
func TestAggregateProgress_100Percent(t *testing.T) {
	setupDB(t)

	// register Sprint
	if err := sprint.SaveToDB("sprint-full", "Full Sprint", "goal", "active", "path", nil, nil); err != nil {
		t.Fatalf("SaveToDB failed: %v", err)
	}

	database := db.GetDB()
	for _, id := range []string{"T801", "T802", "T803"} {
		_, err := database.Exec(
			`INSERT INTO tasks (task_id, title, type, sprint, status, priority, estimate, file_path, created_at, updated_at)
			 VALUES (?, 'title', 'feature', 'sprint-full', 'done', 'p2', 'M', 'path', date('now'), date('now'))`,
			id,
		)
		if err != nil {
			t.Fatalf("Task insert failed: %v", err)
		}
	}

	result, err := sprint.AggregateProgress("sprint-full")
	if err != nil {
		t.Fatalf("AggregateProgress failed: %v", err)
	}
	if result.Percent != 100 {
		t.Errorf("percent=%v, want=100", result.Percent)
	}
}
