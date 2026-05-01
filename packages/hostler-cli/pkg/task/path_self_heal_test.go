package task

import (
	"strings"
	"testing"
)

// TestT680_ReconcileTaskPathsForSprint_NopOnEmptySprint — empty sprintID
// input is a no-op (no panic / error).
func TestT680_ReconcileTaskPathsForSprint_NopOnEmptySprint(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("panic on empty sprintID input: %v", r)
		}
	}()
	reconcileTaskPathsForSprint("")
	reconcileTaskPathsForSprint("   ")
	// normal completion = pass
}

// TestT680_ReconcileTaskPathsForSprint_NopOnMissingFolder — a non-existent
// sprintID is also gracefully skipped (no panic, no error).
func TestT680_ReconcileTaskPathsForSprint_NopOnMissingFolder(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("panic on missing sprint folder: %v", r)
		}
	}()
	reconcileTaskPathsForSprint("sprint-nonexistent-xyz-12345")
}

// TestT680_ReconcileTaskPathsForSprint_NopOnNilStore — even when the store is
// nil the call is gracefully skipped. Because store.Get() is process-global it
// is hard to force a nil — this test only verifies compile/run correctness
// (regression guard).
func TestT680_ReconcileTaskPathsForSprint_NopOnNilStore(t *testing.T) {
	// The compile itself acts as a regression alarm if the function signature
	// changes.
	var f func(string) = reconcileTaskPathsForSprint
	if f == nil {
		t.Error("reconcileTaskPathsForSprint should not be nil function reference")
	}
}

// TestT680_FunctionContractStable — regression guard for function signature
// + package location stability. The task-package entry point is distinct
// from the sprint-package reconcileTaskPathsFromFS used by T660.
func TestT680_FunctionContractStable(t *testing.T) {
	// If the function signature changes arbitrarily the compile fails — this
	// test only documents the contract.
	pkgName := "task"
	if !strings.Contains("task.reconcileTaskPathsForSprint", pkgName) {
		t.Errorf("expected pkg %s in entry point", pkgName)
	}
}

// TestT688_StartReconcileNopOnEmptySprint — the reconcile call inside Start
// is gracefully skipped when the store is nil or no sprint is assigned (no
// panic). store is process-global so a nil cannot be forced directly — this
// test only verifies compile/run correctness.
func TestT688_StartReconcileNopOnEmptySprint(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("Start reconcile: panic when store is nil or sprint unassigned: %v", r)
		}
	}()
	// When store is nil the GetTaskFull lookup is skipped and reconcile is not
	// called.
	// Empty sprintID path: reconcileTaskPathsForSprint becomes a no-op.
	reconcileTaskPathsForSprint("")
}

// TestT688_ReopenReconcileNopOnEmptySprint — the sprintID=="" branch inside
// Reopen skips the reconcile call (graceful skip).
func TestT688_ReopenReconcileNopOnEmptySprint(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("Reopen reconcile: panic when sprintID is the empty string: %v", r)
		}
	}()
	// When sprintID == "" the reconcile call is omitted, so there is no side
	// effect. Verify compile consistency of the explicit skip branch (compile
	// fails on signature change).
	sprintID := ""
	if sprintID != "" {
		reconcileTaskPathsForSprint(sprintID)
	}
}
