package fileutil

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestT642_CheckFrontmatterIDSanity_ArchivePath_Block(t *testing.T) {
	r := CheckFrontmatterIDSanity("T100", "archive/some/T100.md", "task")
	if !strings.HasPrefix(r.Reason, "archive_path_block") {
		t.Errorf("expected archive_path_block, got %q", r.Reason)
	}
}

func TestT642_CheckFrontmatterIDSanity_NestedArchivePath_Block(t *testing.T) {
	r := CheckFrontmatterIDSanity("T100", "/abs/path/archive/sub/T100.md", "task")
	if !strings.HasPrefix(r.Reason, "archive_path_block") {
		t.Errorf("expected archive_path_block, got %q", r.Reason)
	}
}

func TestT642_CheckFrontmatterIDSanity_Match_Pass(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "T100.md")
	body := "---\nid: T100\nstatus: todo\n---\n# task body\n"
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	r := CheckFrontmatterIDSanity("T100", path, "task")
	if r.Reason != "" {
		t.Errorf("expected pass (empty reason), got %q", r.Reason)
	}
	if r.FmID != "T100" {
		t.Errorf("expected FmID=T100, got %q", r.FmID)
	}
}

func TestT642_CheckFrontmatterIDSanity_Mismatch_Block(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "T100.md")
	body := "---\nid: T999\n---\n"
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	r := CheckFrontmatterIDSanity("T100", path, "task")
	if !strings.HasPrefix(r.Reason, "frontmatter_id_mismatch") {
		t.Errorf("expected frontmatter_id_mismatch, got %q", r.Reason)
	}
}

func TestT642_CheckFrontmatterIDSanity_NoFrontmatter_Pass(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "T100.md")
	body := "# raw body without frontmatter\n"
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	r := CheckFrontmatterIDSanity("T100", path, "task")
	if r.Reason != "" {
		t.Errorf("expected pass (frontmatter absent = normal), got %q", r.Reason)
	}
}

func TestT642_CheckFrontmatterIDSanity_SprintEntity(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "SPRINT.md")
	body := "---\nid: sprint-99\n---\n"
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	r := CheckFrontmatterIDSanity("sprint-100", path, "sprint")
	if !strings.HasPrefix(r.Reason, "frontmatter_id_mismatch") {
		t.Errorf("expected mismatch for sprint, got %q", r.Reason)
	}
	if !strings.Contains(r.Reason, "sprint_id") {
		t.Errorf("expected sprint_id in reason (entityType), got %q", r.Reason)
	}
}

func TestT642_ExtractFrontmatterID_QuotedValues(t *testing.T) {
	cases := []struct {
		name, body, want string
	}{
		{"plain", "---\nid: T100\n---\n", "T100"},
		{"quoted", `---` + "\n" + `id: "T200"` + "\n---\n", "T200"},
		{"single-quoted", "---\nid: 'T300'\n---\n", "T300"},
		{"no-frontmatter", "# body\n", ""},
		{"id-missing", "---\ntitle: foo\n---\n", ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			tmp := t.TempDir()
			p := filepath.Join(tmp, "x.md")
			os.WriteFile(p, []byte(c.body), 0o644)
			got, _ := ExtractFrontmatterID(p)
			if got != c.want {
				t.Errorf("got %q, want %q", got, c.want)
			}
		})
	}
}
