package rules

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/pkg/envalias"
)

// Anthropic skills v2.0 spec — SKILL.md body length guard.
//
// Purpose:
//   Block (at pre-commit) any skills/*/SKILL.md whose body exceeds the
//   Anthropic skills v2.0 recommended limit (~500 lines). SKILL.md is
//   loaded quickly as the first layer of progressive disclosure, so a
//   bulky body increases trigger latency, token consumption, and hurts
//   readability.
//
// Recommendation on violation:
//   - Split part of the body into references/<topic>.md (progressive
//     disclosure 2nd layer)
//   - Extract normative policy into a standard
//     (`docs/08-references/standards/`) and cross-link
//   - Keep maintainer-traceability such as changelogs in
//     references/changelog.md (isolated area)
//
// Threshold:
//   - default: 500 (Anthropic skills v2.0 recommendation)
//   - env override: HSTL_SKILL_SIZE_MAX (positive integer)
//
// trac: HAR-CM029
const (
	skillSizeRuleID     = "precommit.skill.size"
	skillSizeCategory   = "precommit"
	skillSizeMaxDefault = 500
)

type skillSizeRule struct{}

func (skillSizeRule) ID() string       { return skillSizeRuleID }
func (skillSizeRule) Category() string { return skillSizeCategory }
func (skillSizeRule) Description() string {
	return "skill SKILL.md body ≤ 500 lines (Anthropic skills v2.0 recommendation)"
}
func (skillSizeRule) DefaultSeverity() Severity { return SeverityBlock }

func (r skillSizeRule) Check(ctx *RuleContext) *RuleResult {
	res := &RuleResult{RuleID: r.ID(), Severity: r.DefaultSeverity()}
	staged := precommitStagedFiles(ctx)
	if len(staged) == 0 {
		res.Status = StatusOK
		return res
	}
	targets := filterSkillMDStrict(staged)
	if len(targets) == 0 {
		res.Status = StatusOK
		return res
	}

	maxLines := skillSizeMaxFromEnv()

	var hits []string
	for _, rel := range targets {
		full := filepath.Join(ctx.ProjectRoot, rel)
		n, err := countSkillLines(full)
		if err != nil {
			continue
		}
		if n > maxLines {
			hits = append(hits, fmt.Sprintf("%s: %d lines > %d (≤ %d recommended — split into references/)",
				rel, n, maxLines, maxLines))
		}
		if len(hits) >= skillOSSIdentityEvidenceMax {
			break
		}
	}

	if len(hits) > 0 {
		res.Status = StatusViolated
		res.Message = fmt.Sprintf("skill SKILL.md exceeds the size limit (Anthropic skills v2.0 recommends ≤ %d lines) — split into references/ for progressive disclosure", maxLines)
		res.Evidence = hits
	} else {
		res.Status = StatusOK
	}
	return res
}

// skillSizeMaxFromEnv — env > default precedence. Reads HSTL_SKILL_SIZE_MAX.
func skillSizeMaxFromEnv() int {
	if v := strings.TrimSpace(envalias.Lookup("SKILL_SIZE_MAX")); v != "" {
		var n int
		if _, err := fmt.Sscanf(v, "%d", &n); err == nil && n > 0 {
			return n
		}
	}
	return skillSizeMaxDefault
}

// countSkillLines returns the line count of a file. An empty file is 0;
// a missing file returns an error.
func countSkillLines(path string) (int, error) {
	f, err := os.Open(path)
	if err != nil {
		return 0, err
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 64*1024), 1024*1024)
	n := 0
	for sc.Scan() {
		n++
	}
	if err := sc.Err(); err != nil {
		return n, err
	}
	return n, nil
}

func init() {
	Register(skillSizeRule{})
}
