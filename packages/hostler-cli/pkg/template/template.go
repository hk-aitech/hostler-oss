// Package template provides standardised body templates for Tasks.
// The template emits English section headings and placeholder markers so the
// parsing pipeline (`HasPlaceholderBody`, audit scripts, etc.) can match them
// consistently across the OSS surface.
package template

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// RenderTaskTemplate composes a Task body from the standard template.
//
// taskID    : Task ID (e.g. "T009")
// title     : task title
// taskType  : feature/bugfix/refactor/infra/docs/test/chore/spike/hotfix
// sprint    : Sprint ID (e.g. "sprint-01") or empty for backlog
// priority  : p0–p3
// estimate  : XS/S/M/L/XL
// dependsOn : depending Task IDs (nil ok)
// summary   : one-line summary; empty falls back to a placeholder string.
func RenderTaskTemplate(
	taskID, title, taskType, sprint, priority, estimate string,
	dependsOn []string,
	summary string,
) string {
	today := time.Now().Format("2006-01-02")

	sprintValue := sprint
	if sprintValue == "" {
		sprintValue = "backlog"
	}

	if dependsOn == nil {
		dependsOn = []string{}
	}

	dependsOnJSON := mustMarshal(dependsOn)

	// YAML frontmatter title escape — guard against double-quote and newline.
	safeTitle := strings.ReplaceAll(title, `"`, `\"`)
	safeTitle = strings.ReplaceAll(safeTitle, "\n", " ")

	// Default text for the ## Summary section if none was supplied.
	summaryText := summary
	if summaryText == "" {
		summaryText = "{One-line summary: what / why / success criterion draft}"
	}

	// hotfix-only section: ## Rollback (required, prevents an empty section).
	hotfixRollback := ""
	if taskType == "hotfix" {
		hotfixRollback = `
## Rollback

{Rollback procedure for this hotfix. Files / commits / settings to revert. At least 20 characters.}
`
	}

	// bugfix-only section — three-axis (reproduction / root cause / resolution)
	// fixture template, aligned with the bugfix Harness Gate items
	// (reproduction / root_cause).
	bugfixFixture := ""
	if taskType == "bugfix" {
		bugfixFixture = `
## Reproduction

**Environment**: {OS / CLI version / relevant env vars}

**Reproduction steps**:
1. {Step 1 — exact command}
2. {Step 2}
3. {Observed: incorrect result}
4. {Expected: correct result}

**Logs / error messages**:
` + "```" + `
{full log — do not truncate}
` + "```" + `

## Root Cause

**Direct cause**: {code path + condition + result}

**Root cause**: {why such code existed — missing design / violated assumption / unhandled condition}

**Related files**: ` + "`path:line`" + `

## Resolution

**Direction**: {single-line fix / logic restructure / guard add / etc.}

**Alternatives**: {rejected alternatives and reasons — clarify selection}
`
	}

	// spike type ## Result sub-headings — narrative mode by default; sub-headings
	// are not file-path scan targets.
	var resultSection string
	if taskType == "spike" {
		resultSection = `## Result

> Spike result section. Authored at task complete time.
> All sub-headings are narrative (not file-path scan targets).

### Subject

{Technology / approach / library evaluated}

### Method

{How the evaluation was done — PoC, doc review, benchmark, comparison test, etc.}

### Conclusion

{Evaluation outcome and key findings}

### Recommendation

{Adopt / reject decision and follow-up recommendation}

### Commits

- {commit SHA}: {message}
`
	} else {
		resultSection = `## Result

> Author this section at task complete time. Three required sub-headings:
> Design Decisions / Artifacts / Verification. The parser only scans
> ` + "`### Artifacts`" + ` for file paths; backtick paths under
> ` + "`### Design Decisions`" + ` are treated as narrative and excluded.

### Design Decisions

{Major design choices and trade-offs in prose. Backtick file references are allowed — the parser treats them as narrative.}

### Artifacts

- {full file path} (added / modified / deleted)

### Verification

- Build: {result}
- Tests: {result}
- Other checks: {command + result}

### Commits

- {commit SHA}: {message}
`
	}

	return fmt.Sprintf(`---
id: %s
title: "%s"
type: %s
sprint: %s
status: todo
priority: %s
estimate: %s
depends_on: %s
created: %s
---

# %s %s

## Type Tags

> Tick at least one option below before sprint assignment so the scope is unambiguous.

- [ ] New structure (new types.go struct, new package, new CLI subcommand, etc.)
- [ ] Existing structure update (field additions, logic changes, doc edits, etc.)
- [ ] Measurement / verification / deployment (Dev verification, prod rollout, regression tests)
- [ ] Mixed (some new + some updates)

## Summary

%s

## Purpose

{The problem this Task solves or the goal it achieves}

## Requirements

> **Measure first**: do not write requirements from guesswork — verify the
> current state with grep / test commands first, then derive "what to do".
> Example: ` + "`grep -rn \"old_pattern\" src/`" + ` → N matches → enumerate replacement targets.
> Record measurements under ` + "`## References`" + ` or as sub-notes under requirements.
>
> **Interface signature change**: changing only the definition breaks the build.
> Run ` + "`grep -rn \"func .*) MethodName(\" packages/`" + ` to enumerate every
> implementer → register mock / ServiceAdapter / composition / call sites
> together as checkboxes → fix in one commit.
>
> **L/XL estimate measurement**: before assigning M-or-larger, measure the
> target file / line scope with grep first. If actual N is larger than expected,
> split into Phase 1/2 or scope a separate Task.

- [ ] {Requirement 1}
%s
## Done Criteria

- [ ] {Criterion 1}

## Scope Limits

### In This Sprint

- {Work items confirmed at sprint planning}

### Next Sprint or Later (Deferred)

- {Items deferred outside this Sprint — "none" if no items}
%s
## References

{Related documents, ADRs, reference implementations, etc.}

%s`,
		taskID, safeTitle, taskType, sprintValue,
		priority, estimate,
		dependsOnJSON, today,
		taskID, title,
		summaryText,
		bugfixFixture,
		hotfixRollback,
		resultSection,
	)
}

// mustMarshal serialises v as JSON and falls back to "[]" on failure.
func mustMarshal(v any) string {
	b, err := json.Marshal(v)
	if err != nil {
		return "[]"
	}
	return string(b)
}
