// Package events — defines the hstl workflow event enum.
//
// Previously, project configuration mixed kebab-case (task-start),
// snake_case (current_sprint), and dot notation (sprint.started) for event
// keys (D07). The valid event list was nowhere documented, so typos were
// silently ignored (D08).
//
// This package: 1) standardises event names on dot notation, 2) declares
// them as Go constants for compile-time safety, and 3) exposes
// `IsKnownEvent` / `SuggestEvent` for runtime validation and typo hints.
//
// Convention:
//   - event name = "{scope}.{action}" (e.g. task.start, sprint.complete)
//   - scope: task | sprint (future scopes may include workflow, session, ...)
//   - action: start | complete | reopen, ...
//
// See: docs/08-references/research/project-config-yaml-redesign-research.md
// §2.2 D07/D08, §4.2 I03.
package events

import "strings"

// Event is the string type for hstl workflow events. Used for constant
// comparisons and map keys.
type Event string

// Known event names for use in reminders / hooks / etc.
const (
	EventTaskStart      Event = "task.start"
	EventTaskComplete   Event = "task.complete"
	EventSprintStart    Event = "sprint.start"
	EventSprintComplete Event = "sprint.complete"
)

// knownEventsOrder is the recommended enumeration order for documentation
// and schema generation. Maps iterate in nondeterministic order, so this
// slice is maintained separately.
var knownEventsOrder = []Event{
	EventTaskStart,
	EventTaskComplete,
	EventSprintStart,
	EventSprintComplete,
}

// knownEventsSet provides O(1) membership checks.
var knownEventsSet = map[Event]struct{}{
	EventTaskStart:      {},
	EventTaskComplete:   {},
	EventSprintStart:    {},
	EventSprintComplete: {},
}

// KnownEvents returns the ordered list of events. Used for schema enum
// generation and typo suggestions. Returns a fresh slice on each call so
// callers may mutate it safely.
func KnownEvents() []Event {
	out := make([]Event, len(knownEventsOrder))
	copy(out, knownEventsOrder)
	return out
}

// KnownEventStrings is used wherever JSON Schema generation needs a string slice.
func KnownEventStrings() []string {
	out := make([]string, len(knownEventsOrder))
	for i, e := range knownEventsOrder {
		out[i] = string(e)
	}
	return out
}

// IsKnown reports whether the given string is a valid event name.
func IsKnown(name string) bool {
	_, ok := knownEventsSet[Event(name)]
	return ok
}

// Suggest returns the known event name closest to the given string.
// Used as a typo hint when no exact match exists. An empty return value
// means no candidate was found.
//
// Matching strategy (simple and predictable):
//  1. Highest priority: differs only in letter case.
//  2. Convert kebab-case / snake_case to dot notation, then match.
//  3. Substring containment — partial match.
//  4. No Levenshtein distance (keeps it simple and avoids external deps).
func Suggest(name string) Event {
	if name == "" {
		return ""
	}

	lower := strings.ToLower(name)

	// 1. Case-insensitive match.
	for _, e := range knownEventsOrder {
		if strings.EqualFold(string(e), name) {
			return e
		}
	}

	// 2. kebab/snake -> dot normalisation.
	normalized := strings.ReplaceAll(strings.ReplaceAll(lower, "-", "."), "_", ".")
	for _, e := range knownEventsOrder {
		if string(e) == normalized {
			return e
		}
	}

	// 3. Substring containment (e.g. "task" matches both "task.start" and "task.complete").
	for _, e := range knownEventsOrder {
		if strings.Contains(string(e), lower) || strings.Contains(lower, string(e)) {
			return e
		}
	}

	return ""
}
