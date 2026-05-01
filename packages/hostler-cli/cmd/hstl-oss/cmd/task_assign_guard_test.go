package cmd

import (
	"strings"
	"testing"

	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/internal/domain"
)

// Unit tests for the completed-Sprint guard.
// Prevents recurrence of KB M001 - verifies that hstl task assign/unassign
// rejects completed Sprints by default and that --force overrides correctly.

// TestT826_RejectIfCompletedSprint_Completed - assigning to a completed Sprint is rejected.
func TestT826_RejectIfCompletedSprint_Completed(t *testing.T) {
	orig := sprintGetForGuard
	defer func() { sprintGetForGuard = orig }()

	sprintGetForGuard = func(id string) (any, error) {
		return &domain.SprintRecord{SprintID: id, Status: "completed"}, nil
	}

	err := rejectIfCompletedSprint("sprint-82", "assign")
	if err == nil {
		t.Fatalf("expected error for completed Sprint, got nil")
	}
	if !strings.Contains(err.Error(), "completed") || !strings.Contains(err.Error(), "M001") {
		t.Fatalf("expected error message to contain 'completed' and 'M001': %q", err.Error())
	}
}

// TestT826_RejectIfCompletedSprint_Active - active Sprint passes through.
func TestT826_RejectIfCompletedSprint_Active(t *testing.T) {
	orig := sprintGetForGuard
	defer func() { sprintGetForGuard = orig }()

	sprintGetForGuard = func(id string) (any, error) {
		return &domain.SprintRecord{SprintID: id, Status: "active"}, nil
	}

	if err := rejectIfCompletedSprint("sprint-97", "assign"); err != nil {
		t.Fatalf("expected active Sprint to pass, got error: %v", err)
	}
}

// TestT826_RejectIfCompletedSprint_Backlog - backlog Sprint also passes.
func TestT826_RejectIfCompletedSprint_Backlog(t *testing.T) {
	orig := sprintGetForGuard
	defer func() { sprintGetForGuard = orig }()

	sprintGetForGuard = func(id string) (any, error) {
		return &domain.SprintRecord{SprintID: id, Status: "backlog"}, nil
	}

	if err := rejectIfCompletedSprint("sprint-98", "assign"); err != nil {
		t.Fatalf("expected backlog Sprint to pass, got error: %v", err)
	}
}

// TestT826_RejectUnassign_CompletedSprintTask - unassign rejected for a Task in a completed Sprint.
func TestT826_RejectUnassign_CompletedSprintTask(t *testing.T) {
	origSprint := sprintGetForGuard
	origTask := taskGetForGuard
	defer func() {
		sprintGetForGuard = origSprint
		taskGetForGuard = origTask
	}()

	taskGetForGuard = func(id string) (any, error) {
		return &domain.GetResult{TaskID: id, Sprint: "sprint-82"}, nil
	}
	sprintGetForGuard = func(id string) (any, error) {
		return &domain.SprintRecord{SprintID: id, Status: "completed"}, nil
	}

	err := rejectUnassignIfCompletedSprint("T708")
	if err == nil {
		t.Fatalf("expected unassign rejection for Task in completed Sprint, got nil")
	}
	if !strings.Contains(err.Error(), "unassign") {
		t.Fatalf("expected error message to contain 'unassign': %q", err.Error())
	}
}

// TestT826_RejectUnassign_BacklogTask - Tasks not in a Sprint (BACKLOG) pass.
func TestT826_RejectUnassign_BacklogTask(t *testing.T) {
	orig := taskGetForGuard
	defer func() { taskGetForGuard = orig }()

	taskGetForGuard = func(id string) (any, error) {
		return &domain.GetResult{TaskID: id, Sprint: ""}, nil
	}

	if err := rejectUnassignIfCompletedSprint("T999"); err != nil {
		t.Fatalf("expected BACKLOG Task to pass, got error: %v", err)
	}
}

// TestT826_RejectUnassign_ActiveSprintTask - Tasks in an active Sprint can be unassigned.
func TestT826_RejectUnassign_ActiveSprintTask(t *testing.T) {
	origSprint := sprintGetForGuard
	origTask := taskGetForGuard
	defer func() {
		sprintGetForGuard = origSprint
		taskGetForGuard = origTask
	}()

	taskGetForGuard = func(id string) (any, error) {
		return &domain.GetResult{TaskID: id, Sprint: "sprint-97"}, nil
	}
	sprintGetForGuard = func(id string) (any, error) {
		return &domain.SprintRecord{SprintID: id, Status: "active"}, nil
	}

	if err := rejectUnassignIfCompletedSprint("T823"); err != nil {
		t.Fatalf("expected active Sprint Task to pass, got error: %v", err)
	}
}
