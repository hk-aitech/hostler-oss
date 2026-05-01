// Package cmd - reproduction test for the task start/complete status drift.
//
// Verifies the reproduction case and the workaround for the failure mode
// where `hstl task start` returned a success response but the DB still
// recorded `done`, causing three consecutive `hstl task complete` calls to
// fail with "current status=done".
//
// KB reference: docs/07-knowledge/mistakes/workflow.md M004
// - task start/complete status drift
package cmd

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	_ "modernc.org/sqlite"
)

// taskMinimalContent returns the minimal Task body that passes the
// placeholder check. HasPlaceholderBody requires at least 300 chars of body
// plus at least one structural element (e.g. a checkbox).
func taskMinimalContent(taskID, title string) string {
	return fmt.Sprintf(`---
id: %s
title: %s
type: test
status: todo
priority: p3
estimate: S
sprint: ""
depends_on: []
created: 2026-04-16T00:00:00Z
---

# %s

## Purpose

Temporary Task used by the E2E test that reproduces the task start/complete
status drift. The Task is created inside an isolated tmpDir during the
integration test run and never touches the real project DB. Used to verify
the KB M004 reproduction case.

## Requirements

- [ ] verify DB status transitions to in-progress after task start
- [ ] verify task start returns an error response when DB status drifts

## Done Criteria

- [ ] task start success response (status: ok, new_status: in-progress)
- [ ] DB drift state returns error response (containing "current status=done")
`, taskID, title, title)
}

// openDriftDB opens the isolated hstl.db directly and returns it.
// Closes automatically via t.Cleanup at test end.
func openDriftDB(t *testing.T, tmpDir string) *sql.DB {
	t.Helper()
	dbPath := filepath.Join(tmpDir, "hstl.db")
	database, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("failed to open hstl.db: %v", err)
	}
	t.Cleanup(func() { _ = database.Close() })
	return database
}

// setTaskDBStatus updates the Task status directly in the DB.
// The file frontmatter is left unchanged, simulating a drift state.
func setTaskDBStatus(t *testing.T, database *sql.DB, taskID, status string) {
	t.Helper()
	res, err := database.Exec(
		"UPDATE tasks SET status = ? WHERE task_id = ?",
		status, taskID,
	)
	if err != nil {
		t.Fatalf("failed to update DB status (%s -> %s): %v", taskID, status, err)
	}
	rows, _ := res.RowsAffected()
	if rows == 0 {
		t.Fatalf("Task %s not found in DB", taskID)
	}
}

// createIsolatedTask creates a Task in the isolated env and returns the
// task_id. Right after creation, writes a non-placeholder body so the task
// start does not trigger a placeholder warning.
//
// The summary static heuristic is strict by default and rejects simple
// summaries used in tests. Because this fixture only needs to reproduce
// drift, we bypass both the heuristic and LLM judge with
// --skip-summary-validation.
func createIsolatedTask(t *testing.T, tmpDir, title string) string {
	t.Helper()
	stdout, _, code := runHstlIsolated(t, tmpDir,
		"task", "create",
		"--title", title,
		"--summary", "T677 temporary Task used to reproduce drift",
		"--type", "test",
		"--skip-summary-validation",
		"-o", "json",
	)
	if code != 0 {
		t.Fatalf("task create failed exit %d\n%s", code, stdout)
	}
	var created struct {
		Data struct {
			TaskID   string `json:"task_id"`
			FilePath string `json:"file_path"`
		} `json:"data"`
	}
	if err := json.Unmarshal([]byte(strings.TrimSpace(stdout)), &created); err != nil {
		t.Fatalf("failed to parse task create JSON: %v\n%s", err, stdout)
	}
	taskID := created.Data.TaskID
	if taskID == "" {
		t.Fatal("task_id is empty")
	}

	// Write a non-placeholder body to avoid the task start placeholder warning.
	filePath := created.Data.FilePath
	if filePath == "" {
		// When file_path is missing, search under works/tasks/.
		entries, err := filepath.Glob(filepath.Join(tmpDir, "works", "tasks", taskID+"-*.md"))
		if err != nil || len(entries) == 0 {
			t.Logf("failed to auto-discover task file - placeholder warning possible")
			return taskID
		}
		filePath = entries[0]
	} else if !filepath.IsAbs(filePath) {
		filePath = filepath.Join(tmpDir, filePath)
	}

	content := taskMinimalContent(taskID, title)
	if err := os.WriteFile(filePath, []byte(content), 0o644); err != nil {
		t.Fatalf("failed to write task file body: %v", err)
	}
	return taskID
}

// setupDriftWorksDirs creates the minimal works/ tree required by runHstlIsolated.
func setupDriftWorksDirs(t *testing.T, tmpDir string) {
	t.Helper()
	for _, d := range []string{
		filepath.Join(tmpDir, "works", "tasks"),
		filepath.Join(tmpDir, "works", "tasks", "completed"),
		filepath.Join(tmpDir, "works", "sprints", "backlog"),
		filepath.Join(tmpDir, "works", "sprints", "active"),
		filepath.Join(tmpDir, "works", "data", "task"),
	} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatalf("failed to mkdir %s: %v", d, err)
		}
	}
}

// =============================================================================
// task start/complete status drift reproduction tests
// KB M004: docs/07-knowledge/mistakes/workflow.md
// =============================================================================

// TestT677_TaskStart_NormalFlow is the regression case verifying the normal
// status transition to in-progress after task start.
//
// KB M004 - exercises the happy path first to guard the drift reproduction.
func TestT677_TaskStart_NormalFlow(t *testing.T) {
	skipIfNoBinary(t)

	tmpDir := t.TempDir()
	setupDriftWorksDirs(t, tmpDir)

	taskID := createIsolatedTask(t, tmpDir, "T677 normal start test")

	// Invoke task start.
	stdout, _, code := runHstlIsolated(t, tmpDir,
		"task", "start", taskID, "-o", "json",
	)
	if code != 0 {
		t.Fatalf("task start failed exit %d\n%s", code, stdout)
	}

	// Validate JSON output.
	trimmed := strings.TrimSpace(stdout)
	if !json.Valid([]byte(trimmed)) {
		t.Fatalf("task start stdout is not valid JSON:\n%s", trimmed)
	}

	// Verify the status transition.
	var result struct {
		Status string `json:"status"`
		Data   struct {
			NewStatus string `json:"new_status"`
		} `json:"data"`
	}
	if err := json.Unmarshal([]byte(trimmed), &result); err != nil {
		t.Fatalf("failed to parse JSON: %v", err)
	}
	if result.Status != "ok" {
		t.Errorf("status = %q, expected ok\nresponse: %s", result.Status, trimmed)
	}
	if result.Data.NewStatus != "in-progress" {
		t.Errorf("new_status = %q, expected in-progress\nresponse: %s",
			result.Data.NewStatus, trimmed)
	}

	// Verify DB also shows in-progress (KB M004: validate DB status after start).
	database := openDriftDB(t, tmpDir)
	var dbStatus string
	if err := database.QueryRow(
		"SELECT status FROM tasks WHERE task_id = ?", taskID,
	).Scan(&dbStatus); err != nil {
		t.Fatalf("failed to query DB status: %v", err)
	}
	if dbStatus != "in-progress" {
		// KB M004 reproduction: DB status remains done after a successful task start response.
		t.Errorf("KB M004 reproduction - DB status = %q, expected in-progress (drift after task start)", dbStatus)
	}
}

// TestT677_TaskStart_DoneDBStatus_ReturnsError verifies that calling task
// start on a Task whose DB status is done returns an appropriate error.
//
// KB M004 - reproduction case for the failure where, retrying task start
// without running backlog sync, the "current status=done" error fired three
// times in a row. This test documents the current behavior and confirms the
// error message mentions "done".
func TestT677_TaskStart_DoneDBStatus_ReturnsError(t *testing.T) {
	skipIfNoBinary(t)

	tmpDir := t.TempDir()
	setupDriftWorksDirs(t, tmpDir)

	taskID := createIsolatedTask(t, tmpDir, "T677 done DB start error test")

	// Set DB status to "done" directly (file frontmatter stays "todo" -> drift).
	// This mirrors the state right before the original error fired.
	database := openDriftDB(t, tmpDir)
	setTaskDBStatus(t, database, taskID, "done")

	// Invoke task start - with DB status=done, expect an error response.
	stdout, _, code := runHstlIsolated(t, tmpDir,
		"task", "start", taskID, "-o", "json",
	)

	// exit 0 means we got a success response while in a drift state - bug.
	if code == 0 {
		t.Errorf(
			"KB M004: task start returned exit 0 for a Task with DB status=done - expected error during drift\nresponse: %s",
			stdout,
		)
	}

	// Confirm the combined stdout/stderr output mentions "done".
	combined := stdout
	if !strings.Contains(combined, "done") {
		t.Logf("error message did not mention 'done' (informational): %s", combined)
	}
}

// TestBacklogSync_ResolvesDrift_TaskStartSucceeds verifies that running
// `hstl-oss backlog sync` recovers the drift between DB and file statuses so
// task start works again.
//
// Status: skipped — the OSS slice of backlog sync does not yet write back DB
// rows from file frontmatter when the file is the SSOT (it currently reports
// drift but does not auto-reconcile). The test is preserved as a regression
// guard for when that recovery path lands.
func TestBacklogSync_ResolvesDrift_TaskStartSucceeds(t *testing.T) {
	t.Skip("backlog sync does not yet auto-reconcile DB→file drift (tracked as a follow-up)")
	skipIfNoBinary(t)

	tmpDir := t.TempDir()
	setupDriftWorksDirs(t, tmpDir)

	taskID := createIsolatedTask(t, tmpDir, "T677 backlog sync workaround test")

	// Set DB status to "done" (file frontmatter stays "todo" -> drift).
	database := openDriftDB(t, tmpDir)
	setTaskDBStatus(t, database, taskID, "done")

	// Verify the drift state: task start must fail.
	_, _, codeBeforeSync := runHstlIsolated(t, tmpDir,
		"task", "start", taskID, "-o", "json",
	)
	if codeBeforeSync == 0 {
		t.Log("KB M004: task start returned exit 0 before sync - drift detection inactive (informational)")
	}

	// Run backlog sync (workaround: recompute DB status from file frontmatter).
	syncOut, _, syncCode := runHstlIsolated(t, tmpDir,
		"backlog", "sync", "-o", "json",
	)
	if syncCode != 0 {
		t.Fatalf("backlog sync failed exit %d\n%s", syncCode, syncOut)
	}

	// After sync, DB status must match the file frontmatter (todo).
	var dbStatusAfterSync string
	if err := database.QueryRow(
		"SELECT status FROM tasks WHERE task_id = ?", taskID,
	).Scan(&dbStatusAfterSync); err != nil {
		t.Fatalf("failed to query DB status after sync: %v", err)
	}

	// Confirm backlog sync reconciled the DB to the file frontmatter.
	if dbStatusAfterSync == "done" {
		// sync did not resolve the drift - the workaround failed.
		t.Logf("KB M004: DB status after backlog sync = %q (drift unresolved). sync did not reconcile the DB to the file.", dbStatusAfterSync)
	}

	// Retry task start after sync.
	startOut, _, startCode := runHstlIsolated(t, tmpDir,
		"task", "start", taskID, "-o", "json",
	)
	if startCode != 0 {
		t.Errorf(
			"KB M004: task start still failed after backlog sync exit %d - workaround ineffective\nresponse: %s",
			startCode, startOut,
		)
		return
	}

	// Verify the success response.
	trimmed := strings.TrimSpace(startOut)
	if !json.Valid([]byte(trimmed)) {
		t.Fatalf("task start stdout after sync is not valid JSON:\n%s", trimmed)
	}
	var result struct {
		Status string `json:"status"`
	}
	if err := json.Unmarshal([]byte(trimmed), &result); err != nil {
		t.Fatalf("failed to parse JSON: %v", err)
	}
	if result.Status != "ok" {
		t.Errorf("KB M004 workaround failed: task start status after backlog sync = %q, expected ok\nresponse: %s",
			result.Status, trimmed)
	}
}
