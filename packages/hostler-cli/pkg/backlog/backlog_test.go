package backlog

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/internal/adapters/sqlite"
	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/pkg/db"
	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/pkg/store"
)

// ---------------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------------

// setupTestDir builds an isolated tmpDir-based environment and returns a
// cleanup function.
func setupTestDir(t *testing.T) (string, func()) {
	t.Helper()
	dir := t.TempDir()

	// override the project root via env var
	t.Setenv("HSTL_PROJECT_ROOT", dir)

	// configure an isolated DB via env var
	dbPath := filepath.Join(dir, "test.db")
	t.Setenv("HSTL_DB_PATH", dbPath)

	// create the standard directory structure
	dirs := []string{
		filepath.Join(dir, "works", "tasks"),
		filepath.Join(dir, "works", "sprints", "active", "sprint-01", "tasks"),
		filepath.Join(dir, "works", "sprints", "completed", "sprint-00", "tasks"),
	}
	for _, d := range dirs {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatalf("directory create failed: %v", err)
		}
	}

	// initialise DB
	if err := db.InitDB(); err != nil {
		t.Fatalf("DB initialisation failed: %v", err)
	}
	sqliteStore := sqlite.New(db.GetDB())
	store.Init(sqliteStore, sqliteStore)

	cleanup := func() {
		store.Reset()
		db.Close()
	}
	return dir, cleanup
}

// writeTaskFile creates a Task markdown file for tests.
func writeTaskFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("directory create failed: %v", err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("file write failed: %v", err)
	}
}

// insertDBTask inserts a tasks record for tests.
func insertDBTask(t *testing.T, taskID, status, sprint, filePath string) {
	t.Helper()
	database := db.GetDB()
	now := time.Now().Format("2006-01-02")
	var sprintVal interface{}
	if sprint != "" {
		sprintVal = sprint
	}
	_, err := database.Exec(
		`INSERT INTO tasks (task_id, title, type, sprint, status, priority, estimate, file_path, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		taskID, "test Task", "feature", sprintVal, status, "p2", "S", filePath, now, now,
	)
	if err != nil {
		t.Fatalf("DB Task insert failed: %v", err)
	}
}

// ---------------------------------------------------------------------------
// TestSync_DryRun: detection only, DB unchanged
// ---------------------------------------------------------------------------

func TestSync_DryRun(t *testing.T) {
	dir, cleanup := setupTestDir(t)
	defer cleanup()

	// Scenario:
	// - T001: file in works/tasks/, missing in DB → db_missing_task
	// - T002: file in works/tasks/, DB status mismatch → db_status_mismatch
	taskDir := filepath.Join(dir, "works", "tasks")

	writeTaskFile(t, filepath.Join(taskDir, "T001-test-task.md"), `---
title: test Task 1
type: feature
status: todo
priority: p2
estimate: S
---
body
`)

	writeTaskFile(t, filepath.Join(taskDir, "T002-another-task.md"), `---
title: test Task 2
type: bugfix
status: in-progress
priority: p1
estimate: M
---
body
`)
	// T002 is in DB but with a different status
	insertDBTask(t, "T002", "todo", "", "works/tasks/T002-another-task.md")

	result, err := Sync(true)
	if err != nil {
		t.Fatalf("Sync failed: %v", err)
	}

	if result.Summary.DryRun != true {
		t.Errorf("DryRun must be true")
	}
	if result.Summary.TaskFilesScanned != 2 {
		t.Errorf("expected TaskFilesScanned=2, got=%d", result.Summary.TaskFilesScanned)
	}
	if result.Summary.AutoFixed != 0 {
		t.Errorf("expected AutoFixed=0 in dry-run mode, got=%d", result.Summary.AutoFixed)
	}
	if len(result.Fixes) != 0 {
		t.Errorf("Fixes must be empty in dry-run mode, got=%d", len(result.Fixes))
	}

	// confirm db_missing_task issue
	hasMissingTask := false
	hasStatusMismatch := false
	for _, iss := range result.Issues {
		if iss.Type == "db_missing_task" && iss.TaskID == "T001" {
			hasMissingTask = true
			if !iss.AutoFixable {
				t.Errorf("T001 db_missing_task must be auto_fixable=true")
			}
		}
		if iss.Type == "db_status_mismatch" && iss.TaskID == "T002" {
			hasStatusMismatch = true
			if iss.DBStatus != "todo" {
				t.Errorf("T002 expected DBStatus=todo, got=%s", iss.DBStatus)
			}
			if iss.FileStatus != "in-progress" {
				t.Errorf("T002 expected FileStatus=in-progress, got=%s", iss.FileStatus)
			}
		}
	}
	if !hasMissingTask {
		t.Errorf("db_missing_task(T001) issue must be detected")
	}
	if !hasStatusMismatch {
		t.Errorf("db_status_mismatch(T002) issue must be detected")
	}

	// DryRun → DB must not change
	database := db.GetDB()
	row := database.QueryRow("SELECT COUNT(*) FROM tasks WHERE task_id = 'T001'")
	var cnt int
	if err := row.Scan(&cnt); err != nil || cnt != 0 {
		t.Errorf("T001 must not be inserted in DryRun (cnt=%d)", cnt)
	}
}

// ---------------------------------------------------------------------------
// TestSync_Apply: actually applies auto-fixes
// ---------------------------------------------------------------------------

func TestSync_Apply(t *testing.T) {
	dir, cleanup := setupTestDir(t)
	defer cleanup()

	taskDir := filepath.Join(dir, "works", "tasks")

	// T003: missing in DB, frontmatter present → INSERT fix
	writeTaskFile(t, filepath.Join(taskDir, "T003-apply-test.md"), `---
title: apply test Task
type: chore
status: todo
priority: p2
estimate: S
---
body
`)

	// T004: DB status mismatch → UPDATE fix
	writeTaskFile(t, filepath.Join(taskDir, "T004-status-fix.md"), `---
title: status-fix Task
type: feature
status: done
priority: p1
estimate: L
---
body
`)
	insertDBTask(t, "T004", "in-progress", "", "works/tasks/T004-status-fix.md")

	result, err := Sync(false)
	if err != nil {
		t.Fatalf("Sync(apply) failed: %v", err)
	}

	if result.Summary.DryRun != false {
		t.Errorf("DryRun must be false")
	}
	if result.Summary.AutoFixed == 0 {
		t.Errorf("expected AutoFixed > 0 in apply mode")
	}

	// confirm T003 was inserted
	database := db.GetDB()
	row := database.QueryRow("SELECT status FROM tasks WHERE task_id = 'T003'")
	var status string
	if err := row.Scan(&status); err != nil {
		t.Errorf("T003 was not inserted into DB: %v", err)
	} else if status != "todo" {
		t.Errorf("expected T003 status=todo, got=%s", status)
	}

	// confirm T004 status updated to done
	row = database.QueryRow("SELECT status FROM tasks WHERE task_id = 'T004'")
	if err := row.Scan(&status); err != nil {
		t.Fatalf("T004 lookup failed: %v", err)
	}
	if status != "done" {
		t.Errorf("expected T004 status=done after fix, got=%s", status)
	}

	// confirm fix records
	insertFix := false
	statusFix := false
	for _, f := range result.Fixes {
		if f.Type == "db_task_inserted" && f.TaskID == "T003" {
			insertFix = true
		}
		if f.Type == "db_status_fixed" && f.TaskID == "T004" {
			statusFix = true
		}
	}
	if !insertFix {
		t.Errorf("missing db_task_inserted(T003) Fix record")
	}
	if !statusFix {
		t.Errorf("missing db_status_fixed(T004) Fix record")
	}
}

// ---------------------------------------------------------------------------
// TestSync_LocationFrontmatterAutoFix (T214)
// ---------------------------------------------------------------------------

// TestSync_LocationFrontmatterAutoFix verifies that when a file under
// works/tasks/ has frontmatter.sprint set to a real Sprint ID, auto-fix
// rewrites the field to "backlog" and aligns the DB sprint column to NULL
// (T214).
func TestSync_LocationFrontmatterAutoFix(t *testing.T) {
	dir, cleanup := setupTestDir(t)
	defer cleanup()

	taskDir := filepath.Join(dir, "works", "tasks")

	// invalid state: file lives under works/tasks/ but frontmatter.sprint=sprint-01
	writeTaskFile(t, filepath.Join(taskDir, "T007-location-mismatch.md"), `---
id: T007
title: location-mismatch Task
type: feature
status: todo
sprint: sprint-01
priority: p2
estimate: S
---
body
`)
	// per T212 fix, DB sprint is NULL (mismatched with file frontmatter)
	insertDBTask(t, "T007", "todo", "", "works/tasks/T007-location-mismatch.md")

	// DryRun — confirm issue detection
	result, err := Sync(true)
	if err != nil {
		t.Fatalf("Sync(dry) failed: %v", err)
	}

	foundDryIssue := false
	for _, iss := range result.Issues {
		if iss.Type == "location_frontmatter_mismatch" && iss.TaskID == "T007" {
			if !iss.AutoFixable {
				t.Errorf("location_frontmatter_mismatch should be auto_fixable=true (T214)")
			}
			foundDryIssue = true
		}
	}
	if !foundDryIssue {
		t.Fatalf("location_frontmatter_mismatch issue not detected for T007 in dry run")
	}

	// Apply — actually run the fix
	result, err = Sync(false)
	if err != nil {
		t.Fatalf("Sync(apply) failed: %v", err)
	}

	foundFix := false
	for _, f := range result.Fixes {
		if f.Type == "location_frontmatter_sprint_cleared" && f.TaskID == "T007" {
			foundFix = true
			if f.OldSprint != "sprint-01" {
				t.Errorf("expected OldSprint=sprint-01, got=%s", f.OldSprint)
			}
			if f.NewSprint != "backlog" {
				t.Errorf("expected NewSprint=backlog, got=%s", f.NewSprint)
			}
		}
	}
	if !foundFix {
		t.Errorf("location_frontmatter_sprint_cleared Fix record missing")
	}

	// confirm file frontmatter actually changed
	data, err := os.ReadFile(filepath.Join(taskDir, "T007-location-mismatch.md"))
	if err != nil {
		t.Fatalf("file read failed: %v", err)
	}
	content := string(data)
	if !containsStr(content, "sprint: backlog") {
		t.Errorf("frontmatter must record sprint: backlog:\n%s", content)
	}
	if containsStr(content, "sprint: sprint-01") {
		t.Errorf("frontmatter still has sprint: sprint-01:\n%s", content)
	}

	// confirm DB updated to NULL
	database := db.GetDB()
	var dbSprint *string
	if err := database.QueryRow(
		"SELECT sprint FROM tasks WHERE task_id = ?", "T007",
	).Scan(&dbSprint); err != nil {
		t.Fatalf("DB query failed: %v", err)
	}
	if dbSprint != nil {
		t.Errorf("expected DB sprint=NULL, got=%q", *dbSprint)
	}
}

// containsStr is a test helper — strings.Contains wrapper.
func containsStr(haystack, needle string) bool {
	return len(haystack) >= len(needle) && func() bool {
		for i := 0; i+len(needle) <= len(haystack); i++ {
			if haystack[i:i+len(needle)] == needle {
				return true
			}
		}
		return false
	}()
}

// ---------------------------------------------------------------------------
// TestCheckDuplicateIDs
// ---------------------------------------------------------------------------

func TestCheckDuplicateIDs(t *testing.T) {
	dir := t.TempDir()
	files := []string{
		filepath.Join(dir, "T010-first.md"),
		filepath.Join(dir, "T010-duplicate.md"),
		filepath.Join(dir, "T011-unique.md"),
	}
	for _, f := range files {
		os.WriteFile(f, []byte(""), 0o644)
	}

	issues := checkDuplicateIDs(files)
	if len(issues) != 1 {
		t.Fatalf("expected 1 duplicate issue, got=%d", len(issues))
	}
	if issues[0].TaskID != "T010" {
		t.Errorf("expected duplicate TaskID=T010, got=%s", issues[0].TaskID)
	}
	if issues[0].AutoFixable != false {
		t.Errorf("duplicate_task_id must be auto_fixable=false")
	}
}

// ---------------------------------------------------------------------------
// TestGetDirLocation
// ---------------------------------------------------------------------------

func TestGetDirLocation(t *testing.T) {
	root := "/project"

	cases := []struct {
		path             string
		wantBacklog      bool
		wantSprint       bool
		wantSprintID     string
		wantSprintStatus string
	}{
		{
			path:        "/project/works/tasks/T001-test.md",
			wantBacklog: true,
			wantSprint:  false,
		},
		{
			path:             "/project/works/sprints/active/sprint-01/tasks/T002-test.md",
			wantBacklog:      false,
			wantSprint:       true,
			wantSprintID:     "sprint-01",
			wantSprintStatus: "active",
		},
		{
			path:             "/project/works/sprints/completed/sprint-00/tasks/T003-test.md",
			wantBacklog:      false,
			wantSprint:       true,
			wantSprintID:     "sprint-00",
			wantSprintStatus: "completed",
		},
		{
			path:        "/project/works/sprints/backlog/sprint-02/SPRINT.md",
			wantBacklog: false,
			wantSprint:  false, // not under tasks/
		},
		// T608: works/tasks/completed/T*.md → inBacklog=false (completed
		// move target).
		{
			path:        "/project/works/tasks/completed/T004-test.md",
			wantBacklog: false,
			wantSprint:  false,
		},
		// T608: an additional subdir under sprint-tasks subdir (future
		// extension) → inSprint=false.
		{
			path:        "/project/works/sprints/active/sprint-01/tasks/sub/T005-test.md",
			wantBacklog: false,
			wantSprint:  false,
		},
	}

	for _, tc := range cases {
		loc := getDirLocation(tc.path, root)
		if loc.inBacklogTasks != tc.wantBacklog {
			t.Errorf("%s: expected inBacklogTasks=%v, got=%v", tc.path, tc.wantBacklog, loc.inBacklogTasks)
		}
		if loc.inSprintTasks != tc.wantSprint {
			t.Errorf("%s: expected inSprintTasks=%v, got=%v", tc.path, tc.wantSprint, loc.inSprintTasks)
		}
		if tc.wantSprint {
			if loc.sprintID != tc.wantSprintID {
				t.Errorf("%s: expected sprintID=%s, got=%s", tc.path, tc.wantSprintID, loc.sprintID)
			}
			if loc.sprintStatus != tc.wantSprintStatus {
				t.Errorf("%s: expected sprintStatus=%s, got=%s", tc.path, tc.wantSprintStatus, loc.sprintStatus)
			}
		}
	}
}

// TestFmString_YAMLNull verifies that a YAML null value resolves to an
// empty string (T176). Earlier bug: when fmString got nil it took the
// default branch fmt.Sprintf("%v", nil) and produced the literal "<nil>",
// which made backlog_sync raise spurious warnings.
func TestFmString_YAMLNull(t *testing.T) {
	cases := []struct {
		name string
		fm   map[string]any
		key  string
		want string
	}{
		{"nil map", nil, "sprint", ""},
		{"missing key", map[string]any{"status": "todo"}, "sprint", ""},
		{"nil value (YAML null/~)", map[string]any{"sprint": nil}, "sprint", ""},
		{"string value", map[string]any{"sprint": "sprint-01"}, "sprint", "sprint-01"},
		{"empty string", map[string]any{"sprint": ""}, "sprint", ""},
		{"int value", map[string]any{"priority": 2}, "priority", "2"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := fmString(tc.fm, tc.key)
			if got != tc.want {
				t.Errorf("fmString(%v, %q) = %q, want %q", tc.fm, tc.key, got, tc.want)
			}
		})
	}
}

// TestSync_T304_FileStatusStale_AfterComplete is the regression guard for
// T304 Bug C. When the file lives under completed/sprint-XX/tasks/ with
// DB=done but the frontmatter says in-progress, do NOT overwrite the DB —
// fix the file frontmatter to done instead.
func TestSync_T304_FileStatusStale_AfterComplete(t *testing.T) {
	dir, cleanup := setupTestDir(t)
	defer cleanup()

	// T304: place a stale-frontmatter file under completed/sprint-00/tasks/
	completedTaskDir := filepath.Join(dir, "works", "sprints", "completed", "sprint-00", "tasks")
	taskFile := filepath.Join(completedTaskDir, "T900-legacy-stale-task.md")
	content := `---
id: T900
title: legacy stale task
type: feature
status: in-progress
priority: p2
estimate: S
sprint: sprint-00
---
body
`
	writeTaskFile(t, taskFile, content)
	insertDBTask(t, "T900", "done", "sprint-00",
		"works/sprints/completed/sprint-00/tasks/T900-legacy-stale-task.md")

	result, err := Sync(false)
	if err != nil {
		t.Fatalf("Sync failed: %v", err)
	}

	// Bug C regression: must classify as file_status_stale_after_complete
	// rather than db_status_mismatch.
	var found bool
	for _, iss := range result.Issues {
		if iss.TaskID != "T900" {
			continue
		}
		if iss.Type == issueDBStatusMismatch {
			t.Errorf("T900 must not be classified as db_status_mismatch (SSOT inversion): %+v", iss)
		}
		if iss.Type == issueFileStatusStale {
			found = true
			if iss.DBStatus != taskStatusDone {
				t.Errorf("expected T900 DBStatus=done, got=%s", iss.DBStatus)
			}
			if iss.FileStatus != "in-progress" {
				t.Errorf("expected T900 FileStatus=in-progress, got=%s", iss.FileStatus)
			}
			// T212 (Sprint-09): completed-Sprint files are HMAC-signed —
			// auto-fix is blocked. Issue detection remains, but no automatic
			// modification is performed (manual re-sign required).
			if iss.AutoFixable {
				t.Errorf("T212 policy: T900 file_status_stale must be auto_fixable=false (HMAC protected)")
			}
		}
	}
	if !found {
		t.Errorf("file_status_stale_after_complete issue must be detected")
	}

	// T212: file frontmatter must NOT be auto-fixed (HMAC protected). sync
	// only detects; modification is manual (including rotate-hmac-secret).
	data, rerr := os.ReadFile(taskFile)
	if rerr != nil {
		t.Fatalf("file read failed: %v", rerr)
	}
	if !strings.Contains(string(data), "status: in-progress") {
		t.Errorf("T212: original stale frontmatter (in-progress) must be preserved (auto-fix blocked):\n%s", string(data))
	}

	// DB must remain done (prevent SSOT inversion)
	database := db.GetDB()
	var dbStatus string
	_ = database.QueryRow("SELECT status FROM tasks WHERE task_id = 'T900'").Scan(&dbStatus)
	if dbStatus != taskStatusDone {
		t.Errorf("T900 DB status must remain done: got=%s", dbStatus)
	}

	// T212: no file_status_fixed Fix should be recorded (auto-fix blocked).
	for _, fx := range result.Fixes {
		if fx.TaskID == "T900" && fx.Type == fixFileStatusFixed {
			t.Errorf("T212 policy: must block auto-fix but Fix recorded: %+v", fx)
		}
	}
	_ = fixFileStatusFixed // keep import used (no fix applied after T212)
}

// TestSync_T304_CompletedLocationAnomaly is the regression guard for the
// abnormal state where the file lives under completed/ but DB status is
// not done (manual review needed).
func TestSync_T304_CompletedLocationAnomaly(t *testing.T) {
	dir, cleanup := setupTestDir(t)
	defer cleanup()

	completedTaskDir := filepath.Join(dir, "works", "sprints", "completed", "sprint-00", "tasks")
	taskFile := filepath.Join(completedTaskDir, "T901-anomaly-task.md")
	content := `---
id: T901
title: anomaly
type: feature
status: todo
priority: p2
estimate: S
sprint: sprint-00
---
body
`
	writeTaskFile(t, taskFile, content)
	insertDBTask(t, "T901", "in-progress", "sprint-00",
		"works/sprints/completed/sprint-00/tasks/T901-anomaly-task.md")

	result, err := Sync(false)
	if err != nil {
		t.Fatalf("Sync failed: %v", err)
	}

	var found bool
	for _, iss := range result.Issues {
		if iss.TaskID != "T901" || iss.Type != issueCompletedAnomaly {
			continue
		}
		found = true
		if iss.AutoFixable {
			t.Errorf("T901 completed anomaly must be auto_fixable=false")
		}
	}
	if !found {
		t.Errorf("completed_location_status_anomaly issue must be detected")
	}
}

// TestT440_CompletedAnomaly_FrontmatterDone_AutoFix — file status=done in
// completed/ folder + DB=in-progress drift should resolve as auto_fixable=true
// because the frontmatter alone marks the task as completed.
func TestT440_CompletedAnomaly_FrontmatterDone_AutoFix(t *testing.T) {
	dir, cleanup := setupTestDir(t)
	defer cleanup()

	completedTaskDir := filepath.Join(dir, "works", "sprints", "completed", "sprint-00", "tasks")
	taskFile := filepath.Join(completedTaskDir, "T902-frontmatter-done.md")
	baseContent := `---
id: T902
title: frontmatter-done
type: feature
status: done
priority: p2
estimate: S
sprint: sprint-00
---
body done
`
	writeTaskFile(t, taskFile, baseContent)

	insertDBTask(t, "T902", "in-progress", "sprint-00",
		"works/sprints/completed/sprint-00/tasks/T902-frontmatter-done.md")

	result, err := Sync(false)
	if err != nil {
		t.Fatalf("Sync failed: %v", err)
	}

	var foundIssue bool
	for _, iss := range result.Issues {
		if iss.TaskID != "T902" || iss.Type != issueCompletedAnomaly {
			continue
		}
		foundIssue = true
		if !iss.AutoFixable {
			t.Errorf("T902 frontmatter-done completed anomaly should be auto_fixable=true: %+v", iss)
		}
	}
	if !foundIssue {
		t.Errorf("completed_location_status_anomaly issue should be detected")
	}

	var foundFix bool
	for _, fx := range result.Fixes {
		if fx.TaskID == "T902" && fx.Type == "completed_anomaly_fixed" {
			foundFix = true
			if fx.NewStatus != taskStatusDone {
				t.Errorf("T902 fix NewStatus expected done, got=%s", fx.NewStatus)
			}
		}
	}
	if !foundFix {
		t.Errorf("T902 completed_anomaly_fixed should be applied: fixes=%+v", result.Fixes)
	}

	database := db.GetDB()
	var dbStatus string
	_ = database.QueryRow("SELECT status FROM tasks WHERE task_id = 'T902'").Scan(&dbStatus)
	if dbStatus != taskStatusDone {
		t.Errorf("T902 DB status should be reconciled to done: got=%s", dbStatus)
	}
}

// TestT440_CompletedAnomaly_FrontmatterNotDone_ManualReview — when the
// completed-folder file's frontmatter is *not* "done", the issue must remain
// auto_fixable=false (manual review required).
func TestT440_CompletedAnomaly_FrontmatterNotDone_ManualReview(t *testing.T) {
	dir, cleanup := setupTestDir(t)
	defer cleanup()

	completedTaskDir := filepath.Join(dir, "works", "sprints", "completed", "sprint-00", "tasks")
	taskFile := filepath.Join(completedTaskDir, "T903-frontmatter-todo.md")
	// frontmatter status=todo (not done) — drift requiring manual review.
	content := `---
id: T903
title: frontmatter-todo
type: feature
status: todo
priority: p2
estimate: S
sprint: sprint-00
---
body
`
	writeTaskFile(t, taskFile, content)
	insertDBTask(t, "T903", "in-progress", "sprint-00",
		"works/sprints/completed/sprint-00/tasks/T903-frontmatter-todo.md")

	result, err := Sync(false)
	if err != nil {
		t.Fatalf("Sync failed: %v", err)
	}

	// Frontmatter is not "done", so the issue must remain auto_fixable=false.
	for _, iss := range result.Issues {
		if iss.TaskID != "T903" || iss.Type != issueCompletedAnomaly {
			continue
		}
		if iss.AutoFixable {
			t.Errorf("T903 frontmatter-not-done case should be auto_fixable=false (manual review)")
		}
	}
}

// TestT410_SprintFileSync_DryRun verifies that a Sprint file present
// without a DB row is detected as db_missing_sprint (ISS-20260413-001).
func TestT410_SprintFileSync_DryRun(t *testing.T) {
	dir, cleanup := setupTestDir(t)
	defer cleanup()

	sprintDir := filepath.Join(dir, "works", "sprints", "active", "sprint-99")
	if err := os.MkdirAll(filepath.Join(sprintDir, "tasks"), 0o755); err != nil {
		t.Fatal(err)
	}
	sprintMD := filepath.Join(sprintDir, "SPRINT.md")
	content := `---
id: sprint-99
title: "T410 test Sprint"
status: active
goal: "verify Sprint file sync recovery"
created: 2026-04-13
started: 2026-04-13
completed: ~
---

# sprint-99: T410 test Sprint
`
	if err := os.WriteFile(sprintMD, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	result, err := Sync(true)
	if err != nil {
		t.Fatalf("Sync dry-run failed: %v", err)
	}

	var found bool
	for _, iss := range result.Issues {
		if iss.Type == issueDBMissingSprint && iss.SprintID == "sprint-99" {
			found = true
			if !iss.AutoFixable {
				t.Errorf("expected auto_fixable=true")
			}
			break
		}
	}
	if !found {
		t.Fatalf("db_missing_sprint issue not detected: %+v", result.Issues)
	}

	// confirm DB was not modified in dry-run
	var count int
	db.GetDB().QueryRow("SELECT COUNT(*) FROM sprints WHERE sprint_id = 'sprint-99'").Scan(&count)
	if count != 0 {
		t.Errorf("dry-run inserted into DB — destructive side effect")
	}
}

// TestT410_SprintFileSync_Apply verifies that with dry-run=false the
// Sprint is actually INSERTed into DB.
func TestT410_SprintFileSync_Apply(t *testing.T) {
	dir, cleanup := setupTestDir(t)
	defer cleanup()

	sprintDir := filepath.Join(dir, "works", "sprints", "completed", "sprint-77")
	if err := os.MkdirAll(filepath.Join(sprintDir, "tasks"), 0o755); err != nil {
		t.Fatal(err)
	}
	sprintMD := filepath.Join(sprintDir, "SPRINT.md")
	content := `---
id: sprint-77
title: "completed Sprint"
status: completed
goal: "T410 recovery verification"
created: 2026-03-01
started: 2026-03-02
completed: 2026-03-10
---
`
	if err := os.WriteFile(sprintMD, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	result, err := Sync(false)
	if err != nil {
		t.Fatalf("Sync apply failed: %v", err)
	}

	var fixFound bool
	for _, fix := range result.Fixes {
		if fix.Type == fixSprintInserted && fix.SprintID == "sprint-77" {
			fixFound = true
			break
		}
	}
	if !fixFound {
		t.Fatalf("db_sprint_inserted fix not recorded: %+v", result.Fixes)
	}

	var gotTitle, gotStatus, gotGoal string
	err = db.GetDB().QueryRow(
		"SELECT title, status, goal FROM sprints WHERE sprint_id = 'sprint-77'",
	).Scan(&gotTitle, &gotStatus, &gotGoal)
	if err != nil {
		t.Fatalf("inserted sprint row lookup failed: %v", err)
	}
	if gotTitle != "completed Sprint" {
		t.Errorf("title: got %q, want %q", gotTitle, "completed Sprint")
	}
	if gotStatus != "completed" {
		t.Errorf("status: got %q, want %q (decided by folder location)", gotStatus, "completed")
	}
	if gotGoal != "T410 recovery verification" {
		t.Errorf("goal: got %q, want %q", gotGoal, "T410 recovery verification")
	}
}

// TestT475_SprintMetadata_Insert_FullFields verifies that inserting a
// missing sprint from file frontmatter restores started_at and
// completed_at as well (regression guard for ISS-20260413-003 — the T410
// test only covered title/status/goal and missed date-field coverage).
func TestT475_SprintMetadata_Insert_FullFields(t *testing.T) {
	dir, cleanup := setupTestDir(t)
	defer cleanup()

	sprintDir := filepath.Join(dir, "works", "sprints", "completed", "sprint-75")
	if err := os.MkdirAll(filepath.Join(sprintDir, "tasks"), 0o755); err != nil {
		t.Fatal(err)
	}
	content := `---
id: sprint-75
title: "T475 regression-guard Sprint"
status: completed
goal: "metadata-loss check"
created: 2026-03-20
started: 2026-03-21
completed: 2026-03-25
---
`
	if err := os.WriteFile(filepath.Join(sprintDir, "SPRINT.md"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	if _, err := Sync(false); err != nil {
		t.Fatalf("Sync failed: %v", err)
	}

	var title, startedAt, completedAt string
	err := db.GetDB().QueryRow(
		"SELECT title, COALESCE(started_at,''), COALESCE(completed_at,'') FROM sprints WHERE sprint_id = 'sprint-75'",
	).Scan(&title, &startedAt, &completedAt)
	if err != nil {
		t.Fatalf("sprint-75 lookup failed: %v", err)
	}
	if title != "T475 regression-guard Sprint" {
		t.Errorf("title: got %q", title)
	}
	if startedAt != "2026-03-21" {
		t.Errorf("started_at: got %q, want 2026-03-21", startedAt)
	}
	if completedAt != "2026-03-25" {
		t.Errorf("completed_at: got %q, want 2026-03-25", completedAt)
	}
}

// TestT475_SprintMetadata_StaleUpdate verifies that a sprint already in DB
// with title equal to sprint_id and started/completed NULL is updated
// from the file frontmatter (regression guard for the
// applySprintMetadataFix path).
func TestT475_SprintMetadata_StaleUpdate(t *testing.T) {
	dir, cleanup := setupTestDir(t)
	defer cleanup()

	sprintDir := filepath.Join(dir, "works", "sprints", "completed", "sprint-76")
	if err := os.MkdirAll(filepath.Join(sprintDir, "tasks"), 0o755); err != nil {
		t.Fatal(err)
	}
	content := `---
id: sprint-76
title: "the real title that needs to be restored"
status: completed
goal: "stale update"
created: 2026-04-01
started: 2026-04-02
completed: 2026-04-05
---
`
	if err := os.WriteFile(filepath.Join(sprintDir, "SPRINT.md"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	// INSERT a stale row first (title=sprint_id, dates NULL)
	_, err := db.GetDB().Exec(
		`INSERT INTO sprints (sprint_id, title, status, folder_path, goal, started_at, completed_at, created_at, updated_at)
		 VALUES ('sprint-76', 'sprint-76', 'completed', 'works/sprints/completed/sprint-76', '', NULL, NULL, '2026-04-01', '2026-04-01')`,
	)
	if err != nil {
		t.Fatalf("stale row INSERT failed: %v", err)
	}

	if _, err := Sync(false); err != nil {
		t.Fatalf("Sync failed: %v", err)
	}

	var title, startedAt, completedAt string
	err = db.GetDB().QueryRow(
		"SELECT title, COALESCE(started_at,''), COALESCE(completed_at,'') FROM sprints WHERE sprint_id = 'sprint-76'",
	).Scan(&title, &startedAt, &completedAt)
	if err != nil {
		t.Fatalf("sprint-76 lookup failed: %v", err)
	}
	if title != "the real title that needs to be restored" {
		t.Errorf("title not updated: got %q", title)
	}
	if startedAt != "2026-04-02" {
		t.Errorf("started_at not updated: got %q, want 2026-04-02", startedAt)
	}
	if completedAt != "2026-04-05" {
		t.Errorf("completed_at not updated: got %q, want 2026-04-05", completedAt)
	}
}

// TestNormalizeSprintField verifies that various "unassigned" notations
// normalise to the empty string (T176).
func TestNormalizeSprintField(t *testing.T) {
	cases := []struct {
		input string
		want  string
	}{
		{"", ""},
		{"backlog", ""},
		{"~", ""},
		{"null", ""},
		{"<nil>", ""},
		{"sprint-01", "sprint-01"},
		{"sprint-23", "sprint-23"},
		{"custom", "custom"},
	}
	for _, tc := range cases {
		got := normalizeSprintField(tc.input)
		if got != tc.want {
			t.Errorf("normalizeSprintField(%q) = %q, want %q", tc.input, got, tc.want)
		}
	}
}

// TestT529_SprintTableStale_NoSectionSkips — when the BACKLOG.md has no
// Sprint Assignment Status section, the
// backlog_summary_sprint_table_stale issue must NOT fire (avoids the
// non-converging state where rebuildSprintSummaryTable silently no-ops).
func TestT529_SprintTableStale_NoSectionSkips(t *testing.T) {
	dir, cleanup := setupTestDir(t)
	defer cleanup()

	// BACKLOG.md: no Sprint section
	writeTaskFile(t, filepath.Join(dir, "works", "tasks", "BACKLOG.md"), `# Backlog

## Unassigned Tasks

- T001
`)
	// insert an active sprint into DB → enables the stale check
	database := db.GetDB()
	_, err := database.Exec(
		`INSERT INTO sprints (sprint_id, title, status, folder_path, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		"sprint-69", "test sprint", "active", "works/sprints/active/sprint-69", "2026-04-15", "2026-04-15",
	)
	if err != nil {
		t.Fatalf("sprint insert: %v", err)
	}

	result, err := Sync(true)
	if err != nil {
		t.Fatalf("Sync: %v", err)
	}
	for _, iss := range result.Issues {
		if iss.Type == "backlog_summary_sprint_table_stale" {
			t.Errorf("stale issue raised even though Sprint Assignment Status section is missing: %+v", iss)
		}
	}
}

// TestT529_SprintTableStale_SectionPresent_Triggers — when the section
// exists and an active sprint is missing, the stale issue must be
// detected as before.
func TestT529_SprintTableStale_SectionPresent_Triggers(t *testing.T) {
	dir, cleanup := setupTestDir(t)
	defer cleanup()

	writeTaskFile(t, filepath.Join(dir, "works", "tasks", "BACKLOG.md"), `# Backlog

## Unassigned Tasks

- T001

## Sprint Assignment Status

| Sprint | Title | Status | Tasks |
|--------|-------|--------|-------|
`)
	database := db.GetDB()
	_, err := database.Exec(
		`INSERT INTO sprints (sprint_id, title, status, folder_path, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		"sprint-69", "test sprint", "active", "works/sprints/active/sprint-69", "2026-04-15", "2026-04-15",
	)
	if err != nil {
		t.Fatalf("sprint insert: %v", err)
	}

	result, err := Sync(true)
	if err != nil {
		t.Fatalf("Sync: %v", err)
	}
	found := false
	for _, iss := range result.Issues {
		if iss.Type == "backlog_summary_sprint_table_stale" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("stale issue not detected even though section exists and active sprint is missing")
	}
}

// ---------------------------------------------------------------------------
// T356 (Sprint-29, ISS-20260422-005): DB orphan Task detection.
// Detects fully orphan rows where the DB has the row but neither file nor
// BACKLOG.md mentions it.
// ---------------------------------------------------------------------------

// TestT356_DBOrphan_Detect_DryRun — orphan is detected and DB preserved
// in dry-run.
func TestT356_DBOrphan_Detect_DryRun(t *testing.T) {
	dir, cleanup := setupTestDir(t)
	defer cleanup()

	// T100 absent in BACKLOG.md, no file, only in DB → orphan
	writeTaskFile(t, filepath.Join(dir, "works", "tasks", "BACKLOG.md"), `# Backlog

## Unassigned Tasks

- T999
`)
	insertDBTask(t, "T100", "todo", "", "works/tasks/T100-ghost-task.md")

	result, err := Sync(true)
	if err != nil {
		t.Fatalf("Sync: %v", err)
	}

	var orphan *Issue
	for i, iss := range result.Issues {
		if iss.Type == "db_orphan_task" && iss.TaskID == "T100" {
			orphan = &result.Issues[i]
			break
		}
	}
	if orphan == nil {
		t.Fatalf("T100 orphan detection failed (none of the %d issues matched)", len(result.Issues))
	}
	if !orphan.AutoFixable {
		t.Errorf("expected orphan.AutoFixable=true, got false")
	}
	if !strings.Contains(orphan.Message, "orphan") {
		t.Errorf("expected orphan Message to contain 'orphan', got=%q", orphan.Message)
	}

	// dry-run preserves DB
	database := db.GetDB()
	row := database.QueryRow("SELECT task_id FROM tasks WHERE task_id = 'T100'")
	var got string
	if err := row.Scan(&got); err != nil {
		t.Errorf("DB row was deleted under dry-run: %v", err)
	}
}

// TestT356_DBOrphan_Apply_DeletesRow — apply removes the DB row and
// records an audit Fix.
func TestT356_DBOrphan_Apply_DeletesRow(t *testing.T) {
	dir, cleanup := setupTestDir(t)
	defer cleanup()

	writeTaskFile(t, filepath.Join(dir, "works", "tasks", "BACKLOG.md"), `# Backlog

## Unassigned Tasks

`)
	insertDBTask(t, "T200", "todo", "", "works/tasks/T200-ghost.md")

	result, err := Sync(false) // apply
	if err != nil {
		t.Fatalf("Sync(apply): %v", err)
	}

	// verify Fix
	deleted := false
	for _, f := range result.Fixes {
		if f.Type == "db_orphan_task_deleted" && f.TaskID == "T200" {
			deleted = true
			break
		}
	}
	if !deleted {
		t.Errorf("missing db_orphan_task_deleted(T200) Fix record")
	}

	// confirm DB row actually deleted
	database := db.GetDB()
	row := database.QueryRow("SELECT COUNT(*) FROM tasks WHERE task_id = 'T200'")
	var cnt int
	if err := row.Scan(&cnt); err != nil {
		t.Fatalf("lookup failed: %v", err)
	}
	if cnt != 0 {
		t.Errorf("T200 DB row was not deleted (count=%d)", cnt)
	}
}

// TestT356_DBOrphan_NotOrphan_WhenFileExists — when the file exists, it
// is not an orphan (regression guard).
func TestT356_DBOrphan_NotOrphan_WhenFileExists(t *testing.T) {
	dir, cleanup := setupTestDir(t)
	defer cleanup()

	writeTaskFile(t, filepath.Join(dir, "works", "tasks", "T300-exists.md"), `---
title: existing Task
type: chore
status: todo
priority: p2
estimate: S
---
body
`)
	insertDBTask(t, "T300", "todo", "", "works/tasks/T300-exists.md")

	result, err := Sync(true)
	if err != nil {
		t.Fatalf("Sync: %v", err)
	}
	for _, iss := range result.Issues {
		if iss.Type == "db_orphan_task" && iss.TaskID == "T300" {
			t.Errorf("T300 misclassified as orphan even though the file exists")
		}
	}
}

// TestT356_DBOrphan_NotOrphan_WhenInBacklogMD — when listed in
// BACKLOG.md, it is not an orphan (without the file present, it is
// classified separately as backlog_md_missing_file).
func TestT356_DBOrphan_NotOrphan_WhenInBacklogMD(t *testing.T) {
	dir, cleanup := setupTestDir(t)
	defer cleanup()

	writeTaskFile(t, filepath.Join(dir, "works", "tasks", "BACKLOG.md"), `# Backlog

## Unassigned Tasks

| T400 | test | chore | S | p2 |
`)
	insertDBTask(t, "T400", "todo", "", "works/tasks/T400-listed.md")

	result, err := Sync(true)
	if err != nil {
		t.Fatalf("Sync: %v", err)
	}
	for _, iss := range result.Issues {
		if iss.Type == "db_orphan_task" && iss.TaskID == "T400" {
			t.Errorf("T400 misclassified as orphan even though it is listed in BACKLOG.md")
		}
	}
}
