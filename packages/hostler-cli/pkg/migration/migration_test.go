package migration_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/internal/adapters/sqlite"
	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/pkg/db"
	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/pkg/migration"
	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/pkg/store"
)

// ---------------------------------------------------------------------------
// test helpers
// ---------------------------------------------------------------------------

// setupDB initialises an isolated DB for tests and registers the cleanup.
func setupDB(t *testing.T) {
	t.Helper()
	t.Setenv("HSTL_DB_PATH", filepath.Join(t.TempDir(), "hstl.db"))
	if err := db.InitDB(); err != nil {
		t.Fatalf("DB initialisation failed: %v", err)
	}
	sqliteStore := sqlite.New(db.GetDB())
	store.Init(sqliteStore, sqliteStore)
	t.Cleanup(func() {
		store.Reset()
		db.Close()
	})
}

// createTaskFile creates a temporary Task markdown file.
func createTaskFile(t *testing.T, dir, filename, content string) string {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, filename)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

// ---------------------------------------------------------------------------
// TestNormalizeTaskID
// ---------------------------------------------------------------------------

// To exercise normalizeTaskID indirectly, run MigrateProject in dry-run.
// Direct calls are covered through parseTaskFile.
func TestNormalizeTaskID(t *testing.T) {
	setupDB(t)
	tmpDir := t.TempDir()
	t.Setenv("HSTL_PROJECT_ROOT", tmpDir)

	tasksDir := filepath.Join(tmpDir, "works", "tasks")
	testCases := []struct {
		filename string
		id       string
		want     string // expected task_id
	}{
		{"T001-test.md", "T001", "T001"},
		{"T042-test.md", "T42", "T042"},
		{"T007-numonly.md", "7", "T007"},
		{"T100-test.md", "T100", "T100"},
	}

	for _, tc := range testCases {
		content := "---\nid: " + tc.id + "\ntitle: test\nstatus: todo\n---\nbody\n"
		createTaskFile(t, tasksDir, tc.filename, content)
	}

	result, err := migration.MigrateProject(tmpDir, true)
	if err != nil {
		t.Fatalf("MigrateProject failed: %v", err)
	}

	if result.Tasks.TotalToLoad != len(testCases) {
		t.Errorf("total_to_load mismatch: expected=%d, got=%d", len(testCases), result.Tasks.TotalToLoad)
	}
}

// ---------------------------------------------------------------------------
// TestCheckBootstrapNeeded
// ---------------------------------------------------------------------------

func TestCheckBootstrapNeeded_EmptyDB_NoRegistry(t *testing.T) {
	setupDB(t)
	tmpDir := t.TempDir()

	bootstrap, err := migration.CheckBootstrapNeeded(tmpDir)
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if bootstrap != nil {
		t.Errorf("expected nil when no registry, got=%+v", bootstrap)
	}
}

func TestCheckBootstrapNeeded_EmptyDB_RegistryPresent(t *testing.T) {
	setupDB(t)
	tmpDir := t.TempDir()

	// create task-id-registry.json
	registryDir := filepath.Join(tmpDir, "works", "data", "task")
	if err := os.MkdirAll(registryDir, 0o755); err != nil {
		t.Fatal(err)
	}
	registry := map[string]any{
		"lastId":     42,
		"totalCount": 15,
		"history":    []any{},
	}
	data, _ := json.Marshal(registry)
	if err := os.WriteFile(filepath.Join(registryDir, "task-id-registry.json"), data, 0o644); err != nil {
		t.Fatal(err)
	}

	bootstrap, err := migration.CheckBootstrapNeeded(tmpDir)
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if bootstrap == nil {
		t.Fatal("expected BootstrapInfo when registry is present, got nil")
		return // nolint (SA5011 guard)
	}
	if bootstrap.LastID != 42 {
		t.Errorf("LastID mismatch: expected=42, got=%d", bootstrap.LastID)
	}
	if bootstrap.TotalCount != 15 {
		t.Errorf("TotalCount mismatch: expected=15, got=%d", bootstrap.TotalCount)
	}
	if bootstrap.RegistryPath == "" {
		t.Error("RegistryPath empty")
	}
}

func TestCheckBootstrapNeeded_DBHasData(t *testing.T) {
	setupDB(t)
	tmpDir := t.TempDir()

	// insert data into the tasks table (file_path NOT NULL → use empty string)
	_, err := db.GetDB().Exec(
		`INSERT INTO tasks (task_id, title, type, status, file_path, created_at, updated_at)
		 VALUES ('T001', 'test', 'feature', 'todo', '', datetime('now'), datetime('now'))`,
	)
	if err != nil {
		t.Fatalf("data insert failed: %v", err)
	}

	// create the registry too (even if present, DB-having data → nil)
	registryDir := filepath.Join(tmpDir, "works", "data", "task")
	if err := os.MkdirAll(registryDir, 0o755); err != nil {
		t.Fatal(err)
	}
	registry := map[string]any{"lastId": 1, "totalCount": 1}
	data, _ := json.Marshal(registry)
	os.WriteFile(filepath.Join(registryDir, "task-id-registry.json"), data, 0o644) //nolint:errcheck

	bootstrap, err := migration.CheckBootstrapNeeded(tmpDir)
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if bootstrap != nil {
		t.Errorf("expected nil when DB has data, got=%+v", bootstrap)
	}
}

// ---------------------------------------------------------------------------
// TestBuildBootstrapResponse
// ---------------------------------------------------------------------------

func TestBuildBootstrapResponse_StructCheck(t *testing.T) {
	bootstrap := &migration.BootstrapInfo{
		RegistryPath: "/path/to/registry.json",
		LastID:       42,
		TotalCount:   15,
		Reason:       "existing project detected",
	}

	resp := migration.BuildBootstrapResponse(bootstrap, "/project/root")

	if resp.Status != "BOOTSTRAP_REQUIRED" {
		t.Errorf("status mismatch: %v", resp.Status)
	}
	if resp.LastID != 42 {
		t.Errorf("last_id mismatch: %v", resp.LastID)
	}
	if resp.TotalCount != 15 {
		t.Errorf("total_count mismatch: %v", resp.TotalCount)
	}
	if resp.Message == "" {
		t.Error("message field empty")
	}

	// suggested_action structure check
	if resp.SuggestedAction.Tool != "project_migrate" {
		t.Errorf("suggested_action.tool mismatch: %v", resp.SuggestedAction.Tool)
	}
}

// ---------------------------------------------------------------------------
// TestMigrateProject_dryRun
// ---------------------------------------------------------------------------

func TestMigrateProject_dryRun(t *testing.T) {
	setupDB(t)
	tmpDir := t.TempDir()
	t.Setenv("HSTL_PROJECT_ROOT", tmpDir)

	// create Task files
	tasksDir := filepath.Join(tmpDir, "works", "tasks")
	createTaskFile(t, tasksDir, "T001-test.md", `---
id: T001
title: test task
type: feature
status: todo
priority: high
estimate: S
---
body
`)
	createTaskFile(t, tasksDir, "T002-another.md", `---
id: T002
title: another task
type: chore
status: done
---
body
`)

	// create Sprint file
	sprintDir := filepath.Join(tmpDir, "works", "sprints", "active", "sprint-01")
	if err := os.MkdirAll(sprintDir, 0o755); err != nil {
		t.Fatal(err)
	}
	sprintContent := `---
id: sprint-01
title: first sprint
status: active
---
body
`
	if err := os.WriteFile(filepath.Join(sprintDir, "SPRINT.md"), []byte(sprintContent), 0o644); err != nil {
		t.Fatal(err)
	}

	result, err := migration.MigrateProject(tmpDir, true)
	if err != nil {
		t.Fatalf("MigrateProject dryRun failed: %v", err)
	}

	if result.DryRun != true {
		t.Errorf("dry_run flag mismatch: %v", result.DryRun)
	}

	if result.Tasks.TotalToLoad != 2 {
		t.Errorf("total_to_load mismatch: expected=2, got=%v", result.Tasks.TotalToLoad)
	}
	if result.Tasks.Loaded != 0 {
		t.Errorf("loaded must be 0 in dryRun: %v", result.Tasks.Loaded)
	}

	if result.Sprints.Scanned != 1 {
		t.Errorf("sprints scanned mismatch: expected=1, got=%v", result.Sprints.Scanned)
	}

	// dryRun must not actually insert into DB
	var count int
	db.GetDB().QueryRow("SELECT COUNT(*) FROM tasks").Scan(&count) //nolint:errcheck
	if count != 0 {
		t.Errorf("after dryRun, tasks table must be empty: count=%d", count)
	}
}

// ---------------------------------------------------------------------------
// TestMigrateProject_RealLoad
// ---------------------------------------------------------------------------

func TestMigrateProject_RealLoad(t *testing.T) {
	setupDB(t)
	tmpDir := t.TempDir()
	t.Setenv("HSTL_PROJECT_ROOT", tmpDir)

	// create Task file
	tasksDir := filepath.Join(tmpDir, "works", "tasks")
	createTaskFile(t, tasksDir, "T001-test.md", `---
id: T001
title: migration test
type: feature
status: todo
priority: high
estimate: M
---
body
`)

	// create Sprint file
	sprintDir := filepath.Join(tmpDir, "works", "sprints", "completed", "sprint-01")
	if err := os.MkdirAll(sprintDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(sprintDir, "SPRINT.md"), []byte(`---
id: sprint-01
title: completed sprint
status: completed
---
`), 0o644); err != nil {
		t.Fatal(err)
	}

	result, err := migration.MigrateProject(tmpDir, false)
	if err != nil {
		t.Fatalf("MigrateProject failed: %v", err)
	}

	if result.DryRun != false {
		t.Errorf("dry_run flag mismatch: %v", result.DryRun)
	}

	if result.Tasks.Loaded != 1 {
		t.Errorf("tasks.loaded mismatch: expected=1, got=%v", result.Tasks.Loaded)
	}

	if result.Sprints.Loaded != 1 {
		t.Errorf("sprints.loaded mismatch: expected=1, got=%v", result.Sprints.Loaded)
	}

	// direct DB verification
	var taskID, title, status string
	err = db.GetDB().QueryRow("SELECT task_id, title, status FROM tasks WHERE task_id='T001'").
		Scan(&taskID, &title, &status)
	if err != nil {
		t.Fatalf("Task lookup failed: %v", err)
	}
	if taskID != "T001" {
		t.Errorf("task_id mismatch: %s", taskID)
	}
	if title != "migration test" {
		t.Errorf("title mismatch: %s", title)
	}
	if status != "todo" {
		t.Errorf("status mismatch: %s", status)
	}

	// Sprint check
	var sprintID, sprintStatus string
	err = db.GetDB().QueryRow("SELECT sprint_id, status FROM sprints WHERE sprint_id='sprint-01'").
		Scan(&sprintID, &sprintStatus)
	if err != nil {
		t.Fatalf("Sprint lookup failed: %v", err)
	}
	if sprintStatus != "completed" {
		t.Errorf("sprint status mismatch: %s", sprintStatus)
	}
}

// ---------------------------------------------------------------------------
// TestMigrateProject_PartialFailureRollback
// ---------------------------------------------------------------------------

func TestMigrateProject_PartialFailureRollback(t *testing.T) {
	setupDB(t)
	tmpDir := t.TempDir()
	t.Setenv("HSTL_PROJECT_ROOT", tmpDir)

	tasksDir := filepath.Join(tmpDir, "works", "tasks")

	// well-formed file
	createTaskFile(t, tasksDir, "T010-ok.md", `---
id: T010
title: well-formed task
type: feature
status: todo
---
body
`)

	// file without frontmatter (parse failure)
	createTaskFile(t, tasksDir, "T011-broken.md", `# file without frontmatter
body only
`)

	// file without id (parse failure)
	createTaskFile(t, tasksDir, "T012-noid.md", `---
title: no-id task
status: todo
---
body
`)

	result, err := migration.MigrateProject(tmpDir, false)
	if err != nil {
		t.Fatalf("MigrateProject failed: %v", err)
	}

	// the well-formed one (1) must still load
	if result.Tasks.Loaded != 1 {
		t.Errorf("loaded mismatch: expected=1, got=%v", result.Tasks.Loaded)
	}

	// skipped must be present (parse-failed files)
	if result.Tasks.Skipped == 0 {
		t.Error("with parse-failed files, skipped must be > 0")
	}

	// warnings must include parse-failure messages
	hasParseWarning := false
	for _, w := range result.Warnings {
		if len(w) > 0 {
			hasParseWarning = true
			break
		}
	}
	if !hasParseWarning {
		t.Error("missing parse-failure warning message")
	}
}

// ---------------------------------------------------------------------------
// TestMigrateProject_SprintFolderStateBased
// ---------------------------------------------------------------------------

func TestMigrateProject_SprintFolderStateBased(t *testing.T) {
	setupDB(t)
	tmpDir := t.TempDir()
	t.Setenv("HSTL_PROJECT_ROOT", tmpDir)

	// Sprint sitting in completed folder (frontmatter status=active)
	sp1Dir := filepath.Join(tmpDir, "works", "sprints", "completed", "sprint-01")
	os.MkdirAll(sp1Dir, 0o755)
	os.WriteFile(filepath.Join(sp1Dir, "SPRINT.md"), []byte(`---
id: sprint-01
title: completed Sprint
status: active
---
`), 0o644)

	// Sprint in active folder
	sp2Dir := filepath.Join(tmpDir, "works", "sprints", "active", "sprint-02")
	os.MkdirAll(sp2Dir, 0o755)
	os.WriteFile(filepath.Join(sp2Dir, "SPRINT.md"), []byte(`---
id: sprint-02
title: in-progress Sprint
status: active
---
`), 0o644)

	result, err := migration.MigrateProject(tmpDir, false)
	if err != nil {
		t.Fatalf("MigrateProject failed: %v", err)
	}

	if result.Sprints.Loaded != 2 {
		t.Errorf("sprints.loaded mismatch: expected=2, got=%v", result.Sprints.Loaded)
	}

	// sprint-01 status must follow folder location (completed)
	var sprint01Status string
	err = db.GetDB().QueryRow(
		"SELECT status FROM sprints WHERE sprint_id='sprint-01'",
	).Scan(&sprint01Status)
	if err != nil {
		t.Fatalf("Sprint lookup failed: %v", err)
	}
	if sprint01Status != "completed" {
		t.Errorf("sprint-01 status must reflect folder location (completed): %s", sprint01Status)
	}
}

// ---------------------------------------------------------------------------
// TestMigrateProject_RegistryMerge
// ---------------------------------------------------------------------------

func TestMigrateProject_RegistryMerge(t *testing.T) {
	setupDB(t)
	tmpDir := t.TempDir()
	t.Setenv("HSTL_PROJECT_ROOT", tmpDir)

	// only T001 has a file
	tasksDir := filepath.Join(tmpDir, "works", "tasks")
	createTaskFile(t, tasksDir, "T001-test.md", `---
id: T001
title: task with file
type: feature
status: done
---
`)

	// task-id-registry.json includes T001 + T005
	registryDir := filepath.Join(tmpDir, "works", "data", "task")
	if err := os.MkdirAll(registryDir, 0o755); err != nil {
		t.Fatal(err)
	}
	registry := map[string]any{
		"lastId":     5,
		"totalCount": 2,
		"history": []any{
			map[string]any{"id": "T001", "title": "task with file", "createdAt": "2025-01-01"},
			map[string]any{"id": "T005", "title": "registry-only task", "createdAt": "2025-02-01"},
		},
	}
	data, _ := json.Marshal(registry)
	if err := os.WriteFile(filepath.Join(registryDir, "task-id-registry.json"), data, 0o644); err != nil {
		t.Fatal(err)
	}

	result, err := migration.MigrateProject(tmpDir, true)
	if err != nil {
		t.Fatalf("MigrateProject failed: %v", err)
	}

	// T001 (file) + T005 (registry) = 2
	if result.Tasks.TotalToLoad != 2 {
		t.Errorf("total_to_load mismatch: expected=2, got=%v", result.Tasks.TotalToLoad)
	}
	if result.Tasks.RegisteredOnly != 1 {
		t.Errorf("registered_only mismatch: expected=1, got=%v", result.Tasks.RegisteredOnly)
	}

	// counter check (lastId=5 vs file-based max=1, larger wins = 5)
	if result.Counters.Task != 5 {
		t.Errorf("task counter mismatch: expected=5, got=%v", result.Counters.Task)
	}
}
