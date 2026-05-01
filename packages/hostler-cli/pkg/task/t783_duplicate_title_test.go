// Package task — T783 (Sprint-91) unit tests for the title-duplicate
// detection guard.
//
// Prevents recurrence of the duplicate-create pattern observed downstream.
// Four scenarios:
//  1. same title called again within the window → BLOCK
//  2. window elapsed → allow
//  3. bypass env set → allow
//  4. title normalization (case / whitespace) → still detected as
//     duplicate
package task_test

import (
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/internal/adapters/sqlite"
	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/internal/ports"
	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/pkg/db"
	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/pkg/store"
	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/pkg/task"
)

// setupT783DB initializes an isolated DB and registers the store.
func setupT783DB(t *testing.T) string {
	t.Helper()
	tmpDir := t.TempDir()
	t.Setenv("HSTL_DB_PATH", filepath.Join(tmpDir, "hstl.db"))
	t.Setenv("HSTL_PROJECT_ROOT", tmpDir)
	if err := db.InitDB(); err != nil {
		t.Fatalf("DB init failed: %v", err)
	}
	s := sqlite.New(db.GetDB())
	store.Init(s, s)
	t.Cleanup(func() {
		store.Reset()
		db.Close()
	})
	return tmpDir
}

// seedTask inserts one Task with the given title (CreatedAt = now).
// "Out-of-window" scenarios are simulated by setting a very short window
// env var and sleeping.
func seedTask(t *testing.T, taskID, title, _ string) {
	t.Helper()
	gs := store.Get()
	if gs == nil {
		t.Fatal("store not initialized")
	}
	now := time.Now().UTC().Format("2006-01-02 15:04:05")
	if err := gs.CreateTask(&ports.TaskRecord{
		TaskID:    taskID,
		Title:     title,
		Type:      "feature",
		Status:    "todo",
		Priority:  "p2",
		Estimate:  "S",
		FilePath:  "works/tasks/" + taskID + ".md",
		CreatedAt: now,
	}); err != nil {
		t.Fatalf("CreateTask: %v", err)
	}
}

// TestT783_Duplicate_Block — same title repeated within the default
// 60-second window must BLOCK.
func TestT783_Duplicate_Block(t *testing.T) {
	setupT783DB(t)
	seedTask(t, "T999A", "Duplicate Title Test", "")

	err := task.CheckDuplicateTitleOrReject("Duplicate Title Test")
	if err == nil {
		t.Fatal("did not detect duplicate — expected RejectedError")
	}
	if !strings.Contains(err.Error(), "T999A") {
		t.Errorf("error message should include the existing Task ID: %v", err)
	}
}

// TestT783_Normalize_CaseSpace — case and whitespace variants are also
// classified as duplicates.
func TestT783_Normalize_CaseSpace(t *testing.T) {
	setupT783DB(t)
	seedTask(t, "T999B", "Normalize Title", "")

	// case difference
	if err := task.CheckDuplicateTitleOrReject("normalize title"); err == nil {
		t.Error("did not detect case-only duplicate")
	}

	// multiple spaces
	if err := task.CheckDuplicateTitleOrReject("Normalize    Title"); err == nil {
		t.Error("did not detect multi-space duplicate")
	}

	// leading/trailing spaces
	if err := task.CheckDuplicateTitleOrReject("  Normalize Title  "); err == nil {
		t.Error("did not detect surrounding-whitespace duplicate")
	}
}

// TestT783_BypassEnv — HSTL_TASK_CREATE_ALLOW_DUPLICATE_TITLE=1 allows.
func TestT783_BypassEnv(t *testing.T) {
	setupT783DB(t)
	seedTask(t, "T999C", "Bypass Title Test", "")
	t.Setenv(task.TaskCreateAllowDuplicateEnv, "1")

	if err := task.CheckDuplicateTitleOrReject("Bypass Title Test"); err != nil {
		t.Errorf("rejected even though bypass env set: %v", err)
	}
}

// TestT783_WindowZero — window=0 allows all duplicates.
func TestT783_WindowZero(t *testing.T) {
	setupT783DB(t)
	seedTask(t, "T999D", "Window Zero Title", "")
	t.Setenv(task.TaskCreateDuplicateWindowEnv, "0")

	if err := task.CheckDuplicateTitleOrReject("Window Zero Title"); err != nil {
		t.Errorf("rejected with window=0: %v", err)
	}
}

// TestT783_NoDuplicate_DifferentTitle — a different title passes.
func TestT783_NoDuplicate_DifferentTitle(t *testing.T) {
	setupT783DB(t)
	seedTask(t, "T999E", "Original Title", "")

	if err := task.CheckDuplicateTitleOrReject("Completely Different"); err != nil {
		t.Errorf("rejected even though title is different: %v", err)
	}
}

// TestT783_WindowShortCut — set window to 1 second + sleep 2 seconds →
// allow. Verifies the out-of-window path with real sleep instead of clock
// manipulation.
func TestT783_WindowShortCut(t *testing.T) {
	if testing.Short() {
		t.Skip("short mode: sleep-based test skipped")
	}
	setupT783DB(t)
	seedTask(t, "T999F", "Short Window Test", "")
	t.Setenv(task.TaskCreateDuplicateWindowEnv, "1")
	time.Sleep(2100 * time.Millisecond)

	if err := task.CheckDuplicateTitleOrReject("Short Window Test"); err != nil {
		t.Errorf("rejected after window elapsed: %v", err)
	}
}
