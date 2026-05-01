package ceremony

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

// TestFindBacktickRefs_SlashFilter (T729) verifies that only slash-bearing
// paths are extracted and code-ish snippets without a slash are excluded.
func TestFindBacktickRefs_SlashFilter(t *testing.T) {
	content := `# Task

related files: ` + "`cli/pkg/foo.go`" + ` modified along with ` + "`const Foo = 1`" + ` added.
also see ` + "`docs/07-knowledge/x.md`" + `. ` + "`func Bar()`" + ` is internal.

- ` + "`a/b.md`" + `
- ` + "`c/d.md`" + `
- duplicate ` + "`a/b.md`" + `
`
	refs := findBacktickRefs(content)
	want := []string{"cli/pkg/foo.go", "docs/07-knowledge/x.md", "a/b.md", "c/d.md"}
	if !reflect.DeepEqual(refs, want) {
		t.Errorf("refs: got %v, want %v", refs, want)
	}
}

// TestFindBacktickRefs_CommandExclusion (T729) — patterns like
// `go test ./...` are excluded.
func TestFindBacktickRefs_CommandExclusion(t *testing.T) {
	content := "run: `go test ./...` and `./cli/...` plus `foo bar`."
	refs := findBacktickRefs(content)
	if len(refs) != 0 {
		t.Errorf("commands and whitespace must be excluded — got %v", refs)
	}
}

// TestFindBacktickRefs_TrailingPunct (T729) — trailing punctuation is
// trimmed.
func TestFindBacktickRefs_TrailingPunct(t *testing.T) {
	content := "see: `cli/foo.go`,"
	refs := findBacktickRefs(content)
	if len(refs) != 1 || refs[0] != "cli/foo.go" {
		t.Errorf("trailing comma trim failed: got %v", refs)
	}
}

// TestFindBacktickRefs_BraceExpansionExcluded (T414) — shell brace notation
// is excluded.
func TestFindBacktickRefs_BraceExpansionExcluded(t *testing.T) {
	content := "related: `docs/{ADR-023,ADR-024}.md` and `cli/real.go`"
	refs := findBacktickRefs(content)
	want := []string{"cli/real.go"}
	if !reflect.DeepEqual(refs, want) {
		t.Errorf("brace expansion not excluded: got %v, want %v", refs, want)
	}
}

// TestCheckBacktickRefsExist (T729) verifies actual filesystem cross-check.
func TestCheckBacktickRefsExist(t *testing.T) {
	tmp := t.TempDir()
	// create one existing file
	existPath := "real/file.md"
	_ = os.MkdirAll(filepath.Join(tmp, "real"), 0o755)
	_ = os.WriteFile(filepath.Join(tmp, existPath), []byte("x"), 0o644)

	refs := []string{existPath, "no/such.md", "also/missing.go"}
	stale := checkBacktickRefsExist(tmp, refs)
	want := []string{"no/such.md", "also/missing.go"}
	if !reflect.DeepEqual(stale, want) {
		t.Errorf("stale: got %v, want %v", stale, want)
	}
}

// TestCheckBacktickRefsExist_packages_prefix (T397, Sprint-33) verifies
// that for paths under packages/hostler-cli/* / packages/hostler-plugin/*
// after the Sprint 5 fork the prefix-omission pattern is resolved. Tasks
// often write `pkg/task/status.go` for short, so this support reduces
// stale false positives.
func TestCheckBacktickRefsExist_packages_prefix(t *testing.T) {
	tmp := t.TempDir()
	files := map[string]string{
		"packages/hostler-cli/pkg/task/status.go":                 "package task",
		"packages/hostler-cli/cmd/hstl-oss/cmd/task.go":              "package cmd",
		"packages/hostler-cli/internal/app/app.go":                "package app",
		"packages/hostler-plugin/skills/task-management/SKILL.md": "# SKILL",
		"packages/hostler-plugin/commands/task/complete.md":       "# complete",
	}
	for relPath, content := range files {
		dir := filepath.Dir(filepath.Join(tmp, relPath))
		_ = os.MkdirAll(dir, 0o755)
		_ = os.WriteFile(filepath.Join(tmp, relPath), []byte(content), 0o644)
	}

	refs := []string{
		"pkg/task/status.go",              // packages/hostler-cli/ prefix auto-detected
		"cmd/hstl-oss/cmd/task.go",           // packages/hostler-cli/ prefix
		"internal/app/app.go",             // packages/hostler-cli/ prefix
		"skills/task-management/SKILL.md", // packages/hostler-plugin/ prefix
		"commands/task/complete.md",       // packages/hostler-plugin/ prefix
		"no/such/file.go",                 // missing — stale
	}
	stale := checkBacktickRefsExist(tmp, refs)
	want := []string{"no/such/file.go"}
	if !reflect.DeepEqual(stale, want) {
		t.Errorf("packages/* prefix stale: got %v, want %v", stale, want)
	}
}

// TestExtractArtifactSection_Basic (T414) — only "## Result → ### Artifacts"
// is extracted.
func TestExtractArtifactSection_Basic(t *testing.T) {
	content := `# Task

## Purpose

narrative mentioning ` + "`cli/nonexistent.go`" + ` must be excluded.

## Result

### Design Decisions

trade-off uses ` + "`archive/old-plugin-*/`" + ` pattern — glob, not a file.

### Artifacts

- ` + "`cli/real.go`" + ` (new)
- ` + "`docs/guide.md`" + ` (modified)

### Validation

- ` + "`go test ./...`" + ` PASS
- benchmark: ` + "`scripts/bench.sh:42-58`" + ` line-range mention.

## Status change history

tail narrative.
`
	section := ExtractArtifactSection(content)
	// the two artefact paths must appear in the extracted section
	if !strings.Contains(section, "cli/real.go") {
		t.Errorf("expected artefact path included: got %q", section)
	}
	if !strings.Contains(section, "docs/guide.md") {
		t.Errorf("expected artefact path included: got %q", section)
	}
	// narrative paths must be excluded
	if strings.Contains(section, "cli/nonexistent.go") {
		t.Errorf("narrative (Purpose) path must not be included: got %q", section)
	}
	if strings.Contains(section, "archive/old-plugin-*") {
		t.Errorf("narrative (Design Decisions) path must not be included: got %q", section)
	}
	if strings.Contains(section, "scripts/bench.sh") {
		t.Errorf("narrative (Validation) path must not be included: got %q", section)
	}
}

// TestFindBacktickRefsInArtifacts_NarrativeExcluded (T414) — narrative
// paths are excluded (regression).
func TestFindBacktickRefsInArtifacts_NarrativeExcluded(t *testing.T) {
	content := `## Result

### Design Decisions

filenames such as ` + "`archive/old/*.md`" + ` were a false-positive glob example.
also ` + "`docs/06-reports/X`" + ` placeholder and ` + "`~/.local/bin/hstl`" + ` home-dir.

### Artifacts

- ` + "`packages/hostler-cli/pkg/ceremony/backtick_refs.go`" + ` (modified)
- ` + "`packages/hostler-cli/cmd/hstl-oss/cmd/doc_review.go`" + ` (modified)

### Validation

- line range ` + "`doc_review.go:108-128`" + ` marks the change site.
`
	refs := FindBacktickRefsInArtifacts(content)
	want := []string{
		"packages/hostler-cli/pkg/ceremony/backtick_refs.go",
		"packages/hostler-cli/cmd/hstl-oss/cmd/doc_review.go",
	}
	if !reflect.DeepEqual(refs, want) {
		t.Errorf("artifact refs: got %v, want %v", refs, want)
	}
}

// TestFindBacktickRefsInArtifacts_NoArtifactSection (T414) — in-progress
// Tasks return an empty result.
func TestFindBacktickRefsInArtifacts_NoArtifactSection(t *testing.T) {
	// no Result section at all (in-progress Task)
	content := `## Purpose

` + "`cli/foo.go`" + ` must be modified.

## Requirements

- ` + "`cli/bar.go`" + ` reference.
`
	refs := FindBacktickRefsInArtifacts(content)
	if len(refs) != 0 {
		t.Errorf("expected empty result when no Result section: got %v", refs)
	}
}

// TestFindBacktickRefsInArtifacts_NoSubHeading (T414) — backward compat.
// Writing directly under "## Result" without a sub-heading is not treated
// as artifacts (in-progress convention). This is intentional — ArtifactOnly
// requires an explicit "### Artifacts" heading.
func TestFindBacktickRefsInArtifacts_NoSubHeading(t *testing.T) {
	content := `## Result

- ` + "`cli/foo.go`" + ` (modified)
- ` + "`cli/bar.go`" + ` (new)
`
	refs := FindBacktickRefsInArtifacts(content)
	if len(refs) != 0 {
		t.Errorf("expected empty result without sub-heading (explicit required): got %v", refs)
	}
}

// TestFindBacktickRefsInArtifacts_SubHeadingVariants (T414) — variants of
// the artifact keyword.
func TestFindBacktickRefsInArtifacts_SubHeadingVariants(t *testing.T) {
	cases := []struct {
		name    string
		heading string
	}{
		{"Artifacts", "Artifacts"},
		{"Artifact", "Artifact"},
		{"Files", "Files"},
		{"Changed files", "Changed files"},
		{"Changed Files", "Changed Files"},
		{"Artifacts (8 items) — first-word match", "Artifacts (8 items)"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			content := "## Result\n\n### " + c.heading + "\n\n- `cli/x.go`\n"
			refs := FindBacktickRefsInArtifacts(content)
			want := []string{"cli/x.go"}
			if !reflect.DeepEqual(refs, want) {
				t.Errorf("heading=%q: got %v, want %v", c.heading, refs, want)
			}
		})
	}
}

// TestFullScanFindBacktickRefs_CompatRegression (T414) — FullScan behaviour
// stays unchanged. sprint start design_readiness still scans the entire
// body as before.
func TestFullScanFindBacktickRefs_CompatRegression(t *testing.T) {
	content := `## Purpose

` + "`cli/objective.go`" + ` modification needed.

## Result

### Design Decisions

` + "`cli/design.go`" + ` alternatives reviewed.

### Artifacts

- ` + "`cli/artifact.go`" + ` (new)
`
	refs := FindBacktickRefs(content) // FullScan
	// FullScan includes everything regardless of narrative/artifact split
	seen := map[string]bool{}
	for _, r := range refs {
		seen[r] = true
	}
	for _, must := range []string{"cli/objective.go", "cli/design.go", "cli/artifact.go"} {
		if !seen[must] {
			t.Errorf("FullScan must include %q: got %v", must, refs)
		}
	}
}

// TestT431_StaleScanner_NarrativeExcluded — verifies the doc_review stale
// scanner does not catch backtick paths under narrative sections
// (Purpose / Design Decisions / Validation) as false positives.
//
// Background: in Sprint 36 there were 3 recurrences — FullScan was
// reporting paths under "### Design Decisions" as stale, leading to
// doc-review BLOCKED. T414 (Sprint-35) replaced that with
// FindBacktickRefsInArtifacts to block this structurally. This test
// guards against regression.
func TestT431_StaleScanner_NarrativeExcluded(t *testing.T) {
	// completed Task body (with Result section) processed by doc_review.go
	content := `# T999 test Task

## Purpose

` + "`narrative/nonexistent.go`" + ` and ` + "`docs/does-not-exist.md`" + ` are narrative mentions.

## Result

### Design Decisions

` + "`design/also-nonexistent.go`" + ` is a Design Decisions narrative — was a stale false positive.
also ` + "`archive/old-pattern-*/`" + ` glob examples are narrative.

### Artifacts

- ` + "`real/artifact.go`" + ` (new)

### Validation

- ` + "`go test ./...`" + ` PASS
- check: ` + "`scripts/verify.sh:10-20`" + ` line-range marker.
`
	// FindBacktickRefsInArtifacts must return only what is under
	// ### Artifacts.
	refs := FindBacktickRefsInArtifacts(content)

	// narrative paths must not be included
	for _, r := range refs {
		if r == "narrative/nonexistent.go" || r == "docs/does-not-exist.md" ||
			r == "design/also-nonexistent.go" {
			t.Errorf("narrative path included by stale scanner (false positive): %q", r)
		}
	}

	// the artifact path must be present
	found := false
	for _, r := range refs {
		if r == "real/artifact.go" {
			found = true
		}
	}
	if !found {
		t.Errorf("artifact path missing from scanner: got %v", refs)
	}
}

// TestCheckBacktickRefsExist_DeepPrefix verifies that commonPathPrefixes
// resolves deep paths under packages/hostler-cli/* and packages/hostler-plugin/*
// when Task bodies omit the leading `packages/...` segment.
func TestCheckBacktickRefsExist_DeepPrefix(t *testing.T) {
	tmp := t.TempDir()
	// create files in the canonical post-fork layout
	files := map[string]string{
		"packages/hostler-cli/cmd/hstl-oss/cmd/foo.go": "package cmd",
		"packages/hostler-cli/pkg/task/bar.go":         "package task",
		"packages/hostler-cli/internal/ports/baz.go":   "package ports",
	}
	for relPath, content := range files {
		dir := filepath.Dir(filepath.Join(tmp, relPath))
		_ = os.MkdirAll(dir, 0o755)
		_ = os.WriteFile(filepath.Join(tmp, relPath), []byte(content), 0o644)
	}

	// prefix-omission patterns frequently used in Task bodies
	refs := []string{
		"cmd/hstl-oss/cmd/foo.go", // packages/hostler-cli/ + cmd/hstl-oss/cmd/ prefix auto-search
		"pkg/task/bar.go",         // packages/hostler-cli/ prefix auto-search
		"internal/ports/baz.go",   // packages/hostler-cli/ prefix auto-search
		"no/such/file.go",         // not present — stale
	}
	stale := checkBacktickRefsExist(tmp, refs)
	want := []string{"no/such/file.go"}
	if !reflect.DeepEqual(stale, want) {
		t.Errorf("deep prefix stale: got %v, want %v", stale, want)
	}
}
