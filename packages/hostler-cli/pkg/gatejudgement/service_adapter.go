// Package gatejudgement — GateLogService ServiceAdapter.
//
// ADR-001 §4.2 Phase D. Routes the GateJudgement entry points used by
// cmd/gate_log.go and cmd/rules_run.go — GlobalStore() admin operations
// and RecordJudgement sampling — through app.GateLogService.
//
// Pass-through principles:
//   - Record calls pkg/gatejudgement.RecordJudgement (reuses the
//     disabled/sampling helper instead of re-implementing it).
//   - The rest (QueryRecord / Query / Stats / MarkFalsePositive) are
//     admin operations through GlobalStore.
//
// The GlobalStore singleton stays — preserves the testing convention of
// path isolation via ResetGlobalStore/SetGlobalStore.
package gatejudgement

import (
	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/internal/ports"
)

// ServiceAdapter is the default implementation of the GateLogService port.
// Stateless — pass-through over GlobalStore() and RecordJudgement().
type ServiceAdapter struct{}

// NewServiceAdapter constructs the default ServiceAdapter.
func NewServiceAdapter() *ServiceAdapter { return &ServiceAdapter{} }

// Record is a pass-through to pkg/gatejudgement.RecordJudgement.
// Includes the disabled check and per-verdict sampling (WARN+ all, PASS sampled, SKIP skipped).
func (ServiceAdapter) Record(r ports.JudgementRecord) {
	RecordJudgement(r)
}

// RecordDirect is a pass-through to GlobalStore().Record — bypasses
// sampling, records all entries. Used by the admin path (cmd/gate_log.go
// inject/record).
func (ServiceAdapter) RecordDirect(r ports.JudgementRecord) error {
	return GlobalStore().Record(r)
}

// Query is a pass-through to GlobalStore().Query.
func (ServiceAdapter) Query(filter ports.JudgementQuery) ([]ports.JudgementRecord, error) {
	return GlobalStore().Query(filter)
}

// Stats is a pass-through to GlobalStore().Stats.
func (ServiceAdapter) Stats(filter ports.JudgementQuery) (*ports.JudgementStats, error) {
	return GlobalStore().Stats(filter)
}

// MarkFalsePositive is a pass-through to GlobalStore().MarkFalsePositive.
func (ServiceAdapter) MarkFalsePositive(recordID string, label ports.FalsePositiveLabel) error {
	return GlobalStore().MarkFalsePositive(recordID, label)
}
