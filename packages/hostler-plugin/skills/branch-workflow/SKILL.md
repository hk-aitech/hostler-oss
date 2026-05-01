---
name: branch-workflow
description: Use this skill whenever the user asks about Git branch strategy, merge policy, hotfix workflow, sprint branch lifecycle, main/dev/develop topology, back-merges, whether to use release branches, feature flag vs branch tradeoffs, or comparisons of the four industry-standard workflows (Gitflow / GitHub flow / GitLab flow / Trunk-based). This is also the source of truth when documenting a project's branch policy or when other skills (sprint-management, hotfix-workflow, getting-started, session-management, project-management) need to reference common branch conventions. Out of scope: VCS systems other than Git, and basic single-command git usage.
---

# Branch Workflow — Git Branch Strategy Reference

The right Git branch strategy depends on team size, release cadence, and
automation maturity. This skill **compares the four industry-standard strategies
to justify the choice**, and documents the structure and rationale of the
"Sprint-Aware GitHub flow" that hostler-plugin adopts by default. The actual
operational conventions live in `references/branch-policy.md`.

## When to use this skill

The skill is referenced automatically in any of these contexts:

- Choosing a branch strategy for a new project — Gitflow? GitHub flow? Trunk-based?
- Designing a hotfix procedure — branch from main? how to back-merge?
- Discussing Sprint branch lifecycle (where to branch from, where to merge back)
- When other skills/commands (sprint:start, hotfix:start, etc.) reference conventions
- When you need to explain industry-standard tradeoffs to the team
- When writing or updating project SSOT docs (e.g. `docs/.../branch-policy.md`)

## Why this skill exists as a standalone reference

Sprint branches, hotfix branches, main merges, and back-merges are **not the
exclusive responsibility of any single skill**. Both sprint-management and
hotfix-workflow need to reference the same policy. Putting it under one skill's
references would force the other skill to reach across with awkward relative
paths. Splitting it out as a **shared reference slot** means:

- sprint-management, hotfix-workflow, etc. all point to one place
- Updating branch policy doesn't require touching multiple skills
- The industry-standard comparison and rationale is written exactly once

So this skill is a **reference hub, not a policy SSOT**. The actual string-level
conventions live in `references/branch-policy.md`. This SKILL.md focuses on
"which standard we picked and why".

## Comparison of the four industry standards

| Strategy | Layers | Branch lifetime | Best fit | Current assessment |
|----------|--------|-----------------|----------|--------------------|
| **Gitflow** (Driessen, 2010) | main + develop + feature/* + release/* + hotfix/* | feature long-lived, release short-lived | Scheduled releases, multi-version parallel | Legacy. Atlassian official: "trunk-based approaches are now recommended" |
| **GitHub flow** (GitHub, 2011) | main + short-lived feature | feature ~1 week | CD, SaaS, single-version | Modern standard. Concise but no staging separation |
| **GitLab flow** (GitLab) | main + pre-prod + production environment branches **or** main + release-N release branches | environment/release persistent | Staging/production split operations | The upstream-first principle is useful. Fowler does not recommend environment branches |
| **Trunk-based** | Single main (trunk) + 24h short-lived branches | <2 days | High-frequency CI/CD, large scale (Google monorepo, 35K developers) | Modern leader. Feature flags required |

### One-line summary of each strategy

**Gitflow** — Rich layering / limited CD compatibility. The long-lived
`develop` branch drifts from `main` and produces merge hell — a structural flaw
that's been widely criticized.

**GitHub flow** — main + topic branch. PR review → merge. Light and agile.
Lacks a separated staging environment, so it falls short for teams that need a
"bake time" before release.

**GitLab flow** — GitHub flow + environment branches (pre-prod → production)
or release branches. **upstream-first**: patches land on main first, then
propagate to lower environments. Reverse-direction patches (production→main)
are forbidden. Even production incident fixes start upstream.

**Trunk-based** — Single trunk + short-lived feature + feature flags. Forced
24h integration → eliminates merge hell at the root. The core discipline is
"every developer commits to trunk at least once a day" (recommended by
`Continuous Delivery` and `DevOps Handbook`).

## Martin Fowler's higher-order principle

Fowler emphasizes that **integration frequency** matters far more than the
choice of branching strategy
(`martinfowler.com/articles/branching-patterns.html`).

> "Integration frequency matters more than integration strategy. The choice
> between feature branching and continuous integration is less important than
> how often you integrate."

Concrete recommendations:

- **Healthy Branch**: every commit passes automated build + tests
- **Mainline Integration**: integrate to mainline at least once a day (= continuous integration)
- **Environment branches are not recommended** — handle env-specific config via build-time variables
- **Release branches** should be temporary only. Long-lived release branches accumulate drift and lose their purpose
- **Hotfix branches** are a valid pattern. Branch from main + merge into both sides (main, mainline) is required

## hostler-plugin default — "Sprint-Aware GitHub flow"

3 layers + a hotfix exit. Keeps the simplicity of GitHub flow while adding
the time boundary of a Sprint.

```
sprint-{id}  →  dev  →  main         (+ hotfix-{ISSUE}-{slug})
(work)         (integration)  (Prod)
```

### Rationale per element

| Element | Rationale |
|---------|-----------|
| `main` Prod-only | Adopts the GitHub flow `main` + Gitflow `master` convention. The externally visible baseline |
| `dev` integration branch | Similar to Gitflow `develop` but **without release branches**. The Sprint boundary substitutes for release units |
| `sprint-{id}` | GitHub flow's short-lived feature + a Sprint-sized lifetime (days to weeks) |
| `hotfix-{ISSUE}-{slug}` | Fowler's Hotfix Branch pattern verbatim. Branch from main + merge into both sides |
| No environment branches | Follows Fowler's "environment branches not recommended" |
| No release branches | Influenced by trunk-based. Release tracking via main tags is sufficient |

### What this combination solves

It avoids the long-lived `develop` drift of Gitflow while giving plain GitHub
flow what it lacks — a "validation window before Sprint completion" provided
by `dev`. Trunk-based's mandatory 24h integration was judged excessive for a
1-person + AI-pair scale, so it's relaxed to Sprint cadence. The dev→main
merge stays manual (triggered at external release / major feature milestones)
to protect the stability of main.

## Actual conventions live in references/branch-policy.md

Once you've reviewed the rationale above, read `references/branch-policy.md`
for actual commands, naming, and merge order. **If a project defines its own
SSOT, the project SSOT overrides this template.** This template is the default
for projects that haven't defined their own policy.

## Skills / commands that reference this skill

| Reference site | Purpose |
|----------------|---------|
| `skills/sprint-management/SKILL.md` | Sprint branch lifecycle (branch from dev / merge to dev) |
| `skills/hotfix-workflow/SKILL.md` | Hotfix branch conventions (branch from main / merge both sides) |
| `skills/getting-started/SKILL.md` | 3-layer overview during new-user onboarding |
| `skills/session-management/SKILL.md` | session-start branch state interpretation |
| `skills/project-management/SKILL.md` | One axis in the project conventions list |
| `commands/sprint/start.md` · `complete.md` | Sprint branch/merge entry points |
| `commands/hotfix/start.md` · `complete.md` | Hotfix entry points |

## References (industry standard sources)

- Atlassian Gitflow: `https://www.atlassian.com/git/tutorials/comparing-workflows/gitflow-workflow`
- GitHub flow: `https://docs.github.com/en/get-started/using-github/github-flow`
- GitLab flow: `https://docs.gitlab.com/topics/gitlab_flow/`
- Trunk-based development: `https://trunkbaseddevelopment.com/`
- Martin Fowler — Branching Patterns: `https://martinfowler.com/articles/branching-patterns.html`
- git-scm official workflows: `https://git-scm.com/docs/gitworkflows`
