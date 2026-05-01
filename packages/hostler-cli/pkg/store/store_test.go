// Package store smoke test. Verifies the Init/Get/MustGet/Reset
// lifecycle of the GraphStore/IDStore global registry.
package store

import (
	"testing"

	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/internal/ports"
)

// nopGraphStore is a nil-behaviour GraphStore used in tests. Since this
// suite verifies registry lifecycle only (no real implementation), a
// shim via interface embedding is sufficient.
type nopGraphStore struct{ ports.GraphStore }

type nopIDStore struct{ ports.IDStore }

// TestInitAndGet verifies that Init sets the global instances and that
// Get returns the same instances.
func TestInitAndGet(t *testing.T) {
	defer Reset()

	gs := &nopGraphStore{}
	ids := &nopIDStore{}
	Init(gs, ids)

	if got := Get(); got != gs {
		t.Errorf("Get() did not return the Init instance — pointer mismatch")
	}
	if got := GetIDStore(); got != ids {
		t.Errorf("GetIDStore() did not return the Init instance")
	}
}

// TestMustGetUninitialized verifies that MustGet returns an error before
// Init is called.
func TestMustGetUninitialized(t *testing.T) {
	Reset() // ensure a clean state.
	defer Reset()

	gs, err := MustGet()
	if err == nil {
		t.Fatalf("MustGet() must return an error before Init")
	}
	if gs != nil {
		t.Errorf("MustGet() should return nil on error, got: %T", gs)
	}

	ids, err := MustGetIDStore()
	if err == nil {
		t.Fatalf("MustGetIDStore() must return an error before Init")
	}
	if ids != nil {
		t.Errorf("MustGetIDStore() should return nil on error, got: %T", ids)
	}
}

// TestReset verifies that Reset clears the global instances back to nil.
func TestReset(t *testing.T) {
	Init(&nopGraphStore{}, &nopIDStore{})
	if Get() == nil {
		t.Fatal("Get() returned nil right after Init — test precondition violated")
	}

	Reset()

	if got := Get(); got != nil {
		t.Errorf("Get() should be nil after Reset, got non-nil")
	}
	if got := GetIDStore(); got != nil {
		t.Errorf("GetIDStore() should be nil after Reset, got non-nil")
	}
}
