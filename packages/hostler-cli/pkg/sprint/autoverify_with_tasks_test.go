package sprint

import (
	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/internal/domain"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// T456 (Sprint-71): InjectAutoVerifyBodyWithTasks injects the
// implementation Task list as checkboxes into the body's requirements
// section.

const testAutoVerifyFixture = `---
id: T999
title: test Dev environment deployment verification
---

# T999 test Dev environment deployment verification

## Type Tags

- [x] Measurement / verification / deployment

## Purpose

placeholder

## Requirements

- [ ] placeholder 1
- [ ] placeholder 2

## Done Criteria

- [ ] placeholder

## Scope Limits

### In this Sprint

- verification
`

func TestT456_InjectAutoVerifyBodyWithTasks_ImplementationTaskInject(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "T999.md")
	if err := os.WriteFile(path, []byte(testAutoVerifyFixture), 0o644); err != nil {
		t.Fatal(err)
	}

	tasks := []domain.TaskSummary{
		{TaskID: "T001", Title: "implement feature A"},
		{TaskID: "T002", Title: "fix bug B"},
	}
	if err := InjectAutoVerifyBodyWithTasks(path, "sprint-99", "", tasks); err != nil {
		t.Fatalf("inject failed: %v", err)
	}

	data, _ := os.ReadFile(path)
	body := string(data)

	wants := []string{
		"sprint-99",
		"T001 — implement feature A",
		"T002 — fix bug B",
		"Regression test",
		"## Scope Limits",
	}
	for _, w := range wants {
		if !strings.Contains(body, w) {
			t.Errorf("missing %q\n\n%s", w, body)
		}
	}
}

func TestT456_InjectAutoVerifyBodyWithTasks_EmptyFallback(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "T999.md")
	if err := os.WriteFile(path, []byte(testAutoVerifyFixture), 0o644); err != nil {
		t.Fatal(err)
	}

	// empty list → generic template fallback
	if err := InjectAutoVerifyBodyWithTasks(path, "sprint-99", "", nil); err != nil {
		t.Fatalf("inject failed: %v", err)
	}
	data, _ := os.ReadFile(path)
	body := string(data)
	// stable phrase from the generic template
	if !strings.Contains(body, "Per-Task manual smoke test") {
		t.Errorf("generic-template fallback did not run\n\n%s", body)
	}
}

// TestT313_AutoVerify_GoalIncluded verifies that passing goal to
// InjectAutoVerifyBodyWithTasks adds a "Sprint goal: ..." line to the
// ## Purpose section (T313, Sprint-79). When goal is empty the line must
// be omitted.
func TestT313_AutoVerify_GoalIncluded(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "T999.md")

	t.Run("goal_present", func(t *testing.T) {
		if err := os.WriteFile(path, []byte(testAutoVerifyFixture), 0o644); err != nil {
			t.Fatal(err)
		}
		tasks := []domain.TaskSummary{
			{TaskID: "T001", Title: "implement feature A"},
			{TaskID: "T002", Title: "set up infra B"},
		}
		goal := "triad refactor + template hardening"
		if err := InjectAutoVerifyBodyWithTasks(path, "sprint-79", goal, tasks); err != nil {
			t.Fatalf("inject failed: %v", err)
		}
		data, _ := os.ReadFile(path)
		body := string(data)

		wants := []string{
			"Sprint goal: " + goal,
			"sprint-79",
			"T001 — implement feature A",
			"T002 — set up infra B",
		}
		for _, w := range wants {
			if !strings.Contains(body, w) {
				t.Errorf("missing %q\n\n%s", w, body)
			}
		}
	})

	t.Run("goal_absent_line_omitted", func(t *testing.T) {
		if err := os.WriteFile(path, []byte(testAutoVerifyFixture), 0o644); err != nil {
			t.Fatal(err)
		}
		tasks := []domain.TaskSummary{{TaskID: "T003", Title: "chore C"}}
		if err := InjectAutoVerifyBodyWithTasks(path, "sprint-80", "", tasks); err != nil {
			t.Fatalf("inject failed: %v", err)
		}
		data, _ := os.ReadFile(path)
		body := string(data)

		if strings.Contains(body, "Sprint goal:") {
			t.Errorf("when goal is absent, no 'Sprint goal:' line must exist\n\n%s", body)
		}
	})
}

// TestT313_AutoVerify_Generic_GoalIncluded verifies the generic template
// (InjectAutoVerifyBody) also includes goal.
func TestT313_AutoVerify_Generic_GoalIncluded(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "T998.md")
	if err := os.WriteFile(path, []byte(testAutoVerifyFixture), 0o644); err != nil {
		t.Fatal(err)
	}

	goal := "API stabilisation and performance improvements"
	if err := InjectAutoVerifyBody(path, "sprint-81", goal); err != nil {
		t.Fatalf("inject failed: %v", err)
	}
	data, _ := os.ReadFile(path)
	body := string(data)

	if !strings.Contains(body, "Sprint goal: "+goal) {
		t.Errorf("generic template missing the Sprint goal line\n\n%s", body)
	}
}
