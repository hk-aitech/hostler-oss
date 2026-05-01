// trac: HAR-CM005
package task

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestAssignSanityCheck_ArchivePath — archive paths fast-fail.
func TestAssignSanityCheck_ArchivePath(t *testing.T) {
	cases := []struct{ path string }{
		{"archive/backups/old/T100-foo.md"},
		{"/abs/repo/archive/legacy/T100-foo.md"},
		{"works/sprints/archive/sprint-NN/T100-foo.md"}, // contains /archive/ in the middle
	}
	for _, c := range cases {
		reason := assignSanityCheck("T100", c.path)
		if !strings.Contains(reason, "archive_path_block") {
			t.Errorf("path %q should be archive blocked, got %q", c.path, reason)
		}
	}
}

// TestAssignSanityCheck_FrontmatterMismatch — frontmatter id != task_id BLOCK.
func TestAssignSanityCheck_FrontmatterMismatch(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "T200-foo.md")
	body := "---\nid: T999\ntitle: foo\n---\n\n# T999 ..."
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	reason := assignSanityCheck("T200", path)
	if !strings.Contains(reason, "frontmatter_id_mismatch") {
		t.Errorf("expected frontmatter mismatch BLOCK, got %q", reason)
	}
}

// TestAssignSanityCheck_FrontmatterMatch — valid frontmatter passes.
func TestAssignSanityCheck_FrontmatterMatch(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "T300-foo.md")
	body := "---\nid: T300\ntitle: foo\n---\n\n# T300 ..."
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	reason := assignSanityCheck("T300", path)
	if reason != "" {
		t.Errorf("expected pass, got %q", reason)
	}
}

// TestAssignSanityCheck_NoFrontmatter — passes when frontmatter is absent
// (normal immediately after creation).
func TestAssignSanityCheck_NoFrontmatter(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "T400-foo.md")
	if err := os.WriteFile(path, []byte("# T400 plain markdown\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	reason := assignSanityCheck("T400", path)
	if reason != "" {
		t.Errorf("no frontmatter should pass, got %q", reason)
	}
}

// TestExtractFrontmatterID_QuotedValue — yaml quoted id value.
func TestExtractFrontmatterID_QuotedValue(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "T500.md")
	body := `---
id: "T500"
title: foo
---
`
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	id, err := extractFrontmatterID(path)
	if err != nil {
		t.Fatal(err)
	}
	if id != "T500" {
		t.Errorf("expected T500, got %q", id)
	}
}
