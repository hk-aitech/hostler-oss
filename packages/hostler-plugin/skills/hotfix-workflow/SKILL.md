---
name: hotfix-workflow
description: |
  The hotfix branch workflow — branch from main, project SSOT defines the
  branch naming convention, then on completion merge to main, back-merge to
  dev, and return to the original branch. Use this skill whenever the user
  mentions "hotfix", "urgent fix", "production incident", "branched from
  main", "urgent defect", "emergency fix", "emergency patch", "hotfix start",
  "hotfix complete", "back-merge", or anything about an emergency fix outside the
  current Sprint scope — even when they don't explicitly say "hotfix". Also
  invoke after a production-incident or post-release defect is mentioned. Do
  NOT use for ordinary Sprint Tasks (use task-management).
allowed-tools: [Read, Write, Edit, Bash, Glob, Grep]
paths: ["works/tasks/**/*", "docs/02-architecture/**/*", "docs/architecture/**/*"]
---

# Hotfix workflow

Fix urgent defects discovered during a Sprint on a **hotfix branch based on
main**, then integrate via main merge → dev back-merge → return to the
original Sprint branch.

> **Branch policy**: detailed naming, branching, and merge rules live in
> `../branch-workflow/references/branch-policy.md` (the shared default
> template). If the project defines its own SSOT, the project SSOT wins.

## Terminology

- ✅ Use **hotfix** consistently
- ❌ `quickfix`, `quick-fix`, `emergency-fix`, `urgent-fix` are forbidden

## When to use

| Situation | Response |
|-----------|----------|
| Urgent defect found during a Sprint (e.g. missing file / package / registration to restore) | hotfix workflow |
| Bug planned within the current Sprint | regular Sprint Task (`/hstl-oss:task:start`) |
| Post-release production incident | hotfix workflow (+ release tag reset) |
| Code cleanup / refactoring | regular Sprint Task |

A hotfix is **off the Sprint path**. It runs independently of Sprint
BACKLOG / active Tasks and bypasses Sprint ceremony.

## Branch conventions

### Naming

Default convention (recommended when the project SSOT is undefined):

```
hotfix-{ISSUE_ID}-{slug}

Examples:
  hotfix-{TASK_ID}-{slug}                    # defect with a tracked Task ID
  hotfix-ISS{YYYYMMDD}{NNN}-{slug}           # external issue format
  hotfix-panic-cli-startup                   # immediate response without an ISSUE identifier
```

`{ISSUE_ID}` is a Task ID or a short identifier. `{slug}` is 3–5 words in
kebab-case. The separator (hyphen `-` or slash `/`) follows the project
SSOT.

### Branching point

**Always branch from `main`** (not `dev`).

```bash
git fetch --all
git checkout main
git pull --ff-only
git checkout -b hotfix-{ISSUE_ID}-{slug}
```

Reason: a hotfix is by definition "a fix to the current production (= main)".
Branching from dev mixes in unmerged Sprint work, breaking release
independence.

## Procedure

### 1. Create the hotfix branch

```
/hstl-oss:hotfix:start <ISSUE_ID> <slug>
```

Internally, the original branch is remembered, then we move to main and
create the new branch.

### 2. Commit the fix

```bash
# fix
git add -A
git commit -m "hotfix(<ISSUE_ID>): description"
```

The commit prefix is fixed at **`hotfix`**. Reason: distinguishes from
regular `fix` when parsing release notes.

### 3. Complete + integrate

```
/hstl-oss:hotfix:complete
```

What happens internally:
1. Merge hotfix branch → main (fast-forward or no-ff merge commit)
2. Back-merge main → dev (so the hotfix is reflected in the in-progress Sprint)
3. Return to the original branch we saved earlier
4. Delete the hotfix branch (optional)

## Hotfix Task registration principle

If you fixed code via the hotfix workflow, **post-hoc Task registration** is
recommended:

- Track the completed hotfix commit by Task ID
- `hstl-oss task create --type hotfix` — use the Task type `hotfix`
- Write the Task body even though the fix is already done, then immediately mark task complete

This matches the principle "every finding gets recorded as a Task" (preserves
post-hoc traceability).

## Example scenario — restoring missing files

While working you discover that a package file from a previous unit / earlier
release was missing. It's outside the current scope and urgent, so apply the
hotfix workflow:

```bash
# Branch from main — the discovery happens on the Sprint branch, but the fix is based on main
git checkout main
git pull --ff-only
git checkout -b hotfix-{ISSUE_ID}-package-x-restore

# Add + commit the missing files
git add <repo>/pkg/<pkg-x>/
git commit -m "hotfix(<ISSUE_ID>): restore missing package X"

# Merge to main + back-merge to dev + return to original branch
git checkout main && git merge --no-ff hotfix-{ISSUE_ID}-package-x-restore
git checkout dev && git merge --no-ff main
git checkout sprint-{N} && git merge --no-ff dev
git branch -d hotfix-{ISSUE_ID}-package-x-restore
```

Recommended: all of the steps above are performed automatically by the
`/hstl-oss:hotfix:start` + `/hstl-oss:hotfix:complete` slash commands.
Manual execution is the fallback when the commands aren't deployed.

## Cautions

- **No force push**: never roll back a hotfix already merged into main.
  If you make a mistake, correct it with a revert commit.
- **Avoid leaking secrets**: hotfixes are emergencies, so diff review tends
  to slip. Re-verify whether `.env`, credentials.json, etc. are staged.
- **Don't skip the back-merge**: merging only to main and not updating dev
  causes a conflict at the next Sprint merge.

## Pre-completion secondary verification checklist

Hotfixes can fix the surface symptom while missing a downstream second-order
defect (which then resurfaces a week later in a different environment).
Right before hotfix completion (before merging to main), empirically verify
the following 3 items:

1. **grep related registry / action / external-script execution paths**:
   verify that downstream files / APIs / rule actions referenced by the
   restored or fixed component actually exist on the deployment path
2. **Cross-verify in other environments / external projects**: if more than
   one environment uses this plugin, run the hotfixed feature in each.
   Verify not just the surface symptom but **end-to-end normal return**.
3. **Add at least one regression test**: reproduce the failure first
   (failing-test-first) and then verify the fix.

Without these checks — the first hotfix resolves the surface symptom but
masks a downstream second-order defect, which then resurfaces 1–2 Sprints
later.

## Response interpretation conventions (AI-friendly)

- **Success detection**: if the JSON response's `status` field is `"ok"` /
  `"success"`, or core result fields like `new_status` / `task_id` /
  `work_ticket` are present, **confirm success**
- **No re-invocation**: after success is confirmed, **do not re-invoke** the
  same command. "Already done / active / completed" errors are mostly caused
  by re-invocation. Re-invocations log a `*.retried` event in the audit
- **On failure**: check the JSON `error_category` + `recovery_hint`, take
  action, then re-invoke. Look up `error_category` values in the hostler CLI
  error-category catalog (project standard doc)

## Retry rules (idempotent)

- **No retry on "already done / active / completed"**:
  `error_category=ALREADY_IN_STATE` → **ignore and proceed to the next step**
- **Idempotent commands**: `hstl-oss harness check` / `hstl-oss harness auto-check` / `hstl-oss backlog sync`
- **Non-idempotent commands**: `hstl-oss task create` / `hstl-oss sprint create` / `hstl-oss task complete`

## Response table by BLOCKED reason

| Situation | Cause | Concrete response |
|-----------|-------|-------------------|
| Branch name violates the convention | Missing hotfix- prefix or contains whitespace | Recreate as `hotfix-{ISSUE_ID}-{slug}` (details: `../branch-workflow/references/branch-policy.md`) |
| main sync fails (pull --ff-only) | local main diverged from origin/main | `git fetch && git log HEAD..origin/main` to confirm cause → manual rebase / merge, then retry |
| `.hostler/.hotfix-origin` already exists | Previous hotfix incomplete | Verify state, then either remove or finish with `/hstl-oss:hotfix:complete` |
| precommit.branch.sync BLOCK | dev divergence ≥ 30 | hotfix-* branches are auto-downgraded on the sync gate. If still BLOCKed, set `HSTL_BRANCH_SYNC_STRICT=0` to bypass once (with explicit user approval) |

## Related documents

- **Branch policy (shared template)**: `skills/branch-workflow/references/branch-policy.md`
- `commands/hotfix/start.md` — hotfix branch start entry point
- `commands/hotfix/complete.md` — main merge + back-merge + branch-restoration entry point
- Project user guide §hotfix — top-level project summary (per-project)
- `skills/task-management/SKILL.md` — boundary with regular Sprint Tasks
