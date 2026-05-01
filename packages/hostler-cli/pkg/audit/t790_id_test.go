// Package audit — T790 (Sprint-92) regression tests for the ID field
// mapping.
//
// Follow-up to a downstream T1474 bug: `hstl audit -o json` always returned
// id = 0. The fix adds ID int64 to ports.AuditEvent and includes the id
// column in the SQLite adapter's SELECT/Scan. This test verifies that on a
// LogEvent → QueryEvents round trip, id is populated by the SQLite
// AUTOINCREMENT PK.
package audit_test

import (
	"testing"

	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/pkg/audit"
)

// TestT790_ID_AutoIncrement — a newly logged event must have id > 0.
func TestT790_ID_AutoIncrement(t *testing.T) {
	setupT784DB(t) // reuse T784 setup — same DB init pattern

	if err := audit.LogEvent("task.created", "task", "T790FX1", "claude", nil, ""); err != nil {
		t.Fatalf("LogEvent failed: %v", err)
	}

	events, err := audit.QueryEvents(audit.QueryFilter{EntityID: "T790FX1"})
	if err != nil {
		t.Fatalf("QueryEvents failed: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("event count: expected 1, got %d", len(events))
	}
	if events[0].ID <= 0 {
		t.Errorf("ID: expected > 0, got %d — likely missing SQLite adapter Scan extension", events[0].ID)
	}
}

// TestT790_ID_Unique — IDs from sequential LogEvent calls must increase
// without duplicates.
func TestT790_ID_Unique(t *testing.T) {
	setupT784DB(t)

	for i := 0; i < 3; i++ {
		if err := audit.LogEvent("task.created", "task", "T790UNIQ", "claude", nil, ""); err != nil {
			t.Fatalf("LogEvent %d failed: %v", i, err)
		}
	}

	events, err := audit.QueryEvents(audit.QueryFilter{EntityID: "T790UNIQ"})
	if err != nil {
		t.Fatalf("QueryEvents failed: %v", err)
	}
	if len(events) != 3 {
		t.Fatalf("event count: expected 3, got %d", len(events))
	}

	ids := map[int64]bool{}
	for _, e := range events {
		if e.ID <= 0 {
			t.Errorf("ID <= 0: %d", e.ID)
		}
		if ids[e.ID] {
			t.Errorf("duplicate ID: %d", e.ID)
		}
		ids[e.ID] = true
	}
}

// TestT790_ID_DifferentEntities — events from different entities must each
// get a unique ID.
func TestT790_ID_DifferentEntities(t *testing.T) {
	setupT784DB(t)

	_ = audit.LogEvent("task.created", "task", "T790A", "claude", nil, "")
	_ = audit.LogEvent("task.created", "task", "T790B", "claude", nil, "")
	_ = audit.LogEvent("sprint.created", "sprint", "sprint-test", "claude", nil, "")

	evA, _ := audit.QueryEvents(audit.QueryFilter{EntityID: "T790A"})
	evB, _ := audit.QueryEvents(audit.QueryFilter{EntityID: "T790B"})
	evS, _ := audit.QueryEvents(audit.QueryFilter{EntityID: "sprint-test"})

	if len(evA) == 0 || len(evB) == 0 || len(evS) == 0 {
		t.Fatalf("event query failed: A=%d B=%d S=%d", len(evA), len(evB), len(evS))
	}
	if evA[0].ID == evB[0].ID || evA[0].ID == evS[0].ID || evB[0].ID == evS[0].ID {
		t.Errorf("ID collision across entities: A=%d B=%d S=%d", evA[0].ID, evB[0].ID, evS[0].ID)
	}
	// Verify monotonic increase (Sprint is logged last).
	if !(evA[0].ID < evB[0].ID && evB[0].ID < evS[0].ID) {
		t.Errorf("ID monotonic ordering violated: A=%d B=%d S=%d", evA[0].ID, evB[0].ID, evS[0].ID)
	}
}
