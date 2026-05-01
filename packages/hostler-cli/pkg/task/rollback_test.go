package task

import (
	"os"
	"path/filepath"
	"testing"
)

// TestVerifyRollbackSection_Present verifies a sufficiently filled
// ## Rollback section passes.
func TestVerifyRollbackSection_Present(t *testing.T) {
	tmpDir := t.TempDir()
	taskFile := filepath.Join(tmpDir, "T999.md")
	content := `---
id: T999
title: "hotfix test"
type: hotfix
---

# T999 hotfix test

## Purpose

emergency fix.

## Rollback

If a problem occurs, run git revert abc1234 to restore. The rollback has no side effects.

## References
`
	if err := os.WriteFile(taskFile, []byte(content), 0o644); err != nil {
		t.Fatalf("test file create failed: %v", err)
	}

	ok, reason := verifyRollbackSection(taskFile)
	if !ok {
		t.Errorf("a sufficient rollback section was rejected: %s", reason)
	}
}

// TestVerifyRollbackSection_SectionMissing verifies that the absence of a
// ## Rollback section is rejected.
func TestVerifyRollbackSection_SectionMissing(t *testing.T) {
	tmpDir := t.TempDir()
	taskFile := filepath.Join(tmpDir, "T999.md")
	content := `---
id: T999
title: "section missing"
type: hotfix
---

# T999 section missing

## Purpose

emergency fix.

## References
`
	if err := os.WriteFile(taskFile, []byte(content), 0o644); err != nil {
		t.Fatalf("test file create failed: %v", err)
	}

	ok, reason := verifyRollbackSection(taskFile)
	if ok {
		t.Errorf("a missing rollback section was accepted")
	}
	if reason == "" {
		t.Errorf("rejection reason is empty")
	}
}

// TestVerifyRollbackSection_EmptySection verifies a ## Rollback section
// whose body is too short is rejected.
func TestVerifyRollbackSection_EmptySection(t *testing.T) {
	tmpDir := t.TempDir()
	taskFile := filepath.Join(tmpDir, "T999.md")
	content := `---
id: T999
title: "empty section"
type: hotfix
---

# T999 empty section

## Rollback

short

## References
`
	if err := os.WriteFile(taskFile, []byte(content), 0o644); err != nil {
		t.Fatalf("test file create failed: %v", err)
	}

	ok, reason := verifyRollbackSection(taskFile)
	if ok {
		t.Errorf("an empty rollback section was accepted")
	}
	if reason == "" {
		t.Errorf("rejection reason is empty")
	}
}

// TestVerifyRollbackSection_RollbackPlanHeader verifies that the
// ## Rollback Plan header is recognised.
func TestVerifyRollbackSection_RollbackPlanHeader(t *testing.T) {
	tmpDir := t.TempDir()
	taskFile := filepath.Join(tmpDir, "T999.md")
	content := `---
id: T999
type: hotfix
---

# T999 plan

## Rollback Plan

Revert commit abc1234 to restore previous state. No side effects expected.

## References
`
	if err := os.WriteFile(taskFile, []byte(content), 0o644); err != nil {
		t.Fatalf("test file create failed: %v", err)
	}

	ok, reason := verifyRollbackSection(taskFile)
	if !ok {
		t.Errorf("Rollback Plan header was rejected: %s", reason)
	}
}

// TestVerifyRollbackSection_FileMissing verifies error handling for a
// non-existent file.
func TestVerifyRollbackSection_FileMissing(t *testing.T) {
	ok, reason := verifyRollbackSection("/nonexistent/path/T999.md")
	if ok {
		t.Errorf("a non-existent file was accepted")
	}
	if reason == "" {
		t.Errorf("error reason is empty")
	}
}
