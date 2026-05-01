package backlog

import (
	"os"
	"path/filepath"
	"testing"
)

// T431 (Sprint-71): SPRINT.md `## Goal` section parsing fallback.

func TestT431_ExtractGoalFromMarkdown_English(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "SPRINT.md")
	content := `# sprint-99

## Goal

Ship quality fixes.

## Tasks
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	got := extractGoalFromMarkdown(path)
	if got != "Ship quality fixes." {
		t.Errorf("got %q", got)
	}
}

func TestT431_ExtractGoalFromMarkdown_NoSection(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "SPRINT.md")
	if err := os.WriteFile(path, []byte("# sprint-99\n\n## Tasks\n- T001\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if got := extractGoalFromMarkdown(path); got != "" {
		t.Errorf("expected empty when section is missing, got %q", got)
	}
}

func TestT431_ExtractGoalFromMarkdown_MissingFile(t *testing.T) {
	if got := extractGoalFromMarkdown("/nonexistent/SPRINT.md"); got != "" {
		t.Errorf("expected empty when file is missing, got %q", got)
	}
}

func TestT431_ExtractGoalFromMarkdown_IgnoresFrontmatter(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "SPRINT.md")
	content := `---
## Goal: this is inside frontmatter and must be ignored
---

## Goal

real goal.
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	if got := extractGoalFromMarkdown(path); got != "real goal." {
		t.Errorf("got %q", got)
	}
}
