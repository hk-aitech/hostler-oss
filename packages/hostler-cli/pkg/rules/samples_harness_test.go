package rules

import (
	"testing"
)

// G1 harness item Rule regression tests.

func TestT591_HarnessRulesRegistered(t *testing.T) {
	want := 16
	got := 0
	for _, r := range List() {
		if r.Category() == "harness" {
			got++
		}
	}
	if got != want {
		t.Errorf("harness Rule count = %d, want %d", got, want)
	}
}

func TestT591_HarnessRule_ID_Prefix(t *testing.T) {
	const prefix = "harness.task."
	for _, r := range List() {
		if r.Category() != "harness" {
			continue
		}
		id := r.ID()
		if len(id) < len(prefix) || id[:len(prefix)] != prefix {
			t.Errorf("Rule ID prefix mismatch: %s", id)
		}
	}
}

func TestT591_HarnessRule_AlwaysSkipped(t *testing.T) {
	// observability-only: result is always Skipped.
	for _, r := range List() {
		if r.Category() != "harness" {
			continue
		}
		res := r.Check(&RuleContext{})
		if res.Status != StatusSkipped {
			t.Errorf("Rule %s status = %v, want Skipped (observability-only)", r.ID(), res.Status)
		}
	}
}

func TestT591_HarnessRule_KeyItemsPresent(t *testing.T) {
	expected := []string{
		"harness.task.criteria_checked",
		"harness.task.build_passed",
		"harness.task.tests_passed",
		"harness.task.code_review",
		"harness.task.context_acknowledged",
		"harness.task.lint_passed",
	}
	for _, id := range expected {
		if _, ok := Get(id); !ok {
			t.Errorf("Rule %q not registered", id)
		}
	}
}
