// Package harness — HarnessService adapter.
// Wraps the package-level high-level harness functions in a struct so
// the app.HarnessService interface is satisfied. The concrete return
// types (domain.HarnessGetResult / HarnessResult / domain.CriteriaResult)
// are exposed as `any` and cmd re-interprets them via type assertion
// (the same strategy as AuditFileOps).
// A later phase will neutralise the return types and merge this with
// the ports.HarnessStore primitive into a single adapter.
package harness

// ServiceAdapter exposes the pkg/harness package-level functions
// through the app.HarnessService contract. Stateless.
type ServiceAdapter struct{}

// NewServiceAdapter constructs the default Harness service adapter.
func NewServiceAdapter() *ServiceAdapter { return &ServiceAdapter{} }

// Get delegates to the HarnessGet package-level function.
// Returns *domain.HarnessGetResult (a pointer — existing cmd code
// expects this).
func (ServiceAdapter) Get(entityType, entityID string) (any, error) {
	r, err := HarnessGet(entityType, entityID)
	if err != nil {
		return nil, err
	}
	return &r, nil
}

// Check delegates to the HarnessCheck package-level function.
func (ServiceAdapter) Check(entityType, entityID, itemID, evidence, actor string) (any, error) {
	return HarnessCheck(entityType, entityID, itemID, evidence, actor)
}

// CheckAll delegates to the CheckHarness package-level function.
func (ServiceAdapter) CheckAll(entityType, entityID string) (any, error) {
	return CheckHarness(entityType, entityID)
}

// AutoCheckCriteria delegates to the AutoCheckCriteria package-level
// function.
func (ServiceAdapter) AutoCheckCriteria(taskID string) (any, error) {
	return AutoCheckCriteria(taskID)
}
