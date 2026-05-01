// Package audit — audit_events history reconstruction.
//
// Walks `git log` for works/sprints/completed/*/SPRINT.md and each Task
// file to recover the first-commit timestamp, then rebuilds
// sprint.completed / task.completed audit events. Used to fill gaps lost
// during manual DB-drift merges.
//
// Policy:
//   - Defaults to dry-run — without --apply, no changes are made.
//   - natural-key dedupe — when an audit log already contains the same
//     (event_type, entity_type, entity_id) combination, skip. Existing
//     event timestamps are never overwritten.
//   - task.completed only targets files whose frontmatter has status=done.
package audit

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/internal/ports"
	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/pkg/fileutil"
	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/pkg/store"
)

// RestoreEntry is a single restoration target.
type RestoreEntry struct {
	EventType  string `json:"event_type"`
	EntityType string `json:"entity_type"`
	EntityID   string `json:"entity_id"`
	Timestamp  string `json:"timestamp"`
	Source     string `json:"source"` // file path used as the git log subject
	Action     string `json:"action"` // "restore" | "skip_existing"
}

// RestoreResult summarises a Restore call's results.
type RestoreResult struct {
	DryRun         bool           `json:"dry_run"`
	ScannedSprints int            `json:"scanned_sprints"`
	ScannedTasks   int            `json:"scanned_tasks"`
	Entries        []RestoreEntry `json:"entries"`
	Restored       int            `json:"restored"`
	SkippedExists  int            `json:"skipped_existing"`
}

// statusFrontmatterRe extracts the status field from a Task file's
// frontmatter.
var statusFrontmatterRe = regexp.MustCompile(`(?m)^status:\s*(\w[\w-]*)\s*$`)

// Restore performs an audit restoration based on git log.
// dryRun=true returns the target list without modifying the DB.
//
// trac: TRC-CM004
func Restore(dryRun bool) (*RestoreResult, error) {
	root := fileutil.GetProjectRoot()
	if root == "" {
		return nil, fmt.Errorf("failed to detect the project root")
	}
	gs := store.Get()
	if gs == nil {
		return nil, fmt.Errorf("store not initialised")
	}

	result := &RestoreResult{DryRun: dryRun}

	sprintFiles, err := filepath.Glob(filepath.Join(root, "works", "sprints", "completed", "*", "SPRINT.md"))
	if err != nil {
		return nil, fmt.Errorf("sprint glob failed: %w", err)
	}
	sort.Strings(sprintFiles)
	result.ScannedSprints = len(sprintFiles)

	// Restore sprint.completed events.
	for _, sprintPath := range sprintFiles {
		sprintID := filepath.Base(filepath.Dir(sprintPath))
		entry, appendErr := restoreSprintCompleted(gs, root, sprintPath, sprintID, dryRun)
		if appendErr != nil {
			return nil, appendErr
		}
		if entry != nil {
			result.Entries = append(result.Entries, *entry)
			if entry.Action == "restore" {
				result.Restored++
			} else {
				result.SkippedExists++
			}
		}
	}

	// Restore task.completed events.
	taskFiles, err := filepath.Glob(filepath.Join(root, "works", "sprints", "completed", "*", "tasks", "T*.md"))
	if err != nil {
		return nil, fmt.Errorf("task glob failed: %w", err)
	}
	// Also include works/tasks/completed/T*.md (done Tasks not assigned to a Sprint).
	looseTaskFiles, _ := filepath.Glob(filepath.Join(root, "works", "tasks", "completed", "T*.md"))
	taskFiles = append(taskFiles, looseTaskFiles...)
	sort.Strings(taskFiles)
	result.ScannedTasks = len(taskFiles)

	for _, taskPath := range taskFiles {
		entry, appendErr := restoreTaskCompleted(gs, root, taskPath, dryRun)
		if appendErr != nil {
			return nil, appendErr
		}
		if entry != nil {
			result.Entries = append(result.Entries, *entry)
			if entry.Action == "restore" {
				result.Restored++
			} else {
				result.SkippedExists++
			}
		}
	}

	return result, nil
}

// restoreSprintCompleted evaluates the restoration target for a single
// sprint.
func restoreSprintCompleted(gs ports.GraphStore, root, sprintPath, sprintID string, dryRun bool) (*RestoreEntry, error) {
	exists, err := auditEventExists(gs, "sprint.completed", "sprint", sprintID)
	if err != nil {
		return nil, err
	}
	if exists {
		return &RestoreEntry{
			EventType: "sprint.completed", EntityType: "sprint", EntityID: sprintID,
			Source: relPath(root, sprintPath), Action: "skip_existing",
		}, nil
	}

	ts, err := firstCommitTimestamp(root, sprintPath)
	if err != nil || ts == "" {
		return nil, nil // skip on git-log failure (safe).
	}

	entry := RestoreEntry{
		EventType: "sprint.completed", EntityType: "sprint", EntityID: sprintID,
		Timestamp: ts, Source: relPath(root, sprintPath), Action: "restore",
	}
	if !dryRun {
		if err := gs.AppendAuditEvent(ports.AuditEvent{
			EventType:  "sprint.completed",
			EntityType: "sprint",
			EntityID:   sprintID,
			ActorID:    "restore",
			Timestamp:  ts,
			Details:    map[string]any{"source": "git-log", "path": relPath(root, sprintPath)},
		}); err != nil {
			return nil, fmt.Errorf("AppendAuditEvent sprint.completed %s: %w", sprintID, err)
		}
	}
	return &entry, nil
}

// taskIDRe matches the T\d+ pattern.
var taskIDRe = regexp.MustCompile(`^T\d+`)

// restoreTaskCompleted evaluates the restoration target for a single
// task.
func restoreTaskCompleted(gs ports.GraphStore, root, taskPath string, dryRun bool) (*RestoreEntry, error) {
	base := filepath.Base(taskPath)
	match := taskIDRe.FindString(base)
	if match == "" {
		return nil, nil
	}
	taskID := match

	// Check the frontmatter status.
	content, readErr := readFile(taskPath)
	if readErr != nil {
		return nil, nil
	}
	m := statusFrontmatterRe.FindStringSubmatch(content)
	if len(m) < 2 || m[1] != "done" {
		return nil, nil
	}

	exists, err := auditEventExists(gs, "task.completed", "task", taskID)
	if err != nil {
		return nil, err
	}
	if exists {
		return &RestoreEntry{
			EventType: "task.completed", EntityType: "task", EntityID: taskID,
			Source: relPath(root, taskPath), Action: "skip_existing",
		}, nil
	}

	ts, err := firstCommitTimestamp(root, taskPath)
	if err != nil || ts == "" {
		return nil, nil
	}

	entry := RestoreEntry{
		EventType: "task.completed", EntityType: "task", EntityID: taskID,
		Timestamp: ts, Source: relPath(root, taskPath), Action: "restore",
	}
	if !dryRun {
		if err := gs.AppendAuditEvent(ports.AuditEvent{
			EventType:  "task.completed",
			EntityType: "task",
			EntityID:   taskID,
			ActorID:    "restore",
			Timestamp:  ts,
			Details:    map[string]any{"source": "git-log", "path": relPath(root, taskPath)},
		}); err != nil {
			return nil, fmt.Errorf("AppendAuditEvent task.completed %s: %w", taskID, err)
		}
	}
	return &entry, nil
}

// auditEventExists deduplicates by natural key
// (event_type + entity_type + entity_id).
func auditEventExists(gs ports.GraphStore, eventType, entityType, entityID string) (bool, error) {
	events, err := gs.QueryAuditEvents(ports.AuditQueryFilter{
		EventType:  eventType,
		EntityType: entityType,
		EntityID:   entityID,
		Limit:      1,
	})
	if err != nil {
		return false, fmt.Errorf("QueryAuditEvents dedupe: %w", err)
	}
	return len(events) > 0, nil
}

// firstCommitTimestamp returns the "first commit timestamp" (ISO format)
// from `git log` for a file. follow=true tracks renames to include
// history before the file was moved.
func firstCommitTimestamp(root, path string) (string, error) {
	rel, err := filepath.Rel(root, path)
	if err != nil {
		rel = path
	}
	cmd := exec.Command("git", "log", "--follow", "--format=%cI", "--reverse", "--", rel)
	cmd.Dir = root
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("git log %s: %w", rel, err)
	}
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	if len(lines) == 0 || lines[0] == "" {
		return "", nil
	}
	return strings.TrimSpace(lines[0]), nil
}

// relPath returns the path relative to root (or the original on failure).
func relPath(root, p string) string {
	rel, err := filepath.Rel(root, p)
	if err != nil {
		return p
	}
	return rel
}

// readFile reads a small text file fully and returns its content as a
// string.
func readFile(p string) (string, error) {
	out, err := os.ReadFile(p)
	if err != nil {
		return "", err
	}
	return string(out), nil
}
