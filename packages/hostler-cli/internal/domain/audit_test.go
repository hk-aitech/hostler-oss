package domain

import (
	"testing"
	"time"
)

// T836 ADR-001 A3 — AuditEvent value-object unit tests.

func TestAuditEvent_IsChained_True(t *testing.T) {
	e := AuditEvent{HMAC: "sha256:abcdef"}
	if !e.IsChained() {
		t.Error("a non-empty HMAC must report IsChained true")
	}
}

func TestAuditEvent_IsChained_False(t *testing.T) {
	e := AuditEvent{HMAC: ""}
	if e.IsChained() {
		t.Error("an empty HMAC must report IsChained false")
	}
}

func TestAuditEvent_FullEventFieldsPreserved(t *testing.T) {
	ts := time.Date(2026, 4, 19, 12, 0, 0, 0, time.UTC)
	e := AuditEvent{
		ID:         "evt-001",
		EntityType: "task",
		EntityID:   "T834",
		Action:     "task.completed",
		Actor:      "claude",
		Timestamp:  ts,
		Details:    map[string]string{"from": "in-progress", "to": "done"},
		HMAC:       "sha256:beef",
	}
	if e.ID != "evt-001" || e.EntityID != "T834" || !e.IsChained() {
		t.Errorf("AuditEvent fields not preserved: %+v", e)
	}
	if e.Details["from"] != "in-progress" {
		t.Errorf("Details not preserved: %v", e.Details)
	}
	if !e.Timestamp.Equal(ts) {
		t.Errorf("Timestamp not preserved: %v", e.Timestamp)
	}
}
