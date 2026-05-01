// Package sprint exposes the high-level sprint use cases as a service adapter
// that satisfies the app.SprintService contract. The adapter is stateless and
// forwards each call to the corresponding package-level function.
package sprint

import (
	"fmt"
)

// ServiceAdapter exposes the sprint package functions as the app.SprintService contract.
type ServiceAdapter struct{}

// NewServiceAdapter returns the default sprint service adapter.
func NewServiceAdapter() *ServiceAdapter { return &ServiceAdapter{} }

// List forwards to ListFromDB. Returns []domain.SprintRecord.
func (ServiceAdapter) List(status string) (any, error) {
	return ListFromDB(status)
}

// Get forwards to GetFromDB. Returns *domain.SprintRecord.
func (ServiceAdapter) Get(sprintID string) (any, error) {
	return GetFromDB(sprintID)
}

// Create forwards to Create. Tasks are normalised from []any to []map[string]any.
func (ServiceAdapter) Create(id, title, goal string, tasks []any) (any, error) {
	var converted []map[string]any
	if len(tasks) > 0 {
		converted = make([]map[string]any, 0, len(tasks))
		for _, t := range tasks {
			m, ok := t.(map[string]any)
			if !ok {
				return nil, fmt.Errorf("sprint task entry is not map[string]any: %T", t)
			}
			converted = append(converted, m)
		}
	}
	return Create(id, title, goal, converted)
}

// Start forwards to Start. Returns *domain.SprintStartResult.
func (ServiceAdapter) Start(sprintID string) (any, error) {
	return Start(sprintID)
}

// Complete forwards to Complete. Returns *domain.SprintCompleteResult.
func (ServiceAdapter) Complete(sprintID string) (any, error) {
	return Complete(sprintID)
}

// Discard forwards to Discard. Returns *domain.SprintDiscardResult.
func (ServiceAdapter) Discard(sprintID, reason string, returnTasks, dbOnly bool) (any, error) {
	return Discard(sprintID, reason, returnTasks, dbOnly)
}

// UpdateFields forwards to UpdateFields.
func (ServiceAdapter) UpdateFields(id string, title, goal, status, folder *string) error {
	return UpdateFields(id, title, goal, status, folder)
}

// AggregateProgress forwards to AggregateProgress. Returns *domain.ProgressResult.
func (ServiceAdapter) AggregateProgress(sprintID string) (any, error) {
	return AggregateProgress(sprintID)
}

// CheckCreateGuard forwards to CheckCreateGuard.
func (ServiceAdapter) CheckCreateGuard(id, title string) error {
	return CheckCreateGuard(id, title)
}

// AppendAutoVerifyTask forwards to AppendAutoVerifyTask. Returns *domain.CreateResult.
func (ServiceAdapter) AppendAutoVerifyTask(sprintID string) (any, error) {
	return AppendAutoVerifyTask(sprintID)
}

// FindSprintOnDisk forwards to FindSprintOnDisk. Returns *domain.LocatedSprint.
func (ServiceAdapter) FindSprintOnDisk(sprintID string) (any, error) {
	return FindSprintOnDisk(sprintID)
}

// Reconcile forwards to Reconcile. Returns *domain.ReconcileResult.
func (ServiceAdapter) Reconcile(sprintID string, dryRun bool) (any, error) {
	return Reconcile(sprintID, dryRun)
}

// UpdateMDField forwards to UpdateSprintMDField.
func (ServiceAdapter) UpdateMDField(path, field, value string) error {
	return UpdateSprintMDField(path, field, value)
}
