package ceremony

import (
	"testing"
)

// T488 (Sprint-71): contains helper removed — collectChangedFiles now
// reuses pkg/task.GetGitChangedFiles, so its own dedup logic is no longer
// needed.

// TestSprintStartCeremonyStruct verifies SprintStartCeremony struct
// serialisation.
func TestSprintStartCeremonyStruct(t *testing.T) {
	cer := &SprintStartCeremony{
		Briefing: &SprintBriefing{
			SprintID:  "sprint-99",
			Title:     "test Sprint",
			Goal:      "test goal",
			TaskCount: 2,
			Tasks: []TaskSummary{
				{TaskID: "T001", Title: "Task 1", Type: "feature", Size: "M", Priority: "p1"},
				{TaskID: "T002", Title: "Task 2", Type: "docs", Size: "S", Priority: "p2"},
			},
		},
		DesignReadiness: []ReadinessCheck{
			{Check: "placeholder", Status: "pass", Detail: "0 items"},
		},
		Reminders: []string{"reminder 1"},
	}

	if cer.Briefing.SprintID != "sprint-99" {
		t.Errorf("SprintID mismatch: %s", cer.Briefing.SprintID)
	}
	if len(cer.Briefing.Tasks) != 2 {
		t.Errorf("Task count mismatch: %d", len(cer.Briefing.Tasks))
	}
	if cer.DesignReadiness[0].Status != "pass" {
		t.Errorf("DesignReadiness status mismatch: %s", cer.DesignReadiness[0].Status)
	}
}

// TestTaskStartCeremonyStruct verifies the TaskStartCeremony struct.
func TestTaskStartCeremonyStruct(t *testing.T) {
	cer := &TaskStartCeremony{
		Briefing: &TaskBriefing{
			TaskID:       "T100",
			Title:        "test Task",
			Type:         "feature",
			Estimate:     "L",
			Priority:     "p0",
			CommitPrefix: "feat",
		},
		Reminders: []string{"1 Task = 1 Commit"},
	}

	if cer.Briefing.CommitPrefix != "feat" {
		t.Errorf("CommitPrefix mismatch: %s", cer.Briefing.CommitPrefix)
	}
}

// TestTaskCompleteCeremonyStruct verifies the TaskCompleteCeremony struct.
func TestTaskCompleteCeremonyStruct(t *testing.T) {
	cer := &TaskCompleteCeremony{
		ChangedFiles: []string{"pkg/ceremony/types.go", "pkg/ceremony/collect.go"},
		HarnessGate: &HarnessGate{
			Status:         "blocked",
			UncheckedItems: []string{"build_passed", "tests_passed"},
		},
		ResultSection: &ResultSectionCheck{
			Exists: true,
		},
	}

	if len(cer.ChangedFiles) != 2 {
		t.Errorf("ChangedFiles count mismatch: %d", len(cer.ChangedFiles))
	}
	if cer.HarnessGate.Status != "blocked" {
		t.Errorf("HarnessGate status mismatch: %s", cer.HarnessGate.Status)
	}
	if !cer.ResultSection.Exists {
		t.Error("ResultSection.Exists is false")
	}
}

// TestSprintCompleteCeremonyStruct verifies the SprintCompleteCeremony
// struct.
func TestSprintCompleteCeremonyStruct(t *testing.T) {
	cer := &SprintCompleteCeremony{
		Harness: []PhaseStatus{
			{Phase: "phase1_doc_review", Status: "done", Detail: "review complete"},
			{Phase: "phase2_code_review", Status: "pending"},
		},
		KBCandidates: []string{"ceremony pattern lesson"},
	}

	if len(cer.Harness) != 2 {
		t.Errorf("Harness count mismatch: %d", len(cer.Harness))
	}
	if cer.Harness[0].Status != "done" {
		t.Errorf("Phase1 status mismatch: %s", cer.Harness[0].Status)
	}
}
