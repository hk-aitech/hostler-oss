package consistency_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/internal/adapters/sqlite"
	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/pkg/consistency"
	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/pkg/db"
	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/pkg/store"
)

// setupTestEnv initialises an isolated DB and project root for tests.
func setupTestEnv(t *testing.T) string {
	t.Helper()
	tmpDir := t.TempDir()
	t.Setenv("HSTL_DB_PATH", filepath.Join(tmpDir, "hstl.db"))
	t.Setenv("HSTL_PROJECT_ROOT", tmpDir)

	for _, dir := range []string{
		filepath.Join(tmpDir, "works", "tasks"),
		filepath.Join(tmpDir, "works", "sprints", "backlog"),
		filepath.Join(tmpDir, "works", "sprints", "active"),
	} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatalf("create directory failed: %v", err)
		}
	}

	if err := db.InitDB(); err != nil {
		t.Fatalf("DB init failed: %v", err)
	}
	sqliteStore := sqlite.New(db.GetDB())
	store.Init(sqliteStore, sqliteStore)
	t.Cleanup(func() {
		store.Reset()
		db.Close()
	})
	t.Cleanup(consistency.InvalidateCache)
	return tmpDir
}

// insertPlaceholderTask creates a Task with a placeholder body in both DB and file.
func insertPlaceholderTask(t *testing.T, tmpDir, taskID string) {
	t.Helper()
	database := db.GetDB()

	// Placeholder body markers must match the strings in pkg/task._placeholderMarkers.
	content := `---
id: ` + taskID + `
title: "test"
type: feature
sprint: backlog
status: todo
priority: p2
estimate: M
depends_on: []
created: 2026-04-10
---

# ` + taskID + ` test

## Purpose

{The problem this Task aims to solve or the goal it pursues}

## Requirements

- [ ] {Requirement 1}
`
	relPath := filepath.Join("works", "tasks", taskID+"-test.md")
	absPath := filepath.Join(tmpDir, relPath)
	if err := os.WriteFile(absPath, []byte(content), 0o644); err != nil {
		t.Fatalf("Task file create failed: %v", err)
	}

	_, err := database.Exec(
		`INSERT INTO tasks
			(task_id, title, type, sprint, status, priority, estimate, file_path,
			 depends_on, created_at, updated_at)
		 VALUES (?, 'test', 'feature', '', 'todo', 'p2', 'M', ?, '[]', date('now'), date('now'))`,
		taskID, relPath,
	)
	if err != nil {
		t.Fatalf("Task DB insert failed: %v", err)
	}
}

// TestQuickCheck_Empty verifies that an empty result is returned when no placeholders exist.
func TestQuickCheck_Empty(t *testing.T) {
	setupTestEnv(t)

	warnings := consistency.QuickCheck()
	if len(warnings) != 0 {
		t.Errorf("expected empty result without placeholders, got: %v", warnings)
	}
}

// TestQuickCheck_WithPlaceholders verifies that warnings are emitted when placeholders exist.
func TestQuickCheck_WithPlaceholders(t *testing.T) {
	tmpDir := setupTestEnv(t)
	insertPlaceholderTask(t, tmpDir, "T001")
	insertPlaceholderTask(t, tmpDir, "T002")

	warnings := consistency.QuickCheck()
	if len(warnings) == 0 {
		t.Fatal("no placeholder warnings emitted")
	}

	joined := strings.Join(warnings, "\n")
	if !strings.Contains(joined, "placeholder") {
		t.Errorf("warning missing the 'placeholder' substring: %v", warnings)
	}
	if !strings.Contains(joined, "T001") || !strings.Contains(joined, "T002") {
		t.Errorf("warning missing Task IDs: %v", warnings)
	}
}

// TestQuickCheck_CacheHit verifies that recalls within TTL hit the cache.
// After the first call, mutating the DB must not change the result.
func TestQuickCheck_CacheHit(t *testing.T) {
	tmpDir := setupTestEnv(t)
	insertPlaceholderTask(t, tmpDir, "T001")

	// First call — 1 placeholder.
	first := consistency.QuickCheck()
	if len(first) == 0 {
		t.Fatal("first call returned no warnings")
	}

	// Add T002 — but the second call should return the cached result.
	insertPlaceholderTask(t, tmpDir, "T002")

	second := consistency.QuickCheck()
	if len(second) != len(first) {
		t.Errorf("expected cache hit (len %d), got len %d", len(first), len(second))
	}
	// The cached result must not contain T002.
	if strings.Contains(strings.Join(second, "\n"), "T002") {
		t.Errorf("fresh result returned instead of cache (T002 included)")
	}
}

// TestInvalidateCache verifies that invalidating the cache forces a re-check.
func TestInvalidateCache(t *testing.T) {
	tmpDir := setupTestEnv(t)
	insertPlaceholderTask(t, tmpDir, "T001")

	// Populate the cache.
	_ = consistency.QuickCheck()

	// Add T002 and invalidate the cache.
	insertPlaceholderTask(t, tmpDir, "T002")
	consistency.InvalidateCache()

	// After invalidation, the call should now include T002.
	warnings := consistency.QuickCheck()
	joined := strings.Join(warnings, "\n")
	if !strings.Contains(joined, "T002") {
		t.Errorf("T002 missing after cache invalidation: %v", warnings)
	}
}
