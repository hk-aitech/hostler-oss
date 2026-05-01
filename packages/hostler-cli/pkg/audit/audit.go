// Package audit provides the audit log service.
// All state changes (Task creation/transition, Sprint changes, etc.) are
// recorded into the audit_events table.
package audit

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/internal/ports"
	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/pkg/envalias"
	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/pkg/store"
)

// Event represents a single audit event.
//
// trac: TRC-EV003
type Event struct {
	ID         int64          `json:"id"`
	Timestamp  string         `json:"timestamp"`
	EventType  string         `json:"event_type"`
	EntityType string         `json:"entity_type"`
	EntityID   string         `json:"entity_id"`
	Actor      string         `json:"actor"`
	Details    map[string]any `json:"details,omitempty"`
	SessionID  string         `json:"session_id,omitempty"`
}

// LogEvent records an audit event to the audit_events table.
//
// eventType  : event kind (e.g. "task.created", "task.transitioned")
// entityType : target entity kind (e.g. "task", "sprint")
// entityID   : target entity ID (e.g. "T009", "sprint-01")
// actor      : actor (an empty string defaults to "claude")
// details    : extra JSON details (may be nil)
// sessionID  : Claude session identifier (may be empty)
//
// trac: TRC-EV003
func LogEvent(eventType, entityType, entityID, actor string, details map[string]any, sessionID string) error {
	gs := store.Get()
	if gs == nil {
		return fmt.Errorf("store is not initialized (call db.InitDB() + store.Init() first)")
	}
	if actor == "" {
		actor = "claude"
	}
	return gs.AppendAuditEvent(ports.AuditEvent{
		EventType:  eventType,
		EntityType: entityType,
		EntityID:   entityID,
		ActorID:    actor,
		SessionID:  sessionID,
		Details:    details,
	})
}

// QueryFilter holds filter conditions for QueryEvents.
type QueryFilter struct {
	EntityType string // entity-type filter (e.g. "task")
	EntityID   string // entity-ID filter (e.g. "T009")
	EventType  string // event-type filter (e.g. "task.created")
	Since      string // start time as ISO string (inclusive)
	Until      string // end time as ISO string (inclusive)
	Actor      string // actor filter
	Limit      int    // max records to return (0 uses default)
}

// queryLimitDefault is the default audit-log query maximum.
const queryLimitDefault = 100

// getQueryLimit returns the audit-log query maximum.
// If HSTL_AUDIT_QUERY_LIMIT is set, its value is used.
func getQueryLimit() int {
	if raw := strings.TrimSpace(envalias.Lookup("AUDIT_QUERY_LIMIT")); raw != "" {
		if n, err := strconv.Atoi(raw); err == nil && n > 0 {
			return n
		}
	}
	return queryLimitDefault
}

// QueryEvents returns audit events matching the filter conditions.
// Results are returned in newest-first order.
//
// trac: TRC-QR004
func QueryEvents(f QueryFilter) ([]Event, error) {
	gs := store.Get()
	if gs == nil {
		return nil, fmt.Errorf("store is not initialized (call db.InitDB() + store.Init() first)")
	}
	if f.Limit <= 0 {
		f.Limit = getQueryLimit()
	}

	portEvents, err := gs.QueryAuditEvents(ports.AuditQueryFilter{
		EntityType: f.EntityType,
		EntityID:   f.EntityID,
		EventType:  f.EventType,
		Actor:      f.Actor,
		Since:      f.Since,
		Until:      f.Until,
		Limit:      f.Limit,
	})
	if err != nil {
		return nil, fmt.Errorf("audit_events query failed: %w", err)
	}

	events := make([]Event, 0, len(portEvents))
	for _, pe := range portEvents {
		// Fix the missing Timestamp mapping. The SQLite adapter's Scan
		// populates timestamp on ports.AuditEvent, but this mapping step
		// did not copy it to Event.Timestamp, so `hstl audit -o json`
		// always returned an empty string. Event has a Timestamp field
		// already, so a straight copy fixes it.
		e := Event{
			ID:         pe.ID, // expose audit event rowid
			EventType:  pe.EventType,
			EntityType: pe.EntityType,
			EntityID:   pe.EntityID,
			Actor:      pe.ActorID,
			SessionID:  pe.SessionID,
			Timestamp:  pe.Timestamp,
		}
		if pe.Details != nil {
			if m, ok := pe.Details.(map[string]any); ok {
				e.Details = m
			} else {
				// If the source is a string, try parsing it as JSON.
				if s, ok := pe.Details.(string); ok {
					_ = json.Unmarshal([]byte(s), &e.Details)
				}
			}
		}
		events = append(events, e)
	}
	return events, nil
}

// SummarizeEvents aggregates event counts by event type (token-saving
// summary mode).
// since/until are ISO strings (empty means no filter).
//
// trac: TRC-QR004
func SummarizeEvents(since, until string) (map[string]int, error) {
	gs := store.Get()
	if gs == nil {
		return nil, fmt.Errorf("store is not initialized (call db.InitDB() + store.Init() first)")
	}
	result, err := gs.SummarizeAuditEvents(since, until)
	if err != nil {
		return nil, fmt.Errorf("audit_events aggregation failed: %w", err)
	}
	return result, nil
}
