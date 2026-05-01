// Package task — T757 (Sprint-88) Delete --orphan-file tests.
//
// Verification scenarios (3 requirements):
//
//	(1) normal Task with no drift + OrphanFile=true → delete via the
//	    existing path (status="deleted", not "orphan_deleted")
//	(2) drift Task (file only) + OrphanFile=false → NotFoundError
//	    (regression guard)
//	(3) drift Task + OrphanFile=true + reason → file removed + audit
//	    recorded
package task_test

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/pkg/apperr"
	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/pkg/audit"
	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/pkg/task"
)

// createDriftFile artificially creates a "file-only" drift state (DB has
// no record). The placeholder file is created as
// works/tasks/T{id}-orphan-drift.md.
func createDriftFile(t *testing.T, tmpDir, taskID string) string {
	t.Helper()
	filename := filepath.Join(tmpDir, "works", "tasks", taskID+"-orphan-drift.md")
	if err := os.WriteFile(filename, []byte("# "+taskID+"\n\ndrift test\n"), 0o644); err != nil {
		t.Fatalf("drift file create failed: %v", err)
	}
	return filename
}

// TestDeleteOrphanFile_NormalTask_ExistingPath verifies that when both DB
// and file are present (normal Task), --orphan-file is ignored and Delete
// goes through the existing path (status="deleted", not "orphan_deleted").
func TestDeleteOrphanFile_NormalTask_ExistingPath(t *testing.T) {
	setupDB(t)

	createRaw, err := task.Create("normal delete test", "chore", "", "p3", "XS", "", nil)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	taskID := asMap(createRaw)["task_id"].(string)

	raw, err := task.DeleteWithOptions(taskID, "verifying the normal path", task.DeleteOptions{OrphanFile: true})
	if err != nil {
		t.Fatalf("Delete failed: %v", err)
	}
	result := asMap(raw)
	if result["status"] != "deleted" {
		t.Errorf("status=%v, want=deleted (orphan path was wrongly taken)", result["status"])
	}

	// audit must include only task.deleted (not orphan_deleted).
	orphanEvents, _ := audit.QueryEvents(audit.QueryFilter{
		EntityType: "task",
		EntityID:   taskID,
		EventType:  "task.orphan_deleted",
	})
	if len(orphanEvents) != 0 {
		t.Errorf("normal path but task.orphan_deleted event recorded: %d", len(orphanEvents))
	}
}

// TestDeleteOrphanFile_drift_DefaultReject verifies that when the flag is
// not set on a file-only drift, NotFoundError is returned as before
// (regression guard).
func TestDeleteOrphanFile_drift_DefaultReject(t *testing.T) {
	tmpDir := setupDB(t)

	taskID := "T8888" // not in DB
	createDriftFile(t, tmpDir, taskID)

	_, err := task.DeleteWithOptions(taskID, "verify reject without orphan flag", task.DeleteOptions{OrphanFile: false})
	if err == nil {
		t.Fatal("OrphanFile=false + drift but err is nil")
	}
	var nfe *apperr.NotFoundError
	if !errors.As(err, &nfe) {
		t.Errorf("expected NotFoundError, got=%T: %v", err, err)
	}
}

// TestDeleteOrphanFile_drift_FileRemovalAudit verifies that when a
// file-only drift is given --orphan-file the file is actually removed and
// task.orphan_deleted audit is recorded.
func TestDeleteOrphanFile_drift_FileRemovalAudit(t *testing.T) {
	tmpDir := setupDB(t)

	taskID := "T9191"
	driftPath := createDriftFile(t, tmpDir, taskID)

	raw, err := task.DeleteWithOptions(taskID, "drift cleanup measurement check", task.DeleteOptions{OrphanFile: true})
	if err != nil {
		t.Fatalf("DeleteWithOptions: %v", err)
	}
	result := asMap(raw)
	if result["status"] != "orphan_deleted" {
		t.Errorf("status=%v, want=orphan_deleted", result["status"])
	}

	// confirm file actually removed.
	if _, statErr := os.Stat(driftPath); !os.IsNotExist(statErr) {
		t.Errorf("file was not removed: %s (err=%v)", driftPath, statErr)
	}

	// confirm audit.orphan_deleted event.
	events, _ := audit.QueryEvents(audit.QueryFilter{
		EntityType: "task",
		EntityID:   taskID,
		EventType:  "task.orphan_deleted",
	})
	if len(events) == 0 {
		t.Fatal("task.orphan_deleted event not recorded")
	}
	ev := events[0]
	if ev.Details["reason"] != "drift cleanup measurement check" {
		t.Errorf("audit.reason mismatch: %v", ev.Details["reason"])
	}
	if fp, _ := ev.Details["file_path"].(string); fp == "" {
		t.Error("audit.file_path is empty")
	}
}

// TestDeleteOrphanFile_FileMissing verifies that when an ID has neither DB
// nor file, --orphan-file returns NotFoundError (drift-flag misuse hint).
func TestDeleteOrphanFile_FileMissing(t *testing.T) {
	setupDB(t)

	_, err := task.DeleteWithOptions("T7777", "attempt to clean up non-existent drift", task.DeleteOptions{OrphanFile: true})
	if err == nil {
		t.Fatal("err nil — succeeded on non-existent ID")
	}
	var nfe *apperr.NotFoundError
	if !errors.As(err, &nfe) {
		t.Errorf("expected NotFoundError, got=%T: %v", err, err)
	}
}
