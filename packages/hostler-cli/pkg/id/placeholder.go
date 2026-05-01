package id

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// Scan SPRINT.md / BACKLOG.md for placeholder Task IDs.
//
// Background: HealAndGetNextTaskID issues the next ID based only on the
// MAX of `tasks.task_id` in the DB. However, an ID like "T130" can be
// tentatively recorded in SPRINT.md before the DB has it (mid-step in a
// mailbox accept / a manual user reservation). If a mailbox accept or
// task create requests a new ID during that interval, the same number is
// reissued and a collision occurs (the boundary case).
//
// This module scans Task ID literals at the following locations and
// returns the maximum number seen:
//
//	works/tasks/BACKLOG.md (unassigned task table)
//	works/sprints/{active,backlog,completed}/*/SPRINT.md (Sprint-body task tables)
//	works/sprints/active/*/tasks/T*.md (filenames) (already-created task files)
//	works/tasks/T*.md / works/tasks/completed/T*.md (filenames)
//
// Design trade-offs:
//
//   - Regex `\bT(\d{1,5})\b` matching -> potential false positives (a body
//     might cite an example ID). Acceptable — BumpTaskCounter only
//     raises the floor, so a slightly elevated false-positive max merely
//     skips an ID; there is no collision.
//   - File I/O cost is full SPRINT.md scan (currently ~20 files) +
//     Task filenames (100-200) -> tens of KB, a few ms. Cheap enough to
//     run on every TaskIDNext call.
//   - On error returns (0, err) — callers log the error, treat as 0, and
//     skip the bump (falling back to HealAndGetNextTaskID's DB-only path).

var taskIDLineRe = regexp.MustCompile(`\bT(\d{1,5})\b`)

// ScanPlaceholderTaskIDs recursively scans works/ under workRoot and
// returns the largest Task ID number found. Returns (0, nil) when none.
// workRoot points to the repo root.
//
// Excludes works/sprints/completed/ from the scan. History areas are
// immutable post-rename (KB O003), and IDs there are already covered by
// `tasks.task_id` MAX, so placeholder bump is unnecessary. Root-cause
// fix for the drift seen during dogfood where each invocation re-bumped
// the counter past 10000 from oversized placeholder references.
func ScanPlaceholderTaskIDs(workRoot string) (int64, error) {
	if workRoot == "" {
		return 0, nil
	}
	worksDir := filepath.Join(workRoot, "works")
	if _, err := os.Stat(worksDir); err != nil {
		if os.IsNotExist(err) {
			return 0, nil
		}
		return 0, fmt.Errorf("works directory stat failed: %w", err)
	}

	// Excluded path — all of works/sprints/completed/ (history).
	completedSprintsDir := filepath.Join(worksDir, "sprints", "completed")

	var maxN int64

	// 1. Markdown body scan (SPRINT.md, BACKLOG.md, Task body).
	err := filepath.WalkDir(worksDir, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return nil // skip — this is a monitoring tool
		}
		if d.IsDir() {
			// Skip the entire completed-sprint history area.
			if path == completedSprintsDir {
				return filepath.SkipDir
			}
			return nil
		}
		name := d.Name()
		// Extract a Task ID from the filename (e.g. patterns like
		// works/sprints/active/sprint-14/tasks/T219-...).
		if strings.HasPrefix(name, "T") && strings.HasSuffix(name, ".md") {
			if n := parseLeadingTaskID(name); n > maxN {
				maxN = n
			}
		}
		// SPRINT.md / BACKLOG.md body scanning is retired. Reason: Task
		// IDs cited in narratives like KPT retros, lessons, and
		// follow-up Task tables were caught as max and made the counter
		// self-reinforce. Filename-only scanning still satisfies the
		// placeholder-protection intent — once a Task is actually issued,
		// its file is created and captured by the filename scan.
		// Placeholders that exist only in SPRINT.md bodies are handled
		// by the backlog sync / task assign flow.
		return nil
	})
	if err != nil {
		return maxN, fmt.Errorf("works scan failed: %w", err)
	}
	return maxN, nil
}

// parseLeadingTaskID extracts the numeric portion (e.g. 219 from "T219-xxx.md").
// Returns 0 on no match.
func parseLeadingTaskID(fileName string) int64 {
	// Starts with T, followed by digits, followed by a non-digit
	// (hyphen / dot / etc.).
	m := taskIDLineRe.FindStringSubmatch(fileName)
	if len(m) < 2 {
		return 0
	}
	var n int64
	if _, err := fmt.Sscanf(m[1], "%d", &n); err != nil {
		return 0
	}
	return n
}
