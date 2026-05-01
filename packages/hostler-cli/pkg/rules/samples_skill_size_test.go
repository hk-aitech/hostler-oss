package rules

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// precommit.skill.size Rule regression tests.
//
// SKILL.md body ≤ 500 (default) guard.

// setupSkillSizeTree creates a skills/<name>/SKILL.md single-file tree.
func setupSkillSizeTree(t *testing.T, rel string, lines int) string {
	t.Helper()
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	full := filepath.Join(root, rel)
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		t.Fatal(err)
	}
	body := strings.Repeat("a\n", lines)
	if err := os.WriteFile(full, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	return root
}

// runSkillSize — Rule execution helper.
func runSkillSize(t *testing.T, root string) *RuleResult {
	t.Helper()
	r, ok := Get(skillSizeRuleID)
	if !ok {
		t.Fatalf("Rule %s not registered", skillSizeRuleID)
	}
	return r.Check(&RuleContext{ProjectRoot: root})
}

// Boundary PASS — exactly 500 lines.
func TestT760_SkillSize_Boundary_PASS(t *testing.T) {
	root := setupSkillSizeTree(t, "skills/foo/SKILL.md", 500)
	stubStaged(t, []string{"skills/foo/SKILL.md"})
	res := runSkillSize(t, root)
	if res.Status != StatusOK {
		t.Errorf("500 lines should PASS, got=%v evidence=%v", res.Status, res.Evidence)
	}
}

// Violation — 501 lines.
func TestT760_SkillSize_OverLimit_BLOCK(t *testing.T) {
	root := setupSkillSizeTree(t, "skills/foo/SKILL.md", 501)
	stubStaged(t, []string{"skills/foo/SKILL.md"})
	res := runSkillSize(t, root)
	if res.Status != StatusViolated {
		t.Errorf("501 lines should BLOCK, got=%v", res.Status)
	}
	if len(res.Evidence) == 0 || !strings.Contains(res.Evidence[0], "501") {
		t.Errorf("evidence missing '501': %v", res.Evidence)
	}
}

// scope skip — files other than SKILL.md, e.g. references/changelog.md.
func TestT760_SkillSize_NonSkillMD_skip(t *testing.T) {
	root := setupSkillSizeTree(t, "skills/foo/references/changelog.md", 1000)
	stubStaged(t, []string{"skills/foo/references/changelog.md"})
	res := runSkillSize(t, root)
	if res.Status != StatusOK {
		t.Errorf("references/ files should skip, got=%v", res.Status)
	}
}

// env override — HSTL_SKILL_SIZE_MAX=300 → 400 lines BLOCK.
func TestT760_SkillSize_EnvOverride(t *testing.T) {
	t.Setenv("HSTL_SKILL_SIZE_MAX", "300")
	root := setupSkillSizeTree(t, "skills/foo/SKILL.md", 400)
	stubStaged(t, []string{"skills/foo/SKILL.md"})
	res := runSkillSize(t, root)
	if res.Status != StatusViolated {
		t.Errorf("env=300 + 400 lines should BLOCK, got=%v", res.Status)
	}
}

// staged 0 → graceful OK.
func TestT760_SkillSize_NoStaged(t *testing.T) {
	root := setupSkillSizeTree(t, "skills/foo/SKILL.md", 600)
	stubStaged(t, []string{})
	res := runSkillSize(t, root)
	if res.Status != StatusOK {
		t.Errorf("no staged files should yield OK, got=%v", res.Status)
	}
}
