// Hash chain unit test.
package audit

import (
	"strings"
	"testing"

	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/internal/ports"
)

func TestCanonicalEventJSON_Deterministic(t *testing.T) {
	e := ports.AuditEvent{
		ID:         42,
		Timestamp:  "2026-04-21 15:00:00",
		EventType:  "task.created",
		EntityType: "task",
		EntityID:   "T268",
		ActorID:    "claude",
		SessionID:  "sess-abc",
		Details:    map[string]any{"b": 2, "a": 1},
	}
	a := CanonicalEventJSON(e)
	b := CanonicalEventJSON(e)
	if a != b {
		t.Errorf("canonical JSON not deterministic")
	}
	// details keys must be sorted alphabetically.
	if !strings.Contains(a, `"details":{"a":1,"b":2}`) {
		t.Errorf("details key sort failed: %s", a)
	}
}

func TestCalculateHash_StableWithPrev(t *testing.T) {
	e := ports.AuditEvent{
		ID:         1,
		Timestamp:  "2026-04-21 15:00:00",
		EventType:  "task.created",
		EntityType: "task",
		EntityID:   "T001",
		ActorID:    "claude",
	}
	h1 := CalculateHash("", e)
	h2 := CalculateHash("", e)
	if h1 != h2 {
		t.Errorf("hash not deterministic")
	}
	if len(h1) != 64 {
		t.Errorf("SHA256 hex length: want 64, got %d", len(h1))
	}

	// Changing prev_hash must change the resulting hash.
	h3 := CalculateHash("abc123", e)
	if h3 == h1 {
		t.Errorf("prev_hash had no effect — tamper detection impossible")
	}
}

func TestCalculateHash_TamperDetection(t *testing.T) {
	e1 := ports.AuditEvent{ID: 1, EventType: "task.created", EntityID: "T001", ActorID: "claude"}
	e2 := ports.AuditEvent{ID: 1, EventType: "task.deleted", EntityID: "T001", ActorID: "claude"} // tampered
	h1 := CalculateHash("", e1)
	h2 := CalculateHash("", e2)
	if h1 == h2 {
		t.Errorf("tamper not detected — different event_type produced same hash")
	}
}
