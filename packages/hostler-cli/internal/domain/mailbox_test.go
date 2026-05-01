package domain

import (
	"testing"
	"time"
)

// T836 ADR-001 A3 — MailboxItem value-object unit tests.

func TestMailboxItem_IsPending_True(t *testing.T) {
	m := MailboxItem{Status: "pending"}
	if !m.IsPending() {
		t.Error("status=pending must report IsPending true")
	}
}

func TestMailboxItem_IsPending_False(t *testing.T) {
	m := MailboxItem{Status: "accepted"}
	if m.IsPending() {
		t.Error("status=accepted must report IsPending false")
	}
}

func TestMailboxItem_IsProcessed_True(t *testing.T) {
	now := time.Now()
	m := MailboxItem{ProcessedAt: &now}
	if !m.IsProcessed() {
		t.Error("a non-nil ProcessedAt must report IsProcessed true")
	}
}

func TestMailboxItem_IsProcessed_False(t *testing.T) {
	m := MailboxItem{ProcessedAt: nil}
	if m.IsProcessed() {
		t.Error("a nil ProcessedAt must report IsProcessed false")
	}
}

func TestMailboxItem_PendingNotProcessed(t *testing.T) {
	m := MailboxItem{Status: "pending", ProcessedAt: nil}
	if m.IsProcessed() {
		t.Error("pending state must report IsProcessed false")
	}
	if !m.IsPending() {
		t.Error("pending state must report IsPending true")
	}
}
