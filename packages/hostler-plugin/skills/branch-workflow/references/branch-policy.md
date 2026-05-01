# Branch Policy (Shared default template)

> **Default template bundled with hostler-plugin.** Each project may define
> its own SSOT to override or customize this template.
> Conventional SSOT location: `docs/08-references/standards/branch-policy.md`
> or `.claude/skills/<project>-policy/references/branch-policy.md`.
>
> This file ships with the plugin and stays project-neutral.

## Topology (default)

```
sprint-{id}  →  dev  →  main
(work branch)   (integration)   (Prod-only)
```

| Branch | Purpose | Lifetime | Merge source | Protection |
|--------|---------|----------|--------------|------------|
| `sprint-{id}` | Sprint-only work | Branch from `dev` at Sprint start, merge back to `dev` and delete locally on completion | none | low |
| `dev` | Integration — accepts Sprint completion + bake-time before promotion | persistent | `sprint-{id}` / `hotfix-*` back-merge | medium |
| `main` | Prod — externally visible stable baseline | persistent | `dev` / `hotfix-*` | highest |
| `hotfix-{ISSUE}-{slug}` | Emergency fix — prod incident response | Branch from `main`, merge into both `main` and `dev`, then delete | none (branched from main) | medium |

## Standard Workflow

### Sprint work

```bash
# Sprint start
git checkout dev
git pull
git checkout -b sprint-{id}

# Work — 1 Task = 1 Commit
git add <specific files>
git commit -m "feat(T{id}): ..."

# Sprint complete
git checkout dev
git merge sprint-{id} --no-ff -m "merge: sprint-{id} → dev"
git push origin dev
git branch -d sprint-{id}
```

### main merge (manual)

`dev → main` merges are **never automatic per Sprint**. Perform manually only
when the trigger defined by the project is met (just before external release /
major feature milestone / N accumulated Sprints / etc.):

```bash
git checkout main
git pull
git merge dev --no-ff -m "merge: dev → main ({one-line context})"
git push origin main
```

> **Terminology note**: this action is named "main merge" / "dev → main merge".
> Do not use the term "Milestone promotion" — "Milestone" belongs to advance
> planning (roadmap, GitLab milestone) and is not a name for a git branch
> policy action.

### Hotfix (prod incident)

```bash
# Start
git checkout main
git pull
git checkout -b hotfix-{ISSUE}-{slug}

# Fix + commit
git commit -m "fix(T{id}): hotfix — {summary}"

# Merge into both sides
git checkout main
git merge hotfix-{ISSUE}-{slug} --no-ff
git push origin main

git checkout dev
git merge hotfix-{ISSUE}-{slug} --no-ff    # back-merge required
git push origin dev

git branch -d hotfix-{ISSUE}-{slug}
```

**Hotfix branch rules (default)**:

| Change size | Branch | Reason |
|-------------|--------|--------|
| XS / S | direct commit on `dev` allowed | isolation unnecessary |
| M or larger | `hotfix-{ISSUE}-{slug}` required | review / rollback ease |

> After `hotfix → main` is applied, **do not skip the `hotfix → dev` back-merge** — this is the #1 cause of drift.

## main merge rules (enforced)

- **Only `dev → main` merges are allowed.** Direct merges from `sprint-{id}` to main are forbidden.
- **`hotfix → main` is allowed as an exception** — but the `hotfix → dev` back-merge immediately afterward is required.
- If the project runs a CI `validate-main-source` job, this can be enforced
  with a `git merge-base --is-ancestor origin/dev HEAD` gate.

## Naming separator flexibility

| Separator | Example | Note |
|-----------|---------|------|
| Hyphen `-` | `hotfix--harnessdefaults-missing` | This template's default |
| Slash `/` | `hotfix/-harnessdefaults-missing` | Git flow family convention |

hostler-plugin commands accept either separator with a permissive regex
(`^hotfix[-/]`). If the project SSOT specifies one, follow that convention.

## Commit conventions (related to branch policy)

1. **Body in Korean** — only the `Co-Authored-By:` line allows English (project default)
2. **Task ID required** — format `<type>(Txxx): ...`. Enforced by the `commit_msg_shape` gate
3. **1 Task = 1 Commit** — commit only at task completion, no intermediate commits
4. **Type prefixes**: `feat:` · `fix:` · `docs:` · `refactor:` · `infra:` · `test:` · `chore:` · `merge:`

## Branch state inspection

```bash
# dev / main delta
git log --oneline origin/main..origin/dev | wc -l

# Current branch sync state (auto-surfaced by session-start hook)
git rev-list --left-right --count origin/dev...HEAD
```

## Per-project customization guide

This template is just the default. A project may override the following in its
own SSOT (inside the repo):

- **main merge trigger** — pick from "just before external release / N accumulated Sprints / manual judgement"
- **Naming separator** — hyphen `-` vs slash `/`
- **Hotfix size threshold** — at which of XS/S/M to require a hotfix branch
- **CI gate** — whether to adopt `validate-main-source`
- **Protected branch policy** — MR approval count and force-push blocking on main/dev

If the project SSOT conflicts with this template, **the project SSOT wins**.

## Related

- hostler-plugin related skills: `sprint-management` · `hotfix-workflow` · `getting-started` · `session-management` · `project-management`
- hostler-plugin related commands: `commands/sprint/start.md` · `commands/sprint/complete.md` · `commands/hotfix/start.md` · `commands/hotfix/complete.md`
