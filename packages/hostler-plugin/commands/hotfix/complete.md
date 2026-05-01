---
description: Complete a hotfix — merge into main, back-merge to dev, return to original branch, and clean up. Use when the user finishes hotfix work and asks to "complete the hotfix", "merge the hotfix", or "wrap up the urgent fix".
argument-hint: [--keep-branch]
allowed-tools: Bash(git:*)
---

# Complete Hotfix: $ARGUMENTS

> **Command-first entry point**: After `/hstl-oss:hotfix:start`, always integrate
> through this Command. Do not call `git merge` directly — follow this Command's procedure.

> **Command First STRICT**: This Command is the official entry point for hotfix
> completion. Calling `git merge` / `git push` directly via Bash skips parts of the
> three-step main merge → dev back-merge → original-branch return, and a `ceremony.bypass`
> event is auto-recorded in the audit log. Both-side merging is required for hotfixes —
> bypassing is not allowed. Policy SSOT: `CLAUDE.md` §Command First mapping table.

Merges the hotfix branch into main, back-merges into dev, and returns to the original branch.

> **Branch policy**: Merge order and the back-merge obligation follow the shared
> template `skills/branch-workflow/references/branch-policy.md`. A project's own SSOT takes precedence.

## Preconditions

- Current branch matches `hotfix-*` or `hotfix/*` (per SSOT convention)
- Working tree is clean (no staged/unstaged changes)
- `.hostler/.hotfix-origin` file exists (recorded by `start`)

## Automated steps

### 1. Pre-checks

```bash
CURRENT=$(git branch --show-current)
[[ ! "$CURRENT" =~ ^hotfix[-/] ]] && { echo "Current branch is not in hotfix-* or hotfix/* form"; exit 1; }
[ -z "$(git status --porcelain)" ] || { echo "Working tree dirty — commit/stash and retry"; exit 1; }
[ ! -f .hostler/.hotfix-origin ] && { echo ".hostler/.hotfix-origin missing — return point unknown"; exit 1; }
ORIGIN=$(cat .hostler/.hotfix-origin)
```

### 2. Merge into main (--no-ff merge commit)

```bash
git fetch --all --prune
git checkout main
git pull --ff-only
git merge --no-ff "$CURRENT" -m "merge: hotfix $CURRENT -> main"
```

Why `--no-ff`: easier to track hotfixes as a unit in release notes.

### 3. Back-merge into dev

```bash
git checkout dev
git pull --ff-only
git merge --no-ff main -m "merge: main (hotfix) -> dev"
```

Skipping the back-merge causes merge conflicts in the next Sprint. **Do not skip.**

### 4. Return to the original branch

```bash
git checkout "$ORIGIN"

# If the original branch is a Sprint branch, merge dev changes back in
if [[ "$ORIGIN" =~ ^sprint- ]]; then
    git merge --no-ff dev -m "merge: dev (hotfix applied) -> $ORIGIN"
fi

rm -f .hostler/.hotfix-origin
```

### 5. Clean up the hotfix branch

```bash
# Default: delete
[ "$1" != "--keep-branch" ] && git branch -d "$CURRENT"
```

Use `--keep-branch` to preserve the branch (for audit/reference).

## Output format

```markdown
=============================================
  Hotfix integration complete
=============================================

  Merge path:
  {hotfix branch} -> main OK ({main merge SHA})
                  -> dev back-merge OK ({dev merge SHA})
                  -> {original branch} applied OK

  Current branch: {original branch}
  Branch cleanup: hotfix-* deleted (or kept)

  Next steps:
  - Run git push individually if needed (main / dev / original branch)
  - Register a Task: hstl-oss task create --type hotfix --title "..."
=============================================
```

## Caveats

- **Push is manual**: this Command does not run `git push` automatically. The user
  pushes each branch individually — prevents accidents.
- **No force push**: main/dev are shared branches. Force-pushing after a hotfix
  merge risks losing other people's work. Resolve divergence with additional merge commits.
- **Conflict resolution is manual**: when main/dev merge conflicts arise, abort the
  Command, resolve with `git mergetool` etc., then `git merge --continue` and proceed manually.

## Related Commands

- `/hstl-oss:hotfix:start` — create the hotfix branch (precursor)
- `/hstl-oss:task:create` — register the Task post-hoc (type=hotfix)

Detailed workflow: `skills/hotfix-workflow/SKILL.md`

