package task_test

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/internal/adapters/sqlite"
	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/pkg/db"
	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/pkg/store"
	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/pkg/task"
)

// setupBenchDB initialises an isolated DB and project root for benchmarks.
func setupBenchDB(b *testing.B) (tmpDir string, cleanup func()) {
	b.Helper()
	tmpDir = b.TempDir()
	os.Setenv("HSTL_DB_PATH", filepath.Join(tmpDir, "hstl.db")) //nolint:errcheck
	os.Setenv("HSTL_PROJECT_ROOT", tmpDir)                       //nolint:errcheck

	for _, dir := range []string{
		filepath.Join(tmpDir, "works", "tasks"),
		filepath.Join(tmpDir, "works", "sprints", "backlog"),
		filepath.Join(tmpDir, "works", "sprints", "active"),
		filepath.Join(tmpDir, "works", "data", "task"),
	} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			b.Fatalf("failed to create directory: %v", err)
		}
	}

	if err := db.InitDB(); err != nil {
		b.Fatalf("DB initialisation failed: %v", err)
	}
	sqliteStore := sqlite.New(db.GetDB())
	store.Init(sqliteStore, sqliteStore)

	cleanup = func() {
		store.Reset()
		db.Close()
		os.Unsetenv("HSTL_PROJECT_ROOT") //nolint:errcheck
		os.Unsetenv("HSTL_DB_PATH")      //nolint:errcheck
	}
	return tmpDir, cleanup
}

// BenchmarkTaskCreate measures the performance of task.Create.
func BenchmarkTaskCreate(b *testing.B) {
	_, cleanup := setupBenchDB(b)
	defer cleanup()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		title := fmt.Sprintf("benchmark task %d", i)
		if _, err := task.Create(title, "feature", "", "p2", "M", "", nil); err != nil {
			b.Fatalf("Create failed: %v", err)
		}
	}
}

// BenchmarkTaskList measures the performance of task.List (10 Tasks
// pre-inserted).
func BenchmarkTaskList(b *testing.B) {
	_, cleanup := setupBenchDB(b)
	defer cleanup()

	// pre-insert 10 test rows
	for i := 0; i < 10; i++ {
		title := fmt.Sprintf("list test task %d", i)
		if _, err := task.Create(title, "feature", "", "p2", "M", "", nil); err != nil {
			b.Fatalf("seed Create failed: %v", err)
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := task.List(nil, nil); err != nil {
			b.Fatalf("List failed: %v", err)
		}
	}
}

// BenchmarkTaskGet measures the performance of task.Get (1 Task pre-created).
func BenchmarkTaskGet(b *testing.B) {
	_, cleanup := setupBenchDB(b)
	defer cleanup()

	raw, err := task.Create("get benchmark task", "feature", "", "p2", "M", "", nil)
	if err != nil {
		b.Fatalf("seed Create failed: %v", err)
	}
	result := asMap(raw)
	taskID, ok := result["task_id"].(string)
	if !ok || taskID == "" {
		b.Fatalf("failed to extract task_id: %v", result)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := task.Get(taskID); err != nil {
			b.Fatalf("Get failed: %v", err)
		}
	}
}

// BenchmarkTaskNext measures the performance of task.Next (5 Tasks with
// dependencies pre-inserted, some marked done).
func BenchmarkTaskNext(b *testing.B) {
	_, cleanup := setupBenchDB(b)
	defer cleanup()

	// insert 5 Tasks — mark the first 2 as done
	var taskIDs []string
	for i := 0; i < 5; i++ {
		title := fmt.Sprintf("Next benchmark task %d", i)
		raw, err := task.Create(title, "feature", "", "p2", "M", "", nil)
		if err != nil {
			b.Fatalf("seed Create failed: %v", err)
		}
		result := asMap(raw)
		if id, ok := result["task_id"].(string); ok {
			taskIDs = append(taskIDs, id)
		}
	}

	// update the first 2 to done status
	database := db.GetDB()
	for _, id := range taskIDs[:2] {
		if _, err := database.Exec(`UPDATE tasks SET status='done' WHERE task_id=?`, id); err != nil {
			b.Fatalf("failed to update Task status: %v", err)
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := task.Next(); err != nil {
			b.Fatalf("Next failed: %v", err)
		}
	}
}
