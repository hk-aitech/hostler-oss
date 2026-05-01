// Package ceremony — T422 (Sprint-36) regression tests for the
// narrative/artifact separation rule of the placeholder-marker scanner.
//
// Goal: ensure the doc-review scanner never produces a false-positive
// (recursive BLOCK) when it quotes a marker literal under a narrative
// sub-heading (Design Decisions / Validation / Commits / Follow-up).
// Marker detection in the core sections (Purpose / Requirements / Done
// Criteria) and in the artefact section must still fire.
package ceremony

import (
	"strings"
	"testing"
)

// Markers quoted under a narrative sub-heading must not be included in
// the scan output.
func TestT422_PlaceholderScan_Narrative_Allows_MarkerLiteral(t *testing.T) {
	content := `# T999 Example

## Purpose
normal purpose description.

## Result

### Design Decisions

example of marker literal quotation: "{The problem this Task", "{TODO",
"{One-line summary" mentioned for descriptive purposes.

### Validation
"{Requirement " is also mentioned as a marker example.
`
	scanned := ExtractPlaceholderScanContent(content)
	for _, marker := range []string{"{The problem this Task", "{TODO", "{One-line summary", "{Requirement "} {
		if strings.Contains(scanned, marker) {
			t.Errorf("marker %q under a narrative sub-heading must not appear in scan output", marker)
		}
	}
}

// Marker literals under the core sections (Purpose / Requirements / Done
// Criteria) must still be detected.
func TestT422_PlaceholderScan_CoreSections_Detect_Marker(t *testing.T) {
	content := `# T998 placeholder leftover case

## Purpose
{The problem this Task addresses goes here}

## Requirements
- [ ] {Requirement 1}
- [ ] {Requirement 2}

## Done Criteria
- [ ] {Criterion 1}

## Result
### Design Decisions
(to be authored)
`
	scanned := ExtractPlaceholderScanContent(content)
	for _, marker := range []string{"{The problem this Task", "{Requirement ", "{Criterion "} {
		if !strings.Contains(scanned, marker) {
			t.Errorf("marker %q in a core section should appear in scan output", marker)
		}
	}
}

// Markers appearing under "## Result -> ### Artifacts" (artifact section)
// must still be detected. In real Tasks, a placeholder remaining inside
// the artefact list is suspicious.
func TestT422_PlaceholderScan_ArtifactSection_Detects_Marker(t *testing.T) {
	content := `# T997

## Result

### Artifacts
- ` + "`{TODO path}`" + ` (new)

### Validation
verified.
`
	scanned := ExtractPlaceholderScanContent(content)
	if !strings.Contains(scanned, "{TODO") {
		t.Error("marker '{TODO' in artifact section must be detected")
	}
}

// Sprint 35 T421 regression fixture — scanner self-description doc
// scenario. A Task that describes the markers in narrative form must not
// be BLOCKED.
//
// Convention: do NOT describe markers in prose under the core sections
// (Purpose / Requirements / Done Criteria) — they cannot be told apart
// from real placeholders. Literal quotation is allowed only under
// narrative sub-headings (Design Decisions / Validation, etc.).
func TestT422_Regression_T421_ScannerDocTask(t *testing.T) {
	content := `# T421 scanner narrative port

## Purpose
fix the defect where the placeholder scanner recursively BLOCKS its own
self-description document.

## Requirements
- [x] port the narrative/artifact separation rule

## Done Criteria
- [x] regression tests PASS

## Result

### Design Decisions
The 6 marker literal strings "{The problem this Task", "{Requirement ",
"{Criterion ", "{TODO", "{One-line summary", "{Items deferred outside"
are quoted from doc_review.go's docReviewPlaceholderMarkers. Previously
this very description triggered the BLOCK.

### Validation
go test PASS.

### Commits
refactor(T421): port placeholder scanner

## References
- KB M003 audit-process.md

## Status change history
| timestamp | change | reason |
|-----------|--------|--------|
`
	scanned := ExtractPlaceholderScanContent(content)
	// All narrative marker quotations must be removed (under Design
	// Decisions / Validation).
	for _, m := range []string{"{The problem this Task", "{Requirement ", "{Criterion ", "{TODO", "{One-line summary", "{Items deferred outside"} {
		if strings.Contains(scanned, m) {
			t.Errorf("marker %q under a narrative sub-heading appeared in scan output — recursive BLOCK", m)
		}
	}
}

// backward compat: when narrative is written directly under `## Result`
// without a sub-heading, the marker is still scannable (suspicious
// leftover). I.e. narrative exclusion only activates with an explicit
// sub-heading.
func TestT422_PlaceholderScan_BackwardCompat_NoSubheading(t *testing.T) {
	content := `# T996 legacy form

## Result
{The problem this Task did} - written directly with no sub-heading
`
	scanned := ExtractPlaceholderScanContent(content)
	if !strings.Contains(scanned, "{The problem this Task") {
		t.Error("marker in a Result section without sub-heading must be detected (backward compat)")
	}
}
