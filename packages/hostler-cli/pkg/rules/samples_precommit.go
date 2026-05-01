package rules

import (
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// G5 pre-commit checks migrated to the Rule Registry, with enforcement
// promoted and conditional execution.
//
// Inventory G5 — `scripts/git-hooks/pre-commit` previously hosted 11
// independent checks as bash branches. They were first registered in
// the Rule Registry as observability-only entries; actual enforcement
// was later migrated onto the Rule Engine.
//
// Design:
//   - Each Rule runs an external command from ProjectRoot and decides via
//     exit code.
//   - severity is `block` — the Rule Engine is the enforcement entry point.
//   - Each Rule has a `shouldRun` predicate that inspects the staged files
//     (e.g. Go checks run only when *.go changed). This mirrors the bash
//     hook's has_*_files branches.
//   - The bash hook collapses to a single `hstl rules run --phase
//     precommit` invocation.

// ── constants ────────────────────────────────────────────────────────

const (
	// precommitWrapperCategory is the Category of the Rules in this file.
	precommitWrapperCategory = "precommit"
)

// precommitCheck is the definition of a single pre-commit Rule.
// A nil shouldRun means the check runs unconditionally regardless of
// staged files.
type precommitCheck struct {
	id          string
	description string
	command     []string
	shouldRun   func(staged []string) bool // nil → always run
}

var precommitChecks = []precommitCheck{
	{
		id:          "precommit.go.vet",
		description: "go vet ./... passes",
		command:     []string{"bash", "-c", "cd cli && go vet ./... > /dev/null 2>&1"},
		shouldRun:   hasGoFiles,
	},
	{
		id:          "precommit.go.staticcheck",
		description: "staticcheck ./... passes (cli directory)",
		command:     []string{"bash", "-c", "cd cli && command -v staticcheck >/dev/null && staticcheck ./... > /dev/null 2>&1 || true"},
		shouldRun:   hasGoFiles,
	},
	{
		id:          "precommit.go.test_short",
		description: "go test -short -count=1 passes (cli directory)",
		command:     []string{"bash", "-c", "cd cli && go test ./... -count=1 -short > /dev/null 2>&1"},
		shouldRun:   hasGoFiles,
	},
	{
		// HSTL_PRECOMMIT_MANIFEST_AUTOREGEN=1 immediately regenerates +
		// `git add`s on drift detection so the commit cycle is shorter.
		// Default off (auto `git add` requires explicit consent).
		id:          "precommit.manifest.drift",
		description: "manifest.json drift check (auto-regenerates when AUTOREGEN=1)",
		command:     []string{"bash", "-c", "cd cli && if make -s check-manifest > /dev/null 2>&1; then :; elif [ \"${HSTL_PRECOMMIT_MANIFEST_AUTOREGEN:-0}\" = \"1\" ]; then make -s generate-manifest > /dev/null && cd .. && git add docs/generated/manifest.json && echo '[precommit.manifest.drift] auto-regenerated + staged'; else exit 1; fi"},
		shouldRun:   hasGoFiles,
	},
	{
		// `test -f X && Y` blocks (exits 1) when X is missing. Switching
		// to `[ ! -f X ] || Y` returns exit 0 (skip) when X is missing
		// and propagates Y's exit code when X exists. The plugin's home
		// repo is unaffected, while monorepo / external repos skip.
		id:          "precommit.skill_trigger.drift",
		description: "skill trigger index drift check",
		command:     []string{"bash", "-c", "[ ! -f scripts/generate-skill-triggers.py ] || { python3 scripts/generate-skill-triggers.py > /dev/null 2>&1 && git diff --exit-code docs/03-design/skill-trigger-index.md > /dev/null 2>&1; }"},
		shouldRun:   hasSkillFiles,
	},
	{
		// Same pattern as skill_trigger.drift.
		// External projects without audit.py skip gracefully.
		// `2>&1` is removed per the rule-engine-development convention.
		id:          "precommit.claudemd.audit",
		description: "CLAUDE.md audit passes",
		command:     []string{"bash", "-c", "if [ ! -f skills/claude-md-audit/scripts/audit.py ]; then exit 0; fi; python3 skills/claude-md-audit/scripts/audit.py CLAUDE.md > /dev/null"},
		shouldRun:   hasClaudeMdOrRules,
	},
	{
		// External projects do not have scripts/check-version-sync.sh,
		// causing `bash nonexistent.sh` to BLOCK with exit 127. We skip
		// gracefully when the script is missing.
		id:          "precommit.version.sync",
		description: "plugin.json / marketplace.json / CLAUDE.md version sync",
		command:     []string{"bash", "-c", "if [ ! -f scripts/check-version-sync.sh ]; then exit 0; fi; bash scripts/check-version-sync.sh"},
		shouldRun:   hasVersionFiles,
	},
	{
		id:          "precommit.manifest.skill_command_sync",
		description: "manifest ↔ Skill/Command consistency check",
		command:     []string{"bash", "scripts/check-manifest-sync.sh"},
		shouldRun:   hasGoOrSkillFiles,
	},
	{
		// Warn (exit 0) when test artefacts (gate-judgement.jsonl,
		// session-id, logs/) appear as untracked. HSTL_ARTIFACTS_CHECK=strict
		// turns this into a block. Prevents recurrence of the artefact leak
		// observed during cleanup.
		id:          "precommit.artifacts.untracked",
		description: "test artefact untracked detection (.gitignore-blocked, warn-only)",
		command:     []string{"bash", "scripts/check-untracked-artifacts.sh"},
		shouldRun:   hasGoFiles,
	},
	{
		// main/dev divergence monitor + block. Prevents the recurrence of
		// the case in which a hotfix could not enter because main fell
		// 561 commits behind dev. HSTL_BRANCH_SYNC=off opts out.
		// The actual threshold is read from
		// .hstl-oss/project-config.yaml precommit.branch_sync.threshold
		// (default 30) — scripts/check-branch-sync.sh prefers config.
		id:          "precommit.branch.sync",
		description: "main/dev branch divergence monitor (blocks above the config threshold — default 30)",
		command:     []string{"bash", "scripts/check-branch-sync.sh"},
		shouldRun:   nil, // independent of staged files — always run (the script's internal skip branch avoids `git fetch` cost)
	},
	{
		id:          "precommit.harness.skill_drift",
		description: "harness_defaults.json ↔ task-management SKILL.md drift",
		command:     []string{"bash", "scripts/check-harness-skill-drift.sh"},
		shouldRun:   hasHarnessOrTaskMgmtSkill,
	},
	{
		id:          "precommit.sprint_phase.drift",
		description: "Sprint Phase triple-definition drift",
		command:     []string{"bash", "scripts/check-sprint-phase-drift.sh"},
		shouldRun:   hasHarnessOrSprintCompleteCmd,
	},
	{
		// Visibility improvement.
		// The first migration used `|| true` and `>/dev/null 2>&1`, swallowing
		// failures and output entirely. With those swallows in place, an
		// HMAC drift would still leave the Rule reporting OK, neutralising
		// enforcement. We restore visibility by exposing failures and
		// preserving verify-hmac's own output. shouldRun=hasTaskFiles
		// already gates this on Task files, so noise stays at 0 when none
		// are staged.
		id:          "precommit.task.hmac",
		description: "HMAC verification of done Task bodies",
		// `--diff-filter=d` excludes deleted files (when a file is moved
		// during Sprint inclusion without `git mv`, both staged delete
		// and staged add appear; passing the deleted old path to
		// verify-hmac fails because the file is gone).
		command:   []string{"bash", "-c", "command -v hstl >/dev/null || { echo '[precommit.task.hmac] hstl not installed — skip' >&2; exit 0; }; files=\"$(git diff --cached --name-only --diff-filter=d | grep -E 'works/(tasks|sprints/.*/tasks)/.*T[0-9]+.*\\.md$' | tr '\\n' ',' | sed 's/,$//')\"; [ -z \"$files\" ] && { echo '[precommit.task.hmac] 0 Task files — skip' >&2; exit 0; }; count=$(echo \"$files\" | tr ',' '\\n' | wc -l); echo \"[precommit.task.hmac] verifying ${count} files\" >&2; hstl task verify-hmac --files \"$files\""},
		shouldRun: hasTaskFiles,
	},
	{
		// Sprint HMAC verification.
		// Verifies the sprint_hmac of staged SPRINT.md files (those that
		// belong to completed). verify-hmac itself passes legacy / active
		// / backlog Sprints through as not-completed / legacy-no-hmac.
		id:          "precommit.sprint.hmac",
		description: "HMAC verification of completed Sprint SPRINT.md files",
		command: []string{"bash", "-c", `command -v hstl >/dev/null || { echo '[precommit.sprint.hmac] hstl not installed — skip' >&2; exit 0; }
files="$(git diff --cached --name-only --diff-filter=d | grep -E 'works/sprints/(completed|active)/sprint-[0-9]+/SPRINT\.md$')"
[ -z "$files" ] && { echo '[precommit.sprint.hmac] 0 SPRINT.md files — skip' >&2; exit 0; }
fail=0
for f in $files; do
  sprint_id="$(basename "$(dirname "$f")")"
  if ! hstl sprint verify-hmac "$sprint_id" >&2; then
    fail=1
  fi
done
exit $fail`},
		shouldRun: hasSprintMDFiles,
	},
	{
		// Detect Task frontmatter-only edits.
		// When a user uses Write/Edit to modify the frontmatter `status` of
		// a Task file directly, the legitimate CLI paths (task
		// start/complete/reopen) are bypassed and a file/DB drift is
		// created at the source. This rule detects staged Task files in
		// which only the frontmatter status changed and the body was not
		// modified, and warns. It does not block — the goal is to point
		// the user back to the legitimate path.
		// env HSTL_FRONTMATTER_EDIT_POLICY: off (skip) / warn (default) / block (exit 1).
		id:          "precommit.task.frontmatter_only_edit",
		description: "Detect staged Task files where only the frontmatter status changed (prevents DB drift at source)",
		command: []string{"bash", "-c", `policy="${HSTL_FRONTMATTER_EDIT_POLICY:-warn}"
if [ "$policy" = "off" ]; then
  echo "[precommit.task.frontmatter_only_edit] policy=off — skip" >&2
  exit 0
fi
files="$(git diff --cached --name-only --diff-filter=d | grep -E 'works/(tasks|sprints/.*/tasks)/.*T[0-9]+.*\.md$' || true)"
[ -z "$files" ] && { echo "[precommit.task.frontmatter_only_edit] 0 Task files — skip" >&2; exit 0; }
detected=""
for f in $files; do
  diff="$(git diff --cached -U0 -- "$f")"
  # Extract changed lines only: exclude '---' diff headers (--- a/...), hunk headers '@@...', and the frontmatter delimiter '---' from real change (+|-) lines.
  changed="$(echo "$diff" | awk '/^[+-][^+-]/ && !/^[+-][+-][+-]/' | grep -vE '^[+-]---$' || true)"
  [ -z "$changed" ] && continue
  status_lines="$(echo "$changed" | grep -cE '^[+-]status:' || true)"
  other_lines="$(echo "$changed" | grep -cvE '^[+-]status:' || true)"
  if [ "${status_lines:-0}" -gt 0 ] && [ "${other_lines:-0}" -eq 0 ]; then
    detected="${detected}  - $f\n"
  fi
done
if [ -z "$detected" ]; then
  echo "[precommit.task.frontmatter_only_edit] no frontmatter-only edits" >&2
  exit 0
fi
echo "[precommit.task.frontmatter_only_edit] WARN: detected staged Task files with frontmatter status only:" >&2
printf "%b" "$detected" >&2
echo "" >&2
echo "  Please use the proper path: hstl task {start|complete|reopen} <ID>" >&2
echo "  Direct edits can cause DB drift (see KB W002)." >&2
echo "  To recover from drift: hstl task update --status (escape hatch, audited)" >&2
echo "  To upgrade this warning to a block: HSTL_FRONTMATTER_EDIT_POLICY=block" >&2
echo "  To disable the warning:         HSTL_FRONTMATTER_EDIT_POLICY=off" >&2
if [ "$policy" = "block" ]; then
  exit 1
fi
exit 0`},
		shouldRun: hasTaskFiles,
	},
	{
		// JSON stdout pollution drift monitor.
		// The default monorepo CLI output is JSON. When AI tools pipe the
		// CLI, mixed stderr breaks parsing.
		// scripts/check-cli-json-stdout.sh detects non-allowlisted direct
		// stderr usage and exits 1 to block the commit.
		// Skip when the script is absent (opt-in for external projects).
		id:          "precommit.cli.json_stdout",
		description: "Monitor stderr pollution drift in CLI JSON mode",
		command: []string{"bash", "-c", `script="scripts/check-cli-json-stdout.sh"
[ -f "$script" ] || { echo "[precommit.cli.json_stdout] $script absent — skip" >&2; exit 0; }
bash "$script"`},
		shouldRun: hasGoFiles,
	},
	{
		// pre-commit hook for full TRAC SSOT integrity verification.
		//
		// When trac-catalog-*.yaml / trac-contexts.yaml / trac-members.yaml
		// change, run `hstl trac validate --quick` to verify integrity
		// across the four catalogues + Registry. Frontmatter scanning is
		// skipped (commit speed). Skip when hstl is not installed.
		id:          "precommit.trac.validate",
		description: "TRAC SSOT integrity check (4 catalogues + Context/Members Registry)",
		command: []string{"bash", "-c", `command -v hstl >/dev/null || { echo '[precommit.trac.validate] hstl not installed — skip' >&2; exit 0; }
hstl trac validate --quick >/dev/null 2>&1 || { hstl trac validate --quick >&2; exit 1; }`},
		shouldRun: hasTracStandardsFiles,
	},
	{
		// SPRINT.md cross_track_deps integrity 4-rule check.
		//
		// SSOT: docs/08-references/standards/cross-track-sprint-deps.md
		// Four rules: referential integrity / cycles / kind enum / non-empty reason.
		//
		// Policy: HSTL_CROSS_TRACK_DEPS_POLICY=strict → BLOCK,
		// HSTL_CROSS_TRACK_DEPS_POLICY=warn → stderr only.
		// Strict (auto-block on standard violations) is the default at
		// commit time.
		id:          "precommit.sprint.cross_track_deps",
		description: "SPRINT.md cross_track_deps integrity check (referential integrity / cycles / kind enum / reason)",
		command: []string{"bash", "-c", `command -v hstl >/dev/null || { echo '[precommit.sprint.cross_track_deps] hstl not installed — skip' >&2; exit 0; }
out=$(HSTL_CROSS_TRACK_DEPS_POLICY=strict hstl track validate -o json 2>/dev/null) || { echo '[precommit.sprint.cross_track_deps] track validate failed to run' >&2; exit 0; }
ctd_errors=$(echo "$out" | jq -r '[.data.issues[] | select((.type | startswith("cross_track_deps_")) and (.severity == "error"))] | length' 2>/dev/null)
[ -z "$ctd_errors" ] && exit 0
if [ "$ctd_errors" -gt 0 ]; then
  echo "[precommit.sprint.cross_track_deps] cross_track_deps integrity violations: ${ctd_errors}" >&2
  echo "$out" | jq -r '.data.issues[] | select((.type | startswith("cross_track_deps_")) and (.severity == "error")) | "  - " + .type + ": " + .detail' >&2
  exit 1
fi
exit 0`},
		shouldRun: hasSprintMDFiles,
	},
	{
		// manifest-schema-contract.md §7 drift detection.
		// When skill SKILL.md files / commands/ / docs/08-references/standards/command-first-mapping.md
		// change, validate against the external profile YAML if it exists
		// via `hstl manifest validate`. Skip when the profile is absent
		// (the current project does not include one — opt-in).
		id:          "precommit.manifest.external.drift",
		description: "Detect drift between the external manifest profile (docs/08-references/standards/manifest-profile.yaml) and the actual CLI",
		command: []string{"bash", "-c", `command -v hstl >/dev/null || { echo '[precommit.manifest.external.drift] hstl not installed — skip' >&2; exit 0; }
profile="docs/08-references/standards/manifest-profile.yaml"
[ -f "$profile" ] || { echo "[precommit.manifest.external.drift] $profile absent — skip (opt-in)" >&2; exit 0; }
hstl manifest validate "$profile"`},
		shouldRun: hasSkillOrCommandsOrStandards,
	},
}

// ── shouldRun predicates ──────────────────────────────────────────────
// Equivalent to the bash hook's has_*_files branches.

// hasSkillOrCommandsOrStandards reports true when skill / commands /
// standards docs change. Used by precommit.manifest.external.drift.
func hasSkillOrCommandsOrStandards(staged []string) bool {
	for _, f := range staged {
		if strings.Contains(f, "/skills/") && strings.HasSuffix(f, ".md") {
			return true
		}
		if strings.Contains(f, "/commands/") && strings.HasSuffix(f, ".md") {
			return true
		}
		if strings.HasPrefix(f, "docs/08-references/standards/") {
			return true
		}
	}
	return false
}

func hasGoFiles(staged []string) bool {
	for _, f := range staged {
		if strings.HasPrefix(f, "cli/") && strings.HasSuffix(f, ".go") {
			return true
		}
	}
	return false
}

func hasSkillFiles(staged []string) bool {
	for _, f := range staged {
		if strings.HasPrefix(f, "skills/") {
			return true
		}
	}
	return false
}

func hasClaudeMdOrRules(staged []string) bool {
	for _, f := range staged {
		if f == "CLAUDE.md" || strings.HasPrefix(f, ".claude/rules/") {
			return true
		}
	}
	return false
}

func hasVersionFiles(staged []string) bool {
	for _, f := range staged {
		switch f {
		case ".claude-plugin/plugin.json", ".claude-plugin/marketplace.json", "CLAUDE.md":
			return true
		}
	}
	return false
}

func hasGoOrSkillFiles(staged []string) bool {
	return hasGoFiles(staged) || hasSkillFiles(staged)
}

func hasHarnessOrTaskMgmtSkill(staged []string) bool {
	for _, f := range staged {
		if f == "cli/pkg/db/schemas/harness_defaults.json" || f == "skills/task-management/SKILL.md" {
			return true
		}
	}
	return false
}

func hasHarnessOrSprintCompleteCmd(staged []string) bool {
	for _, f := range staged {
		if f == "cli/pkg/db/schemas/harness_defaults.json" || f == "commands/sprint/complete.md" {
			return true
		}
	}
	return false
}

// taskFilePathPattern matches works/(tasks|sprints/.../tasks)/T...md.
var taskFilePathPattern = regexp.MustCompile(`^works/(tasks|sprints/.*/tasks)/.*T\d+.*\.md$`)

func hasTaskFiles(staged []string) bool {
	for _, f := range staged {
		if taskFilePathPattern.MatchString(f) {
			return true
		}
	}
	return false
}

// sprintMDFilePattern matches works/sprints/{completed,active}/sprint-*/SPRINT.md.
var sprintMDFilePattern = regexp.MustCompile(`^works/sprints/(completed|active)/sprint-[0-9]+/SPRINT\.md$`)

func hasSprintMDFiles(staged []string) bool {
	for _, f := range staged {
		if sprintMDFilePattern.MatchString(f) {
			return true
		}
	}
	return false
}

// tracStandardsPattern matches docs/08-references/standards/trac-(catalog-*|contexts|members).yaml.
var tracStandardsPattern = regexp.MustCompile(`^docs/08-references/standards/trac-(catalog-[A-Z]{3,4}|contexts|members)\.yaml$`)

func hasTracStandardsFiles(staged []string) bool {
	for _, f := range staged {
		if tracStandardsPattern.MatchString(f) {
			return true
		}
	}
	return false
}

// precommitStagedFiles reads the staged-diff text from ctx.Extra and
// extracts the file list. When Extra is missing (e.g. audit mode), it
// falls back to running `git diff --cached --name-only` for fresh data.
// Defined as a variable so tests can stub it.
var precommitStagedFiles = func(ctx *RuleContext) []string {
	// 1) When a staged_diff Extra is present, parse paths from the diff
	//    headers.
	if diffAny, ok := ctx.Extra[ExtraKeyStagedDiff]; ok {
		if diff, ok := diffAny.(string); ok && diff != "" {
			return stagedFilesFromDiff(diff)
		}
	}
	// 2) Fallback — call git directly.
	out, err := exec.Command("git", "diff", "--cached", "--name-only").Output()
	if err != nil {
		return nil
	}
	var result []string
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		if line != "" {
			result = append(result, line)
		}
	}
	return result
}

// stagedFilesFromDiff extracts the `+++ b/<path>` paths from unified
// diff text.
func stagedFilesFromDiff(diff string) []string {
	var files []string
	for _, line := range strings.Split(diff, "\n") {
		if strings.HasPrefix(line, "+++ b/") {
			files = append(files, strings.TrimPrefix(line, "+++ b/"))
		}
	}
	return files
}

// ── precommitWrapperRule — common implementation for the 11 Rules ─────

type precommitWrapperRule struct {
	check precommitCheck
}

func (r precommitWrapperRule) ID() string                { return r.check.id }
func (r precommitWrapperRule) Category() string          { return precommitWrapperCategory }
func (r precommitWrapperRule) Description() string       { return r.check.description }
func (r precommitWrapperRule) DefaultSeverity() Severity { return SeverityBlock }

func (r precommitWrapperRule) Check(ctx *RuleContext) *RuleResult {
	start := time.Now()
	res := &RuleResult{RuleID: r.ID(), Severity: r.DefaultSeverity()}

	root := ctx.ProjectRoot
	if root == "" {
		res.Status = StatusSkipped
		res.Duration = time.Since(start)
		return res
	}

	// staged-files-conditional execution.
	if r.check.shouldRun != nil {
		staged := precommitStagedFiles(ctx)
		if !r.check.shouldRun(staged) {
			res.Status = StatusSkipped
			res.Duration = time.Since(start)
			return res
		}
	}

	// Verify that the project root actually exists (defends test
	// environments).
	if _, err := os.Stat(filepath.Join(root, ".git")); err != nil {
		res.Status = StatusSkipped
		res.Duration = time.Since(start)
		return res
	}

	// hotfix — when the command is ["bash", "scripts/XXX.sh"] and
	// scripts/XXX.sh is in the embed map, write it to a temp file and
	// invoke it by absolute path. Fixes the exit-127 problem in external
	// projects where the relative-path script does not exist.
	cmdArgs := make([]string, len(r.check.command))
	copy(cmdArgs, r.check.command)
	var tmpScript string
	if len(cmdArgs) >= 2 && cmdArgs[0] == "bash" {
		if _, ok := embeddedRuleScripts[cmdArgs[1]]; ok {
			path, err := writeEmbeddedScript(cmdArgs[1])
			if err != nil {
				res.Status = StatusError
				res.Message = r.check.description + " failed to prepare for execution"
				res.Evidence = []string{err.Error()}
				res.Err = err
				res.Duration = time.Since(start)
				return res
			}
			tmpScript = path
			cmdArgs[1] = path
			defer os.Remove(tmpScript)
		}
	}

	cmd := exec.Command(cmdArgs[0], cmdArgs[1:]...)
	cmd.Dir = root
	// Project-config.yaml precommit thresholds are injected as env vars.
	// Priority: env > project-config > default.
	if extra := buildPrecommitEnv(r.check.id); len(extra) > 0 {
		cmd.Env = append(os.Environ(), extra...)
	}
	if err := cmd.Run(); err != nil {
		res.Status = StatusViolated
		res.Message = r.check.description + " failed"
		res.Evidence = []string{err.Error()}
	} else {
		res.Status = StatusOK
	}
	res.Duration = time.Since(start)
	return res
}

func init() {
	for _, c := range precommitChecks {
		Register(precommitWrapperRule{check: c})
	}
}
