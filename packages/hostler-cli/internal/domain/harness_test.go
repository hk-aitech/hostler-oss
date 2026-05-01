package domain

// T622 — HarnessGate domain invariant unit tests (5+ cases).
// No file / DB / external process calls.

import "testing"

// TestIsSatisfied_AllItemsDone verifies IsSatisfied=true when every required item is complete.
func TestIsSatisfied_AllItemsDone(t *testing.T) {
	gate := HarnessGate{
		Items: []HarnessItem{
			{ID: "build_passed", Required: true, Done: true},
			{ID: "tests_passed", Required: true, Done: true},
		},
	}
	if !gate.IsSatisfied() {
		t.Error("IsSatisfied must be true when every required item is complete")
	}
}

// TestIsSatisfied_RequiredNotDone verifies IsSatisfied=false when any required item is incomplete.
func TestIsSatisfied_RequiredNotDone(t *testing.T) {
	gate := HarnessGate{
		Items: []HarnessItem{
			{ID: "build_passed", Required: true, Done: true},
			{ID: "tests_passed", Required: true, Done: false}, // incomplete
		},
	}
	if gate.IsSatisfied() {
		t.Error("IsSatisfied must be false when a required item is incomplete")
	}
}

// TestIsSatisfied_NonRequiredNotDone verifies optional incomplete items do not block the gate.
func TestIsSatisfied_NonRequiredNotDone(t *testing.T) {
	gate := HarnessGate{
		Items: []HarnessItem{
			{ID: "build_passed", Required: true, Done: true},
			{ID: "optional_check", Required: false, Done: false}, // optional, incomplete
		},
	}
	if !gate.IsSatisfied() {
		t.Error("optional incomplete items must not block the gate")
	}
}

// TestIsSatisfied_EmptyGate verifies that an item-less gate passes.
func TestIsSatisfied_EmptyGate(t *testing.T) {
	gate := HarnessGate{}
	if !gate.IsSatisfied() {
		t.Error("an item-less gate must report IsSatisfied=true")
	}
}

// TestBlockingItems_ReturnsUnfinishedRequired verifies only incomplete required items are returned.
func TestBlockingItems_ReturnsUnfinishedRequired(t *testing.T) {
	gate := HarnessGate{
		Items: []HarnessItem{
			{ID: "build_passed", Required: true, Done: true},  // complete → excluded
			{ID: "tests_passed", Required: true, Done: false}, // required + incomplete → included
			{ID: "code_review", Required: true, Done: false},  // required + incomplete → included
			{ID: "optional", Required: false, Done: false},    // optional → excluded
		},
	}
	blocked := gate.BlockingItems()
	if len(blocked) != 2 {
		t.Errorf("BlockingItems count = %d, want 2", len(blocked))
	}
	ids := make(map[string]bool)
	for _, item := range blocked {
		ids[item.ID] = true
	}
	if !ids["tests_passed"] || !ids["code_review"] {
		t.Errorf("BlockingItems = %v, expected: tests_passed, code_review", blocked)
	}
}

// TestBlockingItems_NoneBlocking verifies an empty slice is returned when every required item is complete.
func TestBlockingItems_NoneBlocking(t *testing.T) {
	gate := HarnessGate{
		Items: []HarnessItem{
			{ID: "build_passed", Required: true, Done: true},
		},
	}
	if blocked := gate.BlockingItems(); len(blocked) != 0 {
		t.Errorf("BlockingItems count = %d, want 0", len(blocked))
	}
}

// T835 ADR-001 A2 — verify the AllRequiredDone alias equals IsSatisfied.
func TestAllRequiredDone_AliasForIsSatisfied(t *testing.T) {
	gates := []HarnessGate{
		{},
		{Items: []HarnessItem{{Required: true, Done: true}}},
		{Items: []HarnessItem{{Required: true, Done: false}}},
		{Items: []HarnessItem{{Required: false, Done: false}}},
	}
	for i, g := range gates {
		if g.AllRequiredDone() != g.IsSatisfied() {
			t.Errorf("gate[%d]: AllRequiredDone %v != IsSatisfied %v",
				i, g.AllRequiredDone(), g.IsSatisfied())
		}
	}
}
