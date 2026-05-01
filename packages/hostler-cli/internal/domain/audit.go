// Package domain defines hstl domain types — cross-cutting types such as
// audit / harness / task.
package domain

import "time"

// AuditEvent — immutable audit-event value object.
type AuditEvent struct {
	ID         string
	EntityType string
	EntityID   string
	Action     string
	Actor      string
	Timestamp  time.Time
	Details    map[string]string
	HMAC       string // signature linking to the previous event in the chain
}

// IsChained reports whether the event is part of an HMAC chain.
func (e AuditEvent) IsChained() bool {
	return e.HMAC != ""
}
