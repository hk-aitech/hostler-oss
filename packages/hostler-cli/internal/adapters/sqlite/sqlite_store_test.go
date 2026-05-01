// Package sqlite_test contains integration tests for SQLiteStore.
// Tests run against an in-memory SQLite database.
package sqlite_test

import (
	"database/sql"
	"testing"

	_ "modernc.org/sqlite"

	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/internal/adapters/sqlite"
	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/internal/ports"
)

// testSchemaStatements is the minimal test schema split into individual
// statements. modernc.org/sqlite's Exec processes only one statement at
// a time, so multiple statements must run independently.
var testSchemaStatements = []string{
	`CREATE TABLE IF NOT EXISTS tasks (
		task_id      TEXT PRIMARY KEY,
		title        TEXT NOT NULL DEFAULT '',
		type         TEXT NOT NULL DEFAULT 'feature',
		sprint       TEXT,
		status       TEXT NOT NULL DEFAULT 'todo',
		priority     TEXT NOT NULL DEFAULT 'p2',
		estimate     TEXT NOT NULL DEFAULT '',
		file_path    TEXT NOT NULL DEFAULT '',
		depends_on   TEXT NOT NULL DEFAULT '',
		work_ticket  TEXT,
		created_at   TEXT NOT NULL DEFAULT '',
		updated_at   TEXT NOT NULL DEFAULT ''
	)`,
	`CREATE TABLE IF NOT EXISTS sprints (
		sprint_id    TEXT PRIMARY KEY,
		title        TEXT NOT NULL DEFAULT '',
		status       TEXT NOT NULL DEFAULT 'planning',
		folder_path  TEXT,
		started_at   TEXT,
		completed_at TEXT,
		goal         TEXT,
		created_at   TEXT NOT NULL DEFAULT '',
		updated_at   TEXT NOT NULL DEFAULT ''
	)`,
	`CREATE TABLE IF NOT EXISTS audit_events (
		id          INTEGER PRIMARY KEY AUTOINCREMENT,
		event_type  TEXT NOT NULL DEFAULT '',
		entity_type TEXT NOT NULL DEFAULT '',
		entity_id   TEXT NOT NULL DEFAULT '',
		actor       TEXT NOT NULL DEFAULT '',
		details     TEXT NOT NULL DEFAULT '',
		session_id  TEXT NOT NULL DEFAULT '',
		timestamp   TEXT NOT NULL DEFAULT ''
	)`,
	`CREATE TABLE IF NOT EXISTS harness_items (
		id          INTEGER PRIMARY KEY AUTOINCREMENT,
		entity_type TEXT NOT NULL,
		entity_id   TEXT NOT NULL,
		item_id     TEXT NOT NULL,
		item_name   TEXT NOT NULL DEFAULT '',
		required    INTEGER NOT NULL DEFAULT 1,
		done        INTEGER NOT NULL DEFAULT 0,
		evidence    TEXT NOT NULL DEFAULT '',
		checked_at  TEXT NOT NULL DEFAULT '',
		checked_by  TEXT NOT NULL DEFAULT '',
		UNIQUE(entity_type, entity_id, item_id)
	)`,
	`CREATE TABLE IF NOT EXISTS harness_templates (
		id           INTEGER PRIMARY KEY AUTOINCREMENT,
		template_key TEXT NOT NULL UNIQUE,
		config       TEXT NOT NULL DEFAULT '[]'
	)`,
	`CREATE TABLE IF NOT EXISTS id_counters (
		counter_key   TEXT PRIMARY KEY,
		current_value INTEGER NOT NULL DEFAULT 0,
		updated_at    TEXT NOT NULL DEFAULT ''
	)`,
	`CREATE TABLE IF NOT EXISTS project_meta (
		key   TEXT PRIMARY KEY,
		value TEXT NOT NULL DEFAULT ''
	)`,
}

func openTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("DB open: %v", err)
	}
	for _, stmt := range testSchemaStatements {
		if _, execErr := db.Exec(stmt); execErr != nil {
			t.Fatalf("schema: %v", execErr)
		}
	}
	t.Cleanup(func() { db.Close() })
	return db
}

// TestNew verifies SQLiteStore construction and the compile-time
// interface assertions.
func TestNew(t *testing.T) {
	db := openTestDB(t)
	store := sqlite.New(db)

	// Runtime type assertion confirms the interface implementation.
	var _ ports.GraphStore = store
	var _ ports.IDStore = store

	if store == nil {
		t.Fatal("New returned nil")
	}
}

// TestCreateTask_And_GetTaskFull tests task creation followed by a full read.
func TestCreateTask_And_GetTaskFull(t *testing.T) {
	store := sqlite.New(openTestDB(t))

	task := &ports.TaskRecord{
		TaskID:    "T001",
		Title:     "test task",
		Type:      "feature",
		Status:    "todo",
		Priority:  "p1",
		Estimate:  "2h",
		Sprint:    "S01",
		FilePath:  "works/tasks/T001.md",
		DependsOn: "",
		CreatedAt: "2026-01-01T00:00:00Z",
	}

	if err := store.CreateTask(task); err != nil {
		t.Fatalf("CreateTask: %v", err)
	}

	got, err := store.GetTaskFull("T001")
	if err != nil {
		t.Fatalf("GetTaskFull: %v", err)
	}

	if got.TaskID != "T001" {
		t.Errorf("TaskID: want T001, got %s", got.TaskID)
	}
	if got.Title != "test task" {
		t.Errorf("Title: want 'test task', got %s", got.Title)
	}
	if got.Status != "todo" {
		t.Errorf("Status: want todo, got %s", got.Status)
	}
}

// TestSaveSprint_And_GetSprint tests sprint save followed by read.
func TestSaveSprint_And_GetSprint(t *testing.T) {
	store := sqlite.New(openTestDB(t))

	sprint := &ports.SprintRecord{
		SprintID:   "S76",
		Title:      "sprint 76",
		Status:     "active",
		FolderPath: "works/sprints/active/sprint-76",
		Goal:       "adapter implementation",
		CreatedAt:  "2026-01-01T00:00:00Z",
		UpdatedAt:  "2026-01-01T00:00:00Z",
	}

	if err := store.SaveSprint(sprint); err != nil {
		t.Fatalf("SaveSprint: %v", err)
	}

	got, err := store.GetSprint("S76")
	if err != nil {
		t.Fatalf("GetSprint: %v", err)
	}

	if got.SprintID != "S76" {
		t.Errorf("SprintID: want S76, got %s", got.SprintID)
	}
	if got.Title != "sprint 76" {
		t.Errorf("Title: want 'sprint 76', got %s", got.Title)
	}
	if got.Status != "active" {
		t.Errorf("Status: want active, got %s", got.Status)
	}

	// upsert verification
	sprint.Status = "completed"
	if err := store.SaveSprint(sprint); err != nil {
		t.Fatalf("SaveSprint upsert: %v", err)
	}
	got2, _ := store.GetSprint("S76")
	if got2.Status != "completed" {
		t.Errorf("upsert Status: want completed, got %s", got2.Status)
	}
}

// TestNextCounter verifies that the counter increments atomically.
func TestNextCounter(t *testing.T) {
	store := sqlite.New(openTestDB(t))

	seq1, err := store.NextCounter("test_key")
	if err != nil {
		t.Fatalf("NextCounter first: %v", err)
	}
	if seq1 != 1 {
		t.Errorf("first counter: want 1, got %d", seq1)
	}

	seq2, err := store.NextCounter("test_key")
	if err != nil {
		t.Fatalf("NextCounter second: %v", err)
	}
	if seq2 != 2 {
		t.Errorf("second counter: want 2, got %d", seq2)
	}

	// Different counter keys are independent.
	seq3, err := store.NextCounter("other_key")
	if err != nil {
		t.Fatalf("NextCounter other_key: %v", err)
	}
	if seq3 != 1 {
		t.Errorf("other_key counter: want 1, got %d", seq3)
	}
}

// TestGetSprint_NotFound verifies that querying a missing sprint returns an error.
func TestGetSprint_NotFound(t *testing.T) {
	store := sqlite.New(openTestDB(t))
	_, err := store.GetSprint("nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent sprint, got nil")
	}
}

// TestTransitionTaskStatus_Blocked verifies that the transition is
// rejected when the expected status does not match.
func TestTransitionTaskStatus_Blocked(t *testing.T) {
	store := sqlite.New(openTestDB(t))

	task := &ports.TaskRecord{
		TaskID: "T002", Title: "transition test", Type: "feature",
		Status: "todo", Priority: "p2", CreatedAt: "2026-01-01T00:00:00Z",
	}
	if err := store.CreateTask(task); err != nil {
		t.Fatalf("CreateTask: %v", err)
	}

	// Current status is todo; trying in-progress → done should fail.
	err := store.TransitionTaskStatus("T002", "in-progress", "done")
	if err == nil {
		t.Fatal("expected error for invalid transition, got nil")
	}

	// todo → in-progress is a valid transition.
	if err := store.TransitionTaskStatus("T002", "todo", "in-progress"); err != nil {
		t.Fatalf("valid transition failed: %v", err)
	}
}

// TestHealAndGetNextTaskID verifies that the sequence is correct after recovery.
func TestHealAndGetNextTaskID(t *testing.T) {
	store := sqlite.New(openTestDB(t))

	id, err := store.HealAndGetNextTaskID()
	if err != nil {
		t.Fatalf("HealAndGetNextTaskID: %v", err)
	}
	if id != "T001" {
		t.Errorf("first ID: want T001, got %s", id)
	}
}
