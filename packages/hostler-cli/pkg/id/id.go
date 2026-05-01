// Package id provides IDStore-backed atomic ID allocation.
package id

import (
	"fmt"
	"os"

	pkglog "github.com/hk-aitech/hostler-oss/packages/hostler-cli/pkg/log"
	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/pkg/store"
)

// idLogger routes id.go diagnostic logs through the global proxy.
var idLogger = pkglog.NewDefault()

// TaskIDNext atomically issues a Task ID.
// Increments the 'task' counter in the id_counters table inside a
// transaction, then returns the "T{NNN}" form (3-digit zero-padded).
// Examples: "T009", "T010", "T100", "T1000".
//
// Self-heals when the counter is smaller than the actual MAX in the tasks
// table. After DB drift the counter could have been reset to 0,
// causing collisions with existing Tasks; this prevents that regression.
//
// Considers SPRINT.md / BACKLOG.md placeholders too. Before calling
// HealAndGetNextTaskID, ScanPlaceholderTaskIDs finds the maximum number
// in the files and BumpTaskCounter raises the counter accordingly. When
// the placeholder is larger than the DB MAX (e.g. a boundary state where
// the higher ID is recorded only in SPRINT.md), the next issued ID is
// bumped past it to prevent a collision. Scan failures are ignored
// (legacy fallback) — a missing works/ directory is normal in CI/test.
// TaskIDSanityMax — WARN threshold for unrealistic jumps. Even after the
// body scan was retired, file-name-level jumps beyond this trigger a
// stderr WARN so operators can trace the cause.
const TaskIDSanityMax = 9999

func TaskIDNext() (string, error) {
	ids, err := store.MustGetIDStore()
	if err != nil {
		return "", fmt.Errorf("task counter self-heal failed: %w", err)
	}
	// Bump the counter to the placeholder max (skip on failure).
	if workRoot, wrErr := os.Getwd(); wrErr == nil {
		if placeholderMax, scanErr := ScanPlaceholderTaskIDs(workRoot); scanErr == nil && placeholderMax > 0 {
			// Detect unrealistic max (e.g. 9999+) — WARN and skip the bump.
			if placeholderMax > TaskIDSanityMax {
				idLogger.Warn("placeholder max exceeds sanity threshold — bump skipped",
					"placeholder_max", placeholderMax, "sanity_max", TaskIDSanityMax,
					"hint", "review oversized Task ID filenames under works/")
			} else {
				_ = ids.BumpTaskCounter(placeholderMax)
			}
		}
	}
	return ids.HealAndGetNextTaskID()
}

// NextCounter atomically increments the counter named counterKey and returns the new value.
func NextCounter(counterKey string) (int64, error) {
	ids, err := store.MustGetIDStore()
	if err != nil {
		return 0, fmt.Errorf("counter increment failed: %w", err)
	}
	return ids.NextCounter(counterKey)
}
