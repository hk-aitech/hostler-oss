package cmd

import (
	"os"
	"path/filepath"
	"testing"
)

// TestT486_RunInitSkeleton verifies that every skeleton entry is created on
// the first run in an empty directory and that a second run is idempotent.
func TestT486_RunInitSkeleton(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HSTL_PROJECT_ROOT", tmp)
	t.Setenv("HSTL_DB_PATH", filepath.Join(tmp, "hstl.db"))
	t.Setenv("HSTL_PROJECT", "test-project")

	// First run.
	result, err := runInitSkeleton(false)
	if err != nil {
		t.Fatalf("first runInitSkeleton failed: %v", err)
	}
	if len(result.CreatedDirs) == 0 {
		t.Error("first run CreatedDirs is empty - no directories created")
	}
	if len(result.CreatedFiles) < 2 {
		t.Errorf("first run CreatedFiles: got %d, want >= 2 (BACKLOG.md + CURRENT-FOCUS.md)", len(result.CreatedFiles))
	}

	// Verify each directory / file actually exists.
	for _, rel := range []string{
		"works/tasks",
		"works/sprints/backlog",
		"works/sprints/active",
		"works/sprints/completed",
		"works/tasks/BACKLOG.md",
		"works/CURRENT-FOCUS.md",
	} {
		if _, err := os.Stat(filepath.Join(tmp, rel)); err != nil {
			t.Errorf("%s does not exist: %v", rel, err)
		}
	}

	// Second run - must be idempotent.
	result2, err := runInitSkeleton(false)
	if err != nil {
		t.Fatalf("second runInitSkeleton failed: %v", err)
	}
	if len(result2.CreatedDirs) != 0 {
		t.Errorf("second run CreatedDirs: got %d, want 0 (all skipped)", len(result2.CreatedDirs))
	}
	if len(result2.CreatedFiles) != 0 {
		t.Errorf("second run CreatedFiles: got %d, want 0 (all skipped)", len(result2.CreatedFiles))
	}
	if len(result2.SkippedExisting) < 2 {
		t.Errorf("second run SkippedExisting: got %d, want >= 2", len(result2.SkippedExisting))
	}
}

// TestT491_RunInitSkeleton_DryRun verifies that dry-run mode populates only
// WouldCreateDirs / WouldCreateFiles without modifying the filesystem.
func TestT491_RunInitSkeleton_DryRun(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HSTL_PROJECT_ROOT", tmp)
	t.Setenv("HSTL_DB_PATH", filepath.Join(tmp, "hstl.db"))
	t.Setenv("HSTL_PROJECT", "dryrun-test")

	result, err := runInitSkeleton(true)
	if err != nil {
		t.Fatalf("dry-run failed: %v", err)
	}
	if !result.DryRun {
		t.Error("DryRun flag is false")
	}
	if len(result.CreatedDirs) != 0 || len(result.CreatedFiles) != 0 {
		t.Errorf("dry-run filled CreatedDirs/CreatedFiles: dirs=%v files=%v",
			result.CreatedDirs, result.CreatedFiles)
	}
	if len(result.WouldCreateDirs) == 0 {
		t.Error("WouldCreateDirs is empty - no planned creations")
	}

	// Verify filesystem - nothing must actually be created.
	for _, rel := range []string{
		"works/tasks",
		"works/sprints/backlog",
		"works/tasks/BACKLOG.md",
		"works/CURRENT-FOCUS.md",
	} {
		if _, err := os.Stat(filepath.Join(tmp, rel)); err == nil {
			t.Errorf("dry-run actually created %s (filesystem side effect)", rel)
		}
	}
}

// TestT486_RunInitSkeleton_PreserveExistingFile verifies that an existing
// BACKLOG.md is not overwritten.
func TestT486_RunInitSkeleton_PreserveExistingFile(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HSTL_PROJECT_ROOT", tmp)
	t.Setenv("HSTL_DB_PATH", filepath.Join(tmp, "hstl.db"))
	t.Setenv("HSTL_PROJECT", "test-project-2")

	_ = os.MkdirAll(filepath.Join(tmp, "works", "tasks"), 0o755)
	existing := "# user edit BACKLOG\npreserve content\n"
	backlogPath := filepath.Join(tmp, "works", "tasks", "BACKLOG.md")
	if err := os.WriteFile(backlogPath, []byte(existing), 0o644); err != nil {
		t.Fatal(err)
	}

	if _, err := runInitSkeleton(false); err != nil {
		t.Fatalf("runInitSkeleton failed: %v", err)
	}

	data, _ := os.ReadFile(backlogPath)
	if string(data) != existing {
		t.Errorf("BACKLOG.md overwritten:\n%s", data)
	}
}
