// Task priority input case normalization plus enum validation.
//
// Background: when users pass uppercase input such as `task create --priority P0`,
// the raw value gets persisted, leaving P0 and p0 mixed in the database and file
// frontmatter and causing drift across queries, sorting, and validation. The
// normalizePriority function below normalizes the value to the canonical form
// (lowercase p0~p3) at the CLI entry point.
package cmd

import (
	"fmt"
	"strings"
)

// validPriorities holds the allowed canonical priority values. SSOT.
var validPriorities = map[string]struct{}{
	"p0": {},
	"p1": {},
	"p2": {},
	"p3": {},
}

// normalizePriority converts a priority input string to the canonical form.
// Trims whitespace, lowercases, and validates against the enum. Empty input
// is allowed (the flag is optional).
//
// Accepted: "p0", "P0", " p0 ", "P1" -> "p0", "p0", "p0", "p1"
// Rejected: "critical", "p4", "0" -> error
func normalizePriority(raw string) (string, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "", nil
	}
	lower := strings.ToLower(trimmed)
	if _, ok := validPriorities[lower]; !ok {
		return "", fmt.Errorf("priority value %q is not allowed - must be one of p0|p1|p2|p3", raw)
	}
	return lower, nil
}
