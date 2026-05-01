// Package audit — T784 (Sprint-91) regression tests for Timestamp mapping.
//
// Background: a downstream T1474 bug. `hstl audit --entity-id T1474 -o json`
// returned an empty Timestamp field, making time-series analysis impossible.
// The cause was a missing ports.AuditEvent.Timestamp → audit.Event.Timestamp
// mapping in QueryEvents. This test checks that Timestamp is populated on a
// LogEvent → QueryEvents round trip.
package audit_test

import (
	"strings"
	"testing"

	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/internal/adapters/sqlite"
	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/pkg/audit"
	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/pkg/db"
	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/pkg/store"
)

// setupT784DB initializes an isolated DB for T784 tests.
// Reuses the setupDB pattern from sprint_test.go.
func setupT784DB(t *testing.T) {
	t.Helper()
	tmpDir := t.TempDir()
	t.Setenv("HSTL_DB_PATH", tmpDir+"/hstl.db")
	t.Setenv("HSTL_PROJECT_ROOT", tmpDir)
	if err := db.InitDB(); err != nil {
		t.Fatalf("DB init failed: %v", err)
	}
	s := sqlite.New(db.GetDB())
	store.Init(s, s)
	t.Cleanup(func() {
		store.Reset()
		db.Close()
	})
}

// TestT784_Timestamp_Populated — an event recorded via LogEvent and queried
// via QueryEvents must come back with a non-empty Timestamp.
func TestT784_Timestamp_Populated(t *testing.T) {
	setupT784DB(t)

	if err := audit.LogEvent("task.created", "task", "T784FX", "claude",
		map[string]any{"title": "fixture"}, ""); err != nil {
		t.Fatalf("LogEvent failed: %v", err)
	}

	events, err := audit.QueryEvents(audit.QueryFilter{EntityID: "T784FX"})
	if err != nil {
		t.Fatalf("QueryEvents failed: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("event count: expected 1, got %d", len(events))
	}
	if events[0].Timestamp == "" {
		t.Error("Timestamp is empty — T784 mapping regression")
	}
}

// TestT784_Timestamp_ISOFormat — accepts the "YYYY-MM-DD HH:MM:SS" output
// from the DB DEFAULT datetime('now') (which may not be RFC3339).
// This test checks "non-empty + contains digits + contains separators"
// rather than the exact format.
func TestT784_Timestamp_ISOFormat(t *testing.T) {
	setupT784DB(t)

	_ = audit.LogEvent("task.created", "task", "T784FMT", "claude", nil, "")
	events, _ := audit.QueryEvents(audit.QueryFilter{EntityID: "T784FMT"})
	if len(events) == 0 {
		t.Fatal("no events")
	}
	ts := events[0].Timestamp
	if ts == "" {
		t.Fatal("Timestamp is empty")
	}
	// Expect either DB DEFAULT 'YYYY-MM-DD HH:MM:SS' or RFC3339 — both
	// contain a hyphen and a colon.
	if !strings.Contains(ts, "-") || !strings.Contains(ts, ":") {
		t.Errorf("unexpected Timestamp format: %q", ts)
	}
}

// TestT784_Timestamp_SinceFilter — once timestamp is actually stored, the
// since/until filter should work. A missing timestamp would break the
// since filter.
func TestT784_Timestamp_SinceFilter(t *testing.T) {
	setupT784DB(t)

	_ = audit.LogEvent("task.created", "task", "T784SINCE", "claude", nil, "")

	// since set in the past → result should be included.
	past := audit.QueryFilter{EntityID: "T784SINCE", Since: "2000-01-01"}
	res, err := audit.QueryEvents(past)
	if err != nil {
		t.Fatalf("since query failed: %v", err)
	}
	if len(res) != 1 {
		t.Errorf("expected 1 event with since=2000-01-01, got %d", len(res))
	}

	// since set in the future → result should be excluded.
	future := audit.QueryFilter{EntityID: "T784SINCE", Since: "2099-12-31"}
	res, err = audit.QueryEvents(future)
	if err != nil {
		t.Fatalf("future since query failed: %v", err)
	}
	if len(res) != 0 {
		t.Errorf("expected 0 events with since=2099-12-31, got %d (Timestamp likely not stored)", len(res))
	}
}
