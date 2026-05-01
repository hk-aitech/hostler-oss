package domain

// T622 — Sprint domain invariant unit tests (5+ cases).
// No file / DB / external process calls.

import "testing"

// TestCanStart_Backlog verifies that a backlog Sprint can be started.
func TestCanStart_Backlog(t *testing.T) {
	s := Sprint{Status: SprintStatusBacklog}
	if !s.CanStart() {
		t.Error("backlog Sprints must be startable")
	}
}

// TestCanStart_AlreadyActive verifies that an already-active Sprint cannot be restarted.
func TestCanStart_AlreadyActive(t *testing.T) {
	s := Sprint{Status: SprintStatusActive}
	if s.CanStart() {
		t.Error("active Sprints must not be restartable")
	}
}

// TestCanStart_Completed verifies that a completed Sprint cannot be started.
func TestCanStart_Completed(t *testing.T) {
	s := Sprint{Status: SprintStatusCompleted}
	if s.CanStart() {
		t.Error("completed Sprints must not be startable")
	}
}

// TestCanComplete_AllTasksDone verifies that completion is allowed when every Task is done.
func TestCanComplete_AllTasksDone(t *testing.T) {
	s := Sprint{
		Status: SprintStatusActive,
		Tasks:  []TaskID{"T001", "T002"},
	}
	done := map[TaskID]bool{"T001": true, "T002": true}
	if !s.CanComplete(done) {
		t.Error("completion must be allowed when all Tasks are done")
	}
}

// TestCanComplete_UnfinishedTask verifies that completion is blocked when any Task is unfinished.
func TestCanComplete_UnfinishedTask(t *testing.T) {
	s := Sprint{
		Status: SprintStatusActive,
		Tasks:  []TaskID{"T001", "T002"},
	}
	done := map[TaskID]bool{"T001": true} // T002 not done
	if s.CanComplete(done) {
		t.Error("completion must be blocked when any Task is unfinished")
	}
}

// TestCanComplete_NotActive verifies that a non-active Sprint cannot be completed.
func TestCanComplete_NotActive(t *testing.T) {
	s := Sprint{
		Status: SprintStatusBacklog,
		Tasks:  []TaskID{"T001"},
	}
	done := map[TaskID]bool{"T001": true}
	if s.CanComplete(done) {
		t.Error("non-active Sprints must not be completable")
	}
}

// TestProgress_AllDone verifies that 1.0 is returned when every Task is done.
func TestProgress_AllDone(t *testing.T) {
	s := Sprint{
		Status: SprintStatusActive,
		Tasks:  []TaskID{"T001", "T002"},
	}
	done := map[TaskID]bool{"T001": true, "T002": true}
	if got := s.Progress(done); got != 1.0 {
		t.Errorf("Progress = %.2f, want 1.0", got)
	}
}

// TestProgress_Half verifies that 0.5 is returned at half completion.
func TestProgress_Half(t *testing.T) {
	s := Sprint{
		Status: SprintStatusActive,
		Tasks:  []TaskID{"T001", "T002"},
	}
	done := map[TaskID]bool{"T001": true}
	if got := s.Progress(done); got != 0.5 {
		t.Errorf("Progress = %.2f, want 0.5", got)
	}
}

// TestProgress_EmptyTasks verifies that an empty Sprint returns 0.0.
func TestProgress_EmptyTasks(t *testing.T) {
	s := Sprint{Status: SprintStatusActive, Tasks: nil}
	if got := s.Progress(nil); got != 0.0 {
		t.Errorf("Progress = %.2f, want 0.0 (empty Sprint)", got)
	}
}

// T835 ADR-001 A2 — IsActive/IsCompleted invariant tests.

func TestIsActive_True(t *testing.T) {
	s := Sprint{Status: SprintStatusActive}
	if !s.IsActive() {
		t.Error("active state must report IsActive true")
	}
}

func TestIsActive_FalseForBacklog(t *testing.T) {
	s := Sprint{Status: SprintStatusBacklog}
	if s.IsActive() {
		t.Error("backlog state must report IsActive false")
	}
}

func TestIsActive_FalseForCompleted(t *testing.T) {
	s := Sprint{Status: SprintStatusCompleted}
	if s.IsActive() {
		t.Error("completed state must report IsActive false")
	}
}

func TestIsCompleted_True(t *testing.T) {
	s := Sprint{Status: SprintStatusCompleted}
	if !s.IsCompleted() {
		t.Error("completed state must report IsCompleted true")
	}
}

func TestIsCompleted_FalseForActive(t *testing.T) {
	s := Sprint{Status: SprintStatusActive}
	if s.IsCompleted() {
		t.Error("active state must report IsCompleted false")
	}
}
