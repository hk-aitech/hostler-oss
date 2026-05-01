// Package task — TaskService adapter.
// Wraps the ~27 cmd-facing public functions of pkg/task in a struct so
// the app.TaskService interface is satisfied. Struct parameters are
// accepted as `any` and the adapter performs concrete-type assertions —
// this keeps app from importing pkg/task and preserves the layering.
// A later phase will merge this with the ports.TaskStore primitive and
// neutralise the return types into domain/task.
package task

import (
	"fmt"

	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/internal/domain"
	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/internal/ports"
)

// ServiceAdapter exposes the pkg/task package-level functions through
// the app.TaskService contract (stateless).
// trac: HAR-CM008
type ServiceAdapter struct{}

// NewServiceAdapter constructs the default Task service adapter.
func NewServiceAdapter() *ServiceAdapter { return &ServiceAdapter{} }

// ── CRUD / status transitions ──────────────────────────────────────

// Create delegates to pkg/task.Create.
func (ServiceAdapter) Create(title, taskType, sprint, priority, estimate, summary string, dependsOn []string) (any, error) {
	return Create(title, taskType, sprint, priority, estimate, summary, dependsOn)
}

// Get delegates to pkg/task.Get.
// trac: WRK-QR001
func (ServiceAdapter) Get(taskID string) (any, error) { return Get(taskID) }

// List delegates to pkg/task.List.
// trac: WRK-QR002
func (ServiceAdapter) List(sprintPtr, statusPtr *string) (any, error) {
	return List(sprintPtr, statusPtr)
}

// ListFiltered delegates to pkg/task.ListFiltered. filter is asserted to
// ports.TaskListFilter.
// trac: WRK-QR002
func (ServiceAdapter) ListFiltered(filter any) (any, error) {
	f, ok := filter.(ports.TaskListFilter)
	if !ok {
		return nil, fmt.Errorf("ListFiltered filter type error: %T (want ports.TaskListFilter)", filter)
	}
	return ListFiltered(f)
}

// Next delegates to pkg/task.Next.
// trac: HAR-CM017
func (ServiceAdapter) Next() (any, error) { return Next() }

// Start delegates to pkg/task.Start.
// trac: HAR-CM009
func (ServiceAdapter) Start(taskID string) (any, error) { return Start(taskID) }

// Complete delegates to pkg/task.Complete.
// trac: HAR-CM010
func (ServiceAdapter) Complete(taskID string, skipHarness bool) (any, error) {
	return Complete(taskID, skipHarness)
}

// Reopen delegates to pkg/task.Reopen.
// trac: HAR-CM011
func (ServiceAdapter) Reopen(taskID, reason, newStatus string) (any, error) {
	return Reopen(taskID, reason, newStatus)
}

// Update delegates to pkg/task.Update. req must be *domain.UpdateRequest.
// trac: HAR-CM015
func (ServiceAdapter) Update(req any) (any, error) {
	r, ok := req.(*domain.UpdateRequest)
	if !ok {
		return nil, fmt.Errorf("Update req type error: %T (want *task.UpdateRequest)", req)
	}
	return Update(r)
}

// DeleteWithOptions delegates to pkg/task.DeleteWithOptions. opts must be
// DeleteOptions.
// trac: HAR-CM016
func (ServiceAdapter) DeleteWithOptions(taskID, reason string, opts any) (any, error) {
	o, ok := opts.(DeleteOptions)
	if !ok {
		return nil, fmt.Errorf("DeleteWithOptions opts type error: %T (want task.DeleteOptions)", opts)
	}
	return DeleteWithOptions(taskID, reason, o)
}

// AssignSprint delegates to pkg/task.AssignSprint.
// trac: HAR-CM013
func (ServiceAdapter) AssignSprint(taskIDs []string, sprintID string) (any, error) {
	return AssignSprint(taskIDs, sprintID)
}

// UnassignSprint delegates to pkg/task.UnassignSprint.
// trac: HAR-CM014
func (ServiceAdapter) UnassignSprint(taskIDs []string) (any, error) {
	return UnassignSprint(taskIDs)
}

// Checkpoint delegates to pkg/task.Checkpoint. Returns
// *domain.CheckpointResult.
// trac: HAR-CM012
func (ServiceAdapter) Checkpoint(taskID, reason string) (any, error) {
	return Checkpoint(taskID, reason)
}

// ── Utilities / helpers ────────────────────────────────────────────

// ResolvePreset returns the summary-heuristic preset config.
// Returns: HeuristicConfig (value).
func (ServiceAdapter) ResolvePreset(name string) any { return ResolvePreset(name) }

// HeuristicCheck runs the summary-heuristic verification. cfg must be a
// HeuristicConfig.
func (ServiceAdapter) HeuristicCheck(summary string, cfg any) ([]string, bool) {
	c, ok := cfg.(HeuristicConfig)
	if !ok {
		return []string{fmt.Sprintf("HeuristicCheck cfg type error: %T", cfg)}, false
	}
	return HeuristicCheck(summary, c)
}

// HasPlaceholderBody reports whether placeholder content remains in the
// Task file.
// trac: WRK-QR001
func (ServiceAdapter) HasPlaceholderBody(filePath string) bool {
	return HasPlaceholderBody(filePath)
}

// GetGitChangedFiles returns the list of files git reports as changed.
func (ServiceAdapter) GetGitChangedFiles() []string { return GetGitChangedFiles() }
