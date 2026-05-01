// Package sqlite_test — benchmark.
//
// Same scenario as the InMemory benchmark
// (inmemory/benchmark_test.go). Differences in measured values
// quantify the impact of switching adapters.
//
// Run: cd cli && go test -bench=. -benchmem ./internal/adapters/sqlite/...
package sqlite_test

import (
	"database/sql"
	"fmt"
	"strings"
	"testing"

	_ "modernc.org/sqlite"

	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/internal/adapters/sqlite"
	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/internal/ports"
)

// benchmarkSeedCount is the number of seed Tasks — matches inmemory.
const benchmarkSeedCount = 100

// benchSchema is the minimal schema shared with the contract test.
// (Keep in sync with inmemory_store_test.go contractSchema.)
const benchSchema = `
CREATE TABLE tasks (
	task_id      TEXT PRIMARY KEY,
	title        TEXT NOT NULL DEFAULT '',
	type         TEXT NOT NULL DEFAULT '',
	sprint       TEXT,
	status       TEXT NOT NULL DEFAULT 'todo',
	priority     TEXT NOT NULL DEFAULT 'p2',
	estimate     TEXT NOT NULL DEFAULT '',
	file_path    TEXT NOT NULL DEFAULT '',
	depends_on   TEXT NOT NULL DEFAULT '',
	work_ticket  TEXT,
	created_at   TEXT NOT NULL DEFAULT '',
	updated_at   TEXT NOT NULL DEFAULT ''
);
CREATE TABLE sprints (
	sprint_id    TEXT PRIMARY KEY,
	title        TEXT NOT NULL DEFAULT '',
	status       TEXT NOT NULL DEFAULT 'planning',
	folder_path  TEXT,
	started_at   TEXT,
	completed_at TEXT,
	goal         TEXT,
	created_at   TEXT NOT NULL DEFAULT '',
	updated_at   TEXT NOT NULL DEFAULT ''
);
CREATE TABLE audit_events (
	id          INTEGER PRIMARY KEY AUTOINCREMENT,
	event_type  TEXT NOT NULL DEFAULT '',
	entity_type TEXT NOT NULL DEFAULT '',
	entity_id   TEXT NOT NULL DEFAULT '',
	actor       TEXT NOT NULL DEFAULT '',
	details     TEXT NOT NULL DEFAULT '',
	session_id  TEXT NOT NULL DEFAULT '',
	timestamp   TEXT NOT NULL DEFAULT ''
);
CREATE TABLE harness_items (
	id          INTEGER PRIMARY KEY AUTOINCREMENT,
	entity_type TEXT NOT NULL,
	entity_id   TEXT NOT NULL,
	item_id     TEXT NOT NULL,
	item_name   TEXT NOT NULL DEFAULT '',
	required    INTEGER NOT NULL DEFAULT 1,
	done        INTEGER NOT NULL DEFAULT 0,
	evidence    TEXT NOT NULL DEFAULT '',
	checked_at  TEXT NOT NULL DEFAULT '',
	checked_by  TEXT NOT NULL DEFAULT ''
);
CREATE TABLE counters (counter_key TEXT PRIMARY KEY, value INTEGER NOT NULL DEFAULT 0);
CREATE TABLE project_meta (key TEXT PRIMARY KEY, value TEXT);
CREATE TABLE work_tickets (task_id TEXT PRIMARY KEY, ticket TEXT NOT NULL);
CREATE TABLE context_acks (ticket TEXT, hash TEXT, size INTEGER, created_at TEXT, PRIMARY KEY(ticket, hash));
`

func newBenchStore(b *testing.B) *sqlite.SQLiteStore {
	b.Helper()
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		b.Fatalf("sqlite open: %v", err)
	}
	for _, stmt := range strings.Split(benchSchema, ";") {
		if strings.TrimSpace(stmt) == "" {
			continue
		}
		if _, err := db.Exec(stmt); err != nil {
			b.Fatalf("schema %q: %v", stmt, err)
		}
	}
	b.Cleanup(func() { db.Close() })
	return sqlite.New(db)
}

func seedTasks(b *testing.B, s *sqlite.SQLiteStore, sprint string, count int) {
	b.Helper()
	for i := 0; i < count; i++ {
		rec := &ports.TaskRecord{
			TaskID:   fmt.Sprintf("B%04d", i),
			Title:    "bench task",
			Type:     "chore",
			Status:   "todo",
			Priority: "p3",
			Estimate: "XS",
			Sprint:   sprint,
			FilePath: fmt.Sprintf("works/tasks/B%04d.md", i),
		}
		if err := s.CreateTask(rec); err != nil {
			b.Fatalf("seed CreateTask: %v", err)
		}
	}
}

// BenchmarkSQLite_CreateTask — repeatedly create a single Task on an empty store.
func BenchmarkSQLite_CreateTask(b *testing.B) {
	s := newBenchStore(b)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rec := &ports.TaskRecord{
			TaskID:   fmt.Sprintf("T%08d", i),
			Title:    "bench",
			Type:     "chore",
			Status:   "todo",
			Priority: "p3",
			Estimate: "XS",
			FilePath: fmt.Sprintf("t%d.md", i),
		}
		if err := s.CreateTask(rec); err != nil {
			b.Fatalf("CreateTask: %v", err)
		}
	}
}

// BenchmarkSQLite_ListTasks — seed 100 Tasks then call ListTasks repeatedly.
func BenchmarkSQLite_ListTasks(b *testing.B) {
	s := newBenchStore(b)
	seedTasks(b, s, "sprint-bench", benchmarkSeedCount)
	sprintFilter := "sprint-bench"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := s.ListTasks(&sprintFilter, nil); err != nil {
			b.Fatalf("ListTasks: %v", err)
		}
	}
}

// BenchmarkSQLite_AggregateSprintProgress — seed 1 Sprint + 100 Tasks, then loop.
func BenchmarkSQLite_AggregateSprintProgress(b *testing.B) {
	s := newBenchStore(b)
	if err := s.SaveSprint(&ports.SprintRecord{
		SprintID: "sprint-bench",
		Title:    "bench sprint",
		Status:   "active",
	}); err != nil {
		b.Fatalf("SaveSprint: %v", err)
	}
	seedTasks(b, s, "sprint-bench", benchmarkSeedCount)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := s.AggregateSprintProgress("sprint-bench"); err != nil {
			b.Fatalf("AggregateSprintProgress: %v", err)
		}
	}
}
