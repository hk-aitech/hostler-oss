package id_test

import (
	"fmt"
	"path/filepath"
	"sync"
	"testing"

	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/internal/adapters/sqlite"
	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/pkg/db"
	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/pkg/id"
	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/pkg/store"
)

// initTestDB initialises a temporary DB for test isolation.
// Sets HSTL_DB_PATH to a file under tmpDir so each test uses an independent DB.
func initTestDB(t *testing.T) {
	t.Helper()
	tmpDir := t.TempDir()
	t.Setenv("HSTL_DB_PATH", filepath.Join(tmpDir, "hstl.db"))
	if err := db.InitDB(); err != nil {
		t.Fatalf("InitDB failed: %v", err)
	}
	sqliteStore := sqlite.New(db.GetDB())
	store.Init(sqliteStore, sqliteStore)
	t.Cleanup(func() {
		store.Reset()
		db.Close()
	})
}

// TestT416_TaskIDNext_SelfHeal_AfterDrift verifies that even when the
// counter is reset to 0 after DB drift, it self-heals by syncing with
// the actual MAX ID in the tasks table and issues the next ID without
// collision (ISS-20260413-005).
func TestT416_TaskIDNext_SelfHeal_AfterDrift(t *testing.T) {
	initTestDB(t)

	database := db.GetDB()
	// Scenario: insert task records directly as if restored from files
	// (T001~T1042) but leave id_counters.task unset — reproduces drift.
	for _, tid := range []string{"T001", "T500", "T999", "T1042"} {
		_, err := database.Exec(
			`INSERT INTO tasks (task_id, title, type, status, priority, estimate, file_path, created_at, updated_at)
			 VALUES (?, 'test', 'chore', 'done', 'p2', 'S', '', datetime('now'), datetime('now'))`,
			tid,
		)
		if err != nil {
			t.Fatalf("seed task %s failed: %v", tid, err)
		}
	}

	// At this point id_counters is empty. The TaskIDNext call must trigger self-heal.
	newID, err := id.TaskIDNext()
	if err != nil {
		t.Fatalf("TaskIDNext failed: %v", err)
	}
	if newID != "T1043" {
		t.Errorf("next ID after self-heal: expected T1043, got %s", newID)
	}

	// Subsequent calls also work.
	newID2, _ := id.TaskIDNext()
	if newID2 != "T1044" {
		t.Errorf("subsequent issue: expected T1044, got %s", newID2)
	}
}

// Verifies that with an empty tasks table, self-heal does not interfere
// and issuing starts from T001 (regression guard).
func TestT416_TaskIDNext_NoRegressionOnEmptyTable(t *testing.T) {
	initTestDB(t)
	newID, err := id.TaskIDNext()
	if err != nil {
		t.Fatalf("TaskIDNext failed: %v", err)
	}
	if newID != "T001" {
		t.Errorf("first ID on empty table: expected T001, got %s", newID)
	}
}

// Verifies that Task IDs are issued sequentially from T001.
func TestTaskIDNext_Sequential(t *testing.T) {
	initTestDB(t)

	id1, err := id.TaskIDNext()
	if err != nil {
		t.Fatalf("TaskIDNext failed: %v", err)
	}
	if id1 != "T001" {
		t.Errorf("first ID: expected T001, got %s", id1)
	}

	id2, err := id.TaskIDNext()
	if err != nil {
		t.Fatalf("TaskIDNext failed: %v", err)
	}
	if id2 != "T002" {
		t.Errorf("second ID: expected T002, got %s", id2)
	}

	id3, err := id.TaskIDNext()
	if err != nil {
		t.Fatalf("TaskIDNext failed: %v", err)
	}
	if id3 != "T003" {
		t.Errorf("third ID: expected T003, got %s", id3)
	}
}

// Verifies that the format remains correct beyond 100 IDs.
func TestTaskIDNext_ZeroPadding(t *testing.T) {
	initTestDB(t)

	// Issue 99 then check the 100th.
	for i := 0; i < 99; i++ {
		if _, err := id.TaskIDNext(); err != nil {
			t.Fatalf("TaskIDNext #%d failed: %v", i+1, err)
		}
	}

	got, err := id.TaskIDNext()
	if err != nil {
		t.Fatalf("TaskIDNext failed: %v", err)
	}
	if got != "T100" {
		t.Errorf("100th ID: expected T100, got %s", got)
	}
}

// Verifies that concurrent calls do not produce duplicate IDs.
func TestTaskIDNext_ConcurrentSafety(t *testing.T) {
	initTestDB(t)

	const n = 20
	results := make([]string, n)
	errs := make([]error, n)
	var wg sync.WaitGroup

	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			results[idx], errs[idx] = id.TaskIDNext()
		}(i)
	}
	wg.Wait()

	for i, err := range errs {
		if err != nil {
			t.Errorf("goroutine %d error: %v", i, err)
		}
	}

	// Check for duplicates.
	seen := make(map[string]bool)
	for _, v := range results {
		if v == "" {
			continue
		}
		if seen[v] {
			t.Errorf("duplicate ID: %s", v)
		}
		seen[v] = true
	}
	if len(seen) != n {
		t.Errorf("issue count mismatch: expected %d, got %d", n, len(seen))
	}
}

// Verifies that an uninitialised DB returns an error.
// Note: this test forces _db to nil so it must run last.
func TestTaskIDNext_DBNotInitialised(t *testing.T) {
	// Close any DB left by other tests so the global is nil.
	db.Close()
	t.Cleanup(db.Close) // clean up if anything reopened

	_, err := id.TaskIDNext()
	if err == nil {
		t.Error("expected error when DB is not initialised, got nil")
	}
}

// Verifies the Task ID format.
func TestTaskIDNext_FormatValidation(t *testing.T) {
	initTestDB(t)

	taskID, err := id.TaskIDNext()
	if err != nil {
		t.Fatalf("TaskIDNext failed: %v", err)
	}

	if len(taskID) < 4 {
		t.Errorf("Task ID too short: %s", taskID)
	}
	if taskID[0] != 'T' {
		t.Errorf("Task ID must start with T: %s", taskID)
	}
	numPart := taskID[1:]
	for _, c := range numPart {
		if c < '0' || c > '9' {
			t.Errorf("non-digit character in Task ID numeric portion: %s", taskID)
			break
		}
	}
}

// Verifies that 50 sequential Task IDs are issued correctly.
func TestTaskIDNext_Up_To_50(t *testing.T) {
	initTestDB(t)

	for i := 1; i <= 50; i++ {
		got, err := id.TaskIDNext()
		if err != nil {
			t.Fatalf("TaskIDNext #%d failed: %v", i, err)
		}
		want := fmt.Sprintf("T%03d", i)
		if got != want {
			t.Errorf("TaskIDNext #%d: expected %s, got %s", i, want, got)
		}
	}
}

// --- helpers ---

var _ = fmt.Sprintf // keeps the fmt import in scope
