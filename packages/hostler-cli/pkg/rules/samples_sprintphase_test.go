package rules

import (
	"testing"
)

// G2 sprint phase Rule regression tests.

func TestT592_SprintPhaseRulesRegistered(t *testing.T) {
	want := 10
	got := 0
	for _, r := range List() {
		if r.Category() == sprintPhaseRuleCategory {
			got++
		}
	}
	if got != want {
		t.Errorf("sprint phase Rule count = %d, want %d", got, want)
	}
}

func TestT592_SprintPhaseRule_AllPhasesCovered(t *testing.T) {
	expected := []string{
		"sprint.phase.phase1_doc_review",
		"sprint.phase.phase5_retro",
		"sprint.phase.phase9_followup_tasks",
		"sprint.phase.phase10_skill_update",
	}
	for _, id := range expected {
		if _, ok := Get(id); !ok {
			t.Errorf("Rule %q not registered", id)
		}
	}
}

func TestT592_SprintPhaseRule_AlwaysSkipped(t *testing.T) {
	for _, r := range List() {
		if r.Category() != sprintPhaseRuleCategory {
			continue
		}
		res := r.Check(&RuleContext{})
		if res.Status != StatusSkipped {
			t.Errorf("Rule %s status=%v, want Skipped", r.ID(), res.Status)
		}
	}
}

// TestT074_RegisterSprintPhase_External — verify that an external plugin
// can add a Phase.
func TestT074_RegisterSprintPhase_External(t *testing.T) {
	RegisterSprintPhase("phase11_custom_verify", "Phase 11 — custom verification")
	r, ok := Get("sprint.phase.phase11_custom_verify")
	if !ok {
		t.Fatal("Get failed after RegisterSprintPhase")
	}
	if r.Description() == "" {
		t.Error("empty Description")
	}
}
