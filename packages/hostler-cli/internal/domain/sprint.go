package domain

// SprintStatus enumerates Sprint statuses.
type SprintStatus string

const (
	SprintStatusBacklog   SprintStatus = "backlog"
	SprintStatusActive    SprintStatus = "active"
	SprintStatusCompleted SprintStatus = "completed"
)

// Sprint — domain Sprint value object.
type Sprint struct {
	ID     SprintID
	Title  string
	Goal   string
	Status SprintStatus
	Tasks  []TaskID
}

// CanStart reports whether the sprint can be started — only backlog sprints qualify.
func (s Sprint) CanStart() bool {
	return s.Status == SprintStatusBacklog
}

// IsActive reports whether the sprint is in progress.
// Invariant method that replaces the Status == "active" literal
// comparisons in pkg/sprint.
func (s Sprint) IsActive() bool {
	return s.Status == SprintStatusActive
}

// IsCompleted reports whether the sprint is completed.
// Invariant method that replaces the Status == "completed" literal
// comparisons in pkg/sprint.
func (s Sprint) IsCompleted() bool {
	return s.Status == SprintStatusCompleted
}

// CanComplete reports whether the sprint can be completed: it must be active
// and every Task must be done or cancelled.
// doneTasks: the set of Task IDs that are done or cancelled.
func (s Sprint) CanComplete(doneTasks map[TaskID]bool) bool {
	if s.Status != SprintStatusActive {
		return false
	}
	for _, tid := range s.Tasks {
		if !doneTasks[tid] {
			return false
		}
	}
	return true
}

// Progress returns the completion ratio (0.0~1.0). Returns 0.0 when Tasks is empty.
// doneTasks: the set of Task IDs that are done or cancelled.
func (s Sprint) Progress(doneTasks map[TaskID]bool) float64 {
	if len(s.Tasks) == 0 {
		return 0.0
	}
	done := 0
	for _, tid := range s.Tasks {
		if doneTasks[tid] {
			done++
		}
	}
	return float64(done) / float64(len(s.Tasks))
}
