package rules

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// multi-project shared skill identity regression-prevention Rule.
//
// Purpose:
//
//	Block new entries that violate the multi-project shared skill
//	identity from skills/*/SKILL.md bodies at pre-commit time. The
//	skills in this plugin are shared across projects (other projects
//	may import them), so project-specific information (T###/Sprint-NN/
//	KB IDs/external project names/external docs/the term "OSS") in a
//	body becomes meaningless maintainer-traceability noise for users
//	from other projects.
//
//	Note: this project itself may be commercialised — it is not a
//	publicly released open source. The term "OSS" has been used loosely
//	and incorrectly, so this rule blocks the term itself and steers
//	authors towards other phrasing (multi-project shared / portability).
//
// 5 patterns:
//   Pattern B — T#### / Sprint-NN / KB ID (maintainer traceability irrelevant to other projects)
//   Pattern C — list of external project names (concrete project identifiers)
//   Pattern D — /home/ absolute path prefix (portability violation)
//   Pattern E — external docs citation (docs/<NN>-<dir>/ + CLAUDE.md) — only the skill's own refs are allowed
//   Pattern F — the term "OSS" itself (this project is not OSS, so the term is forbidden)
//
// Scope:
//   - strict: skills/*/SKILL.md (this Rule's subject)
//   - allow:  skills/*/references/  (maintainer-tracking isolated area)
//             skills/_frozen/        (frozen area)
//             skills/_shared/        (shared methodology)
//             skills/*/examples/     (intentional examples)

// ── constants / regexes ──────────────────────────────────────────────

const (
	skillOSSIdentityRuleID      = "precommit.skill.oss_identity"
	skillOSSIdentityCategory    = "precommit" // automatically mapped via rulesForPhase(PhasePreCommit)
	skillOSSIdentityEvidenceMax = 10          // upper bound to prevent evidence noise
)

// skillOSSPatterns lists the five patterns. The id is used in evidence
// strings.
// (the variable name 'OSS' is historical — kept for rule-ID compatibility;
// the meaning is "multi-project shared".)
var skillOSSPatterns = []struct {
	id      string
	pattern *regexp.Regexp
}{
	// Pattern B — T-ID / Sprint-NN / KB ID (maintainer traceability irrelevant to other projects)
	{"B.t_id", regexp.MustCompile(`\bT\d{3,}\b`)},
	{"B.sprint", regexp.MustCompile(`\bSprint-?\d+\b|\bsprint-\d+\b`)},
	{"B.kb_id", regexp.MustCompile(`\bKB\s+[A-Z]\d{2,3}\b`)},

	// Pattern C — list of external project names (project-specific)
	{"C.external", regexp.MustCompile(`\bsample-internal-project\b`)},

	// Pattern D — absolute-path prefix (portability violation)
	{"D.abspath", regexp.MustCompile(`/home/[a-z][a-z0-9_-]*/`)},

	// Pattern E — external docs citation (project docs directory other than this skill's refs)
	// docs/<NN>-<dir>/ : project docs convention (00-project, 02-architecture, 07-knowledge, etc.)
	// CLAUDE.md : the project root guide (forbidden in skill bodies)
	{"E.ext_docs", regexp.MustCompile(`docs/\d{2}-[a-z][a-z0-9_-]*/|\bCLAUDE\.md\b`)},

	// Pattern F — the term "OSS" itself (this project is not OSS)
	// Use word boundaries to avoid matching OSSL/cross/etc.
	{"F.oss_term", regexp.MustCompile(`\bOSS\b|\boss[-_]?identity\b`)},
}

// skillOSSAllowedSubpaths lists the isolated areas (excluded from checks)
// outside SKILL.md. A path is skipped when it contains any of these
// substrings.
var skillOSSAllowedSubpaths = []string{
	"/references/",
	"/_frozen/",
	"/_shared/",
	"/examples/",
}

// ── Rule implementation ───────────────────────────────────────────────

type skillOSSIdentityRule struct{}

func (skillOSSIdentityRule) ID() string       { return skillOSSIdentityRuleID }
func (skillOSSIdentityRule) Category() string { return skillOSSIdentityCategory }
func (skillOSSIdentityRule) Description() string {
	return "Block multi-project shared identity 6-pattern regressions in skill SKILL.md bodies"
}
func (skillOSSIdentityRule) DefaultSeverity() Severity { return SeverityBlock }

func (r skillOSSIdentityRule) Check(ctx *RuleContext) *RuleResult {
	start := time.Now()
	res := &RuleResult{RuleID: r.ID(), Severity: r.DefaultSeverity()}

	root := ctx.ProjectRoot
	if root == "" {
		res.Status = StatusSkipped
		res.Duration = time.Since(start)
		return res
	}

	// Extract SKILL.md only from staged changes.
	staged := precommitStagedFiles(ctx)
	targets := filterSkillMDStrict(staged)
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
			// Staged but unreadable (deleted/renamed) — skip.
			continue
		}
		raw := string(data)
		// Body inspection — exclude the frontmatter region.
		body := stripYAMLFrontmatter(raw)

		for _, p := range skillOSSPatterns {
			matches := p.pattern.FindAllStringIndex(body, -1)
			if len(matches) == 0 {
				continue
			}
			for _, m := range matches {
				snippet := excerpt(body, m[0], m[1])
				hits = append(hits, rel+":"+p.id+": "+snippet)
				if len(hits) >= skillOSSIdentityEvidenceMax {
					break
				}
			}
			if len(hits) >= skillOSSIdentityEvidenceMax {
				break
			}
		}

		// Pattern G — frontmatter paths project-specific glob check.
		// Inspects only the frontmatter region (the body-strip excludes
		// it). Since the skill is multi-project shared, paths only allow
		// generic globs.
		if len(hits) < skillOSSIdentityEvidenceMax {
			pathsHits := checkFrontmatterPaths(raw, rel)
			for _, h := range pathsHits {
				hits = append(hits, h)
				if len(hits) >= skillOSSIdentityEvidenceMax {
					break
				}
			}
		}

		if len(hits) >= skillOSSIdentityEvidenceMax {
			break
		}
	}

	if len(hits) > 0 {
		res.Status = StatusViolated
		res.Message = "skill SKILL.md body contains multi-project shared identity violations — isolate maintainer-traceability info under references/changelog.md, align with the Anthropic skills v2.0 spec"
		res.Evidence = hits
	} else {
		res.Status = StatusOK
	}
	res.Duration = time.Since(start)
	return res
}

// filterSkillMDStrict selects only the strict-check targets from the
// staged set.
//
//	target: skills/<name>/SKILL.md
//	exclude: subtrees of skills/_frozen/ / _shared/ / references/ / examples/
func filterSkillMDStrict(staged []string) []string {
	var out []string
	for _, f := range staged {
		// Match SKILL.md filename only (the last path segment must be SKILL.md).
		if filepath.Base(f) != "SKILL.md" {
			continue
		}
		// Inside skills/ only.
		if !strings.Contains(f, "/skills/") && !strings.HasPrefix(f, "skills/") {
			continue
		}
		// allow the _frozen / _shared / references / examples subtrees.
		skip := false
		for _, sub := range skillOSSAllowedSubpaths {
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

// stripYAMLFrontmatter removes the `---\n...\n---\n` frontmatter region.
//
//	When no frontmatter is present, returns the input unchanged.
func stripYAMLFrontmatter(body string) string {
	if !strings.HasPrefix(body, "---\n") {
		return body
	}
	_, after, ok := strings.Cut(body[4:], "\n---\n")
	if !ok {
		return body
	}
	return after
}

// extractYAMLFrontmatterRaw returns just the frontmatter region
// (excluding the `---` delimiters). Returns an empty string when no
// frontmatter is present.
func extractYAMLFrontmatterRaw(body string) string {
	if !strings.HasPrefix(body, "---\n") {
		return ""
	}
	before, _, ok := strings.Cut(body[4:], "\n---\n")
	if !ok {
		return ""
	}
	return before
}

// frontmatterPathsSpecificPattern is Pattern G: project-specific globs in
// frontmatter paths. The presence of a docs/<NN>-<dir>/ form in paths
// violates the multi-project shared invariant.
// Detection target: docs/[0-9]{2}-[a-z][a-z0-9_-]*/ in paths lines.
var frontmatterPathsSpecificPattern = regexp.MustCompile(`docs/\d{2}-[a-z][a-z0-9_-]*/`)

// checkFrontmatterPaths runs the precise Pattern G check on frontmatter
// paths. It returns evidence when the paths line in the frontmatter
// contains a project-specific glob.
func checkFrontmatterPaths(raw, rel string) []string {
	fm := extractYAMLFrontmatterRaw(raw)
	if fm == "" {
		return nil
	}
	var hits []string
	// Only single-line `paths` lines are checked. multi-line YAML arrays
	// are a future extension.
	for _, line := range strings.Split(fm, "\n") {
		trimmed := strings.TrimSpace(line)
		if !strings.HasPrefix(trimmed, "paths:") && !strings.HasPrefix(trimmed, "paths :") {
			continue
		}
		matches := frontmatterPathsSpecificPattern.FindAllString(line, -1)
		for _, m := range matches {
			hits = append(hits, rel+":G.fm_paths: "+strings.TrimSpace(line[:min(len(line), 80)])+" → "+m)
		}
	}
	return hits
}

// excerpt returns 30 chars of context centred on body[start:end].
func excerpt(body string, start, end int) string {
	const padding = 15
	from := max(start-padding, 0)
	to := min(end+padding, len(body))
	snippet := body[from:to]
	snippet = strings.ReplaceAll(snippet, "\n", " ")
	return strings.TrimSpace(snippet)
}

func init() {
	Register(skillOSSIdentityRule{})
}
