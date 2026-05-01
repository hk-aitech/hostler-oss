package sprint

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// T449 (Sprint-54) — InjectAutoVerifyBody unit test.
// AppendAutoVerifyTask depends heavily on real DB / file paths and is
// covered by integration tests; this file exercises only the pure
// file-manipulation function.

const t449FixtureTaskFile = `---
id: T999
title: "Sprint-54 Dev environment deployment verification"
type: infra
sprint: sprint-54
status: todo
priority: p1
estimate: S
---

# T999 Sprint-54 Dev environment deployment verification

## Type Tags

- [ ] New structure introduction
- [ ] Existing structure update
- [ ] Measurement / verification / deployment
- [ ] Mixed

## Purpose

{placeholder}

## Requirements

- [ ] {Requirement 1}

## Done Criteria

- [ ] {Criterion 1}

## Scope Limits

### In this Sprint

- existing scope

## References

- ref
`

func TestT449_InjectAutoVerifyBody_Success(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "T999.md")
	if err := os.WriteFile(path, []byte(t449FixtureTaskFile), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := InjectAutoVerifyBody(path, "sprint-54", ""); err != nil {
		t.Fatalf("InjectAutoVerifyBody: %v", err)
	}

	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	content := string(got)

	// confirm new body was injected
	mustContain := []string{
		"Integration-verify the implementation results of the sprint-54 Sprint",
		"Per-Task manual smoke test",
		"go test -count=1 ./...",
		"integration-test.sh",
		"## Scope Limits", // trailing anchor preserved
		"## References",   // section after preserved
		"- ref",
		"## Type Tags", // preceding section preserved
	}
	for _, s := range mustContain {
		if !strings.Contains(content, s) {
			t.Errorf("missing %q after injection:\n---\n%s\n---", s, content)
		}
	}

	// confirm placeholders were removed
	mustNotContain := []string{
		"{placeholder}",
		"{Requirement 1}",
		"{Criterion 1}",
	}
	for _, s := range mustNotContain {
		if strings.Contains(content, s) {
			t.Errorf("placeholder %q still present after injection", s)
		}
	}
}

func TestT449_InjectAutoVerifyBody_EmptyPathError(t *testing.T) {
	if err := InjectAutoVerifyBody("", "sprint-54", ""); err == nil {
		t.Fatal("expected error for empty file path")
	}
}

func TestT449_InjectAutoVerifyBody_FallbackWithoutScopeLimits(t *testing.T) {
	// variant body without ## Scope Limits, with only ## References
	dir := t.TempDir()
	path := filepath.Join(dir, "T998.md")
	variant := strings.Replace(t449FixtureTaskFile, "## Scope Limits\n\n### In this Sprint\n\n- existing scope\n\n", "", 1)
	if err := os.WriteFile(path, []byte(variant), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := InjectAutoVerifyBody(path, "sprint-54", ""); err != nil {
		t.Fatalf("fallback (## References) failed: %v", err)
	}
	got, _ := os.ReadFile(path)
	if !strings.Contains(string(got), "## References") {
		t.Error("fallback trailing anchor not preserved")
	}
}

func TestT449_InjectAutoVerifyBody_NoTypeTagsSectionError(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "T997.md")
	invalid := "# Task\n\n## Scope Limits\n- foo\n"
	if err := os.WriteFile(path, []byte(invalid), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := InjectAutoVerifyBody(path, "sprint-54", ""); err == nil {
		t.Fatal("expected error for file without ## Type Tags")
	}
}
