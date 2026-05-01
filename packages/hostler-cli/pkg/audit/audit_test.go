package audit_test

import (
	"path/filepath"
	"testing"

	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/internal/adapters/sqlite"
	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/pkg/audit"
	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/pkg/db"
	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/pkg/store"
)

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

// TestLogEvent_BasicRecord verifies that an event is recorded as expected.
func TestLogEvent_BasicRecord(t *testing.T) {
	initTestDB(t)

	err := audit.LogEvent(
		"task.created",
		"task",
		"T001",
		"claude",
		map[string]any{"title": "Test Task"},
		"sess-001",
	)
	if err != nil {
		t.Fatalf("LogEvent failed: %v", err)
	}

	events, err := audit.QueryEvents(audit.QueryFilter{EntityID: "T001"})
	if err != nil {
		t.Fatalf("QueryEvents failed: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("event count: want 1, got %d", len(events))
	}
	e := events[0]
	if e.EventType != "task.created" {
		t.Errorf("event_type: want task.created, got %s", e.EventType)
	}
	if e.EntityType != "task" {
		t.Errorf("entity_type: want task, got %s", e.EntityType)
	}
	if e.Actor != "claude" {
		t.Errorf("actor: want claude, got %s", e.Actor)
	}
	if e.SessionID != "sess-001" {
		t.Errorf("session_id: want sess-001, got %s", e.SessionID)
	}
	if title, ok := e.Details["title"]; !ok || title != "Test Task" {
		t.Errorf("details.title: want 'Test Task', got %v", e.Details)
	}
}

// TestLogEvent_DefaultActor verifies that an empty actor defaults to "claude".
func TestLogEvent_DefaultActor(t *testing.T) {
	initTestDB(t)

	if err := audit.LogEvent("sprint.created", "sprint", "sprint-01", "", nil, ""); err != nil {
		t.Fatalf("LogEvent failed: %v", err)
	}

	events, err := audit.QueryEvents(audit.QueryFilter{EntityID: "sprint-01"})
	if err != nil {
		t.Fatalf("QueryEvents failed: %v", err)
	}
	if len(events) == 0 {
		t.Fatal("event was not recorded")
	}
	if events[0].Actor != "claude" {
		t.Errorf("default actor: want claude, got %s", events[0].Actor)
	}
}

// TestQueryEvents_Filter verifies that the entity_type/event_type filters work.
func TestQueryEvents_Filter(t *testing.T) {
	initTestDB(t)

	// Insert several events.
	events := []struct {
		eventType, entityType, entityID string
	}{
		{"task.created", "task", "T001"},
		{"task.transitioned", "task", "T001"},
		{"sprint.created", "sprint", "sprint-01"},
		{"task.created", "task", "T002"},
	}
	for _, ev := range events {
		if err := audit.LogEvent(ev.eventType, ev.entityType, ev.entityID, "claude", nil, ""); err != nil {
			t.Fatalf("LogEvent failed: %v", err)
		}
	}

	// entity_type filter.
	taskEvents, err := audit.QueryEvents(audit.QueryFilter{EntityType: "task"})
	if err != nil {
		t.Fatalf("QueryEvents failed: %v", err)
	}
	if len(taskEvents) != 3 {
		t.Errorf("task event count: want 3, got %d", len(taskEvents))
	}

	// event_type filter.
	createdEvents, err := audit.QueryEvents(audit.QueryFilter{EventType: "task.created"})
	if err != nil {
		t.Fatalf("QueryEvents failed: %v", err)
	}
	if len(createdEvents) != 2 {
		t.Errorf("task.created event count: want 2, got %d", len(createdEvents))
	}

	// limit filter.
	limited, err := audit.QueryEvents(audit.QueryFilter{Limit: 2})
	if err != nil {
		t.Fatalf("QueryEvents failed: %v", err)
	}
	if len(limited) != 2 {
		t.Errorf("limit=2 result count: want 2, got %d", len(limited))
	}
}

// TestQueryEvents_NewestFirst verifies that results come back newest-first.
func TestQueryEvents_NewestFirst(t *testing.T) {
	initTestDB(t)

	for i := 1; i <= 3; i++ {
		if err := audit.LogEvent("task.created", "task", "T001", "claude",
			map[string]any{"seq": i}, ""); err != nil {
			t.Fatalf("LogEvent failed: %v", err)
		}
	}

	events, err := audit.QueryEvents(audit.QueryFilter{EntityID: "T001"})
	if err != nil {
		t.Fatalf("QueryEvents failed: %v", err)
	}
	if len(events) != 3 {
		t.Fatalf("event count: want 3, got %d", len(events))
	}
	// Newest-first means IDs descend.
	if events[0].ID < events[1].ID {
		t.Errorf("newest-first ordering broken: events[0].ID=%d < events[1].ID=%d", events[0].ID, events[1].ID)
	}
}

// TestLogEvent_MultipleEntities verifies recording and querying events
// for several entity types.
func TestLogEvent_MultipleEntities(t *testing.T) {
	initTestDB(t)

	events := []struct {
		eventType, entityType, entityID, actor string
	}{
		{"task.created", "task", "T001", "claude"},
		{"task.transitioned", "task", "T001", "claude"},
		{"task.created", "task", "T002", "human:hyunkim"},
		{"sprint.created", "sprint", "sprint-01", "claude"},
	}
	for _, e := range events {
		if err := audit.LogEvent(e.eventType, e.entityType, e.entityID, e.actor, nil, ""); err != nil {
			t.Fatalf("LogEvent failed: %v", err)
		}
	}

	// entity_id filter.
	t001Events, err := audit.QueryEvents(audit.QueryFilter{EntityID: "T001"})
	if err != nil {
		t.Fatalf("QueryEvents(T001) failed: %v", err)
	}
	if len(t001Events) != 2 {
		t.Errorf("T001 event count: want 2, got %d", len(t001Events))
	}

	// event_type filter.
	createdEvents, err := audit.QueryEvents(audit.QueryFilter{EventType: "task.created"})
	if err != nil {
		t.Fatalf("QueryEvents(task.created) failed: %v", err)
	}
	if len(createdEvents) != 2 {
		t.Errorf("task.created event count: want 2, got %d", len(createdEvents))
	}

	// entity_type + event_type combined filter.
	taskCreated, err := audit.QueryEvents(audit.QueryFilter{
		EntityType: "task",
		EventType:  "task.created",
	})
	if err != nil {
		t.Fatalf("combined filter QueryEvents failed: %v", err)
	}
	if len(taskCreated) != 2 {
		t.Errorf("task+task.created event count: want 2, got %d", len(taskCreated))
	}

	// actor filter.
	humanEvents, err := audit.QueryEvents(audit.QueryFilter{Actor: "human:hyunkim"})
	if err != nil {
		t.Fatalf("QueryEvents(actor) failed: %v", err)
	}
	if len(humanEvents) != 1 {
		t.Errorf("human:hyunkim event count: want 1, got %d", len(humanEvents))
	}
}

// TestQueryEvents_All verifies querying every event without a filter.
func TestQueryEvents_All(t *testing.T) {
	initTestDB(t)

	for i := 0; i < 5; i++ {
		if err := audit.LogEvent("task.created", "task", "T001", "claude", nil, ""); err != nil {
			t.Fatalf("LogEvent failed: %v", err)
		}
	}

	events, err := audit.QueryEvents(audit.QueryFilter{})
	if err != nil {
		t.Fatalf("QueryEvents failed: %v", err)
	}
	if len(events) < 5 {
		t.Errorf("total event count: want >= 5, got %d", len(events))
	}
}

// TestLogEvent_Details verifies that the details field is stored and
// restored correctly.
func TestLogEvent_Details(t *testing.T) {
	initTestDB(t)

	details := map[string]any{
		"title": "Test Task",
		"from":  "todo",
		"to":    "in-progress",
	}
	if err := audit.LogEvent("task.transitioned", "task", "T001", "claude", details, "sess-xyz"); err != nil {
		t.Fatalf("LogEvent failed: %v", err)
	}

	events, err := audit.QueryEvents(audit.QueryFilter{EntityID: "T001"})
	if err != nil {
		t.Fatalf("QueryEvents failed: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("event count: want 1, got %d", len(events))
	}

	e := events[0]
	if e.Details["title"] != "Test Task" {
		t.Errorf("details.title: want 'Test Task', got %v", e.Details["title"])
	}
	if e.Details["from"] != "todo" {
		t.Errorf("details.from: want 'todo', got %v", e.Details["from"])
	}
	if e.SessionID != "sess-xyz" {
		t.Errorf("session_id: want 'sess-xyz', got %s", e.SessionID)
	}
}

// TestQueryEvents_Limit verifies that the limit parameter is honoured.
func TestQueryEvents_Limit(t *testing.T) {
	initTestDB(t)

	for i := 0; i < 10; i++ {
		if err := audit.LogEvent("task.created", "task", "T001", "claude", nil, ""); err != nil {
			t.Fatalf("LogEvent failed: %v", err)
		}
	}

	limited, err := audit.QueryEvents(audit.QueryFilter{Limit: 3})
	if err != nil {
		t.Fatalf("QueryEvents(limit=3) failed: %v", err)
	}
	if len(limited) != 3 {
		t.Errorf("limit=3 result count: want 3, got %d", len(limited))
	}
}

// TestLogEvent_EmptyDetails verifies that nil details still record correctly.
func TestLogEvent_EmptyDetails(t *testing.T) {
	initTestDB(t)

	if err := audit.LogEvent("sprint.completed", "sprint", "sprint-01", "claude", nil, ""); err != nil {
		t.Fatalf("LogEvent failed: %v", err)
	}

	events, err := audit.QueryEvents(audit.QueryFilter{EntityID: "sprint-01"})
	if err != nil {
		t.Fatalf("QueryEvents failed: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("event count: want 1, got %d", len(events))
	}
	if len(events[0].Details) != 0 {
		t.Logf("details: %v", events[0].Details)
	}
}

// TestSummarizeEvents_EmptyDB verifies that an empty DB returns an empty map.
func TestSummarizeEvents_EmptyDB(t *testing.T) {
	initTestDB(t)

	result, err := audit.SummarizeEvents("", "")
	if err != nil {
		t.Fatalf("SummarizeEvents failed: %v", err)
	}
	if len(result) != 0 {
		t.Errorf("non-empty result on empty DB: %v", result)
	}
}

// TestSummarizeEvents_Aggregate verifies per-event-type counts.
func TestSummarizeEvents_Aggregate(t *testing.T) {
	initTestDB(t)

	// Record three event kinds.
	audit.LogEvent("task.created", "task", "T001", "claude", nil, "") //nolint:errcheck
	audit.LogEvent("task.created", "task", "T002", "claude", nil, "") //nolint:errcheck
	audit.LogEvent("task.started", "task", "T001", "claude", nil, "") //nolint:errcheck

	result, err := audit.SummarizeEvents("", "")
	if err != nil {
		t.Fatalf("SummarizeEvents failed: %v", err)
	}
	if result["task.created"] != 2 {
		t.Errorf("task.created=%d, want=2", result["task.created"])
	}
	if result["task.started"] != 1 {
		t.Errorf("task.started=%d, want=1", result["task.started"])
	}
}

// TestSummarizeEvents_SinceFilter verifies that since limits to events
// after the boundary.
func TestSummarizeEvents_SinceFilter(t *testing.T) {
	initTestDB(t)

	// Record events (timestamps default to now).
	audit.LogEvent("task.created", "task", "T001", "claude", nil, "") //nolint:errcheck
	audit.LogEvent("task.created", "task", "T002", "claude", nil, "") //nolint:errcheck

	// A future since must yield no results.
	result, err := audit.SummarizeEvents("2099-12-31T00:00:00Z", "")
	if err != nil {
		t.Fatalf("SummarizeEvents failed: %v", err)
	}
	if len(result) != 0 {
		t.Errorf("future since must produce empty result: %v", result)
	}
}

// TestSummarizeEvents_UntilFilter verifies that until limits to events
// before the boundary.
func TestSummarizeEvents_UntilFilter(t *testing.T) {
	initTestDB(t)

	audit.LogEvent("sprint.started", "sprint", "sprint-01", "claude", nil, "") //nolint:errcheck

	// A past until must yield no results.
	result, err := audit.SummarizeEvents("", "2000-01-01T00:00:00Z")
	if err != nil {
		t.Fatalf("SummarizeEvents failed: %v", err)
	}
	if len(result) != 0 {
		t.Errorf("past until must produce empty result: %v", result)
	}
}

// TestQueryEvents_EntityTypeFilter verifies the entity_type filter.
func TestQueryEvents_EntityTypeFilter(t *testing.T) {
	initTestDB(t)

	audit.LogEvent("task.created", "task", "T001", "claude", nil, "")          //nolint:errcheck
	audit.LogEvent("sprint.created", "sprint", "sprint-01", "claude", nil, "") //nolint:errcheck
	audit.LogEvent("task.started", "task", "T001", "claude", nil, "")          //nolint:errcheck

	events, err := audit.QueryEvents(audit.QueryFilter{EntityType: "sprint"})
	if err != nil {
		t.Fatalf("QueryEvents failed: %v", err)
	}
	for _, e := range events {
		if e.EntityType != "sprint" {
			t.Errorf("entity_type=%v, want=sprint", e.EntityType)
		}
	}
	if len(events) != 1 {
		t.Errorf("sprint event count: want 1, got %d", len(events))
	}
}

// TestQueryEvents_EventTypeFilter verifies the event_type filter.
func TestQueryEvents_EventTypeFilter(t *testing.T) {
	initTestDB(t)

	audit.LogEvent("task.created", "task", "T001", "claude", nil, "") //nolint:errcheck
	audit.LogEvent("task.created", "task", "T002", "claude", nil, "") //nolint:errcheck
	audit.LogEvent("task.started", "task", "T001", "claude", nil, "") //nolint:errcheck

	events, err := audit.QueryEvents(audit.QueryFilter{EventType: "task.created"})
	if err != nil {
		t.Fatalf("QueryEvents failed: %v", err)
	}
	if len(events) != 2 {
		t.Errorf("task.created event count: want 2, got %d", len(events))
	}
	for _, e := range events {
		if e.EventType != "task.created" {
			t.Errorf("event_type=%v, want=task.created", e.EventType)
		}
	}
}
