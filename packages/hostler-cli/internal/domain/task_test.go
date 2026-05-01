package domain

// T622 — Task domain invariant unit tests (8+ cases).
// No file / DB / external process calls — pure-function tests only.

import "testing"

// TestIsPlaceholder_EmptyTitle verifies that an empty title is treated as a placeholder.
func TestIsPlaceholder_EmptyTitle(t *testing.T) {
	task := Task{Title: ""}
	if !task.IsPlaceholder() {
		t.Error("an empty title should be treated as a placeholder")
	}
}

// TestIsPlaceholder_TBD verifies that "TBD" is treated as a placeholder.
func TestIsPlaceholder_TBD(t *testing.T) {
	task := Task{Title: "TBD"}
	if !task.IsPlaceholder() {
		t.Error("TBD should be treated as a placeholder")
	}
}

// TestIsPlaceholder_TemplateMarker verifies that the "{title}" marker is treated as a placeholder.
func TestIsPlaceholder_TemplateMarker(t *testing.T) {
	task := Task{Title: "{title}"}
	if !task.IsPlaceholder() {
		t.Error("{title} should be treated as a placeholder")
	}
}

// TestIsPlaceholder_NormalTitle verifies that a normal title is not flagged as a placeholder.
func TestIsPlaceholder_NormalTitle(t *testing.T) {
	task := Task{Title: "Add a new API endpoint"}
	if task.IsPlaceholder() {
		t.Error("a normal title was incorrectly flagged as a placeholder")
	}
}

// TestCanTransitionTo_TodoToInProgress verifies the todo→in-progress transition.
func TestCanTransitionTo_TodoToInProgress(t *testing.T) {
	task := Task{Status: TaskStatusTodo}
	if !task.CanTransitionTo(TaskStatusInProgress) {
		t.Error("todo → in-progress transition must be allowed")
	}
}

// TestCanTransitionTo_InProgressToDone verifies the in-progress→done transition.
func TestCanTransitionTo_InProgressToDone(t *testing.T) {
	task := Task{Status: TaskStatusInProgress}
	if !task.CanTransitionTo(TaskStatusDone) {
		t.Error("in-progress → done transition must be allowed")
	}
}

// TestCanTransitionTo_DoneToInProgress verifies the done→in-progress (reopen) transition.
func TestCanTransitionTo_DoneToInProgress(t *testing.T) {
	task := Task{Status: TaskStatusDone}
	if !task.CanTransitionTo(TaskStatusInProgress) {
		t.Error("done → in-progress (reopen) transition must be allowed")
	}
}

// TestCanTransitionTo_InvalidTodoToDone verifies that direct todo→done is blocked.
func TestCanTransitionTo_InvalidTodoToDone(t *testing.T) {
	task := Task{Status: TaskStatusTodo}
	if task.CanTransitionTo(TaskStatusDone) {
		t.Error("direct todo → done transition must be blocked")
	}
}

// TestCanTransitionTo_InvalidCancelledToAny verifies every transition out of cancelled is blocked.
func TestCanTransitionTo_InvalidCancelledToAny(t *testing.T) {
	task := Task{Status: TaskStatusCancelled}
	for _, next := range []TaskStatus{TaskStatusTodo, TaskStatusInProgress, TaskStatusDone} {
		if task.CanTransitionTo(next) {
			t.Errorf("cancelled → %s transition must be blocked", next)
		}
	}
}

// TestRequiresHarness_FeatureType verifies the feature type requires a Harness Gate.
func TestRequiresHarness_FeatureType(t *testing.T) {
	task := Task{Type: TaskTypeFeature}
	if !task.RequiresHarness() {
		t.Error("feature type must require a Harness Gate")
	}
}

// TestRequiresHarness_ChoreType verifies the chore type does not require a Harness Gate.
func TestRequiresHarness_ChoreType(t *testing.T) {
	task := Task{Type: TaskTypeChore}
	if task.RequiresHarness() {
		t.Error("chore type must not require a Harness Gate")
	}
}

// TestRequiresHarness_SpikeType verifies the spike type does not require a Harness Gate.
func TestRequiresHarness_SpikeType(t *testing.T) {
	task := Task{Type: TaskTypeSpike}
	if task.RequiresHarness() {
		t.Error("spike type must not require a Harness Gate")
	}
}

// TestIsAssigned_WithSprint verifies a Task assigned to a Sprint is marked assigned.
func TestIsAssigned_WithSprint(t *testing.T) {
	task := Task{Sprint: SprintID("sprint-72")}
	if !task.IsAssigned() {
		t.Error("a Task assigned to a Sprint must report IsAssigned true")
	}
}

// TestIsAssigned_WithoutSprint verifies an unassigned Task is not marked assigned.
func TestIsAssigned_WithoutSprint(t *testing.T) {
	task := Task{Sprint: SprintID("")}
	if task.IsAssigned() {
		t.Error("an unassigned Task must not report IsAssigned true")
	}
}

// T835 ADR-001 A2 — ValidateTransition / IsDone / IsInProgress invariant tests.

func TestValidateTransition_Allowed(t *testing.T) {
	task := Task{Status: TaskStatusInProgress}
	if err := task.ValidateTransition(TaskStatusDone); err != nil {
		t.Errorf("in-progress → done must be allowed, err=%v", err)
	}
}

func TestValidateTransition_Blocked(t *testing.T) {
	task := Task{Status: TaskStatusTodo}
	err := task.ValidateTransition(TaskStatusDone)
	if err == nil {
		t.Error("todo → done must be blocked (error expected)")
	}
}

func TestValidateTransition_ErrorMessageContainsStates(t *testing.T) {
	task := Task{Status: TaskStatusCancelled}
	err := task.ValidateTransition(TaskStatusInProgress)
	if err == nil {
		t.Fatal("cancelled → any must be blocked")
	}
	msg := err.Error()
	if !contains(msg, "cancelled") || !contains(msg, "in-progress") {
		t.Errorf("error message must include current/target states: %s", msg)
	}
}

func TestIsDone_True(t *testing.T) {
	task := Task{Status: TaskStatusDone}
	if !task.IsDone() {
		t.Error("done state must report IsDone true")
	}
}

func TestIsDone_FalseForInProgress(t *testing.T) {
	task := Task{Status: TaskStatusInProgress}
	if task.IsDone() {
		t.Error("in-progress state must report IsDone false")
	}
}

func TestIsInProgress_True(t *testing.T) {
	task := Task{Status: TaskStatusInProgress}
	if !task.IsInProgress() {
		t.Error("in-progress state must report IsInProgress true")
	}
}

func TestIsInProgress_FalseForDone(t *testing.T) {
	task := Task{Status: TaskStatusDone}
	if task.IsInProgress() {
		t.Error("done state must report IsInProgress false")
	}
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && indexOf(s, sub) >= 0
}

func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}
