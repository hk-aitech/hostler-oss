package rules

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// frontmatter UUID v7 mandatory auto-validation Rule.
//
// Purpose:
//
//	Auto-enforces the operating convention from
//	uuid-frontmatter-conventions.md plus the Identity Pattern ADR.
//	When a new .md document falls into the UUID-mandatory area
//	(ADRs / FRSs / UX designs / guides / standards / package READMEs /
//	KB cards), it must carry `uuid: <UUID v7>` in its frontmatter. The
//	format is UUID v7 (timestamp-based + version bit 7).
//
// Subjects (BLOCK):
//
//	staged .md files in the UUID-mandatory area whose frontmatter `uuid`:
//	  - is missing
//	  - violates the format (UUID v7 pattern mismatch)
//
// UUID v7 format:
//
//	`xxxxxxxx-xxxx-7xxx-Vxxx-xxxxxxxxxxxx`, where V ∈ {8, 9, a, b}.
//	(third group's first char "7" indicates version 7; fourth group's
//	first char encodes the variant bits.)
//
// Exceptions (indices / auto-generated artefacts):
//
//	- works/sprints/.../SPRINT.md (id: sprint-NN is the PK)
//	- works/sprints/.../tasks/Txxx-*.md (id: Txxx is the PK)
//	- works/tasks/BACKLOG.md
//	- works/sprints/INDEX.md
//	- docs/07-knowledge/INDEX.md
//	- archive/** (frozen)
//	- packages/hostler-plugin/skills/<name>/SKILL.md (multi-project shared)

const (
	uuidFrontmatterRuleID      = "precommit.uuid.frontmatter"
	uuidFrontmatterCategory    = "precommit"
	uuidFrontmatterEvidenceMax = 10
)

// uuidV7Pattern matches version 7 + variant 10.
var uuidV7Pattern = regexp.MustCompile(
	`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-7[0-9a-fA-F]{3}-[89abAB][0-9a-fA-F]{3}-[0-9a-fA-F]{12}$`,
)

// uuidFrontmatterTargets lists the UUID-mandatory area path globs (substring match).
//
// This rule does not perform precise path matching — it identifies the
// mandatory area via prefix/substring. Excluded areas are extracted into
// a separate variable.
//
// docs/07-knowledge/ currently has KB cards stored as 1 file = N cards
// (h2-delimited), so the absence of frontmatter is the operating pattern.
// The "KB card" entry of uuid-frontmatter-conventions.md will be activated
// later when the structure migrates to 1 file = 1 card. For now the
// mandatory area excludes KB cards (this was discovered via self-validation:
// adding new KB cards by appending an h2 without frontmatter is the
// expected pattern).
var uuidFrontmatterTargets = []string{
	"docs/02-architecture/adrs/",
	"docs/03-design/frs/",
	"docs/03-design/ux/",
	"docs/03-design/ux-extra/",
	"docs/04-guides/",
	"docs/08-references/standards/",
	// "docs/07-knowledge/" — activate after KB migrates to 1-file-1-card
}

// uuidFrontmatterExclude lists explicit exceptions inside the mandatory area.
var uuidFrontmatterExclude = []string{
	"docs/07-knowledge/INDEX.md",
	"docs/07-knowledge/README.md",
}

// uuidFrontmatterREADMEPattern matches packages/<name>/README.md only.
var uuidFrontmatterREADMEPattern = regexp.MustCompile(`^packages/[^/]+/README\.md$`)

type uuidFrontmatterRule struct{}

func (uuidFrontmatterRule) ID() string       { return uuidFrontmatterRuleID }
func (uuidFrontmatterRule) Category() string { return uuidFrontmatterCategory }
func (uuidFrontmatterRule) Description() string {
	return "Validate that new .md frontmatter contains UUID v7 (Identity Pattern ADR)"
}
func (uuidFrontmatterRule) DefaultSeverity() Severity { return SeverityBlock }

func (r uuidFrontmatterRule) Check(ctx *RuleContext) *RuleResult {
	start := time.Now()
	res := &RuleResult{RuleID: r.ID(), Severity: r.DefaultSeverity()}

	root := ctx.ProjectRoot
	if root == "" {
		res.Status = StatusSkipped
		res.Duration = time.Since(start)
		return res
	}

	staged := precommitStagedFiles(ctx)
	targets := filterUUIDTargets(staged)
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
		uuidVal := extractFrontmatterField(body, "uuid")
		if uuidVal == "" {
			hits = append(hits, rel+": uuid field missing")
			if len(hits) >= uuidFrontmatterEvidenceMax {
				break
			}
			continue
		}
		if !uuidV7Pattern.MatchString(uuidVal) {
			hits = append(hits, rel+": uuid format violation ("+uuidVal+") — UUID v7 required")
			if len(hits) >= uuidFrontmatterEvidenceMax {
				break
			}
		}
	}

	if len(hits) > 0 {
		res.Status = StatusViolated
		res.Message = "frontmatter UUID v7 mandatory violation — issue via `hstl uuid generate`. SSOT: docs/08-references/standards/uuid-frontmatter-conventions.md"
		res.Evidence = hits
	} else {
		res.Status = StatusOK
	}
	res.Duration = time.Since(start)
	return res
}

// filterUUIDTargets extracts only the .md files in the UUID-mandatory area
// from the staged set.
func filterUUIDTargets(staged []string) []string {
	var out []string
	for _, f := range staged {
		f = filepath.ToSlash(f)
		if !strings.HasSuffix(f, ".md") {
			continue
		}
		// Explicit exception — always exclude.
		excluded := false
		for _, ex := range uuidFrontmatterExclude {
			if f == ex {
				excluded = true
				break
			}
		}
		if excluded {
			continue
		}
		// Match 1 — package README
		if uuidFrontmatterREADMEPattern.MatchString(f) {
			out = append(out, f)
			continue
		}
		// Match 2 — prefix paths
		matched := false
		for _, t := range uuidFrontmatterTargets {
			if strings.HasPrefix(f, t) {
				matched = true
				break
			}
		}
		if matched {
			out = append(out, f)
		}
	}
	return out
}

// extractFrontmatterField extracts the first occurrence of `key` from the
// YAML frontmatter of a body.
//
// Plain grep based — does not use a real yaml parser (keeps the rules
// package's dependencies minimal). Frontmatter is assumed to be the
// `---\n...\n---\n` region.
//
// pkg/fileutil has a function with the same name and intent; this is an
// isolated helper local to the rules package.
func extractFrontmatterField(body, key string) string {
	body = strings.TrimPrefix(body, "") // BOM
	if !strings.HasPrefix(body, "---") {
		return ""
	}
	// Find the end of the frontmatter region.
	rest := body[3:]
	if !strings.HasPrefix(rest, "\n") {
		return ""
	}
	rest = rest[1:]
	end := strings.Index(rest, "\n---")
	if end < 0 {
		return ""
	}
	fm := rest[:end]
	prefix := key + ":"
	for _, line := range strings.Split(fm, "\n") {
		t := strings.TrimSpace(line)
		if !strings.HasPrefix(t, prefix) {
			continue
		}
		val := strings.TrimSpace(strings.TrimPrefix(t, prefix))
		val = strings.Trim(val, `"'`)
		return val
	}
	return ""
}

func init() {
	Register(uuidFrontmatterRule{})
}
