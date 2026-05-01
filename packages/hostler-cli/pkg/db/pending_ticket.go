// Package db — PENDING work_ticket issuance.
//
// Background: a previous hotfix introduced fail-closed + deferred
// rollback so that a failed task create no longer consumes the ID
// counter. That removed orphan IDs but left no audit-log trace of
// "why did it fail?".
//
// To address this, the create entry point pre-issues
// WT-PENDING-{8hex} so that all paths (success, reject, subprocess
// failure, rollback) can be tracked by the same ticket. On success the
// ticket is promoted to the result of GenerateWorkTicket(taskID).
//
// Format: WT-PENDING-{8hex} (e.g. WT-PENDING-a1b2c3d4).
// After promotion: WT-T{id}-{8hex}.
package db

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
)

// pendingTicketPrefix identifies a pre-promotion temporary ticket.
// `WT-PENDING-` alone is enough to filter failure paths via audit_log
// grep.
const pendingTicketPrefix = "WT-PENDING-"

// GeneratePendingTicket issues a temporary work ticket usable before a
// Task ID is allocated. Format: WT-PENDING-{8hex}.
//
// Use sites:
//   - task.Create entry point — for audit logs across the internal
//     failure paths (file/DB/rollback).
//   - `hstl task create` CLI — for audit logs across the summary
//     heuristic / subagent reject paths.
//
// On the success path the value is replaced by
// db.GenerateWorkTicket(taskID), and the PENDING ticket only survives
// in the audit_log. The mapping between the two tickets is observable
// via details.pending_ticket + details.work_ticket on the
// task.created audit event.
func GeneratePendingTicket() (string, error) {
	buf := make([]byte, workTicketRandomBytes)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("failed to generate pending ticket random bytes: %w", err)
	}
	return pendingTicketPrefix + hex.EncodeToString(buf), nil
}

// IsPendingTicket reports whether a ticket is in the PENDING form.
// Used to separate success from failure paths in audit_log queries.
func IsPendingTicket(ticket string) bool {
	return len(ticket) >= len(pendingTicketPrefix) && ticket[:len(pendingTicketPrefix)] == pendingTicketPrefix
}
