package audit_test

// InMemoryStore PoC migration.
// Three of the SQLite-based integration tests in audit_test.go are
// switched to the InMemory adapter to demonstrate filesystem-dependency
// removal and to measure speed improvements.

import (
	"testing"

	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/internal/adapters/inmemory"
	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/pkg/audit"
	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/pkg/store"
)

// initInMemoryStore initialises a test environment backed by
// InMemoryStore. Provides the same interface as the SQLite-backed
// initTestDB.
func initInMemoryStore(t *testing.T) {
	t.Helper()
	mem := inmemory.NewForTest()
	store.Init(mem, mem)
	t.Cleanup(store.Reset)
}

func TestT716_InMem_LogEvent_BasicRecord(t *testing.T) {
	initInMemoryStore(t)

	err := audit.LogEvent(
		"task.created",
		"task",
		"T001",
		"claude",
		map[string]any{"title": "InMem test"},
		"sess-im-001",
	)
	if err != nil {
		t.Fatalf("LogEvent failed: %v", err)
	}

	events, err := audit.QueryEvents(audit.QueryFilter{EntityID: "T001"})
	if err != nil {
		t.Fatalf("QueryEvents failed: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("event count: expected 1, got %d", len(events))
	}
	e := events[0]
	if e.EventType != "task.created" {
		t.Errorf("event_type=%s", e.EventType)
	}
	if e.Actor != "claude" {
		t.Errorf("actor=%s", e.Actor)
	}
	if title, ok := e.Details["title"]; !ok || title != "InMem test" {
		t.Errorf("details.title: got %v", e.Details)
	}
}

func TestT716_InMem_LogEvent_DefaultActor(t *testing.T) {
	initInMemoryStore(t)

	if err := audit.LogEvent("sprint.created", "sprint", "sprint-01", "", nil, ""); err != nil {
		t.Fatalf("LogEvent failed: %v", err)
	}
	events, err := audit.QueryEvents(audit.QueryFilter{EntityID: "sprint-01"})
	if err != nil || len(events) != 1 {
		t.Fatalf("events=%v err=%v", events, err)
	}
	if events[0].Actor != "claude" {
		t.Errorf("default actor not applied: %s", events[0].Actor)
	}
}

func TestT716_InMem_QueryEvents_FilterQuery(t *testing.T) {
	initInMemoryStore(t)

	// Insert several events.
	_ = audit.LogEvent("task.created", "task", "T100", "claude", nil, "s1")
	_ = audit.LogEvent("task.updated", "task", "T100", "claude", nil, "s1")
	_ = audit.LogEvent("task.created", "task", "T101", "claude", nil, "s1")

	// entity_id=T100 filter.
	events, err := audit.QueryEvents(audit.QueryFilter{EntityID: "T100"})
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 2 {
		t.Errorf("T100 event count: expected 2, got %d", len(events))
	}

	// event_type filter.
	events, err = audit.QueryEvents(audit.QueryFilter{EventType: "task.created"})
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 2 {
		t.Errorf("task.created event count: expected 2, got %d", len(events))
	}
}
