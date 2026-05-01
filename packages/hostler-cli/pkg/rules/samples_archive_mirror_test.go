// trac: HAR-CM015
package rules

import (
	"testing"
)

func TestT526_ArchiveMirror_RootOK(t *testing.T) {
	cases := []string{
		"archive/docs/foo.md",
		"archive/backups/snap1/file.txt",
		"archive/backups/legacy-knowledge-2026-04-25/knowledge/decisions/foo.md",
	}
	for _, c := range cases {
		if isArchiveMirrorViolation(c) {
			t.Errorf("root archive %q wrongly flagged as a violation", c)
		}
	}
}

func TestT526_ArchiveMirror_SubArchive_Violation(t *testing.T) {
	cases := []string{
		"docs/archive/foo.md",
		"packages/cli/archive/old.go",
		"works/sprints/archive/sprint-99/SPRINT.md",
		"docs/02-architecture/archive/legacy/adr.md",
		"a/b/archive/c.md",
	}
	for _, c := range cases {
		if !isArchiveMirrorViolation(c) {
			t.Errorf("sub-archive %q not detected as a violation", c)
		}
	}
}

func TestT526_ArchiveMirror_NoArchive_OK(t *testing.T) {
	cases := []string{
		"docs/02-architecture/foo.md",
		"packages/hostler-cli/cmd/hstl-oss/main.go",
		"works/tasks/T001.md",
		"README.md",
	}
	for _, c := range cases {
		if isArchiveMirrorViolation(c) {
			t.Errorf("legitimate path %q wrongly flagged as a violation", c)
		}
	}
}

func TestT526_ArchiveMirror_EdgeCases(t *testing.T) {
	// empty / single segment.
	if isArchiveMirrorViolation("") {
		t.Errorf("empty path flagged as a violation")
	}
	if isArchiveMirrorViolation("archive") {
		t.Errorf("'archive' alone flagged as a violation")
	}
	if isArchiveMirrorViolation("readme.md") {
		t.Errorf("top-level file flagged as a violation")
	}
	// name contains "archive" but is not a segment.
	if isArchiveMirrorViolation("docs/archived-by-2026.md") {
		t.Errorf("'archived-' prefix flagged as a violation")
	}
}

func TestT526_ArchiveMirror_RuleIntegration(t *testing.T) {
	root := setupKBRefTree(t, map[string]string{
		"docs/archive/old.md": "stale content",
	})
	stubStaged(t, []string{"docs/archive/old.md"})
	r, ok := Get(archiveMirrorRuleID)
	if !ok {
		t.Fatalf("Rule not registered")
	}
	res := r.Check(&RuleContext{ProjectRoot: root})
	if res.Status != StatusViolated {
		t.Errorf("expected Violated, got %v", res.Status)
	}
}

func TestT526_ArchiveMirror_RuleAllowsRoot(t *testing.T) {
	root := setupKBRefTree(t, map[string]string{
		"archive/docs/foo.md": "ok",
	})
	stubStaged(t, []string{"archive/docs/foo.md"})
	r, _ := Get(archiveMirrorRuleID)
	res := r.Check(&RuleContext{ProjectRoot: root})
	if res.Status != StatusOK {
		t.Errorf("expected OK, got %v evidence=%v", res.Status, res.Evidence)
	}
}
