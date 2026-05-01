# Hands-On Lab — Build a Simple Todo Webapp with Hostler

End-to-end walkthrough that takes a brand-new idea ("I want a simple
todo app") all the way to a working SvelteKit application — through
research, planning, design, and a complete Sprint — without leaving the
Claude Code chat.

The lab is also a guided tour of the gates hostler-oss enforces:
research-before-scope, design-readiness before sprint start, the
Harness Gate at task complete, and the 10-Phase Cascade at sprint
complete. Pay attention to where the system pushes back — that is
where the value is.

## Time

About 60–90 minutes if you keep moving; longer if you stop to read
each artefact (recommended on a first run).

## Prerequisites

- `hstl-oss` CLI on PATH — verify with `hstl-oss --help`. If missing,
  run `/hstl-oss:cli:install` or follow `installation.md`.
- Claude Code with the hostler-oss plugin enabled. Add the marketplace
  with `/plugin marketplace add https://github.com/hk-aitech/hostler-oss`
  if needed.
- Node.js 20+ and npm (only required if you want to actually run the
  resulting SvelteKit app).
- An empty parent directory (e.g. `~/projects/`) where the new
  `todo-webapp/` will live.

## Pacing

Each step below is a single prompt that you paste into Claude Code.
Do **not** paste them all at once. Finish one, glance at the artefacts
it produced, then move on. The point of the lab is to feel each gate,
not to race through it.

---

## Step 1 — Research the problem space

**Goal**: scope the feature set by surveying public software (mobile
apps and mobile web), then producing a competitive-comparison report
and an MVP feature list.

**What you learn**: how `/deep-research` runs Perspective Mining,
parallel WebSearch, Knowledge-Gap loops, Chain-of-Verification, and
Grounding Pass — all in a single skill invocation.

**Prompt**:

```
/deep-research Let's gather the requirements for building a simple todo webapp in SvelteKit. Reference existing public software — especially mobile apps and mobile web apps — and produce a report for developing a minimal todo webapp that includes only the truly essential features.
```

**Expect to see**: a research plan up front (Step 1.5 gate), then a
structured report covering the comparison matrix (Apple Reminders /
Microsoft To Do / Todoist / TickTick / Things / Vikunja / etc.), MoSCoW
classification, mobile-UX patterns, SvelteKit-specific notes, and the
P0 / P1 / P2 feature list.

---

## Step 2 — Bootstrap the project skeleton

**Goal**: stand up the standard hostler folder layout and persist the
research report inside it; then write the Project Definition Document
(PDD) and the Roadmap.

**What you learn**: where each kind of document lives in the hostler
standard (`docs/00-project/`, `docs/04-guides/`, `docs/06-reports/`,
`works/sprints/`, etc.) and the `project-structure` skill that owns
the layout SSOT.

**Prompt**:

```
Create the project structure according to the hostler standard (project-structure skill), save the research report into the reports folder, and then author the PDD and Roadmap documents.
```

**Expect to see**:

- Project root scaffolded (`docs/`, `works/`, `archive/`, `tests/`,
  `src/`, `.gitignore`, `README.md`).
- `docs/06-reports/` contains the Step 1 research report.
- `docs/00-project/pdd.md` — Vision, Goals, Scope, Stakeholders,
  Constraints, Risks, Tech stack, Milestones.
- `docs/00-project/roadmap.md` — Phases, milestones, dependencies.

---

## Step 3 — Design layer

**Goal**: produce the architecture, requirements, wireframes, and ERD
documents that feed Sprint planning.

**What you learn**: hostler's design taxonomy — `01-requirements/`
(SRS / FRS / NFR / RTM), `02-architecture/` (ADR-NNN), `03-design/`
(component/UI/data designs).

**Prompt**:

```
Author the architecture document, requirements specification, wireframes, ERD, and related design documents.
```

**Expect to see**:

- `docs/01-requirements/` — `SRS.md`, `FRS.md`, `NFR.md`, `RTM.md`.
- `docs/02-architecture/ADR-001-...md` capturing the major
  architectural decisions (e.g. SvelteKit + IndexedDB + idb).
- `docs/03-design/` — component layout, wireframes, ERD.

---

## Step 4 — Generate CLAUDE.md

**Goal**: produce the project-level guide that conditions every future
Claude session — slim, structured, and within the 200/300/60-line
targets.

**What you learn**: the `claude-md-audit` skill — how it diagnoses
size / structure issues and proposes a migration matrix that pushes
material into `.claude/rules/` or `docs/` rather than letting
`CLAUDE.md` bloat.

**Prompt**:

```
Generate CLAUDE.md using the claude-md-audit skill.
```

**Expect to see**: a `CLAUDE.md` at the project root referencing
existing skills and rules (Command First, 1 Task = 1 Commit, branch
strategy, language policy) without duplicating their bodies.

---

## Step 5 — Confirm the CLI surface

**Goal**: ground the next step (Sprint planning) in the actual CLI
surface rather than memory.

**What you learn**: where to look up CLI flags and option names before
relying on them. The manifest is the SSOT — disagreement with any
guide means the CLI wins.

**Prompt**:

```
Use `hstl-oss manifest --help` to confirm the CLI tool usage.
```

**Expect to see**: the printed manifest, plus a brief read-through that
flags the commands you'll use in Step 7 (`hstl-oss task create`,
`hstl-oss task update`, `/hstl-oss:sprint:create`, etc.).

---

## Step 6 — Initialize the git repository

**Goal**: capture every artefact produced so far in a single Initial
Commit, so the Sprint that follows starts on a clean, traceable
history.

**What you learn**: hostler's audit chain begins at git init. After
this point, `1 Task = 1 Commit` and the hooks built into the CLI rely
on a real working tree.

**Prompt**:

```
git init
```

After init, stage everything from Steps 2–4 and create an initial
commit (Conventional Commits style):

```bash
git add .
git commit -m "chore: bootstrap project structure + design docs"
```

**Expect to see**: an empty repo on the default branch, a single
"chore: bootstrap …" commit containing the entire design layer.
`works/` is empty (Sprints arrive in the next step).

> **Why now and not earlier or later?** Earlier (before Step 2) there
> is no project to track. Later (after Sprint planning) the Sprint
> Tasks would land outside any audit chain. Step 6 is the natural
> seam: the design phase has frozen, but execution has not yet begun.

---

## Step 7 — Plan the Sprint

**Goal**: turn the PDD / Roadmap / design documents into a single
executable Sprint, broken into 5–10 Tasks.

**What you learn**: hostler's two-step Task creation pattern —
**metadata first, body second**. Every Task is created with summary
metadata via `hstl-oss task create`, then its body is filled in
through the `update` tool. This split keeps the create / start /
complete ceremonies auditable while letting bodies grow as the work
clarifies.

**Prompt**:

```
Based on the PDD, Roadmap, and design documents, organize a Sprint for the work.
Create each Task with summary metadata first, then fill in its body using the update tool.
```

**Expect to see**:

- `works/sprints/backlog/sprint-01/SPRINT.md` with goal, member tasks,
  validation matrix.
- 5–10 `T###-<slug>.md` files in the same folder, each carrying both
  metadata frontmatter (status / type / priority / estimate /
  depends_on) **and** a written body (Description, Done Criteria,
  Requirements → Measurement, Scope Limits, Nature).
- `BACKLOG.md` and `CURRENT-FOCUS.md` updated by the CLI.

---

## Step 8 — Start the Sprint

**Goal**: transition the Sprint from `backlog` to `active` and run the
design-readiness checklist.

**What you learn**: the `/hstl-oss:sprint:start` ceremony — readiness
verification, ADR-deferral expiry scan, CURRENT-FOCUS rotation, and
the start briefing.

**Prompt**:

```
/hstl-oss:sprint:start sprint-01
```

**Expect to see**: state transition `backlog → active`,
`works/sprints/active/sprint-01/` materialized, the briefing printed
to stdout, and any deferred ADRs flagged with a warning.

---

## Step 9 — Run the Sprint to completion

**Goal**: execute every Task end-to-end and finally close the Sprint.

**What you learn**: how Task ceremonies chain — `task:start` →
implement → tests → commit → `task:complete` (Harness Gate) — and
how `sprint:complete` fires the 10-Phase Cascade
(doc-review → code-review → work-audit → doc-cross-check → retro →
learned → dev-deploy verify → install → follow-up Tasks → skill
update review).

**Prompt**:

```
Proceed with the sprint to completion. You may run autonomously, except: pause and ask me before the final KB card selection, the follow-up Task selection, and any other important decisions.
```

> **Why the carve-out**: Phase 6 (KB cards) and Phase 9 (follow-up
> Tasks) of `sprint:complete` write durable artefacts that other
> Sprints inherit — the skill body itself mandates final user
> confirmation. The "any other important decisions" clause lets the
> agent escalate genuine surprises (a failing test that hides a
> design defect, a Done Criteria that turns out to be unmeasurable)
> instead of silently working around them.

**Expect to see**:

- One commit per Task, each prefixed by Conventional Commits type
  (`feat:` / `fix:` / `docs:` / …).
- The Harness Gate blocking completion when items are unticked, with
  clear "fix this" guidance.
- A pause at Phase 6 (KB candidates) and Phase 9 (follow-up Task
  candidates) where you confirm or revise the lists before they are
  written.
- Final `sprint:complete sprint-01` running the 10-Phase Cascade,
  including KB card registration and follow-up Task derivation.
- A working SvelteKit app: `npm run dev` serves the todo webapp at
  `http://localhost:5173/`. The LAN URL is enabled in the next step.

---

## Step 10 — Enable LAN listen for mobile testing

**Goal**: bind the SvelteKit dev server to every network interface
(`0.0.0.0`), so the now-completed app is reachable not only from
`localhost` but from other devices on the same Wi-Fi — typically the
phone you want to actually test the mobile UX on.

**What you learn**: Vite's default `host: 'localhost'` exists for
safety (don't expose a dev server by accident); enabling LAN listen
is a deliberate opt-in. Two equivalent ways to flip it: edit
`vite.config.ts` (`server.host = true`) or extend the npm script
(`"dev": "vite dev --host"`). Because the Sprint is already complete,
register the change as a follow-up Task if you want to preserve the
1-Task = 1-commit audit chain.

**Prompt**:

```
Enable LAN listen on the SvelteKit dev server so the todo webapp is reachable from other devices on the same network. Update vite.config.ts (or the dev script in package.json) accordingly. Register the change as a follow-up Task in the backlog so the audit chain stays intact.
```

**Expect to see**:

- `vite.config.ts` with `server: { host: true }` **or** `package.json`
  with `"dev": "vite dev --host"` — pick one, not both.
- `npm run dev` now prints two lines like:

  ```
  ➜  Local:   http://localhost:5173/
  ➜  Network: http://192.168.x.y:5173/
  ```

- A new follow-up Task (e.g. `T###-enable-lan-listen.md`) in
  `works/tasks/` capturing the change with its own `feat(dev):` /
  `chore(dev):` commit.

> **Heads up**: depending on your environment you may need to allow
> port 5173 through the OS firewall (`ufw`, `firewalld`, Windows
> Defender). On WSL the LAN IP is the WSL adapter's, not the host's
> — see Troubleshooting below.

---

## What to watch for as you go

- **Command First** — ceremony commands (`task / sprint create | start
  | complete`) only travel through `/hstl-oss:*` slash commands.
  Direct Bash invocation is reserved for non-ceremony plumbing.
- **1 Task = 1 Commit** — each Task ends in a single Conventional
  Commit at completion time. The Harness keeps that invariant.
- **CURRENT-FOCUS.md is read-only to humans** — hostler manages it.
  If it drifts, run `hstl-oss backlog sync`; never hand-edit.
- **Harness items are type-aware** — a `feature` Task and a `bugfix`
  Task carry different checklists. Always `harness get` first,
  `harness check` second.
- **Use the manifest as the SSOT** — when a guide and the CLI
  disagree, the CLI wins.

---

## After the lab

- `docs/07-knowledge/INDEX.md` should now have a fresh batch of KB
  cards captured by Phase 6 of `sprint:complete`.
- Open the SvelteKit app on a phone (or DevTools mobile mode) using
  the **LAN URL** printed by `npm run dev` (Step 10 enabled this) —
  the research report explicitly targeted mobile UX, so the result
  should feel right on a small screen.
- Re-read the PDD: how much of what you wrote at Step 2 actually
  shipped? The retro from Step 9 already has the answer.

## Troubleshooting

| Symptom | Fix |
|---------|-----|
| `hstl-oss: command not found` | Run `/hstl-oss:cli:install` or follow `installation.md`. |
| Sprint start fails on design-readiness | The skill body lists the missing artefact — usually a doc from Step 3. Add it and retry. |
| Task complete blocked at the Harness | `hstl-oss harness get task <ID>`, address the unticked items, then complete with `--auto-check-criteria` if your `## Done Criteria` is fully `[x]`. |
| `.hostler/` directory appears in the project | You're on a pre-rename CLI build. Pull the latest hostler-oss and reinstall via `/hstl-oss:cli:install` or `go install ./cmd/hstl-oss`. |
| KB card creation skipped at sprint complete | Phase 6 only registers cards confirmed by the user. Re-enter the candidate list and accept explicitly. |
| Phone can't reach the LAN URL | (1) Confirm both devices are on the same Wi-Fi / VLAN. (2) Allow port 5173 through the OS firewall (`sudo ufw allow 5173`, `firewall-cmd`, Windows Defender). (3) On WSL2 the LAN IP shown by Vite is the WSL adapter — bind the Windows host port via `netsh interface portproxy add v4tov4 listenport=5173 connectaddress=<WSL IP>`. (4) On Docker / dev-container, ensure `--publish 5173:5173` and that `vite` is bound to `0.0.0.0` inside the container. |

## Related guides

- [`getting-started.md`](getting-started.md) — five-minute first run.
- [`concepts.md`](concepts.md) — Sprint / Task / Harness primitives.
- [`installation.md`](installation.md) — install paths, env vars,
  config.
- [`cli-reference.md`](cli-reference.md) — flat command index.
