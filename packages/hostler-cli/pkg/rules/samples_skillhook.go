package rules

import (
	"os"
	"path/filepath"
	"strings"
	"time"
)

// G6 Skill Hook 2-rule.
//
// Inventory G6 — verifies, as observability, that skill-hook files such as
// `skills/destructive-task-guard/hooks/destructive-guard.sh` and
// `skills/tdd-mode/hooks/tdd-run.sh` actually exist and are executable.
// Starts as observation rather than blocking (severity: advisory). Lets
// audit reports detect hook breakage immediately.

// ── constants ────────────────────────────────────────────────────────

// skillHookExpectedFiles lists the hook files that must exist.
// Paths are relative to the hstl project root.
var skillHookExpectedFiles = []string{
	"skills/destructive-task-guard/hooks/destructive-guard.sh",
	"skills/tdd-mode/hooks/tdd-run.sh",
}

// skillHookAllowedToolsFrontmatterKey is the field name (compatibility.tools
// or allowed-tools) inspected in SKILL.md frontmatter.
const skillHookAllowedToolsFrontmatterKey = "tools:"

// skillsScanRoot is the skills directory root (relative to project root).
const skillsScanRoot = "skills"

// ── Rule 1: skill.hook.exec_present ───────────────────────────────────

type skillHookExecPresentRule struct{}

func (skillHookExecPresentRule) ID() string       { return "skill.hook.exec_present" }
func (skillHookExecPresentRule) Category() string { return "skill" }
func (skillHookExecPresentRule) Description() string {
	return "Verify mandatory skill hook scripts exist and are executable"
}
func (skillHookExecPresentRule) DefaultSeverity() Severity { return SeverityAdvisory }

func (r skillHookExecPresentRule) Check(ctx *RuleContext) *RuleResult {
	start := time.Now()
	res := &RuleResult{RuleID: r.ID(), Severity: r.DefaultSeverity()}

	root := ctx.ProjectRoot
	if root == "" {
		res.Status = StatusSkipped
		res.Duration = time.Since(start)
		return res
	}

	var missing []string
	var notExec []string
	for _, rel := range skillHookExpectedFiles {
		full := filepath.Join(root, rel)
		info, err := os.Stat(full)
		if err != nil {
			missing = append(missing, rel)
			continue
		}
		if info.Mode()&0o111 == 0 {
			notExec = append(notExec, rel)
		}
	}

	if len(missing) > 0 || len(notExec) > 0 {
		res.Status = StatusViolated
		res.Message = "skill hook missing or not executable"
		for _, m := range missing {
			res.Evidence = append(res.Evidence, "missing: "+m)
		}
		for _, n := range notExec {
			res.Evidence = append(res.Evidence, "not_exec: "+n)
		}
	} else {
		res.Status = StatusOK
	}
	res.Duration = time.Since(start)
	return res
}

// ── Rule 2: skill.frontmatter.tools_declared ──────────────────────────

type skillFrontmatterToolsDeclaredRule struct{}

func (skillFrontmatterToolsDeclaredRule) ID() string       { return "skill.frontmatter.tools_declared" }
func (skillFrontmatterToolsDeclaredRule) Category() string { return "skill" }
func (skillFrontmatterToolsDeclaredRule) Description() string {
	return "Verify that skills/*/SKILL.md frontmatter declares the compatibility.tools field"
}
func (skillFrontmatterToolsDeclaredRule) DefaultSeverity() Severity { return SeverityAdvisory }

func (r skillFrontmatterToolsDeclaredRule) Check(ctx *RuleContext) *RuleResult {
	start := time.Now()
	res := &RuleResult{RuleID: r.ID(), Severity: r.DefaultSeverity()}

	root := ctx.ProjectRoot
	if root == "" {
		res.Status = StatusSkipped
		res.Duration = time.Since(start)
		return res
	}

	skillsDir := filepath.Join(root, skillsScanRoot)
	entries, err := os.ReadDir(skillsDir)
	if err != nil {
		res.Status = StatusSkipped
		res.Err = err
		res.Duration = time.Since(start)
		return res
	}

	var missing []string
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		skillMD := filepath.Join(skillsDir, e.Name(), "SKILL.md")
		data, err := os.ReadFile(skillMD)
		if err != nil {
			continue
		}
		if !strings.Contains(string(data), skillHookAllowedToolsFrontmatterKey) {
			missing = append(missing, e.Name())
		}
	}

	if len(missing) > 0 {
		res.Status = StatusViolated
		res.Message = "SKILLs missing the tools: field"
		// Keep evidence noise low — first 5 only.
		max := skillFrontmatterEvidenceMax
		if len(missing) < max {
			max = len(missing)
		}
		res.Evidence = missing[:max]
	} else {
		res.Status = StatusOK
	}
	res.Duration = time.Since(start)
	return res
}

const skillFrontmatterEvidenceMax = 5

func init() {
	Register(skillHookExecPresentRule{})
	Register(skillFrontmatterToolsDeclaredRule{})
}
