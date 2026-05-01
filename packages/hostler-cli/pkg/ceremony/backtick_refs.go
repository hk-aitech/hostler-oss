// Package ceremony — Task body backtick-path existence verification.
// At sprint start, verifies that backtick paths (`path/to/file`)
// mentioned in the bodies of Sprint-bound Tasks actually exist on
// the filesystem. Prevents the recurrence of the "body assumption !=
// actual code state" problem.
// Target filter:
// Only paths containing a slash (e.g. `cli/pkg/foo.go`,
// `docs/07-knowledge/x.md`).
// Backticks without a slash are treated as code identifiers
// (e.g. `const Foo`) and excluded.
// Tokens ending in parentheses, comma, or full stop are treated
// as narrative context and excluded.
// Narrative/artifact split scan support:
// FindBacktickRefs : FullScan (entire Task body —
// prior behaviour).
// FindBacktickRefsInArtifacts: ArtifactOnly (only under the
// `## Result -> ### Artifacts` sub-tree). Reuses the task
// package convention — narrative sub-headings such as
// `### Design Decisions` / `### Verification` are excluded so
// doc-review false positives are eliminated at the root.
package ceremony

import (
	"bufio"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// backtickPathRe matches text wrapped in backticks.
var backtickPathRe = regexp.MustCompile("`([^`\\n]+)`")

// findBacktickRefs extracts only **file-path candidates** from the
// markdown content (deduplicated, slash-required filter).
func findBacktickRefs(content string) []string {
	matches := backtickPathRe.FindAllStringSubmatch(content, -1)
	seen := map[string]bool{}
	var refs []string
	for _, m := range matches {
		raw := strings.TrimSpace(m[1])
		if !isLikelyPathRef(raw) {
			continue
		}
		// Trim trailing punctuation (e.g. "path/to/x.md)." ->
		// "path/to/x.md").
		raw = strings.TrimRight(raw, ".,);:?!")
		if raw == "" || seen[raw] {
			continue
		}
		seen[raw] = true
		refs = append(refs, raw)
	}
	return refs
}

// isLikelyPathRef reports whether the token inside backticks is
// likely a file path.
func isLikelyPathRef(s string) bool {
	// Slash required — excludes single-token code identifiers.
	if !strings.Contains(s, "/") {
		return false
	}
	// No path with whitespace (e.g. `go test ./...`).
	if strings.ContainsAny(s, " \t") {
		return false
	}
	// Exclude command-like forms (./..., /path/...): "./...",
	// "./cli/..." are Go package patterns.
	if strings.HasSuffix(s, "/...") {
		return false
	}
	// Exclude shell brace expansion: `docs/{a,b}.md` summarises
	// multiple files, not a single path.
	if strings.ContainsAny(s, "{}") {
		return false
	}
	return true
}

// commonPathPrefixes is the auto-tried prefix list for cases where
// the project-root prefix is omitted in Task bodies. Example:
// `pkg/task/foo.go` -> `packages/hostler-cli/pkg/task/foo.go`.
// Reflects the empirical pattern of Task bodies omitting the
// "packages/" prefix (e.g. `pkg/task/status.go`,
// `skills/task-management/SKILL.md`).
var commonPathPrefixes = []string{
	"",
	"packages/hostler-cli/",
	"packages/hostler-cli/cmd/hstl-oss/cmd/",
	"packages/hostler-cli/pkg/",
	"packages/hostler-cli/internal/",
	"packages/hostler-plugin/",
	"packages/hostler-plugin/skills/",
	"packages/hostler-plugin/commands/",
}

// checkBacktickRefsExist returns the paths in refs that do not exist
// on the filesystem (anchored at projectRoot). Input order is
// preserved.
// Niceties:
// Auto-trims `:123` line-number suffixes (e.g.
// `cli/foo.go:43` -> `cli/foo.go`).
// Auto-searches the `cli/` prefix (Task bodies may omit it).
func checkBacktickRefsExist(projectRoot string, refs []string) []string {
	var stale []string
	for _, ref := range refs {
		if refExists(projectRoot, normalizeRef(ref)) {
			continue
		}
		stale = append(stale, ref)
	}
	return stale
}

// normalizeRef strips the `:123` line-number suffix.
func normalizeRef(ref string) string {
	if idx := strings.LastIndex(ref, ":"); idx >= 0 {
		suffix := ref[idx+1:]
		if len(suffix) > 0 && isAllDigits(suffix) {
			return ref[:idx]
		}
	}
	return ref
}

// isAllDigits returns true when s is composed entirely of digits.
func isAllDigits(s string) bool {
	if s == "" {
		return false
	}
	for _, c := range s {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}

// refExists checks whether ref exists relative to projectRoot
// (including the auto-prefix search).
func refExists(projectRoot, ref string) bool {
	if filepath.IsAbs(ref) {
		_, err := os.Stat(ref)
		return err == nil
	}
	for _, prefix := range commonPathPrefixes {
		path := filepath.Join(projectRoot, prefix+ref)
		if _, err := os.Stat(path); err == nil {
			return true
		}
	}
	return false
}

// FindBacktickRefs is the public wrapper around findBacktickRefs.
// Exported so external packages (e.g. cmd/doc-review) can reuse the
// same prefix / filter conventions; the doc-review stale_path logic
// was migrated here to remove duplicate implementations.
// FullScan mode — scans the entire Task body. Used by sprint-start
// design_readiness verification and other "stale references across
// the whole body" detectors. When false positives may arise from
// narrative globs/placeholders, prefer FindBacktickRefsInArtifacts.
func FindBacktickRefs(content string) []string {
	return findBacktickRefs(content)
}

// CheckBacktickRefsExist is the public wrapper around
// checkBacktickRefsExist. Exposed so external packages can reuse the
// same 12-prefix + normalizeRef + refExists logic.
func CheckBacktickRefsExist(projectRoot string, refs []string) []string {
	return checkBacktickRefsExist(projectRoot, refs)
}

// artifactSubHeadings is the set of sub-heading (### ) names inside
// "## Result" that are treated as actual artefact lists.
// Must match pkg/task/task.go's _artifactSubHeadings; both must be
// updated together. A future refactor will unify these constants.
var artifactSubHeadings = map[string]bool{
	"artifact":      true,
	"Artifacts":     true,
	"Artifact":      true,
	"Changed files": true,
	"Changed Files": true,
	"Files":         true,
}

// resultSectionTitles is the set of titles recognised as the
// "## Result" section entry. Must match parseTaskResultFiles
// sectionTitles defaults in the task package.
var resultSectionTitles = map[string]bool{
	"result":    true,
	"artifact":  true,
	"Result":    true,
	"Artifact":  true,
	"Artifacts": true,
}

// ExtractArtifactSection concatenates only the lines under the
// artefact sub-heading (`### Artifacts`, etc.) inside the "## Result"
// section of the Task body content.
// Behaviour:
// On detection of "## Result" (or any resultSectionTitles entry),
// enter the section.
// On entering "### Artifacts" / "### Changed files" (any
// artifactSubHeadings entry) enable the scan.
// On entering narrative sub-headings such as
// "### Design Decisions" / "### Verification", disable the scan.
// Terminate on any other "## ..." section or a "---" separator.
// Tasks that use no sub-headings at all (backward compat) return
// an empty string -> the caller must interpret empty results as
// "no artefact section".
// Sub-headings with extra text (e.g. "### Artifacts (8 items)") are
// matched by the first word.
func ExtractArtifactSection(content string) string {
	var out strings.Builder
	scanner := bufio.NewScanner(strings.NewReader(content))
	// Same scanner-buffer policy used elsewhere — long-line support.
	const (
		initKB = 64
		maxKB  = 1024
		kib    = 1024
	)
	scanner.Buffer(make([]byte, 0, initKB*kib), maxKB*kib)

	inResultSection := false
	inArtifactSub := false

	firstWord := func(s string) string {
		if idx := strings.IndexAny(s, " \t"); idx > 0 {
			return s[:idx]
		}
		return s
	}

	for scanner.Scan() {
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)

		// "## ..." (H2) handling — "### ..." (H3) is excluded.
		if strings.HasPrefix(trimmed, "## ") && !strings.HasPrefix(trimmed, "### ") {
			heading := strings.TrimSpace(strings.TrimPrefix(trimmed, "## "))
			fw := firstWord(heading)
			if resultSectionTitles[heading] || resultSectionTitles[fw] {
				inResultSection = true
				inArtifactSub = false
				continue
			}
			if inResultSection {
				// Another ## entry -> end the Result section.
				break
			}
			continue
		}

		if !inResultSection {
			continue
		}

		// "---" separator -> end the section.
		if trimmed == "---" {
			break
		}

		// "### ..." (H3) — switch between artefact and narrative
		// modes.
		if strings.HasPrefix(trimmed, "### ") {
			sub := strings.TrimSpace(strings.TrimPrefix(trimmed, "### "))
			fw := firstWord(sub)
			inArtifactSub = artifactSubHeadings[sub] || artifactSubHeadings[fw]
			continue
		}

		if inArtifactSub {
			out.WriteString(line)
			out.WriteByte('\n')
		}
	}
	return out.String()
}

// FindBacktickRefsInArtifacts extracts backtick paths only from the
// "## Result -> ### Artifacts" sub-tree of the Task body.
// ArtifactOnly mode — used by doc-review. Narrative sub-headings
// (`### Design Decisions`, `### Verification`, etc.) are treated as
// false positives and excluded. Tasks without an artefact section
// (in-progress or backward-compat) return an empty result.
// Intentionally separated from the FullScan behaviour of
// FindBacktickRefs:
// sprint-start design_readiness needs whole-body stale-reference
// detection (existing behaviour).
// doc-review verifies "what the Task created" — ArtifactOnly
// fits.
// Empirical observation: most Phase 1 stale_path entries appeared in
// narrative paths under `### Design Decisions`
// (glob/line-range/home-dir/placeholder).
func FindBacktickRefsInArtifacts(content string) []string {
	section := ExtractArtifactSection(content)
	if section == "" {
		return nil
	}
	return findBacktickRefs(section)
}

// narrativeSubHeadings — H3 names treated as narrative beneath the
// "## Result" section. The placeholder-marker scanner allows marker
// literals to be quoted in narrative areas (e.g. mentioning `"{TODO"`
// for descriptive purposes inside `### Design Decisions` is excluded
// from BLOCK).
// Heuristic: these sub-headings are prose / description / rationale
// areas where the probability is higher that a marker is mentioned
// naturally rather than left as a real placeholder.
var narrativeSubHeadings = map[string]bool{
	"Design":           true,
	"Design Decisions": true,
	"Validation":       true,
	"Commit":           true,
	"Commits":          true,
	"Trade-offs":       true,
	"Tradeoffs":        true,
	"Follow-up":        true,
	"Follow-Ups":       true,
}

// ExtractPlaceholderScanContent returns only the regions of a Task
// body that are subject to placeholder-marker scanning.
// Behaviour:
// Lines under narrative sub-headings of the `## Result` section
// (`### Design Decisions`, `### Verification`, `### Commits`,
// `### Follow-up`) are excluded.
// Lines under artefact sub-headings (`### Artifacts`, etc.) are
// included (markers in the artefact list are still considered
// placeholder candidates).
// All non-Result H2 sections (Purpose / Requirements / Done
// Criteria / Scope Limits / References, etc.) are included
// (where real placeholder content can hide).
// Reconstructed line-by-line so marker detection via
// strings.Contains stays compatible.
// Design intent: resolves the defect where the doc-review scanner
// recursively BLOCKED on markers quoted descriptively in narrative
// regions. Earlier indirect fixes did not address the root cause;
// the proper fix is to teach the scanner the narrative-separation
// convention.
func ExtractPlaceholderScanContent(content string) string {
	var out strings.Builder
	scanner := bufio.NewScanner(strings.NewReader(content))
	const (
		initKB = 64
		maxKB  = 1024
		kib    = 1024
	)
	scanner.Buffer(make([]byte, 0, initKB*kib), maxKB*kib)

	inResult := false
	inNarrativeSub := false

	firstWord := func(s string) string {
		if idx := strings.IndexAny(s, " \t"); idx > 0 {
			return s[:idx]
		}
		return s
	}

	for scanner.Scan() {
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)

		// H2 handling ("## ..."; not "### ...").
		if strings.HasPrefix(trimmed, "## ") && !strings.HasPrefix(trimmed, "### ") {
			heading := strings.TrimSpace(strings.TrimPrefix(trimmed, "## "))
			fw := firstWord(heading)
			inResult = resultSectionTitles[heading] || resultSectionTitles[fw]
			inNarrativeSub = false
			out.WriteString(line)
			out.WriteByte('\n')
			continue
		}

		// H3 handling.
		if strings.HasPrefix(trimmed, "### ") {
			sub := strings.TrimSpace(strings.TrimPrefix(trimmed, "### "))
			fw := firstWord(sub)
			if inResult {
				inNarrativeSub = narrativeSubHeadings[sub] || narrativeSubHeadings[fw]
			} else {
				inNarrativeSub = false
			}
			// Emit the heading itself (H3 lines almost never contain
			// a marker literal).
			out.WriteString(line)
			out.WriteByte('\n')
			continue
		}

		// Skip lines under narrative sub-headings.
		if inNarrativeSub {
			continue
		}
		out.WriteString(line)
		out.WriteByte('\n')
	}
	return out.String()
}
