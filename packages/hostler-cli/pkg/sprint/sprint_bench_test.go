package sprint_test

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/internal/adapters/sqlite"
	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/pkg/db"
	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/pkg/sprint"
	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/pkg/store"
)

// setupBenchDB initialises an isolated DB and project root for benchmarks.
func setupBenchDB(b *testing.B) (tmpDir string, cleanup func()) {
	b.Helper()
	tmpDir = b.TempDir()
	os.Setenv("HSTL_DB_PATH", filepath.Join(tmpDir, "hstl.db")) //nolint:errcheck
	os.Setenv("HSTL_PROJECT_ROOT", tmpDir)                       //nolint:errcheck

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

// BenchmarkSprintAggregateProgress measures the performance of
// sprint.AggregateProgress. 1 Sprint + 10 Tasks (mix of done / in-progress /
// todo) are pre-inserted.
func BenchmarkSprintAggregateProgress(b *testing.B) {
	_, cleanup := setupBenchDB(b)
	defer cleanup()

	const sprintID = "sprint-bench-01"
	if err := sprint.SaveToDB(sprintID, "benchmark sprint", "goal", "active", "path/"+sprintID, nil, nil); err != nil {
		b.Fatalf("SaveToDB failed: %v", err)
	}

	statuses := []string{"done", "done", "done", "done", "in-progress", "in-progress", "todo", "todo", "todo", "todo"}
	database := db.GetDB()
	for i, status := range statuses {
		_, err := database.Exec(
			`INSERT INTO tasks (task_id, title, type, sprint, status, priority, estimate, file_path, created_at, updated_at)
			 VALUES (?, 'benchmark task', 'feature', ?, ?, 'p2', 'M', 'path', date('now'), date('now'))`,
			fmt.Sprintf("TB%03d", i+1), sprintID, status,
		)
		if err != nil {
			b.Fatalf("Task insert failed: %v", err)
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := sprint.AggregateProgress(sprintID); err != nil {
			b.Fatalf("AggregateProgress failed: %v", err)
		}
	}
}

// BenchmarkSprintListFromDB measures the performance of sprint.ListFromDB
// (3 Sprints pre-inserted).
func BenchmarkSprintListFromDB(b *testing.B) {
	_, cleanup := setupBenchDB(b)
	defer cleanup()

	for i := 1; i <= 3; i++ {
		sid := fmt.Sprintf("sprint-bench-%02d", i)
		if err := sprint.SaveToDB(sid, fmt.Sprintf("sprint %d", i), "goal", "backlog", "path/"+sid, nil, nil); err != nil {
			b.Fatalf("SaveToDB failed: %v", err)
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := sprint.ListFromDB(""); err != nil {
			b.Fatalf("ListFromDB failed: %v", err)
		}
	}
}
