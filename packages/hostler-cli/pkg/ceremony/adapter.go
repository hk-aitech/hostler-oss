// Package ceremony — CeremonyCollector adapter.
// Wraps the existing CollectSprintStart / CollectSprintComplete /
// CollectTaskStart / CollectTaskComplete functions in a struct that
// satisfies ports.CeremonyCollector. CLI commands route through this
// adapter.
package ceremony

import (
	"fmt"

	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/internal/ports"
)

// CeremonyCollectorAdapter dispatches ceremony collection by kind.
type CeremonyCollectorAdapter struct{}

// NewCeremonyCollectorAdapter constructs the default ceremony collector
// adapter.
func NewCeremonyCollectorAdapter() *CeremonyCollectorAdapter {
	return &CeremonyCollectorAdapter{}
}

// Collect gathers the ceremony payload corresponding to kind.
// The result is the domain struct wrapped in any; callers type-assert
// per kind:
//	res, _ := c.Collect(ports.CeremonySprintStart, "sprint-id")
//	brief := res.(*ceremony.SprintStartCeremony)
func (CeremonyCollectorAdapter) Collect(kind ports.CeremonyKind, targetID string) (any, error) {
	switch kind {
	case ports.CeremonySprintStart:
		return CollectSprintStart(targetID), nil
	case ports.CeremonySprintComplete:
		return CollectSprintComplete(targetID), nil
	case ports.CeremonyTaskStart:
		return CollectTaskStart(targetID), nil
	case ports.CeremonyTaskComplete:
		return CollectTaskComplete(targetID), nil
	default:
		return nil, fmt.Errorf("unknown CeremonyKind: %q", string(kind))
	}
}

// Compile-time check — CeremonyCollectorAdapter satisfies the
// ports.CeremonyCollector contract.
var _ ports.CeremonyCollector = (*CeremonyCollectorAdapter)(nil)
