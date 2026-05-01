package task

import (
	"os"
	"path/filepath"
	"testing"
)

// T612 (Sprint-71): drift-less regenerated-file exception handling.

func TestT612_IsDriftLessRegenerated_MatchesPattern(t *testing.T) {
	dir := t.TempDir()
	gen := filepath.Join(dir, "docs", "generated")
	if err := os.MkdirAll(gen, 0o755); err != nil {
		t.Fatal(err)
	}
	manifestPath := filepath.Join(gen, "manifest.json")
	if err := os.WriteFile(manifestPath, []byte("{}"), 0o644); err != nil {
		t.Fatal(err)
	}

	if !isDriftLessRegenerated("docs/generated/manifest.json", dir) {
		t.Error("expected true on existing file + pattern match")
	}
}

func TestT612_IsDriftLessRegenerated_NoMatch(t *testing.T) {
	dir := t.TempDir()
	// file not in the pattern set
	p := filepath.Join(dir, "src", "foo.go")
	_ = os.MkdirAll(filepath.Dir(p), 0o755)
	_ = os.WriteFile(p, []byte("package foo"), 0o644)
	if isDriftLessRegenerated("src/foo.go", dir) {
		t.Error("expected false on no pattern match")
	}
}

func TestT612_IsDriftLessRegenerated_MissingFile(t *testing.T) {
	dir := t.TempDir()
	// pattern matches but file is absent
	if isDriftLessRegenerated("docs/generated/manifest.json", dir) {
		t.Error("expected false when file is missing (existence check fails)")
	}
}

func TestT612_IsDriftLessRegenerated_SuffixMatch(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "docs", "03-design", "skill-trigger-index.md")
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	_ = os.WriteFile(p, []byte("# stub"), 0o644)
	if !isDriftLessRegenerated("docs/03-design/skill-trigger-index.md", dir) {
		t.Error("expected default pattern skill-trigger-index.md to match")
	}
}
