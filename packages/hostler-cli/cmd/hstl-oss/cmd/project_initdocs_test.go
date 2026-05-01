// Package cmd — regression tests for the project init-docs subcommand:
// idempotency, dry-run, and preservation of existing files.
package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// setupDocsTestRoot creates a temporary project root and returns its path.
// app.ProjectRoot() (internal fileutil.GetProjectRoot) walks up from cwd, so
// we t.Chdir() into the temp dir to simulate running from a real repo.
func setupDocsTestRoot(t *testing.T) string {
	t.Helper()
	tmp := t.TempDir()
	// GetProjectRoot looks for the .hstl/ marker (or git root); seed one so
	// tmp is recognised as the project root.
	if err := os.MkdirAll(filepath.Join(tmp, ".hstl"), 0o755); err != nil {
		t.Fatalf("create tmp .hstl: %v", err)
	}
	t.Chdir(tmp)
	return tmp
}

func TestRunInitDocs_CreatesStandardTree(t *testing.T) {
	tmp := setupDocsTestRoot(t)

	result, err := runInitDocs(false)
	if err != nil {
		t.Fatalf("runInitDocs: %v", err)
	}

	// docs/ + 9 prefixes + docs/index.md + 9 INDEX.md = 1+9+1+9 = 20 entries.
	if len(result.CreatedDirs) < 10 { // docs + 9 prefixes
		t.Errorf("CreatedDirs = %d, expected >= 10 (docs + 9 prefixes)", len(result.CreatedDirs))
	}
	if len(result.CreatedFiles) < 10 { // docs/index.md + 9 INDEX.md
		t.Errorf("CreatedFiles = %d, expected >= 10", len(result.CreatedFiles))
	}

	// Spot-check three of the expected files.
	for _, rel := range []string{
		"docs/index.md",
		"docs/00-project/INDEX.md",
		"docs/08-references/INDEX.md",
	} {
		full := filepath.Join(tmp, rel)
		if _, err := os.Stat(full); err != nil {
			t.Errorf("%s missing: %v", rel, err)
		}
	}
}

func TestRunInitDocs_Idempotent(t *testing.T) {
	setupDocsTestRoot(t)

	// First run.
	r1, err := runInitDocs(false)
	if err != nil {
		t.Fatalf("1st runInitDocs: %v", err)
	}
	created1 := len(r1.CreatedFiles) + len(r1.CreatedDirs)
	if created1 == 0 {
		t.Fatal("first run created nothing — initialization failed")
	}

	// Second run — every entry should be skipped (idempotent).
	r2, err := runInitDocs(false)
	if err != nil {
		t.Fatalf("2nd runInitDocs: %v", err)
	}
	if len(r2.CreatedFiles) != 0 || len(r2.CreatedDirs) != 0 {
		t.Errorf("second run produced new entries — idempotency violated: dirs=%v files=%v",
			r2.CreatedDirs, r2.CreatedFiles)
	}
	if len(r2.SkippedExisting) == 0 {
		t.Error("second run reported zero SkippedExisting — existing-file detection failed")
	}
}

func TestRunInitDocs_DryRun_DoesNotWrite(t *testing.T) {
	tmp := setupDocsTestRoot(t)

	result, err := runInitDocs(true)
	if err != nil {
		t.Fatalf("runInitDocs dry: %v", err)
	}

	if !result.DryRun {
		t.Error("DryRun flag not set")
	}
	if len(result.WouldCreateFiles) == 0 {
		t.Error("WouldCreateFiles is empty")
	}
	if len(result.CreatedFiles) != 0 || len(result.CreatedDirs) != 0 {
		t.Errorf("dry-run wrote entries: files=%v dirs=%v",
			result.CreatedFiles, result.CreatedDirs)
	}

	// Confirm nothing actually landed on disk.
	docsPath := filepath.Join(tmp, "docs")
	if _, err := os.Stat(docsPath); err == nil {
		t.Error("dry-run actually created docs/")
	}
}

func TestRunInitDocs_PreservesExistingFile(t *testing.T) {
	tmp := setupDocsTestRoot(t)

	// Simulate a user-authored INDEX.md.
	existingDir := filepath.Join(tmp, "docs", "00-project")
	if err := os.MkdirAll(existingDir, 0o755); err != nil {
		t.Fatalf("pre-setup mkdir: %v", err)
	}
	customContent := "# Custom project definition\n\n(user edits must be preserved)\n"
	existingFile := filepath.Join(existingDir, "INDEX.md")
	if err := os.WriteFile(existingFile, []byte(customContent), 0o644); err != nil {
		t.Fatalf("pre-setup write: %v", err)
	}

	_, err := runInitDocs(false)
	if err != nil {
		t.Fatalf("runInitDocs: %v", err)
	}

	// User content must survive.
	got, err := os.ReadFile(existingFile)
	if err != nil {
		t.Fatalf("re-read file: %v", err)
	}
	if string(got) != customContent {
		t.Errorf("existing file was overwritten — idempotency violated\n got=%q\nwant=%q", got, customContent)
	}
}

func TestRenderIndexTop_ContainsAllPrefixes(t *testing.T) {
	out := renderIndexTop()
	for _, p := range docsPrefixes {
		if !strings.Contains(out, p.Dir) {
			t.Errorf("index top is missing prefix %q", p.Dir)
		}
	}
	if strings.Contains(out, "{{ROWS}}") {
		t.Error("template substitution failed: {{ROWS}} still present")
	}
}

func TestRenderIndexPrefix_FillsTemplate(t *testing.T) {
	spec := docsPrefixSpec{Dir: "00-project", Title: "Project definition", Desc: "test description"}
	out := renderIndexPrefix(spec)

	if !strings.Contains(out, spec.Title) {
		t.Errorf("TITLE substitution failed: %s", out)
	}
	if !strings.Contains(out, spec.Desc) {
		t.Errorf("DESC substitution failed: %s", out)
	}
	if strings.Contains(out, "{{TITLE}}") || strings.Contains(out, "{{DESC}}") {
		t.Errorf("template placeholders still present: %s", out)
	}
}
