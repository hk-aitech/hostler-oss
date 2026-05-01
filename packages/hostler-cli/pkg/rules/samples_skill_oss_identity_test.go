package rules

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// precommit.skill.oss_identity Rule regression tests.
//
// 5 violations + 1 clean + scope skip.

// setupSkillOSSTree — helper that creates a skills/<name>/SKILL.md tree.
//
//	files: relative path → body mapping.
func setupSkillOSSTree(t *testing.T, files map[string]string) string {
	t.Helper()
	root := t.TempDir()
	// Stub .git directory (matches behaviour of other precommit Rule tests
	// that check for the project root; this Rule does not need it but the
	// stub is harmless).
	if err := os.MkdirAll(filepath.Join(root, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	for rel, content := range files {
		full := filepath.Join(root, rel)
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return root
}

// stubStaged temporarily replaces precommitStagedFiles with a fixed list.
func stubStaged(t *testing.T, files []string) {
	t.Helper()
	orig := precommitStagedFiles
	precommitStagedFiles = func(*RuleContext) []string { return files }
	t.Cleanup(func() { precommitStagedFiles = orig })
}

// runSkillOSS — Rule execution helper.
func runSkillOSS(t *testing.T, root string) *RuleResult {
	t.Helper()
	r, ok := Get(skillOSSIdentityRuleID)
	if !ok {
		t.Fatalf("Rule %s not registered", skillOSSIdentityRuleID)
	}
	return r.Check(&RuleContext{ProjectRoot: root})
}

// Violation 2 — Pattern B (T### Task ID inline).
func TestT578_OSSIdentity_PatternB_TaskID_Violation(t *testing.T) {
	root := setupSkillOSSTree(t, map[string]string{
		"skills/foo/SKILL.md": "# Foo\nAdded as part of T578.",
	})
	stubStaged(t, []string{"skills/foo/SKILL.md"})
	res := runSkillOSS(t, root)
	if res.Status != StatusViolated {
		t.Fatalf("status=%v want Violated", res.Status)
	}
	if !containsHit(res.Evidence, "B.t_id") {
		t.Errorf("evidence missing B.t_id: %v", res.Evidence)
	}
}

// Violation 3 — Pattern B (Sprint-NN inline).
func TestT578_OSSIdentity_PatternB_Sprint_Violation(t *testing.T) {
	root := setupSkillOSSTree(t, map[string]string{
		"skills/foo/SKILL.md": "# Foo\nFinalised in Sprint-55.",
	})
	stubStaged(t, []string{"skills/foo/SKILL.md"})
	res := runSkillOSS(t, root)
	if res.Status != StatusViolated {
		t.Fatalf("status=%v want Violated", res.Status)
	}
	if !containsHit(res.Evidence, "B.sprint") {
		t.Errorf("evidence missing B.sprint: %v", res.Evidence)
	}
}

// Violation 4 — Pattern C (external project name reference).
func TestSkillOSSIdentity_PatternC_External_Violation(t *testing.T) {
	root := setupSkillOSSTree(t, map[string]string{
		"skills/foo/SKILL.md": "# Foo\nOriginated in the sample-internal-project repo.",
	})
	stubStaged(t, []string{"skills/foo/SKILL.md"})
	res := runSkillOSS(t, root)
	if res.Status != StatusViolated {
		t.Fatalf("status=%v want Violated", res.Status)
	}
	if !containsHit(res.Evidence, "C.external") {
		t.Errorf("evidence missing C.external hit: %v", res.Evidence)
	}
}

// Violation 5 — Pattern D (absolute-path prefix).
func TestT578_OSSIdentity_PatternD_AbsPath_Violation(t *testing.T) {
	root := setupSkillOSSTree(t, map[string]string{
		"skills/foo/SKILL.md": "# Foo\nPath example: /home/user/projects/foo/bar.go",
	})
	stubStaged(t, []string{"skills/foo/SKILL.md"})
	res := runSkillOSS(t, root)
	if res.Status != StatusViolated {
		t.Fatalf("status=%v want Violated", res.Status)
	}
	if !containsHit(res.Evidence, "D.abspath") {
		t.Errorf("evidence missing D.abspath: %v", res.Evidence)
	}
}

// Clean — no violation patterns.
func TestT578_OSSIdentity_Clean_PASS(t *testing.T) {
	root := setupSkillOSSTree(t, map[string]string{
		"skills/foo/SKILL.md": "---\nname: foo\n---\n# Foo\n\nUses hostler configuration.\nPath: packages/hostler-cli/pkg/rules/.\n",
	})
	stubStaged(t, []string{"skills/foo/SKILL.md"})
	res := runSkillOSS(t, root)
	if res.Status != StatusOK {
		t.Fatalf("status=%v want OK, evidence=%v", res.Status, res.Evidence)
	}
}

// Frontmatter region keywords are ignored — UUID/title/etc. that contain
// external-looking words.
func TestT578_OSSIdentity_Frontmatter_Excluded(t *testing.T) {
	root := setupSkillOSSTree(t, map[string]string{
		// A violation pattern (T999) appears in frontmatter — ignored. Body is clean.
		"skills/foo/SKILL.md": "---\nname: foo\nlegacy: T999 (history)\n---\n# Foo\nClean body.\n",
	})
	stubStaged(t, []string{"skills/foo/SKILL.md"})
	res := runSkillOSS(t, root)
	if res.Status != StatusOK {
		t.Fatalf("status=%v want OK (frontmatter excluded), evidence=%v", res.Status, res.Evidence)
	}
}

// Scope — references/ subtree is excluded from checks.
func TestT578_OSSIdentity_Scope_References_Skip(t *testing.T) {
	root := setupSkillOSSTree(t, map[string]string{
		// Even with violation patterns under references/, it is not
		// SKILL.md so it is skipped.
		"skills/foo/references/changelog.md": "T578 / Sprint-56 / monorepo",
		// SKILL.md itself is clean.
		"skills/foo/SKILL.md": "# Foo\nClean.\n",
	})
	stubStaged(t, []string{"skills/foo/references/changelog.md"})
	res := runSkillOSS(t, root)
	if res.Status != StatusSkipped {
		t.Fatalf("status=%v want Skipped (no SKILL.md staged)", res.Status)
	}
}

// Scope — _frozen / _shared / examples subtree SKILL.md files also skip.
func TestT578_OSSIdentity_Scope_Frozen_Skip(t *testing.T) {
	root := setupSkillOSSTree(t, map[string]string{
		"skills/_frozen/old/SKILL.md": "# Old\nLegacy invocation. T100 work.\n",
	})
	stubStaged(t, []string{"skills/_frozen/old/SKILL.md"})
	res := runSkillOSS(t, root)
	if res.Status != StatusSkipped {
		t.Fatalf("status=%v want Skipped (_frozen scope), evidence=%v", res.Status, res.Evidence)
	}
}

// When staged is empty — Skipped.
func TestT578_OSSIdentity_NoStaged_Skip(t *testing.T) {
	root := setupSkillOSSTree(t, map[string]string{
		"skills/foo/SKILL.md": "# Foo\nT999 violation.\n",
	})
	stubStaged(t, []string{}) // SKILL.md not changed
	res := runSkillOSS(t, root)
	if res.Status != StatusSkipped {
		t.Fatalf("status=%v want Skipped (no staged SKILL.md)", res.Status)
	}
}

// helper — does an evidence entry contain the given pattern ID?
func containsHit(evidence []string, patternID string) bool {
	for _, e := range evidence {
		if strings.Contains(e, ":"+patternID+":") {
			return true
		}
	}
	return false
}

// Pattern G — block project-specific globs in frontmatter paths.
func TestSkillSharedIdentity_PatternG_FrontmatterPaths_Violation(t *testing.T) {
	root := setupSkillOSSTree(t, map[string]string{
		"skills/foo/SKILL.md": "---\nname: foo\npaths: [\"docs/07-knowledge/**/*\", \"works/**/*\"]\n---\n# Foo\nClean body.\n",
	})
	stubStaged(t, []string{"skills/foo/SKILL.md"})
	res := runSkillOSS(t, root)
	if res.Status != StatusViolated {
		t.Fatalf("status=%v want Violated (Pattern G — frontmatter paths)", res.Status)
	}
	found := false
	for _, e := range res.Evidence {
		if strings.Contains(e, "G.fm_paths") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected G.fm_paths in evidence, got %v", res.Evidence)
	}
}

// Pattern G — paths with only generic globs pass.
func TestSkillSharedIdentity_PatternG_FrontmatterPaths_GenericGlob_Pass(t *testing.T) {
	root := setupSkillOSSTree(t, map[string]string{
		"skills/foo/SKILL.md": "---\nname: foo\npaths: [\"**/*.md\", \"works/**/*\"]\n---\n# Foo\nClean body.\n",
	})
	stubStaged(t, []string{"skills/foo/SKILL.md"})
	res := runSkillOSS(t, root)
	if res.Status != StatusOK {
		t.Errorf("expected OK (generic glob), got %v evidence=%v", res.Status, res.Evidence)
	}
}
