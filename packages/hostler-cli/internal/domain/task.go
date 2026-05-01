package domain

import "fmt"

// TaskStatus enumerates Task statuses.
type TaskStatus string

const (
	TaskStatusTodo       TaskStatus = "todo"
	TaskStatusInProgress TaskStatus = "in-progress"
	TaskStatusDone       TaskStatus = "done"
	TaskStatusCancelled  TaskStatus = "cancelled"
)

// TaskType enumerates Task types.
type TaskType string

const (
	TaskTypeFeature  TaskType = "feature"
	TaskTypeRefactor TaskType = "refactor"
	TaskTypeBugfix   TaskType = "bugfix"
	TaskTypeInfra    TaskType = "infra"
	TaskTypeDocs     TaskType = "docs"
	TaskTypeTest     TaskType = "test"
	TaskTypeChore    TaskType = "chore"
	TaskTypeSpike    TaskType = "spike"
)

// Priority enumerates Task priorities.
type Priority string

const (
	PriorityP0 Priority = "p0"
	PriorityP1 Priority = "p1"
	PriorityP2 Priority = "p2"
	PriorityP3 Priority = "p3"
)

// Estimate enumerates Task size estimates.
type Estimate string

const (
	EstimateXS Estimate = "XS"
	EstimateS  Estimate = "S"
	EstimateM  Estimate = "M"
	EstimateL  Estimate = "L"
	EstimateXL Estimate = "XL"
)

// Task — domain Task value object (no file I/O).
type Task struct {
	ID         TaskID
	Title      string
	Status     TaskStatus
	Type       TaskType
	Priority   Priority
	Estimate   Estimate
	Sprint     SprintID
	DependsOn  []TaskID
	WorkTicket string
}

// IsPlaceholder reports whether the title matches a placeholder pattern.
func (t Task) IsPlaceholder() bool {
	return t.Title == "" || t.Title == "TBD" || t.Title == "{title}"
}

// CanTransitionTo enforces the state-transition invariants.
func (t Task) CanTransitionTo(next TaskStatus) bool {
	switch t.Status {
	case TaskStatusTodo:
		return next == TaskStatusInProgress || next == TaskStatusCancelled
	case TaskStatusInProgress:
		return next == TaskStatusDone || next == TaskStatusTodo || next == TaskStatusCancelled
	case TaskStatusDone:
		return next == TaskStatusInProgress // reopen
	case TaskStatusCancelled:
		return false
	}
	return false
}

// ValidateTransition is the error-returning variant of CanTransitionTo.
// Wraps the bool return in an error so users see a specific reason
// when a state transition is rejected. Used as the domain-level check
// before pkg/task/reopen and similar paths wrap it in
// apperr.RejectedError.
func (t Task) ValidateTransition(next TaskStatus) error {
	if t.CanTransitionTo(next) {
		return nil
	}
	return fmt.Errorf("transition not allowed: current=%s → target=%s", t.Status, next)
}

// IsDone reports whether the task is in the done state.
// Invariant method that replaces the status == "done" literal
// comparisons in pkg/task.
func (t Task) IsDone() bool {
	return t.Status == TaskStatusDone
}

// IsInProgress reports whether the task is in the in-progress state.
// Invariant method that replaces the status == "in-progress" literal
// comparisons in pkg/task.
func (t Task) IsInProgress() bool {
	return t.Status == TaskStatusInProgress
}

// RequiresHarness reports whether the Task type requires a Harness Gate check.
func (t Task) RequiresHarness() bool {
	switch t.Type {
	case TaskTypeChore, TaskTypeSpike:
		return false
	}
	return true
}

// IsAssigned reports whether the Task has been assigned to a Sprint.
func (t Task) IsAssigned() bool {
	return !t.Sprint.IsZero()
}
