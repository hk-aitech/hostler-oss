// Package sprint — auto-injection of Phase 5/6/9 ceremony section
// templates.
// Background: with the introduction of two-stage sprint-stamp
// verification, SPRINT.md is rejected by the heuristic when the three
// sections "## Retrospective (KPT)" / "## Lessons Learned" /
// "## Follow-up Tasks" are missing. Users had to write all three
// sections manually before each Sprint complete signature — "ceremony
// is enforced but writing remains manual" was an incomplete UX.
// EnsureCeremonySections is invoked by sprint.Complete just before
// StampWithValidation and **only when the section is missing** injects
// a section template. Users (or AIs) only need to fill in KPT /
// Lessons / Follow-up Tasks inside the empty section template; the
// section structure is guaranteed by sprint.Complete — the section
// titles are chosen to be consistent with the phase5/6/9Pattern
// regular expressions.
// Design principles:
// Existing content is preserved: a section that already has
// content is not touched (idempotent).
// Minimum bullets to avoid heuristic rejection: K/P/T have one
// each, and KB / followup record "0 items, with reason" by default,
// assuming the user / AI will overwrite them ("TODO" placeholder).
// Placeholders may not remain: never write strings that match the
// stamp_heuristic placeholderPatterns (forbidden braces, etc.).
package sprint

import (
	"fmt"
	"os"
	"regexp"
	"strings"
)

// sectionFileMode is the permission used when appending to SPRINT.md
// (re-using an existing file).
const sectionFileMode = 0o644

// Section titles to inject — chosen to match phase5/6/9Pattern in
// the heuristic.
const (
	phase5Heading = "## Retrospective (KPT)"
	phase6Heading = "## Lessons Learned"
	phase9Heading = "## Follow-up Tasks"
)

// Default template bodies — meet the strict-default heuristic with
// minimum bullets. Users (or AI) overwrite these with the Sprint
// Complete Phase 5/6/9 results.
// Phases 6/9 are tabular with an "actual selection" column so that
// candidates that were not chosen are also recorded with their
// rationale, structurally preventing the recurring defect "AI
// auto-registers and skips the user-selection step".
// Actual-selection convention:
// `registered (ID)` — KB card or Task ID specified.
// `dropped (reason 10+)` — concrete rationale for not selecting.
// `merged (existing ID)` — absorbed into an existing card / Task.
// When zero items, still record one row
// ("none | — | — | — | dropped (>=10-char reason) | rationale").
// The guidance lines (`_Write at Sprint Complete Phase N_`, etc.) are
// included intentionally as an affordance the user references while
// filling in the table. **The entire guidance must be deleted after
// the user finishes writing** for the stamp to pass; the heuristic's
// placeholderPatterns BLOCKs lingering guidance lines.
const (
	phase5TemplateBody = `
_Write at Sprint Complete Phase 5 — replace each bullet with the actual retrospective content and remove this guidance._
_The blank line between groups (Keep / Problem / Try) is a readability convention; multiple bullets per group are allowed._

- Keep 1: TODO item to keep.

- Problem 1: TODO problem.

- Try 1: TODO action to try.

`

	phase6TemplateBody = `
_Write at Sprint Complete Phase 6 — record candidate + **actual selection** + rationale on each row._
_Actual-selection convention: ` + "`registered (KB ID)`" + ` / ` + "`dropped (>=10-char reason)`" + ` / ` + "`merged (existing ID)`" + `._
_Even with zero items, write one row "none | — | — | — | dropped (>=10-char reason) | rationale"._
_After completion, delete this entire 3-line guidance for the stamp to pass (the heuristic detects the guidance as a placeholder)._

| # | candidate | category | source Task | recommended | actual selection | rationale |
|---|-----------|----------|-------------|-------------|------------------|-----------|
| 1 | TODO lesson candidate title | mistakes | T### | yes | TODO actual selection | TODO rationale (link ` + "`docs/07-knowledge/...`" + ` on registration) |

`

	phase9TemplateBody = `
_Write at Sprint Complete Phase 9 — record proposal + **actual selection** + rationale on each row._
_Actual-selection convention: ` + "`registered (T###)`" + ` / ` + "`dropped (>=10-char reason)`" + ` / ` + "`merged (existing T###)`" + `._
_Even with zero items, write one row "none | — | — | — | — | dropped (>=10-char reason) | rationale"._
_After completion, delete this entire 3-line guidance for the stamp to pass (the heuristic detects the guidance as a placeholder)._

| # | proposal | type | est | pri | recommended | actual selection | rationale |
|---|----------|------|-----|-----|-------------|------------------|-----------|
| 1 | TODO follow-up Task proposal | feature | S | p3 | yes | TODO actual selection | TODO rationale (record T### on registration) |

`
)

// anchorHeadingPattern locates the "## Done Criteria" section
// boundary so the new section can be inserted after it. Searches the
// range from "## Done Criteria" up to the next `## ` heading. Since
// "## Done Criteria" sits near the end of SPRINT.md, just after it is
// a natural injection point.
var anchorHeadingPattern = regexp.MustCompile(`(?m)^##\s+Done\s+Criteria\s*$`)

// EnsureCeremonySections appends Phase 5/6/9 section templates to
// sprintMDPath when the corresponding section is missing. No-op when
// they exist (idempotent).
// Existence is decided via the phase5/6/9Pattern regular expressions
// reused from the heuristic: Korean / English / Phase-N variants are
// all recognised, so user-written variants are also treated as
// no-ops.
// File I/O failures return an error — the caller
// (sprint.Complete) reports it before StampWithValidation runs but
// preserves Complete's overall success.
func EnsureCeremonySections(sprintMDPath string) error {
	data, err := os.ReadFile(sprintMDPath)
	if err != nil {
		return fmt.Errorf("SPRINT.md read failed (%s): %w", sprintMDPath, err)
	}
	content := string(data)

	toAppend := buildMissingSections(content)
	if toAppend == "" {
		return nil // Idempotent: every section is present.
	}

	newContent := appendAfterAnchor(content, toAppend)
	if newContent == content {
		// No anchor -> append to EOF (defensive).
		newContent = ensureTrailingNewline(content) + toAppend
	}

	if err := os.WriteFile(sprintMDPath, []byte(newContent), sectionFileMode); err != nil {
		return fmt.Errorf("SPRINT.md write failed (%s): %w", sprintMDPath, err)
	}
	return nil
}

// Patterns identifying ceremony sections in SPRINT.md (Korean /
// English / generic Phase-N labels). Kept local to
// ceremony_template.go after the heuristic harness was removed.
var (
	phase5Pattern = regexp.MustCompile(`(?m)^##\s+.*(KPT|[Rr]etro|Phase\s*5)`)
	phase6Pattern = regexp.MustCompile(`(?m)^##\s+.*(KB|[Ll]earned|Phase\s*6)`)
	phase9Pattern = regexp.MustCompile(`(?m)^##\s+.*([Ff]ollow\-?up|Phase\s*9)`)
)

// buildMissingSections concatenates templates for the sections not
// present in content. Returns an empty string when every section
// exists.
func buildMissingSections(content string) string {
	var b strings.Builder
	if !phase5Pattern.MatchString(content) {
		b.WriteString(phase5Heading)
		b.WriteString(phase5TemplateBody)
	}
	if !phase6Pattern.MatchString(content) {
		b.WriteString(phase6Heading)
		b.WriteString(phase6TemplateBody)
	}
	if !phase9Pattern.MatchString(content) {
		b.WriteString(phase9Heading)
		b.WriteString(phase9TemplateBody)
	}
	return b.String()
}

// nextHeaderPattern detects the next `## ` heading (used to delimit
// the anchor section range).
var nextHeaderPattern = regexp.MustCompile(`(?m)^##\s+`)

// appendAfterAnchor inserts toAppend right after the anchor section.
// When the anchor is missing, returns content unchanged so the
// caller can fall back.
func appendAfterAnchor(content, toAppend string) string {
	loc := anchorHeadingPattern.FindStringIndex(content)
	if loc == nil {
		return content
	}
	// The anchor body extends from anchor end to the next `## ` (or
	// EOF).
	rest := content[loc[1]:]
	nextLoc := nextHeaderPattern.FindStringIndex(rest)

	var insertAt int
	if nextLoc != nil {
		insertAt = loc[1] + nextLoc[0]
	} else {
		insertAt = len(content)
	}
	before := ensureTrailingNewline(content[:insertAt])
	after := content[insertAt:]
	return before + toAppend + after
}

// ensureTrailingNewline appends '\n' when s does not end with one.
// Guarantees a blank line between sections so the heuristic regex
// matches via `(?m)^##`.
func ensureTrailingNewline(s string) string {
	if strings.HasSuffix(s, "\n\n") {
		return s
	}
	if strings.HasSuffix(s, "\n") {
		return s + "\n"
	}
	return s + "\n\n"
}
