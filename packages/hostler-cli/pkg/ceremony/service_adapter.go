// Package ceremony — CeremonyService ServiceAdapter.
// Pass-through adapter that lets cmd/sprint.go and cmd/task.go reach
// pkg/ceremony's CollectSprintStart / CollectSprintComplete /
// CollectTaskStart / CollectTaskComplete via app.CeremonyService.
// The actual return values are concrete types (*SprintStartCeremony,
// etc.), but the interface returns `any` so that internal/app does not
// need to import pkg/ceremony — cmd performs the concrete-type assertion.
package ceremony

// ServiceAdapter is the default implementation of the CeremonyService
// port. It is a pass-through to the four pkg/ceremony Collect*
// functions.
type ServiceAdapter struct{}

// NewServiceAdapter constructs the default ServiceAdapter.
func NewServiceAdapter() *ServiceAdapter { return &ServiceAdapter{} }

// CollectSprintStart is a pass-through to pkg/ceremony.CollectSprintStart.
// Returns *SprintStartCeremony.
func (ServiceAdapter) CollectSprintStart(sprintID string) any {
	return CollectSprintStart(sprintID)
}

// CollectSprintComplete is a pass-through to
// pkg/ceremony.CollectSprintComplete.
// Returns *SprintCompleteCeremony.
func (ServiceAdapter) CollectSprintComplete(sprintID string) any {
	return CollectSprintComplete(sprintID)
}

// CollectTaskStart is a pass-through to pkg/ceremony.CollectTaskStart.
// Returns *TaskStartCeremony.
func (ServiceAdapter) CollectTaskStart(taskID string) any {
	return CollectTaskStart(taskID)
}

// CollectTaskComplete is a pass-through to pkg/ceremony.CollectTaskComplete.
// Returns *TaskCompleteCeremony.
func (ServiceAdapter) CollectTaskComplete(taskID string) any {
	return CollectTaskComplete(taskID)
}
