package rules

import (
	"os"
	"path/filepath"
	"testing"
)

// G6 skill hook Rule regression tests.

func setupSkillsTree(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	// Create the hook files.
	for _, rel := range skillHookExpectedFiles {
		full := filepath.Join(root, rel)
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte("#!/bin/bash\n"), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	// skills/foo/SKILL.md with the tools field set.
	good := filepath.Join(root, "skills", "foo", "SKILL.md")
	_ = os.MkdirAll(filepath.Dir(good), 0o755)
	_ = os.WriteFile(good, []byte("---\ntools: [Read]\n---\n# Foo"), 0o644)
	// skills/bar/SKILL.md without the tools field.
	bad := filepath.Join(root, "skills", "bar", "SKILL.md")
	_ = os.MkdirAll(filepath.Dir(bad), 0o755)
	_ = os.WriteFile(bad, []byte("---\nname: bar\n---\n# Bar"), 0o644)
	return root
}

func TestT588_SkillHookExecPresent_OK(t *testing.T) {
	root := setupSkillsTree(t)
	r, _ := Get("skill.hook.exec_present")
	res := r.Check(&RuleContext{ProjectRoot: root})
	if res.Status != StatusOK {
		t.Errorf("status=%v, want OK, evidence=%v", res.Status, res.Evidence)
	}
}

func TestT588_SkillHookExecPresent_Missing_Violation(t *testing.T) {
	root := t.TempDir() // empty tree
	r, _ := Get("skill.hook.exec_present")
	res := r.Check(&RuleContext{ProjectRoot: root})
	if res.Status != StatusViolated {
		t.Errorf("status=%v, want Violated", res.Status)
	}
	if len(res.Evidence) != len(skillHookExpectedFiles) {
		t.Errorf("evidence len=%d, want %d", len(res.Evidence), len(skillHookExpectedFiles))
	}
}

func TestT588_SkillHookExecPresent_NotExecutable_Violation(t *testing.T) {
	root := setupSkillsTree(t)
	// Strip execute permission from one file.
	target := filepath.Join(root, skillHookExpectedFiles[0])
	if err := os.Chmod(target, 0o644); err != nil {
		t.Fatal(err)
	}
	r, _ := Get("skill.hook.exec_present")
	res := r.Check(&RuleContext{ProjectRoot: root})
	if res.Status != StatusViolated {
		t.Errorf("status=%v, want Violated", res.Status)
	}
}

func TestT588_SkillFrontmatterToolsDeclared_Missing_Detected(t *testing.T) {
	root := setupSkillsTree(t)
	r, _ := Get("skill.frontmatter.tools_declared")
	res := r.Check(&RuleContext{ProjectRoot: root})
	if res.Status != StatusViolated {
		t.Errorf("status=%v, want Violated, evidence=%v", res.Status, res.Evidence)
	}
	// bar is the missing candidate.
	found := false
	for _, e := range res.Evidence {
		if e == "bar" {
			found = true
		}
	}
	if !found {
		t.Errorf("failed to detect missing 'bar': %v", res.Evidence)
	}
}

func TestT588_SkillFrontmatterToolsDeclared_ProjectRoot_Missing_skip(t *testing.T) {
	r, _ := Get("skill.frontmatter.tools_declared")
	res := r.Check(&RuleContext{})
	if res.Status != StatusSkipped {
		t.Errorf("status=%v, want Skipped", res.Status)
	}
}
