package inmemory

import (
	"testing"
	"time"
)

// Context ack TTL regression tests.
// Inject test timestamps directly into the InMemoryStore to verify TTL
// semantics equivalent to the SQLite adapter.

func TestT547_HasContextAckFresh_TTLZero_NoExpiry(t *testing.T) {
	s := New()
	_ = s.InsertContextAck("WT-T547-x", "hash1", 10)

	// TTL 0 → identical to HasContextAck (no expiry).
	ok, err := s.HasContextAckFresh("WT-T547-x", "hash1", 0)
	if err != nil || !ok {
		t.Errorf("TTL=0 expected true, got ok=%v err=%v", ok, err)
	}
}

func TestT547_HasContextAckFresh_WithinTTL(t *testing.T) {
	s := New()
	_ = s.InsertContextAck("WT-T547-a", "hash2", 10)

	// Immediate check → within a 5-minute TTL.
	ok, err := s.HasContextAckFresh("WT-T547-a", "hash2", 5)
	if err != nil || !ok {
		t.Errorf("just-created ack expected true within 5-minute TTL, got ok=%v err=%v", ok, err)
	}
}

func TestT547_HasContextAckFresh_StaleRejected(t *testing.T) {
	s := New()
	// Insert an ack while forcing the creation time into the past.
	key := "WT-T547-b|hash3"
	s.mu.Lock()
	s.contextAcks[key] = time.Now().UTC().Add(-10 * time.Minute)
	s.mu.Unlock()

	// TTL 5 minutes → stale (created 10 minutes ago).
	ok, err := s.HasContextAckFresh("WT-T547-b", "hash3", 5)
	if err != nil {
		t.Fatal(err)
	}
	if ok {
		t.Error("10-minute-old ack passed within a 5-minute TTL — stale detection failed")
	}

	// HasContextAck (no TTL) still returns true.
	existsAny, _ := s.HasContextAck("WT-T547-b", "hash3")
	if !existsAny {
		t.Error("HasContextAck must still report existence regardless of TTL")
	}
}

func TestT547_HasContextAckFresh_NotFound(t *testing.T) {
	s := New()
	ok, err := s.HasContextAckFresh("WT-T547-none", "hash", 10)
	if err != nil {
		t.Fatal(err)
	}
	if ok {
		t.Error("a missing ack returned true")
	}
}

func TestT547_InsertContextAck_Idempotent_NoTimeReset(t *testing.T) {
	s := New()
	_ = s.InsertContextAck("WT-T547-c", "hash4", 10)
	// Force the creation time into the past.
	key := "WT-T547-c|hash4"
	past := time.Now().UTC().Add(-20 * time.Minute)
	s.mu.Lock()
	s.contextAcks[key] = past
	s.mu.Unlock()

	// Re-insert — idempotent operations must not reset the timestamp.
	_ = s.InsertContextAck("WT-T547-c", "hash4", 10)

	s.mu.RLock()
	stored := s.contextAcks[key]
	s.mu.RUnlock()
	if !stored.Equal(past) {
		t.Errorf("re-insert reset the timestamp: stored=%v, want=%v (matches SQLite INSERT OR IGNORE)", stored, past)
	}
}
