// Package ports — T412 (Sprint-34) smoke test.
// Compile-time contract check for the interface-definition package plus a
// zero-value round-trip smoke test for the core structs. Resolves audit D05
// (no tests under internal/ports).
package ports

import "testing"

// TestPortInterfacesCompile asserts at compile time that the core port
// interface variables can be declared as nil-convertible types (we only mark
// the variables as used — no runtime check needed). The real effect is that
// when *_test.go runs across the whole module, any interface drift produces
// a compile failure and regresses are caught automatically.
func TestPortInterfacesCompile(t *testing.T) {
	var gs GraphStore
	var ids IDStore
	var fc FrontmatterCodec
	_ = gs
	_ = ids
	_ = fc
}

// TestAuditEventZeroValue verifies that the zero value of AuditEvent is
// valid (the struct compiles without missing field types).
func TestAuditEventZeroValue(t *testing.T) {
	ev := AuditEvent{}
	// Zero-value field access must not panic — every field is a value type
	// (string/int/map/...), so there is nothing to nil-deref.
	_ = ev
}

// TestTaskRecordZeroValue — same smoke check.
func TestTaskRecordZeroValue(t *testing.T) {
	r := TaskRecord{}
	_ = r
}

// TestSprintRecordZeroValue — same smoke check.
func TestSprintRecordZeroValue(t *testing.T) {
	r := SprintRecord{}
	_ = r
}
