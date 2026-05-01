---
description: Start a hotfix branch — branch from main, remember the original branch, and validate the naming convention. Use when the user requests an "urgent fix", says "start a hotfix", or needs an out-of-Sprint emergency patch.
argument-hint: [issue-id] [slug]
allowed-tools: Bash(git:*)
---

# Start Hotfix: $ARGUMENTS

> **Command-first entry point**: official entry for fixing urgent defects outside a Sprint.
> When the user says "start a hotfix" or "urgent fix", do not call git directly —
> use this Command first to apply branch naming, branch-point validation, and original-branch
> bookkeeping, then run the git commands.

> **Command First STRICT**: This Command is the official entry point for hotfix
> initiation. Running `git checkout -b hotfix-...` directly via Bash skips branch-name
> validation and original-branch bookkeeping side effects, and a `ceremony.bypass`
> event is auto-recorded in the audit log. The main branch-point is required for hotfixes —
> bypassing is not allowed. Policy SSOT: `CLAUDE.md` §Command First mapping table.

Handles urgent defects outside the Sprint on a **main-based hotfix branch**.

> **Branch policy**: Naming and branch-point follow the shared template
> `skills/branch-workflow/references/branch-policy.md` (default:
> `hotfix-{ISSUE}-{slug}` hyphen-separated, branched from main). A project's own SSOT takes precedence.

## Arguments

| Argument | Description | Example |
|------|------|------|
| issue-id | Linked Task ID or identifier | `T123`, `panic-cli` |
| slug | 3~5 word kebab-case summary | `harnessdefaults-missing` |

Resulting branch name: `hotfix-{issue-id}-{slug}` (SSOT default; if a project SSOT differs, it wins)

## Automated steps

### 1. Remember the original branch

Records the current branch name in `.hostler/.hotfix-origin` so that `/hstl-oss:hotfix:complete`
can use it as the return point. If the file already exists, the previous hotfix is incomplete — WARN.

```bash
CURRENT=$(git branch --show-current)
[ -f .hostler/.hotfix-origin ] && echo "[WARN] previous hotfix incomplete — inspect .hostler/.hotfix-origin manually"
mkdir -p .hostler
echo "$CURRENT" > .hostler/.hotfix-origin
```

### 2. Refresh main and branch off

```bash
git fetch --all --prune
git checkout main
git pull --ff-only
git checkout -b "hotfix-{issue-id}-{slug}"
```

If fast-forward fails, main has diverged from local — instruct the user to resolve manually and abort.

### 3. Validate branch naming convention

- Format: `^hotfix-[A-Za-z0-9_-]+$` (SSOT default; if the project SSOT uses a slash separator, `^hotfix/[A-Za-z0-9_-]+$`)
- Avoid reserved names: `hotfix-main`, `hotfix-dev`, `hotfix-master` are forbidden
- Prevent duplicates: pre-check with `git show-ref --verify --quiet refs/heads/hotfix-{slug}`

## Output format

```markdown
=============================================
  Hotfix branch created
=============================================

  Branch:        hotfix-{issue-id}-{slug}
  Branch point:  main @ {main HEAD SHA}
  Original:      {user's previous branch} (recorded in .hostler/.hotfix-origin)

  Next steps:
  1. Apply the fix and run git commit -m "hotfix({issue-id}): description"
  2. When done, run /hstl-oss:hotfix:complete
=============================================
```

## Caveats

- **Abort if main is out of sync**: `pull --ff-only` failure means main is dirty or
  local has diverged. Hotfixes must start from a clean main. Tell the user to inspect with
  `git fetch && git log HEAD..origin/main`.
- **Check staged files**: staged/untracked files on the current branch may cause
  checkout to fail. Pre-check with `git status` and stash if needed.
- **Task linkage (optional)**: post-hoc Task registration with
  `hstl-oss task create --type hotfix --title "..."` is recommended (`register-all-findings` principle).

## Related Commands

- `/hstl-oss:hotfix:complete` — main merge + dev back-merge + return to original branch
- `/hstl-oss:task:create` — register Task post-hoc (type=hotfix)

Detailed workflow: `skills/hotfix-workflow/SKILL.md`

