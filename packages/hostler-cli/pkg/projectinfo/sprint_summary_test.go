package projectinfo

import (
	"os"
	"path/filepath"
	"testing"
)

// TestT540_LoadSprintSummary_basic verifies that frontmatter goal +
// `## Goal` body parses correctly.
func TestT540_LoadSprintSummary_basic(t *testing.T) {
	tmp := t.TempDir()
	content := `---
id: sprint-test
title: "test Sprint"
goal: "simple goal"
---

# sprint-test

## Goal

first line
second line
third line

## Task list

| ID | Title |
|----|-------|
| T001 | hello |
| T002 | world |
`
	if err := os.WriteFile(filepath.Join(tmp, "SPRINT.md"), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	s := LoadSprintSummary(tmp)
	if s.Goal != "simple goal" {
		t.Errorf("goal mismatch: got %q", s.Goal)
	}
	// Lines matching the frontmatter goal are not in Summary if absent
	// from the body. All 3 lines preserved here.
	if len(s.Summary) != 3 {
		t.Errorf("summary len: got %d want 3 (%v)", len(s.Summary), s.Summary)
	}
	if s.TaskCount != 2 {
		t.Errorf("task count: got %d want 2", s.TaskCount)
	}
}

// TestT540_LoadSprintSummary_goal_dedup verifies that when the goal and
// the first body line are identical, the duplicate is removed from Summary.
func TestT540_LoadSprintSummary_goal_dedup(t *testing.T) {
	tmp := t.TempDir()
	content := `---
id: sprint-test
goal: "duplicate goal line"
---

## Goal

duplicate goal line
real summary line
`
	if err := os.WriteFile(filepath.Join(tmp, "SPRINT.md"), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	s := LoadSprintSummary(tmp)
	if s.Goal != "duplicate goal line" {
		t.Errorf("goal: got %q", s.Goal)
	}
	for _, line := range s.Summary {
		if line == s.Goal {
			t.Errorf("goal duplicated in summary: %q", line)
		}
	}
}

// TestT540_LoadSprintSummary_fallback_task_count verifies that when the
// Goal body is empty but Task count exists, Summary falls back to a
// "N Tasks scheduled" line.
func TestT540_LoadSprintSummary_fallback_task_count(t *testing.T) {
	tmp := t.TempDir()
	content := `---
id: sprint-test
goal: "only goal"
---

## Goal

## Task list

| T001 | a |
| T002 | b |
| T003 | c |
`
	if err := os.WriteFile(filepath.Join(tmp, "SPRINT.md"), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	s := LoadSprintSummary(tmp)
	if len(s.Summary) != 1 || s.Summary[0] != "3 Tasks scheduled" {
		t.Errorf("fallback summary: got %v", s.Summary)
	}
}

// TestT540_LoadSprintSummary_missing_file returns empty values when the file is missing.
func TestT540_LoadSprintSummary_missing_file(t *testing.T) {
	s := LoadSprintSummary("/nonexistent/path")
	if s.Goal != "" || len(s.Summary) != 0 || s.TaskCount != 0 {
		t.Errorf("expected empty, got %+v", s)
	}
}
