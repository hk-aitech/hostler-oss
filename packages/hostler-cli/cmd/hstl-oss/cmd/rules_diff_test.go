package cmd

import (
	"testing"
)

// Regression tests for the rules diff tool.

func makeReport(entries []auditEntry) *auditReport {
	rep := &auditReport{Version: "1", Entries: entries}
	rep.Summary.Total = len(entries)
	return rep
}

// TestT599_Diff_AllIdentical_Unchanged - identical reports yield only unchanged entries.
func TestT599_Diff_AllIdentical_Unchanged(t *testing.T) {
	entries := []auditEntry{
		{RuleID: "a.rule", Status: "ok", Severity: "warn"},
		{RuleID: "b.rule", Status: "skipped", Severity: "advisory"},
	}
	diff := computeRulesDiff(makeReport(entries), makeReport(entries))
	if diff.Summary.Unchanged != 2 {
		t.Errorf("unchanged=%d, want 2", diff.Summary.Unchanged)
	}
	if diff.Summary.Added+diff.Summary.Removed+diff.Summary.Status+diff.Summary.Severity != 0 {
		t.Errorf("diff detected when nothing changed: %+v", diff.Summary)
	}
}

// TestT599_Diff_added_removed - entries unique to after are added, unique to before are removed.
func TestT599_Diff_added_removed(t *testing.T) {
	before := makeReport([]auditEntry{
		{RuleID: "keep.rule", Status: "ok", Severity: "warn"},
		{RuleID: "removed.rule", Status: "ok", Severity: "warn"},
	})
	after := makeReport([]auditEntry{
		{RuleID: "keep.rule", Status: "ok", Severity: "warn"},
		{RuleID: "added.rule", Status: "violated", Severity: "block"},
	})
	diff := computeRulesDiff(before, after)
	if diff.Summary.Added != 1 {
		t.Errorf("added=%d, want 1", diff.Summary.Added)
	}
	if diff.Summary.Removed != 1 {
		t.Errorf("removed=%d, want 1", diff.Summary.Removed)
	}
	if diff.Summary.Unchanged != 1 {
		t.Errorf("unchanged=%d, want 1", diff.Summary.Unchanged)
	}
}

// TestT599_Diff_StatusChange - detects a status change for the same ID.
func TestT599_Diff_StatusChange(t *testing.T) {
	before := makeReport([]auditEntry{
		{RuleID: "r", Status: "ok", Severity: "warn"},
	})
	after := makeReport([]auditEntry{
		{RuleID: "r", Status: "violated", Severity: "warn"},
	})
	diff := computeRulesDiff(before, after)
	if diff.Summary.Status != 1 {
		t.Errorf("status=%d, want 1", diff.Summary.Status)
	}
	if diff.Entries[0].BeforeStatus != "ok" || diff.Entries[0].AfterStatus != "violated" {
		t.Errorf("wrong entry contents: %+v", diff.Entries[0])
	}
}

// TestT599_Diff_SeverityChange - status unchanged with severity changed yields a severity diff.
func TestT599_Diff_SeverityChange(t *testing.T) {
	before := makeReport([]auditEntry{
		{RuleID: "r", Status: "skipped", Severity: "advisory"},
	})
	after := makeReport([]auditEntry{
		{RuleID: "r", Status: "skipped", Severity: "block"},
	})
	diff := computeRulesDiff(before, after)
	if diff.Summary.Severity != 1 {
		t.Errorf("severity=%d, want 1", diff.Summary.Severity)
	}
	if diff.Entries[0].BeforeSeverity != "advisory" || diff.Entries[0].AfterSeverity != "block" {
		t.Errorf("wrong entry contents: %+v", diff.Entries[0])
	}
}

// TestT599_Diff_EmptyReport - empty input is handled correctly.
func TestT599_Diff_EmptyReport(t *testing.T) {
	empty := makeReport(nil)
	diff := computeRulesDiff(empty, empty)
	if diff.Summary.Added+diff.Summary.Removed+diff.Summary.Status+diff.Summary.Severity+diff.Summary.Unchanged != 0 {
		t.Errorf("change detected in empty diff: %+v", diff.Summary)
	}
}
