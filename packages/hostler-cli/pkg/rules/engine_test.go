package rules

import (
	"testing"
)

// TestRegistry_BasicBehaviour — Register / Get / List work correctly.
func TestRegistry_BasicBehaviour(t *testing.T) {
	// init() should have registered at least 2 sample rules.
	rs := List()
	if len(rs) < 2 {
		t.Fatalf("List length = %d, expected at least 2", len(rs))
	}
	// Sorted alphabetically by ID.
	for i := 1; i < len(rs); i++ {
		if rs[i-1].ID() > rs[i].ID() {
			t.Errorf("List ordering violated: %s > %s", rs[i-1].ID(), rs[i].ID())
		}
	}
	// Get
	r, ok := Get("task.body.not_placeholder")
	if !ok || r == nil {
		t.Error("failed to look up task.body.not_placeholder")
	}
	if _, ok := Get("nonexistent.rule"); ok {
		t.Error("looked up a nonexistent rule successfully")
	}
}

// TestRegistry_DuplicateRegister_Panic — re-registering the same ID panics.
func TestRegistry_DuplicateRegister_Panic(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic on duplicate Register")
		}
	}()
	Register(taskBodyNotPlaceholderRule{})
}

// TestFilterEvidence_SelfRemoved — SelfFilePath is removed from the
// evidence list.
func TestFilterEvidence_SelfRemoved(t *testing.T) {
	ev := []string{"works/tasks/T573.md", "cli/foo.go", "docs/x.md"}
	out := FilterEvidence(ev, "works/tasks/T573.md")
	if len(out) != 2 {
		t.Errorf("after filter = %v, expected 2 entries", out)
	}
	for _, e := range out {
		if e == "works/tasks/T573.md" {
			t.Error("self path was not filtered")
		}
	}
	// An empty selfPath leaves the original untouched.
	out2 := FilterEvidence(ev, "")
	if len(out2) != 3 {
		t.Error("empty selfPath should leave the original unchanged")
	}
}

// dedicated Rule regression tests after removing the bridge pattern.
// Detailed behaviour is verified in samples_test.go.

// TestSampleRule_SelfPath_Missing_Skipped — Skipped when SelfFilePath is empty.
func TestSampleRule_SelfPath_Missing_Skipped(t *testing.T) {
	r, _ := Get("task.result_section.files_exist")
	res := r.Check(&RuleContext{})
	if res.Status != StatusSkipped {
		t.Errorf("missing SelfFilePath status = %v, want Skipped", res.Status)
	}
}

// TestSampleRule_BodyRule_SelfPath_Missing_Skipped — body Rule is also
// Skipped when SelfFilePath is missing.
func TestSampleRule_BodyRule_SelfPath_Missing_Skipped(t *testing.T) {
	r, _ := Get("task.body.not_placeholder")
	res := r.Check(&RuleContext{})
	if res.Status != StatusSkipped {
		t.Errorf("body Rule missing SelfFilePath status = %v, want Skipped", res.Status)
	}
}

// TestRuleMetadata — verify metadata of the sample Rules.
func TestRuleMetadata(t *testing.T) {
	r, _ := Get("task.result_section.files_exist")
	if r.Category() != "task" {
		t.Errorf("Category = %q", r.Category())
	}
	if r.DefaultSeverity() != SeverityBlock {
		t.Errorf("DefaultSeverity = %v, want block", r.DefaultSeverity())
	}
	if r.Description() == "" {
		t.Error("empty Description")
	}
}
