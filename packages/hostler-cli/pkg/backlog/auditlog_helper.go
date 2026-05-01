package backlog

import "github.com/hk-aitech/hostler-oss/packages/hostler-cli/pkg/audit"

// auditLog — local helper. Records audit events from the adapter.
func auditLog(eventType string, details map[string]any) error {
	return audit.LogEvent(eventType, "backlog", "sync", "", details, "")
}
