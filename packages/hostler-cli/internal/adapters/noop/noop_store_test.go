// Package noop_test contains tests for the NoopStore.
package noop_test

import (
	"testing"

	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/internal/adapters/noop"
	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/internal/ports"
)

// TestNoopStoreInterfaces verifies NoopStore implements both interfaces.
func TestNoopStoreInterfaces(t *testing.T) {
	store := noop.New()

	var _ ports.GraphStore = store
	var _ ports.IDStore = store

	if store == nil {
		t.Fatal("New returned nil")
	}
}

// TestNoopStore_AppendAuditEvent_ReturnsNil verifies that appending an audit event returns nil.
func TestNoopStore_AppendAuditEvent_ReturnsNil(t *testing.T) {
	store := noop.New()
	err := store.AppendAuditEvent(ports.AuditEvent{
		EventType: "task.created",
		EntityID:  "T001",
		ActorID:   "test-actor",
	})
	if err != nil {
		t.Errorf("AppendAuditEvent: want nil, got %v", err)
	}
}

// TestNoopStore_HasPlaceholderBody_ReturnsFalseNil verifies that the placeholder check returns false, nil.
func TestNoopStore_HasPlaceholderBody_ReturnsFalseNil(t *testing.T) {
	store := noop.New()
	isPlaceholder, err := store.HasPlaceholderBody("T001")
	if err != nil {
		t.Errorf("HasPlaceholderBody: want nil error, got %v", err)
	}
	if isPlaceholder {
		t.Errorf("HasPlaceholderBody: want false, got true")
	}
}

// TestNoopStore_NextCounter_ReturnsZero verifies the counter always returns 0.
func TestNoopStore_NextCounter_ReturnsZero(t *testing.T) {
	store := noop.New()
	seq, err := store.NextCounter("any_key")
	if err != nil {
		t.Errorf("NextCounter: want nil error, got %v", err)
	}
	if seq != 0 {
		t.Errorf("NextCounter: want 0, got %d", seq)
	}
}

// TestNoopStore_ListTasks_ReturnsEmpty verifies the task list returns an empty result.
func TestNoopStore_ListTasks_ReturnsEmpty(t *testing.T) {
	store := noop.New()
	result, err := store.ListTasks(nil, nil)
	if err != nil {
		t.Errorf("ListTasks: want nil error, got %v", err)
	}
	if result == nil {
		t.Fatal("ListTasks: want non-nil result, got nil")
		return // nolint (SA5011 guard)
	}
	if len(result.Tasks) != 0 {
		t.Errorf("ListTasks: want 0 tasks, got %d", len(result.Tasks))
	}
}
