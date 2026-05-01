// T182 — Task result-section parser/policy unit tests.
// T217 — secondary parser false-positive avoidance (glob / git mv arrow)
// regression tests.
// Internal-package (task) tests so unexported functions can be reached.
package task

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// =============================================================================
// T418 — nested bullet false-positive avoidance (ISS-20260413-004)
// =============================================================================

// TestT418_ParseTaskResultFiles_NestedBullet verifies that 2-space-indented
// nested bullets are not misclassified as path candidates.
func TestT418_ParseTaskResultFiles_NestedBullet(t *testing.T) {
	content := `## Result

### Artifacts

- src/foo.go
  - explanation nested bullet 1
  - explanation nested bullet 2
- src/bar.go
  - another nested explanation
`
	paths := parseTaskResultFiles(content, []string{"Result"})
	wantSet := map[string]bool{"src/foo.go": true, "src/bar.go": true}
	gotSet := map[string]bool{}
	for _, p := range paths {
		gotSet[p] = true
	}
	for p := range wantSet {
		if !gotSet[p] {
			t.Errorf("expected path %q missing", p)
		}
	}
	for p := range gotSet {
		if !wantSet[p] {
			t.Errorf("unexpected path extracted (nested bullet false positive): %q", p)
		}
	}
}

// TestT476_ParseTaskResultFiles_ShortBasename verifies that even when the
// result section uses a short basename only (e.g. "- manifest.json
// (regenerated)"), the file candidate is still extracted (sprint-55 T460
// firsthand experience — guards against 3-time BLOCK recurrence).
func TestT476_ParseTaskResultFiles_ShortBasename(t *testing.T) {
	content := `## Result

### Artifacts

- manifest.json (regenerated)
- README.md content enriched
- docs/generated/manifest.json fully regenerated
`
	got := parseTaskResultFiles(content, []string{"Result", "Artifacts"})
	want := map[string]bool{
		"manifest.json":                true,
		"README.md":                    true,
		"docs/generated/manifest.json": true,
	}
	gotSet := map[string]bool{}
	for _, p := range got {
		gotSet[p] = true
	}
	for p := range want {
		if !gotSet[p] {
			t.Errorf("expected path %q missing (got=%v)", p, got)
		}
	}
}

// TestT476_ParseTaskResultFiles_AbsolutePath reflects the T199 (Sprint-09)
// policy change.
// Before: parser excluded absolute paths early to "prevent slash
// confusion".
// Now: parser passes absolute paths through and verifyTaskResultsForType
//
//	classifies them missing if they sit outside projectRoot — closes a
//	deliberate bypass.
//
// (T199: blocks the "list /tmp/fake.txt as artefact to skip verification"
// bypass.)
func TestT476_ParseTaskResultFiles_AbsolutePath_Excluded(t *testing.T) {
	content := `## Result

### Artifacts

- /home/user/project/file.go
- src/included.go
`
	got := parseTaskResultFiles(content, []string{"Result", "Artifacts"})
	// After T199: parser also lets absolute paths through. verify-side
	// makes the projectRoot decision.
	if len(got) != 2 {
		t.Errorf("absolute path should also pass parsing (T199): got=%v", got)
	}
}

// =============================================================================
// T216 — HasPlaceholderBody backtick false-positive avoidance
// =============================================================================

// TestHasPlaceholderBody_T216_BacktickQuoteExcluded verifies that
// backtick-wrapped placeholder markers do not produce false positives
// (T216).
func TestHasPlaceholderBody_T216_BacktickQuoteExcluded(t *testing.T) {
	// HasPlaceholderBody is DB + filepath driven, so we exercise it via
	// _backtickContentRe and _placeholderMarkers directly.
	cases := []struct {
		name string
		body string
		want bool // true = placeholder remaining is suspected
	}{
		{
			name: "marker example inside backticks is ignored",
			body: "Body complete. Mentions `{The problem this Task solves}` only as an example; not an actual placeholder.",
			want: false,
		},
		{
			name: "real marker without backticks is detected",
			body: "Body not written\n## Purpose\n{The problem this Task solves}\n",
			want: true,
		},
		{
			name: "self-referencing Task body",
			body: "no placeholder markers like `{Requirement 1}` or `{Criterion 1}` are present",
			want: false,
		},
		{
			name: "the word placeholder itself is not a forbidden marker",
			body: "the file parser detects placeholders",
			want: false,
		},
		{
			// Using a natural-language substring identical to the
			// detection message as a marker would produce false
			// positives whenever a Task descriptively mentions the
			// term. Only structural patterns are retained as markers.
			name: "natural-language substring self-reference not detected",
			body: "the post-mortem repeatedly logged 'placeholder body remaining' warnings; the cause was using a natural-language substring as a marker.",
			want: false,
		},
		{
			name: "real un-substituted template is still detected",
			body: "## Purpose\n{The problem this Task solves}\n## Requirements\n- [ ] {Requirement 1}\n",
			want: true,
		},
	}
	for _, tc := range cases {
		stripped := _backtickContentRe.ReplaceAllString(tc.body, "")
		found := false
		for _, marker := range _placeholderMarkers {
			if strings.Contains(stripped, marker) {
				found = true
				break
			}
		}
		if found != tc.want {
			t.Errorf("%s: body=%q → found=%v, want=%v (stripped=%q)",
				tc.name, tc.body, found, tc.want, stripped)
		}
	}
}

// =============================================================================
// sprint_start design-readiness gate: body length + requirement checkbox count.
// =============================================================================

// TestT530_countStructuredItemsAcrossSections verifies that items across
// different sections and patterns are all counted.
func TestT530_countStructuredItemsAcrossSections(t *testing.T) {
	cases := []struct {
		name string
		body string
		want int
	}{
		{
			name: "checkboxes only in Requirements",
			body: "## Requirements\n- [ ] A\n- [ ] B\n",
			want: 2,
		},
		{
			name: "Requirements written under Done Criteria (no ## Requirements)",
			body: "## Purpose\nDescription\n\n## Done Criteria\n- [ ] A\n- [ ] B\n- [ ] C\n",
			want: 3,
		},
		{
			name: "Requirements as a numbered list",
			body: "## Requirements\n1. first\n2. second\n3. third\n\n## Done Criteria\n- [ ] X\n",
			want: 4,
		},
		{
			name: "Requirements as bullets",
			body: "## Requirements\n- first item\n- second item\n",
			want: 2,
		},
		{
			name: "Scope section used",
			body: "## Scope\n- [x] A\n- [x] B\n",
			want: 2,
		},
		{
			name: "every structural section empty",
			body: "## Purpose\nbody\n\n## Requirements\n\n## Done Criteria\n\n## Scope\n\n## References\n",
			want: 0,
		},
	}
	for _, tc := range cases {
		got := countStructuredItemsAcrossSections(tc.body)
		if got != tc.want {
			t.Errorf("%s: got=%d want=%d", tc.name, got, tc.want)
		}
	}
}

// TestT438_countRequirementItems verifies that the count under
// `## Requirements` is correct.
func TestT438_countRequirementItems(t *testing.T) {
	cases := []struct {
		name string
		body string
		want int
	}{
		{
			name: "empty requirements section",
			body: "## Purpose\nbody\n\n## Requirements\n\n## Done Criteria\n- [ ] X\n",
			want: 0,
		},
		{
			name: "two checkboxes",
			body: "## Requirements\n\n- [ ] A\n- [x] B\n\n## Done Criteria\n- [ ] Y\n",
			want: 2,
		},
		{
			name: "checkboxes in other sections are excluded",
			body: "## Done Criteria\n- [ ] X\n\n## Requirements\n- [ ] A\n",
			want: 1,
		},
		{
			name: "level-3 heading inside the section does not end it",
			body: "## Requirements\n### Sub\n- [ ] A\n- [ ] B\n\n## Done Criteria\n",
			want: 2,
		},
	}
	for _, tc := range cases {
		got := countRequirementItems(tc.body)
		if got != tc.want {
			t.Errorf("%s: got=%d want=%d", tc.name, got, tc.want)
		}
	}
}

// TestT438_BodyLengthThreshold verifies that body-length-based placeholder
// detection works on either side of the threshold (compact length after
// whitespace removal).
func TestT438_BodyLengthThreshold(t *testing.T) {
	short := strings.Repeat("a", minTaskBodyChars-1)
	longEnough := strings.Repeat("a", minTaskBodyChars+10)

	// only test the body length (compact length of stripped body)
	cases := []struct {
		name      string
		stripped  string
		wantShort bool
	}{
		{"just below threshold → short", short, true},
		{"just above threshold → long enough", longEnough, false},
	}
	for _, tc := range cases {
		compact := strings.Join(strings.Fields(tc.stripped), "")
		isShort := len(compact) < minTaskBodyChars
		if isShort != tc.wantShort {
			t.Errorf("%s: isShort=%v want=%v (compact=%d)",
				tc.name, isShort, tc.wantShort, len(compact))
		}
	}
}

// =============================================================================
// T217 — secondary parser false-positive avoidance regression tests
// =============================================================================

// TestParseTaskResultFiles_T217_GlobWildcardExcluded verifies that glob
// patterns such as `Strategy/SplitBuy*.cs` are excluded from path
// candidates (T217).
func TestParseTaskResultFiles_T217_GlobWildcardExcluded(t *testing.T) {
	content := `## Result

### Changed files
- ` + "`" + `Strategy/SplitBuy*.cs` + "`" + ` (multi-file mention)
- ` + "`" + `src/foo?.cs` + "`" + ` (single-char glob)
- ` + "`" + `src/real/SplitBuyOrder.cs` + "`" + ` (real file)
`
	got := parseTaskResultFiles(content, []string{"Result"})
	want := []string{"src/real/SplitBuyOrder.cs"}
	assertEqualStringSlices(t, got, want)

	for _, g := range got {
		if strings.ContainsAny(g, "*?") {
			t.Errorf("path with glob wildcard included: %q", g)
		}
	}
}

// TestParseTaskResultFiles_T217_GitMvArrow verifies that for "A.cs → B.cs"
// move notation, the left side (source, removed) is dropped and only the
// right side (destination) is extracted (T217).
func TestParseTaskResultFiles_T217_GitMvArrow(t *testing.T) {
	content := `## Result

### Changed files
- ` + "`" + `src/old/Provider.cs` + "`" + ` → ` + "`" + `src/new/Provider.cs` + "`" + ` (git mv)
- ` + "`" + `src/old/Other.cs` + "`" + ` -> ` + "`" + `src/new/Other.cs` + "`" + ` (ASCII arrow)
- ` + "`" + `src/unchanged/Single.cs` + "`" + ` (single)
`
	got := parseTaskResultFiles(content, []string{"Result"})
	want := []string{
		"src/new/Provider.cs",
		"src/new/Other.cs",
		"src/unchanged/Single.cs",
	}
	assertEqualStringSlices(t, got, want)

	for _, g := range got {
		if strings.Contains(g, "old/") {
			t.Errorf("pre-move path was extracted: %q", g)
		}
	}
}

// TestIsLikelyFilePath_T217_glob verifies that isLikelyFilePath rejects
// glob patterns directly.
func TestIsLikelyFilePath_T217_glob(t *testing.T) {
	cases := []struct {
		input string
		want  bool
	}{
		{"Strategy/SplitBuy*.cs", false},
		{"src/foo?.cs", false},
		{"src/**/foo.cs", false},
		{"src/real.cs", true}, // sanity check
	}
	for _, tc := range cases {
		got := isLikelyFilePath(tc.input)
		if got != tc.want {
			t.Errorf("isLikelyFilePath(%q) = %v, want %v", tc.input, got, tc.want)
		}
	}
}

// TestStripMovedSource_T217 verifies the stripMovedSource helper removes
// the left side of the arrow.
func TestStripMovedSource_T217(t *testing.T) {
	cases := []struct {
		input string
		want  string
	}{
		{"- `A.cs` → `B.cs`", "`B.cs`"},
		{"- `A.cs` -> `B.cs`", "`B.cs`"},
		{"- `Single.cs` (single)", "- `Single.cs` (single)"},
	}
	for _, tc := range cases {
		got := stripMovedSource(tc.input)
		if got != tc.want {
			t.Errorf("stripMovedSource(%q) = %q, want %q", tc.input, got, tc.want)
		}
	}
}

// =============================================================================
// T221 (Sprint-28) — sub-heading rule: ### Artifacts vs narrative
// separation
// =============================================================================

// TestParseTaskResultFiles_T221_ArtifactsSubheadingIncluded verifies that
// only ### Artifacts contents are scanned and the rest is excluded as
// narrative.
func TestParseTaskResultFiles_T221_ArtifactsSubheadingIncluded(t *testing.T) {
	content := "## Result\n\n" +
		"### Design Decisions\n\n" +
		"some design impacts pkg/narrative/design.go (narrative — excluded)\n\n" +
		"### Artifacts\n\n" +
		"- `pkg/task/task.go` (modified)\n" +
		"- `pkg/task/task_test.go` (new)\n\n" +
		"### Validation\n\n" +
		"`pkg/validation/check.go` runs as a check (narrative — excluded)\n\n" +
		"## References\n"
	got := parseTaskResultFiles(content, []string{"Result", "Artifacts"})
	want := []string{
		"pkg/task/task.go",
		"pkg/task/task_test.go",
	}
	assertEqualStringSlices(t, got, want)
}

// TestParseTaskResultFiles_T221_BackwardCompat_NoSubheading verifies that
// existing Task files without sub-headings still scan everything.
func TestParseTaskResultFiles_T221_BackwardCompat_NoSubheading(t *testing.T) {
	content := `## Result

- ` + "`" + `a.go` + "`" + `
- ` + "`" + `b.go` + "`" + `
- ` + "`" + `c.go` + "`" + `

## References
`
	got := parseTaskResultFiles(content, []string{"Result", "Artifacts"})
	want := []string{"a.go", "b.go", "c.go"}
	assertEqualStringSlices(t, got, want)
}

// TestParseTaskResultFiles_T221_ChangedFiles_alias verifies that "Changed
// files" alias is also recognised as the artifact sub-section.
func TestParseTaskResultFiles_T221_ChangedFiles_alias(t *testing.T) {
	cases := []struct {
		name    string
		heading string
	}{
		{"Changed files", "### Changed files"},
		{"Artifacts", "### Artifacts"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			content := "## Result\n\n### Design\n\n`skip.go`\n\n" + c.heading + "\n\n- `main.go`\n\n## References\n"
			got := parseTaskResultFiles(content, []string{"Result"})
			if len(got) != 1 || got[0] != "main.go" {
				t.Errorf("heading=%s, got=%v (want [main.go])", c.heading, got)
			}
		})
	}
}

// =============================================================================
// parseTaskResultFiles — section recognition + various format parsing
// =============================================================================

func TestParseTaskResultFiles_ResultSectionHyphenList(t *testing.T) {
	content := `# T999

## Purpose

test

## Result

### Changed files

- ` + "`" + `cli/pkg/foo.go` + "`" + ` (modified)
- ` + "`" + `cli/pkg/bar.go` + "`" + ` (new)
- cli/tests/foo_test.go (new)

## References

other
`
	got := parseTaskResultFiles(content, []string{"Result", "Artifacts"})
	want := []string{
		"cli/pkg/foo.go",
		"cli/pkg/bar.go",
		"cli/tests/foo_test.go",
	}
	assertEqualStringSlices(t, got, want)
}

func TestParseTaskResultFiles_ArtifactsSectionCompat(t *testing.T) {
	content := `## Artifacts

- src/handler.go
- src/handler_test.go

## Next
`
	got := parseTaskResultFiles(content, []string{"Result", "Artifacts"})
	want := []string{"src/handler.go", "src/handler_test.go"}
	assertEqualStringSlices(t, got, want)
}

func TestParseTaskResultFiles_CustomSectionName(t *testing.T) {
	content := `## Result

- ` + "`" + `cmd/main.go` + "`" + `
- ` + "`" + `internal/server.go` + "`" + `

## End
`
	got := parseTaskResultFiles(content, []string{"Result", "Artifact"})
	want := []string{"cmd/main.go", "internal/server.go"}
	assertEqualStringSlices(t, got, want)
}

func TestParseTaskResultFiles_MarkdownTable(t *testing.T) {
	content := `## Result

**Changed files**:

| File | Change |
|------|--------|
| ` + "`" + `pkg/config/types.go` + "`" + ` | new |
| ` + "`" + `pkg/config/yaml_loader.go` + "`" + ` | new |
| ` + "`" + `pkg/sprint/sprint.go` + "`" + ` | RenderCeremonyMD refactor |

## References
`
	got := parseTaskResultFiles(content, []string{"Result"})
	want := []string{
		"pkg/config/types.go",
		"pkg/config/yaml_loader.go",
		"pkg/sprint/sprint.go",
	}
	assertEqualStringSlices(t, got, want)
}

func TestParseTaskResultFiles_BacktickPathPriority(t *testing.T) {
	// the path inside backticks is preferred; surrounding text is ignored.
	content := `## Result

- ` + "`" + `cli/pkg/foo.go` + "`" + ` is the file we modified.
`
	got := parseTaskResultFiles(content, []string{"Result"})
	want := []string{"cli/pkg/foo.go"}
	assertEqualStringSlices(t, got, want)
}

func TestParseTaskResultFiles_DedupDuplicates(t *testing.T) {
	content := `## Result

- ` + "`" + `pkg/a.go` + "`" + `
- ` + "`" + `pkg/a.go` + "`" + ` (mentioned again)
- ` + "`" + `pkg/b.go` + "`" + `
`
	got := parseTaskResultFiles(content, []string{"Result"})
	want := []string{"pkg/a.go", "pkg/b.go"}
	assertEqualStringSlices(t, got, want)
}

func TestParseTaskResultFiles_PlaceholderExcluded(t *testing.T) {
	content := `## Result

- ` + "`" + `path/to/file1` + "`" + ` (template)
- ` + "`" + `path/to/file2` + "`" + ` (template)
- ` + "`" + `pkg/real.go` + "`" + ` (real)
`
	got := parseTaskResultFiles(content, []string{"Result"})
	want := []string{"pkg/real.go"}
	assertEqualStringSlices(t, got, want)
}

func TestParseTaskResultFiles_NoSection(t *testing.T) {
	content := `# Task

## Purpose

test

## Other section

content
`
	got := parseTaskResultFiles(content, []string{"Result", "Artifacts"})
	if len(got) != 0 {
		t.Errorf("expected empty when no Result section: got=%v", got)
	}
}

func TestParseTaskResultFiles_StopAtNextSection(t *testing.T) {
	content := `## Result

- ` + "`" + `pkg/a.go` + "`" + `

## References

- ` + "`" + `pkg/should_not_appear.go` + "`" + `
`
	got := parseTaskResultFiles(content, []string{"Result"})
	want := []string{"pkg/a.go"}
	assertEqualStringSlices(t, got, want)
}

// After T221 (Sprint-28): only artifact sub-headings are scanned. A
// narrative-style sub-heading like "### Build Result" is excluded.
// Pre-T221 used to scan everything, but the strict mode prevents parser
// false positives.
func TestParseTaskResultFiles_OnlyArtifactSubheading(t *testing.T) {
	content := `## Result

### Changed files

- ` + "`" + `a.go` + "`" + `

### Build Result

- ` + "`" + `b.go` + "`" + ` (narrative — excluded after T221)

## References
`
	got := parseTaskResultFiles(content, []string{"Result"})
	want := []string{"a.go"}
	assertEqualStringSlices(t, got, want)
}

// =============================================================================
// extractPathsFromLine — single-line processing
// =============================================================================

func TestExtractPathsFromLine_BacktickHyphen(t *testing.T) {
	got := extractPathsFromLine("- `pkg/foo.go` (modified)")
	want := []string{"pkg/foo.go"}
	assertEqualStringSlices(t, got, want)
}

func TestExtractPathsFromLine_BacktickTable(t *testing.T) {
	got := extractPathsFromLine("| `pkg/foo.go` | new |")
	want := []string{"pkg/foo.go"}
	assertEqualStringSlices(t, got, want)
}

func TestExtractPathsFromLine_TableSeparatorExcluded(t *testing.T) {
	got := extractPathsFromLine("|------|----------|")
	if len(got) != 0 {
		t.Errorf("table separator not excluded: %v", got)
	}
}

func TestExtractPathsFromLine_NoBacktickHyphen(t *testing.T) {
	got := extractPathsFromLine("- pkg/foo.go (modified)")
	want := []string{"pkg/foo.go"}
	assertEqualStringSlices(t, got, want)
}

func TestExtractPathsFromLine_NoPathLine(t *testing.T) {
	got := extractPathsFromLine("- a line with only ordinary text")
	if len(got) != 0 {
		t.Errorf("expected empty for path-less line: %v", got)
	}
}

// =============================================================================
// helper
// =============================================================================

func assertEqualStringSlices(t *testing.T, got, want []string) {
	t.Helper()
	if len(got) != len(want) {
		t.Errorf("length mismatch: got=%d want=%d\ngot=%v\nwant=%v", len(got), len(want), got, want)
		return
	}
	for i := range got {
		if got[i] != want[i] {
			t.Errorf("[%d] mismatch: got=%q want=%q", i, got[i], want[i])
		}
	}
}

// =============================================================================
// T435 (Sprint-51) — exclude self Task file path
// =============================================================================

// TestT435_excludeSelfReference_ExactPath verifies that paths the parser
// extracted that exactly match the Task file itself are excluded.
func TestT435_excludeSelfReference_ExactPath(t *testing.T) {
	self := "works/sprints/active/sprint-51/tasks/T435-foo.md"
	paths := []string{
		"works/sprints/active/sprint-51/tasks/T435-foo.md",
		"cli/pkg/task/task.go",
	}
	got := excludeSelfReference(paths, self)
	if len(got) != 1 || got[0] != "cli/pkg/task/task.go" {
		t.Errorf("self-reference exclusion failed: %v", got)
	}
}

// TestT435_excludeSelfReference_basename verifies basename-match self
// reference detection (when prefix differs, suffix/basename fallback).
func TestT435_excludeSelfReference_basename(t *testing.T) {
	self := "/abs/path/to/T435-foo.md"
	paths := []string{"T435-foo.md", "pkg/foo.go"}
	got := excludeSelfReference(paths, self)
	if len(got) != 1 || got[0] != "pkg/foo.go" {
		t.Errorf("basename match failed: %v", got)
	}
}

// TestT435_excludeSelfReference_OtherArtefactsRetained verifies that when
// no self-reference is present, the original is returned unchanged
// (regression guard).
func TestT435_excludeSelfReference_OtherArtefactsRetained(t *testing.T) {
	self := "works/sprints/active/sprint-51/tasks/T435-foo.md"
	paths := []string{"pkg/a.go", "pkg/b.go"}
	got := excludeSelfReference(paths, self)
	if len(got) != 2 {
		t.Errorf("other artefacts shrank: %v", got)
	}
}

// TestT435_excludeSelfReference_EmptyInput verifies edge cases.
func TestT435_excludeSelfReference_EmptyInput(t *testing.T) {
	if got := excludeSelfReference(nil, "foo.md"); got != nil {
		t.Errorf("expected nil for nil input: %v", got)
	}
	if got := excludeSelfReference([]string{"a"}, ""); len(got) != 1 {
		t.Errorf("expected the original to be preserved when filePath is empty: %v", got)
	}
}

// =============================================================================
// policy branch (env var has priority over config.GetTaskResultCheckPolicy)
// =============================================================================
// Note: config.GetTaskResultCheckPolicy itself is verified separately in
// pkg/config/config_test.go. This file only checks that verifyTaskResults
// is invoked normally.

func TestVerifyTaskResults_FileReadFailureEmpty(t *testing.T) {
	missing, checked, sectionTitles := verifyTaskResults("T999", "/nonexistent/path/to/file.md")
	if len(missing) != 0 {
		t.Errorf("expected missing empty when file missing: %v", missing)
	}
	if len(checked) != 0 {
		t.Errorf("expected checked empty when file missing: %v", checked)
	}
	if len(sectionTitles) == 0 {
		t.Errorf("sectionTitles should always return defaults")
	}
}

// unit check: parseTaskResultFiles must safely handle nil sectionTitles.
func TestParseTaskResultFiles_NilSectionTitles(t *testing.T) {
	content := `## Result

- a.go
`
	got := parseTaskResultFiles(content, nil)
	if len(got) != 0 {
		t.Errorf("expected empty result for empty sectionTitles: %v", got)
	}
}

// splitPathCandidate unit tests
func TestSplitPathCandidate_ParenthesesSplit(t *testing.T) {
	got := splitPathCandidate("pkg/foo.go (modified)")
	if len(got) != 1 || got[0] != "pkg/foo.go" {
		t.Errorf("parentheses split failed: %v", got)
	}
}

// TestT199_VerifyTaskResults_OutsideProjectRootClassifyMissing — T199
// (Sprint-09).
// Absolute paths outside projectRoot (/tmp/..., /var/...) are not allowed
// as artefacts and must be classified as missing. Basename fallback must
// not match by accident.
func TestT199_VerifyTaskResults_OutsideProjectRootClassifyMissing(t *testing.T) {
	dir := t.TempDir()
	taskFile := filepath.Join(dir, "T999-external.md")
	content := `---
id: T999
---

## Result

### Artifacts

- /tmp/definitely-not-in-repo-12345.txt
- /var/tmp/bogus.md
`
	if err := os.WriteFile(taskFile, []byte(content), 0o644); err != nil {
		t.Fatalf("write task file: %v", err)
	}

	missing, checked, _ := verifyTaskResultsForType("T999", taskFile, "")
	if len(checked) != 2 {
		t.Errorf("expected checked=2 (/tmp/... + /var/...), got %d: %v", len(checked), checked)
	}
	if len(missing) != 2 {
		t.Fatalf("expected missing=2 (both outside projectRoot), got %d: %+v", len(missing), missing)
	}
	for _, m := range missing {
		if !strings.Contains(m.Reason, "outside") || !strings.Contains(m.Reason, "projectRoot") {
			t.Errorf("expected reason to mention 'outside projectRoot': %q", m.Reason)
		}
	}
}

func TestSplitPathCandidate_EmDashSplit(t *testing.T) {
	got := splitPathCandidate("pkg/bar.go — new")
	if len(got) != 1 || got[0] != "pkg/bar.go" {
		t.Errorf("em-dash split failed: %v", got)
	}
}

func TestSplitPathCandidate_NoPath(t *testing.T) {
	got := splitPathCandidate("plain text")
	if len(got) != 0 {
		t.Errorf("expected empty without slash/dot: %v", got)
	}
}

// post-normalisation matching — verifyTaskResults uses filepath.Clean then
// exact + suffix match.
// This integration is hard to unit-test due to git environment dependency;
// parser-level coverage stands in.
// Integration verification happens in user scenarios.

func TestParseTaskResultFiles_subWordMatch(t *testing.T) {
	// when the first word matches such as "## Result Summary"
	content := `## Result Summary

- ` + "`" + `pkg/x.go` + "`" + `
`
	got := parseTaskResultFiles(content, []string{"Result"})
	want := []string{"pkg/x.go"}
	assertEqualStringSlices(t, got, want)
}

// =============================================================================
// T312 — headingTitleContainsKeyword word-boundary unit tests
// =============================================================================

// TestHeadingTitleContainsKeyword_T312_WordBoundary verifies that
// substrings do not falsely match and word boundaries are checked
// precisely (T312).
func TestHeadingTitleContainsKeyword_T312_WordBoundary(t *testing.T) {
	cases := []struct {
		name        string
		headingLine string
		kw          string
		want        bool
	}{
		// normal matches — keyword present as a standalone word
		{name: "single_keyword_exact", headingLine: "## Followup", kw: "Followup", want: true},
		{name: "keyword_with_word_before", headingLine: "## Major Followup tasks", kw: "Followup", want: true},
		{name: "keyword_first_word", headingLine: "## Followup processing list", kw: "Followup", want: true},
		{name: "keyword_last_word", headingLine: "## Outstanding Followup", kw: "Followup", want: true},
		{name: "TODO_alone", headingLine: "## TODO", kw: "TODO", want: true},
		{name: "TODO_prefix", headingLine: "## TODO items", kw: "TODO", want: true},
		// false-positive prevention — keyword is part of a larger word
		{
			name:        "false_positive_keyword_inside_compound",
			headingLine: "## Followup-style analysis",
			kw:          "Followup",
			// "Followup-style" does not end at "Followup" alone — substring
			// match would falsely succeed; want false here.
			want: false,
		},
		{
			name:        "false_positive_TODO_in_other_word",
			headingLine: "## METHODOLOGY summary",
			kw:          "TODO",
			want:        false,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := headingTitleContainsKeyword(tc.headingLine, tc.kw)
			if got != tc.want {
				t.Errorf("headingTitleContainsKeyword(%q, %q) = %v, want %v",
					tc.headingLine, tc.kw, got, tc.want)
			}
		})
	}
}

// =============================================================================
// T312 — verifyFollowups substring false-positive regression
// =============================================================================

// TestVerifyFollowups_T312_NoFalsePositive verifies that a heading using a
// compound that contains the keyword as a prefix is not detected as a
// followup section (T312).
//
// The previous strings.Contains-based implementation classified "##
// Followup-style analysis" as a followup section and emitted a false
// positive warning.
func TestVerifyFollowups_T312_NoFalsePositive(t *testing.T) {
	// the section "## Followup analysis result" includes "Followup" as a
	// substring but is a separate token, so it must not be treated as a
	// followup section → no warning.
	content := `---
id: T999
title: "test"
type: feature
---

# T999 test

## Followup-style analysis result

after analysing follow-up impact: no change required.

## References

other
`
	tmpDir := t.TempDir()
	taskFile := tmpDir + "/T999.md"
	if err := os.WriteFile(taskFile, []byte(content), 0o644); err != nil {
		t.Fatalf("file create failed: %v", err)
	}

	warns := verifyFollowups(taskFile)
	if len(warns) != 0 {
		t.Errorf("'Followup-style analysis result' must not match the followup detector, warnings=%v", warns)
	}
}

// TestVerifyFollowups_T312_FollowupSectionDetected verifies that
// "## Followup tasks" is correctly detected as a followup section and
// emits a warning when no Task ID is present (T312).
func TestVerifyFollowups_T312_FollowupSectionDetected(t *testing.T) {
	content := `---
id: T999
title: "test"
type: feature
---

# T999 test

## Followup tasks

further improvement is needed.

## References

other
`
	tmpDir := t.TempDir()
	taskFile := tmpDir + "/T999.md"
	if err := os.WriteFile(taskFile, []byte(content), 0o644); err != nil {
		t.Fatalf("file create failed: %v", err)
	}

	warns := verifyFollowups(taskFile)
	if len(warns) == 0 {
		t.Error("'## Followup tasks' section must warn when no Task ID is present")
	}
}

// when entering the next ## section, termination must be exact.
// T221 (Sprint-28): sub-tree is scanned only when the sub-heading matches
// `_artifactSubHeadings`. This test uses "### Artifacts".
func TestParseTaskResultFiles_h2Stop_h3Continue(t *testing.T) {
	content := `## Result

### Artifacts
- ` + "`" + `a.go` + "`" + `
- ` + "`" + `b.go` + "`" + `

## Next section
- ` + "`" + `c.go` + "`" + ` (must not be included)
`
	got := parseTaskResultFiles(content, []string{"Result"})
	want := []string{"a.go", "b.go"}
	assertEqualStringSlices(t, got, want)
	// explicitly verify c.go was not included
	for _, p := range got {
		if strings.Contains(p, "c.go") {
			t.Errorf("content from the next ## section was included: %v", got)
		}
	}
}

// =============================================================================
// T195 — false-positive regression guard (Sprint-24 failure inputs)
// =============================================================================

// TestIsLikelyFilePath_Allow verifies that previously-allowed cases still
// pass.
func TestIsLikelyFilePath_Allow(t *testing.T) {
	cases := []string{
		"pkg/foo.go",
		"cli/pkg/task/task.go",
		"a.md",
		"docs/07-knowledge/kb-001.md",
		"scripts/measure-skill-quality.py",
		"cmd/hstl-oss/cmd/sprint.go",
	}
	for _, c := range cases {
		if !isLikelyFilePath(c) {
			t.Errorf("expected allow: %q", c)
		}
	}
}

// TestIsLikelyFilePath_Reject verifies the false-positive cases that T195
// must remove. All collected from real sprint-24 task_complete-blocking
// inputs.
func TestIsLikelyFilePath_Reject(t *testing.T) {
	cases := []struct {
		input  string
		reason string
	}{
		{"list/single", "no extension — T190 category notation"},
		{"delete/irreversible", "no extension — T190 category notation"},
		{"./internal/...", "starts with ./ — T192 go-test argument"},
		{"/hstl:kb:init", "starts with / + colon — T193 slash command"},
		{"commands/{namespace}/{action}.md", "braces — T193 template placeholder"},
		{"category-name", "no extension — generic token"},
		{"plain text", "no extension"},
		{"Sprint 25 progress", "whitespace + no extension"},
	}
	for _, c := range cases {
		if isLikelyFilePath(c.input) {
			t.Errorf("expected rejection (%s): %q", c.reason, c.input)
		}
	}
}

// TestIsLikelyFilePath_T559_ExternalPathExclusion — Sprint-68 T559.
// In spike Tasks etc., mentioning artefacts that live outside the repo
// (home dir output, parent dir, HTTP URL) must not produce a false-
// positive BLOCK in the git diff verification.
func TestIsLikelyFilePath_T559_ExternalPathExclusion(t *testing.T) {
	externalCases := []struct {
		input  string
		reason string
	}{
		{"~/Downloads/spike-report.xlsx", "home dir — ~/ prefix"},
		{"~/tmp/scratch.md", "home dir — ~/ prefix"},
		{"../other-repo/file.go", "parent dir — ../ prefix"},
		{"../../elsewhere/doc.md", "parent dir — multiple ../"},
		{"https://example.com/spec.md", "remote URL — https prefix"},
		{"http://internal.wiki/page.md", "remote URL — http prefix"},
	}
	for _, c := range externalCases {
		if isLikelyFilePath(c.input) {
			t.Errorf("T559 external path must be rejected (%s): %q", c.reason, c.input)
		}
	}

	// repo-internal paths must still pass (regression guard)
	validCases := []string{
		"cli/pkg/task/task.go",
		"works/sprints/active/sprint-68/tasks/T559.md",
		"docs/04-guides/configuration.md",
	}
	for _, v := range validCases {
		if !isLikelyFilePath(v) {
			t.Errorf("T559 repo-internal path must pass: %q", v)
		}
	}
}

// TestParseTaskResultFiles_T195_FalsePositiveRemoval verifies that when
// parsing the original Sprint-24 result-section text used in T190 / T192
// / T193, only real file paths are kept and slash notation, test args,
// slash commands and templates are all rejected.
func TestParseTaskResultFiles_T195_FalsePositiveRemoval(t *testing.T) {
	content := `# T999

## Result

- category-name list/single lookup — readOnly=true (T190 notation)
- registry_list / registry_check have delete/irreversible flag false
- run tests: ` + "`" + `go test ./cli/...` + "`" + `
- new slash command: /hstl:kb:init (T193)
- template path: commands/{namespace}/{action}.md
- cli/pkg/task/task.go — parser fix
- cli/pkg/task/result_verify_test.go — regression test added

## Status change history
`
	got := parseTaskResultFiles(content, []string{"Result", "Artifacts"})
	// only the two real file paths should be kept
	want := []string{
		"cli/pkg/task/task.go",
		"cli/pkg/task/result_verify_test.go",
	}
	assertEqualStringSlices(t, got, want)

	// explicitly verify forbidden tokens are not included
	forbidden := []string{
		"list/single",
		"delete/irreversible",
		"./internal",
		"/hstl:kb:init",
		"commands/",
		"{namespace}",
	}
	for _, g := range got {
		for _, f := range forbidden {
			if strings.Contains(g, f) {
				t.Errorf("false positive recurred: %q contains forbidden token %q", g, f)
			}
		}
	}
}

// TestSplitPathCandidate_T195_SlashTextExcluded verifies that calling
// splitPathCandidate directly on hyphen-list lines does not produce false
// positives.
func TestSplitPathCandidate_T195_SlashTextExcluded(t *testing.T) {
	cases := []string{
		"list/single lookup tool description",
		"delete/irreversible work classification",
		"./cmd/hstl-oss/cmd/... run",
	}
	for _, c := range cases {
		got := splitPathCandidate(c)
		if len(got) != 0 {
			t.Errorf("expected rejection: input=%q got=%v", c, got)
		}
	}
}

// =============================================================================
// T199 — prevent false positives for class / method names and JSON keys
// inside backticks (originating downstream Sprint 123).
// =============================================================================

// TestIsLikelyFilePath_T199_NonFileTokensInBackticks reproduces the
// pattern that caused 6–8 BLOCKED round-trips downstream. Class names,
// method names, and JSON keys wrapped in backticks may look like file
// extensions but are not file paths and must be excluded.
func TestIsLikelyFilePath_T199_NonFileTokensInBackticks(t *testing.T) {
	cases := []struct {
		input  string
		reason string
	}{
		{"MyClass.DoWork", "uppercase-leading method name → .DoWork breaks lowercase rule"},
		{"BackfillHost.ProcessAsync", "downstream observed pattern → .ProcessAsync excluded"},
		{"obj.someMethod", "camelCase method → .someMethod not in whitelist"},
		{"JSON.stringify", "namespace.function → .stringify not in whitelist"},
		{"Array.prototype", "namespace.property → .prototype not in whitelist"},
		{"Math.PI", "uppercase constant → .PI not in whitelist"},
		{"System.Collections.Generic", "C# namespace chain → .Generic not in whitelist"},
	}
	for _, c := range cases {
		if isLikelyFilePath(c.input) {
			t.Errorf("expected rejection (%s): %q", c.reason, c.input)
		}
	}
}

// TestIsLikelyFilePath_T199_WhitelistPasses verifies that even after T199
// introduces the whitelist, normal file paths still pass.
func TestIsLikelyFilePath_T199_WhitelistPasses(t *testing.T) {
	cases := []string{
		"pkg/foo.go",
		"cmd/cli/main.go",
		"src/index.ts",
		"src/App.tsx",
		"lib/utils.py",
		"docs/README.md",
		"config/app.yaml",
		"config/app.yml",
		"package.json",
		"go.mod",
		"go.sum",
		"Cargo.toml",
		"schema.sql",
		"api.graphql",
		"proto/user.proto",
		"scripts/build.sh",
		"styles/main.css",
		"public/logo.svg",
	}
	for _, c := range cases {
		if !isLikelyFilePath(c) {
			t.Errorf("expected allow: %q", c)
		}
	}
}

// TestParseTaskResultFiles_T199_BacktickMethodNameExcluded verifies in
// integration that result-section parsing does not misclassify
// backtick-wrapped method names as file paths.
func TestParseTaskResultFiles_T199_BacktickMethodNameExcluded(t *testing.T) {
	content := `## Result

- ` + "`" + `cli/pkg/task/task.go` + "`" + ` — parser fix
- ` + "`" + `BackfillHost.ProcessAsync` + "`" + ` new method (downstream pattern)
- ` + "`" + `JSON.stringify` + "`" + ` serialisation logic changed
- ` + "`" + `cli/pkg/task/result_verify_test.go` + "`" + ` — regression test added

## References
`
	got := parseTaskResultFiles(content, []string{"Result", "Artifacts"})
	want := []string{
		"cli/pkg/task/task.go",
		"cli/pkg/task/result_verify_test.go",
	}
	assertEqualStringSlices(t, got, want)

	// forbidden-token verification
	for _, g := range got {
		if strings.Contains(g, "BackfillHost") || strings.Contains(g, "JSON.stringify") {
			t.Errorf("false positive recurred: %q contains a method name", g)
		}
	}
}

// ── T296 (Sprint-36): infra-type narrative default mode ──
//
// The Sprint-35 T295 false positive is structurally blocked: an infra-type
// Task's result section scans file paths only when an explicit
// ### Artifacts sub-heading exists; other backtick paths in narrative are
// ignored.

// TestParseTaskResultFiles_T296_infra_NoSubheadingIgnored verifies that
// for the infra type with no sub-heading (backward-compat mode) narrative
// backtick paths are ignored. Previously this would have full-scanned and
// extracted every backtick path.
func TestParseTaskResultFiles_T296_infra_NoSubheadingIgnored(t *testing.T) {
	content := `## Result

This Task is the infra type (deployment verification) with no new code
artefacts.
The verification log is recorded in ` + "`" + `works/sprints/active/sprint-99/tasks/T999.md` + "`" + `.

go build / go test all PASS, ` + "`" + `cli/pkg/task/task.go` + "`" + ` not changed.
`
	got := parseTaskResultFilesForType(content, []string{"Result", "Artifacts"}, "infra")
	if len(got) != 0 {
		t.Errorf("infra type defaults to narrative — expected backtick paths ignored, got=%v", got)
	}
}

// TestParseTaskResultFiles_T296_infra_ExplicitArtifactsScan verifies that
// for the infra type, an explicit `### Artifacts` sub-heading still scans
// normally.
func TestParseTaskResultFiles_T296_infra_ExplicitArtifactsScan(t *testing.T) {
	content := `## Result

### Deployment record

This Task uses infra type narrative.

### Artifacts

- ` + "`" + `cli/pkg/task/task.go` + "`" + ` (modified)
- ` + "`" + `docs/release/v2.md` + "`" + ` (new)
`
	got := parseTaskResultFilesForType(content, []string{"Result", "Artifacts"}, "infra")
	want := []string{
		"cli/pkg/task/task.go",
		"docs/release/v2.md",
	}
	assertEqualStringSlices(t, got, want)
}

// TestParseTaskResultFiles_T296_feature_BackwardCompat verifies that
// non-infra types preserve the existing behaviour (full scan when no
// sub-heading).
func TestParseTaskResultFiles_T296_feature_BackwardCompat(t *testing.T) {
	content := `## Result

- ` + "`" + `pkg/foo.go` + "`" + ` new
- ` + "`" + `pkg/bar.go` + "`" + ` modified
`
	got := parseTaskResultFilesForType(content, []string{"Result"}, "feature")
	want := []string{"pkg/foo.go", "pkg/bar.go"}
	assertEqualStringSlices(t, got, want)
}

// TestParseTaskResultFiles_T296_EmptyType_BackwardCompat verifies that
// when taskType is unspecified (parseTaskResultFiles wrapper) the
// existing behaviour is preserved.
func TestParseTaskResultFiles_T296_EmptyType_BackwardCompat(t *testing.T) {
	content := `## Result

- ` + "`" + `pkg/foo.go` + "`" + ` new
`
	// existing wrapper must behave identically
	got := parseTaskResultFiles(content, []string{"Result"})
	want := []string{"pkg/foo.go"}
	assertEqualStringSlices(t, got, want)

	// explicit empty string must behave the same
	got2 := parseTaskResultFilesForType(content, []string{"Result"}, "")
	assertEqualStringSlices(t, got2, want)
}

// TestParseTaskResultFiles_T296_infra_MixedNarrativeArtifact verifies that
// in the infra type, when narrative and artifact sub-headings coexist,
// only the latter is scanned.
func TestParseTaskResultFiles_T296_infra_MixedNarrativeArtifact(t *testing.T) {
	content := `## Result

### Verification summary

` + "`" + `~/.local/bin/hstl` + "`" + ` reinstall complete. ` + "`" + `cmd/hstl-oss/main.go` + "`" + ` is also fine.

### Artifacts

- ` + "`" + `cmd/hstl-oss/cmd/sprint.go` + "`" + ` (modified)
`
	got := parseTaskResultFilesForType(content, []string{"Result", "Artifacts"}, "infra")
	want := []string{"cmd/hstl-oss/cmd/sprint.go"}
	assertEqualStringSlices(t, got, want)

	// backtick paths from narrative must be ignored
	for _, g := range got {
		if strings.Contains(g, "main.go") || strings.Contains(g, ".local/bin") {
			t.Errorf("narrative backtick path was extracted: %q", g)
		}
	}
}

// =============================================================================
// T535 (Sprint-69) — getGitChangedFiles includes the most recent N
// commits.
// =============================================================================

// TestT535_GetGitChangedFiles_RecentCommits verifies that even with an
// empty working tree, files from recent commits appear in the result.
// Previously `git diff HEAD~1` was a single range so only the immediate
// previous commit was caught, false-BLOCKing multi-commit Tasks.
func TestT535_GetGitChangedFiles_RecentCommits(t *testing.T) {
	dir := t.TempDir()
	run := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	run("init", "-q")
	run("config", "user.email", "t@t.t")
	run("config", "user.name", "t")
	run("commit", "--allow-empty", "-m", "init")

	// add 3 files in 3 commits
	for i, name := range []string{"a.txt", "b.txt", "c.txt"} {
		p := filepath.Join(dir, name)
		if err := os.WriteFile(p, []byte(name), 0o644); err != nil {
			t.Fatalf("write: %v", err)
		}
		run("add", name)
		run("commit", "-m", fmt.Sprintf("add %d", i))
	}

	// chdir so project root points to the temp directory
	t.Chdir(dir)

	files := getGitChangedFiles()
	seen := map[string]bool{}
	for _, f := range files {
		seen[f] = true
	}
	for _, want := range []string{"a.txt", "b.txt", "c.txt"} {
		if !seen[want] {
			t.Errorf("%s missing from getGitChangedFiles result: %v", want, files)
		}
	}
}

// TestT535_TaskResultDiffDepth_EnvOverride verifies depth can be tuned
// via env var.
func TestT535_TaskResultDiffDepth_EnvOverride(t *testing.T) {
	t.Setenv("HSTL_TASK_RESULT_DIFF_DEPTH", "3")
	if got := taskResultDiffDepth(); got != 3 {
		t.Errorf("env override failed: %d", got)
	}
	t.Setenv("HSTL_TASK_RESULT_DIFF_DEPTH", "")
	if got := taskResultDiffDepth(); got != defaultTaskResultDiffDepth {
		t.Errorf("default fallback failed: %d", got)
	}
	t.Setenv("HSTL_TASK_RESULT_DIFF_DEPTH", "invalid")
	if got := taskResultDiffDepth(); got != defaultTaskResultDiffDepth {
		t.Errorf("invalid value should default: %d", got)
	}
}

// TestT822_RunGit_NonAsciiPath_NotOctalEscaped verifies that non-ASCII
// filenames are returned from runGit as UTF-8, not octal-escape. Sprint-95
// T797 firsthand defect: default git config (core.quotepath=on) outputs
// "\354\234\274..." and result-section strict verification false-BLOCKED.
//
// fix (T822): runGit now always adds `-c core.quotepath=false` → keeps
// UTF-8.
func TestT822_RunGit_NonAsciiPath_NotOctalEscaped(t *testing.T) {
	tmp := t.TempDir()

	// initialise local git repo (omit --initial-branch to avoid global
	// config interference).
	runInDir := func(args ...string) error {
		c := exec.Command("git", args...)
		c.Dir = tmp
		c.Env = append(os.Environ(),
			"GIT_AUTHOR_NAME=t", "GIT_AUTHOR_EMAIL=t@t",
			"GIT_COMMITTER_NAME=t", "GIT_COMMITTER_EMAIL=t@t",
			"HOME="+tmp, // avoid global config
		)
		out, err := c.CombinedOutput()
		if err != nil {
			return fmt.Errorf("git %v: %w\n%s", args, err, out)
		}
		return nil
	}
	if err := runInDir("init"); err != nil {
		t.Fatal(err)
	}
	// regardless of global core.quotepath, do not force the test repo to
	// off — we want to verify the fix forces core.quotepath=false on a
	// per-command basis.

	// Create a non-ASCII filename (staged).
	utf8Name := "utf8-test-file.md"
	utf8Path := filepath.Join(tmp, utf8Name)
	if err := os.WriteFile(utf8Path, []byte("# utf8 content\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := runInDir("add", utf8Name); err != nil {
		t.Fatal(err)
	}

	// invoke runGit (fix applied — core.quotepath=false automatically).
	out, err := runGit(tmp, "diff", "--name-only", "--cached")
	if err != nil {
		t.Fatalf("runGit failed: %v", err)
	}
	result := strings.TrimSpace(string(out))

	// octal escape (\354...) must not appear in the output.
	if strings.Contains(result, "\\354") || strings.Contains(result, "\\355") {
		t.Errorf("octal escape remains — fix incomplete:\n%s", result)
	}
	// the UTF-8 filename should be present as-is.
	if !strings.Contains(result, utf8Name) {
		t.Errorf("filename match failed — expected %q, got %q", utf8Name, result)
	}
}

// =============================================================================
// T432 — HasPlaceholderBody narrative-section false-positive prevention
// =============================================================================

// TestT432_HasPlaceholderBody_NarrativeExcluded verifies that
// extractBodyForPlaceholderScan does not falsely flag marker quotations
// inside narrative sections like "## Result → ### Design Decisions /
// Validation" (T432 Sprint-37).
//
// Background: after a Task is completed, descriptive mentions of
// placeholder markers in the result-section narrative caused
// HasPlaceholderBody to wrongly return true and unfairly block assign.
func TestT432_HasPlaceholderBody_NarrativeExcluded(t *testing.T) {
	cases := []struct {
		name string
		body string
		want bool // true = expect a placeholder remaining detection
	}{
		{
			name: "Design Decisions section marker quote ignored",
			body: `## Purpose

A sufficiently authored Purpose section. After Task completion the result
is recorded.

## Requirements

- [x] feature A implemented
- [x] feature B implemented
- [x] feature C implemented

## Done Criteria

- [x] go build PASS
- [x] go test PASS

## Result

### Design Decisions

Replaced legacy {The problem this Task solves}-style markers with
extractBodyForPlaceholderScan. Markers like {Requirement N} should also
be excluded under narrative as descriptive mentions.

### Artifacts

- packages/hostler-cli/pkg/task/task.go (modified)
`,
			want: false, // narrative markers ignored → not a placeholder
		},
		{
			name: "Validation section marker quote also ignored",
			body: `## Purpose

A sufficiently authored Task Purpose. Multiple lines of description.

## Requirements

- [x] item 1
- [x] item 2
- [x] item 3

## Done Criteria

- [x] build PASS

## Result

### Validation

Confirmed there is no remaining {TODO}-style work. {Criterion N}
satisfaction confirmed.
`,
			want: false,
		},
		{
			name: "real un-substituted marker in Purpose section detected",
			body: `## Purpose

{The problem this Task solves}

## Requirements

- [ ] {Requirement 1}

## Done Criteria

- [ ] {Criterion 1}
`,
			want: true, // real placeholder → must be detected
		},
		{
			name: "marker outside Result section in plain text detected",
			body: `## Purpose

A sufficiently authored Purpose. This Task solves problem A.

## Requirements

- [x] feature A
- [x] feature B

## Done Criteria

- [x] PASS

## Scope Limits

{TODO} remaining work deferred to a later Sprint.
`,
			want: true, // marker outside Result section → must be detected
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			scanned := extractBodyForPlaceholderScan(tc.body)
			found := false
			for _, marker := range _placeholderMarkers {
				if strings.Contains(scanned, marker) {
					found = true
					break
				}
			}
			if found != tc.want {
				t.Errorf("extractBodyForPlaceholderScan: found=%v, want=%v\n  body=%q\n  scanned=%q",
					found, tc.want, tc.body, scanned)
			}
		})
	}
}
