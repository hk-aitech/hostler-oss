// Package task — Task status alias normalisation.
// Background: `hstl task list --status in-progress` matches, but
// `--status in_progress` returned count=0. The DB stores "in-progress"
// (hyphen) while the JSON response field convention is "in_progress"
// (snake_case), creating ambiguity for users / AIs about which form to
// use. The fix is to normalise the input form to a canonical form at the
// CLI entry point so every DB query sees a single shape.
// This file provides the single function `Canonicalize` so multiple call
// sites (task list, backlog sync, task reopen --status, etc.) share one
// path. Existing DB data is not migrated — the canonical form already
// matches what the DB stores (hyphen), so this is not a breaking change.
package task

import (
	"fmt"
	"strings"
)

// Task status canonical form. DB storage and Task file frontmatter both
// use the hyphen variants. The `in_progress` field name in brief JSON
// responses is purely a JSON naming convention (snake_case); the status
// value itself stays consistent as "in-progress".
const (
	StatusTodo       = "todo"
	StatusInProgress = "in-progress"
	StatusDone       = "done"
	StatusBlocked    = "blocked"
	StatusReopened   = "reopened"
)

// canonicalStatuses — set of valid values that Canonicalize may return.
// Any normalised result outside this set produces an error.
var canonicalStatuses = map[string]struct{}{
	StatusTodo:       {},
	StatusInProgress: {},
	StatusDone:       {},
	StatusBlocked:    {},
	StatusReopened:   {},
}

// Canonicalize normalises a user-supplied status string to its canonical
// form. Steps:
// 1. strings.TrimSpace — strip leading/trailing whitespace.
// 2. strings.ToLower — normalise case.
// 3. Replace underscores/spaces with hyphens — "in_progress" /
// "in progress" -> "in-progress".
// 4. Validate membership in canonicalStatuses — unknown values error.
// An empty string means "no filter" so it returns ("", nil) instead of
// an error. The caller distinguishes via the *string pointer nil-check
// (preserving the existing CLI flag convention).
// Returned errors must be surfaced with a recovery hint, so callers
// should wrap the error message.
// trac: HAR-CM015
func Canonicalize(raw string) (string, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "", nil
	}
	lower := strings.ToLower(trimmed)
	// Underscore + space -> hyphen. Multiple-hyphen variants are not in
	// the valid set, so they fall out at the membership check below.
	replaced := strings.NewReplacer("_", "-", " ", "-").Replace(lower)
	if _, ok := canonicalStatuses[replaced]; !ok {
		return "", fmt.Errorf(
			"unknown status %q (allowed: todo / in-progress / done / blocked / reopened)",
			raw,
		)
	}
	return replaced, nil
}

// CanonicalizeOrDefault calls Canonicalize but returns the original input
// on error. Used for backward-compat paths (call sites that previously
// passed flag values straight to DB queries without validation) during
// a gradual migration. New code should call Canonicalize directly and
// surface the error to the user.
// trac: HAR-CM015
func CanonicalizeOrDefault(raw string) string {
	if s, err := Canonicalize(raw); err == nil {
		return s
	}
	return raw
}
