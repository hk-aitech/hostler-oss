// Package task — duplicate-title detection guard.
// Reproduced downstream as / — when an AI calls
// `hstl task create` twice with the same title, both Tasks are created
// with independent IDs and no duplicate detection (each call uses a
// separate crypto/rand pending_ticket, making the two calls fully
// independent). Without explicit user intent, this is almost always an
// AI retry-judgement error.
// This module is invoked at the Create entry point and returns a
// RejectedError when the same title is detected within the last N
// seconds. Window/bypass are tunable via env vars.
// Design:
// title normalization: case-insensitive + whitespace-collapsed
// (defends against typo-style variants).
// within the last N seconds (default 60), same title → BLOCK.
// bypass via HSTL_TASK_CREATE_ALLOW_DUPLICATE_TITLE=1 or CLI flag.
// DB uninitialized / query failure → silent fallback (skip the
// duplicate check).
package task

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/pkg/apperr"
	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/pkg/envalias"
	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/pkg/store"
)

// TaskCreateDuplicateWindowEnv is the env var for the duplicate-detection
// window (seconds).
const TaskCreateDuplicateWindowEnv = "HSTL_TASK_CREATE_DUPLICATE_WINDOW_SEC"

// TaskCreateAllowDuplicateEnv is the env var that bypasses duplicate
// detection (1 = skip).
const TaskCreateAllowDuplicateEnv = "HSTL_TASK_CREATE_ALLOW_DUPLICATE_TITLE"

// duplicateWindowDefaultSec is the default detection window.
const duplicateWindowDefaultSec = 60

// resolveDuplicateWindowSec parses the env var as an int. On failure
// returns the default.
func resolveDuplicateWindowSec() int {
	raw := strings.TrimSpace(envalias.Lookup("TASK_CREATE_DUPLICATE_WINDOW_SEC"))
	if raw == "" {
		return duplicateWindowDefaultSec
	}
	n, err := strconv.Atoi(raw)
	if err != nil || n < 0 {
		return duplicateWindowDefaultSec
	}
	return n
}

// isDuplicateCheckBypassed reports whether the bypass env flag is set.
func isDuplicateCheckBypassed() bool {
	return envalias.Lookup("TASK_CREATE_ALLOW_DUPLICATE_TITLE") == "1"
}

// normalizeTitle converts a title into its canonical comparison form.
// case-insensitive (ToLower)
// trim whitespace at both ends
// collapse multiple whitespace runs (tabs/newlines included) to one
// space
func normalizeTitle(s string) string {
	return strings.Join(strings.Fields(strings.ToLower(s)), " ")
}

// parseTaskCreatedAt converts TaskRecord.CreatedAt into a time.Time.
// Prefers the SQLite DEFAULT format "YYYY-MM-DD HH:MM:SS"; also accepts
// date-only ("YYYY-MM-DD") and RFC3339. On failure returns zero time +
// error.
func parseTaskCreatedAt(s string) (time.Time, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return time.Time{}, fmt.Errorf("empty created_at")
	}
	formats := []string{
		"2006-01-02 15:04:05",
		"2006-01-02T15:04:05Z",
		time.RFC3339,
		"2006-01-02",
	}
	for _, f := range formats {
		if t, err := time.Parse(f, s); err == nil {
			return t.UTC(), nil
		}
	}
	return time.Time{}, fmt.Errorf("unrecognized created_at format: %q", s)
}

// DuplicateCheckResult holds the duplicate-detection result.
type DuplicateCheckResult struct {
	Duplicate     bool
	ExistingID    string
	ExistingTitle string
	CreatedAt     string
}

// FindDuplicateTitle searches for a Task with the same normalized title
// within the last windowSec seconds and returns the first match. Returns
// nil on DB query failure (silent fallback).
// This function does not read env vars — the caller passes the result of
// resolveDuplicateWindowSec() (so tests can inject fixed values).
func FindDuplicateTitle(title string, windowSec int) *DuplicateCheckResult {
	gs := store.Get()
	if gs == nil {
		return nil
	}
	all, err := gs.ListTasks(nil, nil)
	if err != nil || all == nil {
		return nil
	}
	norm := normalizeTitle(title)
	if norm == "" {
		return nil
	}
	cutoff := time.Now().UTC().Add(-time.Duration(windowSec) * time.Second)
	for _, t := range all.Tasks {
		if normalizeTitle(t.Title) != norm {
			continue
		}
		ts, err := parseTaskCreatedAt(t.CreatedAt)
		if err != nil {
			continue
		}
		if ts.After(cutoff) {
			return &DuplicateCheckResult{
				Duplicate:     true,
				ExistingID:    t.TaskID,
				ExistingTitle: t.Title,
				CreatedAt:     t.CreatedAt,
			}
		}
	}
	return nil
}

// CheckDuplicateTitleOrReject is invoked at the Create entry point and
// returns a RejectedError on duplicate detection. Returns nil when the
// bypass env is set or when no duplicate is found.
func CheckDuplicateTitleOrReject(title string) error {
	if isDuplicateCheckBypassed() {
		return nil
	}
	windowSec := resolveDuplicateWindowSec()
	if windowSec == 0 {
		return nil
	}
	res := FindDuplicateTitle(title, windowSec)
	if res == nil || !res.Duplicate {
		return nil
	}
	return &apperr.RejectedError{
		EntityID: res.ExistingID,
		Reason: fmt.Sprintf(
			"a Task with the same title already exists in the last %d seconds (existing: %s, created=%s). Likely an AI duplicate-call.",
			windowSec, res.ExistingID, res.CreatedAt,
		),
		RecoveryHint: fmt.Sprintf(
			"If the duplicate is intentional, retry with the --allow-duplicate-title flag (or %s=1). Otherwise reuse the existing Task %s or change the title.",
			TaskCreateAllowDuplicateEnv, res.ExistingID,
		),
	}
}
