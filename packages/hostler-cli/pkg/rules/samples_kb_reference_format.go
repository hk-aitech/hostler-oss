package rules

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// KB bare-reference auto-detection Rule.
//
// Purpose:
//
//	Auto-enforce the KB reference standard defined in
//	docs/08-references/standards/kb-reference-format.md (an extension of
//	the KB-card-id-governance ADR §3). KB card IDs (W001/M001/A001…)
//	are file-local numbering, so a reference must combine a file path
//	with the ID to be unambiguous.
//
// Patterns checked (BLOCK):
//
//	Pattern A — `KB W001` style bare reference (KB followed by a lone ID)
//	Pattern B — `[KB:W001]` style markdown-link bare reference
//
// Allowed forms:
//   - Full: `docs/07-knowledge/mistakes/refactoring.md#m001`
//   - Short: `mistakes/refactoring.md#m001`
//   - Bare (with context): when the same paragraph already mentions the
//     file path. This Rule matches patterns rather than lines, so only
//     the explicit forms Pattern A/B are blocked. A lone "W001" is not
//     in scope (false-positive risk).
//
// Scope:
//   - target: every staged .md file in the workspace (docs/ / works/ / etc.)
//   - allow:  archive/ (external project snapshots)
//     docs/07-knowledge/ (KB card documents themselves — local IDs are
//     defined in their bodies)
//     this standard document (kb-reference-format.md)
//     the KB-card-id-governance ADR — bare forms appear there as
//     negative examples
//     this Rule's docs / test files (samples_kb_reference_format*.go)
//   - new .md only — historical records like sprint retros / Task
//     results are still subject to this Rule, but only at the new-write
//     point.

// ── constants / regexes ──────────────────────────────────────────────

const (
	kbReferenceFormatRuleID      = "precommit.kb.reference-format"
	kbReferenceFormatCategory    = "precommit"
	kbReferenceFormatEvidenceMax = 10
)

// kbReferenceBarePatterns lists the two kinds of bare-reference patterns.
var kbReferenceBarePatterns = []struct {
	id      string
	pattern *regexp.Regexp
}{
	// Pattern A — `KB W001` / `KB  W01` style (KB + space + uppercase letter + 2~3 digits)
	{"A.kb_bare", regexp.MustCompile(`\bKB\s+[A-Z]\d{2,3}\b`)},
	// Pattern B — `[KB:W001]` markdown-link form
	{"B.kb_link", regexp.MustCompile(`\[KB:[A-Z]\d{2,3}\]`)},
}

// kbReferenceAllowedSubpaths lists the excluded areas (substring match).
var kbReferenceAllowedSubpaths = []string{
	"archive/",
	"docs/07-knowledge/",
	"docs/08-references/standards/kb-reference-format.md",
	"docs/02-architecture/adrs/ADR-046",
	"samples_kb_reference_format",
	// All works/ Task/Sprint files — false positives from bare-ID
	// quotations inside reminder blockquotes.
	// works/sprints/sprint-NN/tasks/T*.md is not covered by works/tasks/T,
	// so works/sprints/ is included as well.
	"works/tasks/BACKLOG.md",
	"works/sprints/INDEX.md",
	"works/tasks/T",
	"works/sprints/",
}

// ── Rule implementation ───────────────────────────────────────────────

type kbReferenceFormatRule struct{}

func (kbReferenceFormatRule) ID() string       { return kbReferenceFormatRuleID }
func (kbReferenceFormatRule) Category() string { return kbReferenceFormatCategory }
func (kbReferenceFormatRule) Description() string {
	return "Block KB-card bare references — prohibit using IDs without a file path (KB-card-id-governance ADR §3)"
}
func (kbReferenceFormatRule) DefaultSeverity() Severity { return SeverityBlock }

func (r kbReferenceFormatRule) Check(ctx *RuleContext) *RuleResult {
	start := time.Now()
	res := &RuleResult{RuleID: r.ID(), Severity: r.DefaultSeverity()}

	root := ctx.ProjectRoot
	if root == "" {
		res.Status = StatusSkipped
		res.Duration = time.Since(start)
		return res
	}

	staged := precommitStagedFiles(ctx)
	targets := filterKBReferenceTargets(staged)
	if len(targets) == 0 {
		res.Status = StatusSkipped
		res.Duration = time.Since(start)
		return res
	}

	var hits []string
	for _, rel := range targets {
		full := filepath.Join(root, rel)
		data, err := os.ReadFile(full)
		if err != nil {
			continue
		}
		body := string(data)
		body = stripYAMLFrontmatter(body)

		for _, p := range kbReferenceBarePatterns {
			matches := p.pattern.FindAllStringIndex(body, -1)
			if len(matches) == 0 {
				continue
			}
			for _, m := range matches {
				snippet := excerpt(body, m[0], m[1])
				hits = append(hits, rel+":"+p.id+": "+snippet)
				if len(hits) >= kbReferenceFormatEvidenceMax {
					break
				}
			}
			if len(hits) >= kbReferenceFormatEvidenceMax {
				break
			}
		}
		if len(hits) >= kbReferenceFormatEvidenceMax {
			break
		}
	}

	if len(hits) > 0 {
		res.Status = StatusViolated
		res.Message = "KB bare references blocked — card IDs must be combined with their file path: docs/07-knowledge/<category>/<file>.md#<id>"
		res.Evidence = hits
	} else {
		res.Status = StatusOK
	}
	res.Duration = time.Since(start)
	return res
}

// filterKBReferenceTargets selects the .md files of interest from the
// staged set.
func filterKBReferenceTargets(staged []string) []string {
	var out []string
	for _, f := range staged {
		if !strings.HasSuffix(f, ".md") {
			continue
		}
		skip := false
		for _, sub := range kbReferenceAllowedSubpaths {
			if strings.Contains(f, sub) {
				skip = true
				break
			}
		}
		if skip {
			continue
		}
		out = append(out, f)
	}
	return out
}

func init() {
	Register(kbReferenceFormatRule{})
}
