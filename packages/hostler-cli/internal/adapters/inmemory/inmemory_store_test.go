// Package inmemory_test — InMemoryStore regression + SQLite parity tests.
//
// Verification strategy:
//  1. Standalone — InMemory honours the declared contract.
//  2. Contract parity — the same call sequence produces the same result
//     as SQLite. Five scenarios — Task CRUD / Sprint / Audit / Harness
//     / Transition atomic.
//
// race-detector compatible — must not produce concurrency issues under
// `go test -race`.
package inmemory_test

import (
	"database/sql"
	"errors"
	"strings"
	"sync"
	"testing"

	_ "modernc.org/sqlite"

	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/internal/adapters/inmemory"
	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/internal/adapters/sqlite"
	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/internal/ports"
)

// contractSchema is the minimal schema required by the SQLite adapter.
const contractSchema = `
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
	timestamp   TEXT NOT NULL DEFAULT '',
	prev_hash   TEXT,
	event_hash  TEXT
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

// newSQLiteStore returns an in-memory SQLite adapter for tests.
func newSQLiteStore(t *testing.T) ports.GraphStore {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("sqlite open: %v", err)
	}
	for _, stmt := range splitSQL(contractSchema) {
		if _, err := db.Exec(stmt); err != nil {
			t.Fatalf("schema %q: %v", stmt, err)
		}
	}
	t.Cleanup(func() { db.Close() })
	return sqlite.New(db)
}

func splitSQL(sql string) []string {
	parts := []string{}
	for _, s := range strings.Split(sql, ";") {
		trim := strings.TrimSpace(s)
		if trim != "" {
			parts = append(parts, trim)
		}
	}
	return parts
}

// newInMemoryStore returns an InMemory adapter for tests.
func newInMemoryStore(t *testing.T) ports.GraphStore {
	return inmemory.NewForTest()
}

// --- Standalone behaviour --------------------------------------------------

func TestT697_InMemory_CreateTask_Get(t *testing.T) {
	s := newInMemoryStore(t)
	err := s.CreateTask(&ports.TaskRecord{
		TaskID: "T001", Title: "test", Type: "feature", Status: "todo", Priority: "p2",
	})
	if err != nil {
		t.Fatalf("CreateTask: %v", err)
	}
	got, err := s.GetTaskFull("T001")
	if err != nil || got == nil {
		t.Fatalf("GetTaskFull: got=%v err=%v", got, err)
	}
	if got.TaskID != "T001" || got.Title != "test" {
		t.Errorf("got=%+v", got)
	}
}

func TestT697_InMemory_TransitionTaskAtomic_Commit(t *testing.T) {
	s := newInMemoryStore(t)
	_ = s.CreateTask(&ports.TaskRecord{TaskID: "T002", Title: "x", Type: "chore", Status: "todo"})

	called := false
	err := s.TransitionTaskAtomic("T002", "todo", "in-progress", func() (string, error) {
		called = true
		return "works/tasks/T002.md", nil
	})
	if err != nil {
		t.Fatalf("TransitionTaskAtomic: %v", err)
	}
	if !called {
		t.Fatal("fileOp not called")
	}

	details, _ := s.GetTaskDetails("T002")
	if details.Status != "in-progress" || details.FilePath != "works/tasks/T002.md" {
		t.Errorf("got=%+v", details)
	}
}

func TestT697_InMemory_TransitionTaskAtomic_Rollback(t *testing.T) {
	s := newInMemoryStore(t)
	_ = s.CreateTask(&ports.TaskRecord{TaskID: "T003", Title: "x", Type: "chore", Status: "todo", FilePath: "orig.md"})

	sentinel := errors.New("fileOp fail")
	err := s.TransitionTaskAtomic("T003", "todo", "done", func() (string, error) {
		return "", sentinel
	})
	if err == nil {
		t.Fatal("expected error")
	}
	details, _ := s.GetTaskDetails("T003")
	if details.Status != "todo" || details.FilePath != "orig.md" {
		t.Errorf("rollback failed: got=%+v", details)
	}
}

func TestT697_InMemory_Audit_Query(t *testing.T) {
	s := newInMemoryStore(t)
	_ = s.AppendAuditEvent(ports.AuditEvent{EventType: "task.created", EntityType: "task", EntityID: "T001", ActorID: "test"})
	_ = s.AppendAuditEvent(ports.AuditEvent{EventType: "task.completed", EntityType: "task", EntityID: "T001", ActorID: "test"})

	events, err := s.QueryAuditEvents(ports.AuditQueryFilter{EntityID: "T001"})
	if err != nil {
		t.Fatalf("QueryAuditEvents: %v", err)
	}
	if len(events) != 2 {
		t.Errorf("len=%d, want 2", len(events))
	}
}

func TestT697_InMemory_Harness_Lifecycle(t *testing.T) {
	s := newInMemoryStore(t)
	items := []ports.HarnessItemTemplate{
		{ID: "criteria_checked", Description: "Done criteria", Required: true},
		{ID: "build_passed", Description: "Build", Required: true},
	}
	_ = s.EnsureHarnessItems("task", "T001", items)

	count, _ := s.CountHarnessItems("task", "T001")
	if count != 2 {
		t.Errorf("count=%d, want 2", count)
	}

	_ = s.CheckHarnessItem("task", "T001", "build_passed", "go build OK", "claude")
	got, _ := s.GetHarnessItems("task", "T001")
	var buildDone bool
	for _, it := range got {
		if it.ID == "build_passed" && it.Done {
			buildDone = true
		}
	}
	if !buildDone {
		t.Error("build_passed check failed")
	}
}

func TestT697_InMemory_Concurrency(t *testing.T) {
	s := newInMemoryStore(t)
	_ = s.CreateTask(&ports.TaskRecord{TaskID: "T010", Title: "conc", Type: "test", Status: "todo"})

	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, _ = s.GetTaskDetails("T010")
			_, _ = s.QueryAuditEvents(ports.AuditQueryFilter{})
		}()
	}
	wg.Wait()
}

// --- SQLite parity --------------------------------------------------------

// stores names each implementation so the same scenario can be run
// against both.
func stores(t *testing.T) []struct {
	name  string
	store ports.GraphStore
} {
	return []struct {
		name  string
		store ports.GraphStore
	}{
		{"sqlite", newSQLiteStore(t)},
		{"inmemory", newInMemoryStore(t)},
	}
}

func TestT697_Contract_TaskCRUD(t *testing.T) {
	for _, impl := range stores(t) {
		t.Run(impl.name, func(t *testing.T) {
			s := impl.store
			if err := s.CreateTask(&ports.TaskRecord{
				TaskID: "T100", Title: "contract", Type: "feature", Status: "todo", Priority: "p1",
			}); err != nil {
				t.Fatalf("Create: %v", err)
			}
			got, _ := s.GetTaskFull("T100")
			if got == nil || got.Title != "contract" {
				t.Fatalf("Get: %+v", got)
			}
			if err := s.DeleteTask("T100"); err != nil {
				t.Fatalf("Delete: %v", err)
			}
			if g, _ := s.GetTaskFull("T100"); g != nil {
				t.Errorf("query after delete: %+v", g)
			}
		})
	}
}

func TestT697_Contract_SprintSave_Get(t *testing.T) {
	for _, impl := range stores(t) {
		t.Run(impl.name, func(t *testing.T) {
			s := impl.store
			if err := s.SaveSprint(&ports.SprintRecord{
				SprintID: "sprint-contract", Title: "C", Status: "backlog",
			}); err != nil {
				t.Fatalf("SaveSprint: %v", err)
			}
			got, _ := s.GetSprint("sprint-contract")
			if got == nil || got.Title != "C" {
				t.Fatalf("Get: %+v", got)
			}
		})
	}
}

func TestT697_Contract_AuditAppend_Query(t *testing.T) {
	for _, impl := range stores(t) {
		t.Run(impl.name, func(t *testing.T) {
			s := impl.store
			_ = s.AppendAuditEvent(ports.AuditEvent{EventType: "task.created", EntityType: "task", EntityID: "TX", ActorID: "u"})
			events, _ := s.QueryAuditEvents(ports.AuditQueryFilter{EntityID: "TX"})
			if len(events) < 1 {
				t.Errorf("append/query failed: %d", len(events))
			}
		})
	}
}

func TestT697_Contract_TransitionTaskAtomic(t *testing.T) {
	for _, impl := range stores(t) {
		t.Run(impl.name, func(t *testing.T) {
			s := impl.store
			_ = s.CreateTask(&ports.TaskRecord{TaskID: "T200", Status: "todo", Type: "test"})
			err := s.TransitionTaskAtomic("T200", "todo", "in-progress", func() (string, error) {
				return "new-path.md", nil
			})
			if err != nil {
				t.Fatalf("transition: %v", err)
			}
			got, _ := s.GetTaskDetails("T200")
			if got == nil || got.Status != "in-progress" || got.FilePath != "new-path.md" {
				t.Errorf("got=%+v", got)
			}
		})
	}
}

func TestT697_Contract_HarnessEnsure_Check(t *testing.T) {
	for _, impl := range stores(t) {
		t.Run(impl.name, func(t *testing.T) {
			s := impl.store
			_ = s.EnsureHarnessItems("task", "T300", []ports.HarnessItemTemplate{
				{ID: "criteria_checked", Description: "Done", Required: true},
			})
			if c, _ := s.CountHarnessItems("task", "T300"); c != 1 {
				t.Errorf("count=%d", c)
			}
			if err := s.CheckHarnessItem("task", "T300", "criteria_checked", "ok", "u"); err != nil {
				t.Fatalf("Check: %v", err)
			}
		})
	}
}
