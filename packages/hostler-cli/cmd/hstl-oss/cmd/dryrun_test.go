package cmd

import (
	"testing"
)

// The dry-run builders return a report map without side effects.
// CLI integration tests run separately in integration-test.sh; this file only
// covers edge cases for nonexistent Task/Sprint inputs.

func TestT356_BuildTaskStartDryRun_NonexistentTask(t *testing.T) {
	r := buildTaskStartDryRun("T_NOT_EXIST")
	if r["dry_run"] != true {
		t.Error("expected dry_run=true")
	}
	if r["decision"] != "blocked" {
		t.Errorf("expected blocked decision for nonexistent Task, got %v", r["decision"])
	}
}

func TestT356_BuildTaskCompleteDryRun_ReturnsDryRunFlag(t *testing.T) {
	r := buildTaskCompleteDryRun("T_NOT_EXIST")
	if r["dry_run"] != true {
		t.Error("expected dry_run=true")
	}
	if r["action"] != "complete" {
		t.Error("expected action=complete")
	}
}

func TestT356_BuildSprintStartDryRun_Nonexistent(t *testing.T) {
	r := buildSprintStartDryRun("sprint-99999")
	if r["dry_run"] != true {
		t.Error("expected dry_run=true")
	}
	if r["decision"] != "blocked" {
		t.Errorf("expected blocked decision for nonexistent Sprint, got %v", r["decision"])
	}
}

func TestT356_BuildSprintCompleteDryRun_ReturnsDryRunFlag(t *testing.T) {
	r := buildSprintCompleteDryRun("sprint-99999")
	if r["dry_run"] != true {
		t.Error("expected dry_run=true")
	}
}
