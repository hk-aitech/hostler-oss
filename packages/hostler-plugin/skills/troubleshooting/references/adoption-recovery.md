# hostler plugin initial-adoption recovery playbook

Typical symptoms and recovery steps for an existing project that adopts the
hstl-oss plugin for the first time.

This is the long-form version of SKILL.md's "hostler plugin initial-adoption recovery" section.

## Symptom identification

If **two or more** of the following hold, treat the project as in the "initial adoption" state:

- [ ] Dozens of `works/tasks/T*.md` files exist but `works/tasks/BACKLOG.md` is missing
- [ ] `works/tasks/BACKLOG.md` exists but has fewer rows than files (drift)
- [ ] `works/CURRENT-FOCUS.md` is missing
- [ ] `.hostler/project-config.yaml` is missing
- [ ] `hstl-oss task list` returns fewer items than expected (not indexed in the DB)
- [ ] `hstl-oss brief`'s `active_sprints` and `task_summary.total` don't match the filesystem

## Recovery procedure (canonical order)

### 1. Pre-flight diagnosis

```bash
cd <project-root>
echo "files: $(find works/tasks -maxdepth 1 -name 'T*.md' | wc -l)"
echo "BACKLOG rows: $(grep -cE '^\| T[0-9]+ ' works/tasks/BACKLOG.md 2>/dev/null || echo 0)"
ls works/CURRENT-FOCUS.md 2>&1
ls .hostler/project-config.yaml 2>&1
```

### 2. Auto-create the skeleton

**Recommend a dry-run** — confirm what will be created before actually creating:

```bash
hstl-oss-oss project init-skeleton --dry-run | jq '.would_create_dirs, .would_create_files'
```

After confirming, run for real:

```bash
hstl-oss-oss project init-skeleton
```

The result is created idempotently (existing files are never touched):

- `works/tasks/` + `works/sprints/{backlog,active,completed}/` directories
- `works/tasks/BACKLOG.md` (only when missing)
- `works/CURRENT-FOCUS.md` (only when missing, in idle initial state)
- `.hstl-oss/` directory
- `.hostler/project-config.yaml` (`DetectProjectKey` auto-pins the key it
  computes via the fallback path. From subsequent runs the YAML becomes the SSOT,
  so a binary-version hash-algorithm drift no longer matters)

### 3. File → DB sync

```bash
hstl-oss-oss backlog sync
```

The output's `auto_fixed` field can include:
- `db_task_inserted` — added a Task that existed only on disk to the DB tasks table
- `db_status_fixed` — reconciled mismatch between DB status and file frontmatter status
- `db_sprint_inserted` — added a Sprint that existed only on disk to the DB sprints table
- `db_sprint_metadata_fixed` — fixed Sprint metadata (title/started_at/completed_at)

You can detect issues first with `--dry-run`.

### 4. DB → BACKLOG.md regeneration

**Recommend a dry-run** — confirm the impact before overwriting:

```bash
hstl-oss-oss backlog rebuild-md --dry-run | jq '{count, would_backup}'
```

After confirming, run for real (existing BACKLOG.md is backed up to `.bak`):

```bash
hstl-oss-oss backlog rebuild-md
```

- Lists every unassigned (`sprint IS NULL OR sprint = ''`) Task in BACKLOG.md,
  sorted by priority then ascending task_id
- The existing BACKLOG.md is backed up to `.bak` and overwritten
- The `--include-in-progress` flag also includes in-progress Tasks (default false)

### 5. Verify

```bash
# Files == BACKLOG.md rows (unassigned baseline)
find works/tasks -maxdepth 1 -name 'T*.md' | wc -l
grep -cE '^\| T[0-9]+ ' works/tasks/BACKLOG.md

# Re-check DB ↔ files — must be 0 issues
hstl-oss-oss backlog sync --dry-run | jq '.total_issues'

# Briefing displays correctly
hstl-oss-oss brief | jq '{task_count: .task_summary.total, sprints: .active_sprints}'
```

## Caveats

### Order matters

Stick to `init-skeleton` → `backlog sync` → `rebuild-md`:

1. `init-skeleton` calls `DetectProjectKey` to initialize the DB and pin the YAML
   first. Without this step, the location of the DB after running sync is undetermined.
2. `backlog sync` indexes files into the DB. Without this step, `rebuild-md` would
   read an empty DB and regenerate a BACKLOG.md with 0 rows.
3. `rebuild-md` reads the DB to regenerate BACKLOG.md. Always call after sync.

### Preserving manual edits

- `init-skeleton` — **never overwrites existing files**. Returns a `SkippedExisting`
  array.
- `rebuild-md` — backs up the existing BACKLOG.md to `.bak` and overwrites. If you
  have manual edits, run `diff` first to inspect them.

### Task-only projects (no Sprints)

Safe even when `works/sprints/` is missing or unused:

- `fileutil.UpdateCurrentFocus` no-ops when `works/sprints/` is absent (won't create CURRENT-FOCUS.md)
- `init-skeleton` creates the 3 sprint subfolders by default, but if you don't use the
  sprint feature, they just remain empty without side effects

### Recovery after a binary upgrade

If the project key changed after an hstl-oss binary upgrade and the DB looks like it disappeared:

1. Find the old path: look for a different hash folder under `~/.hostler/data/`
2. Restore from the filesystem: rebuild the DB from files via `hstl-oss backlog sync`
   (data lives in files, so it can be restored)
3. Pin via YAML: run `hstl-oss project init-skeleton` so `.hostler/project-config.yaml`
   is auto-created → the key is pinned across future upgrades

## Related lessons

- KB `mistakes/workflow.md` — YAML time.Time fallback format drift
- KB `mistakes/yaml.md` — same topic as above
- KB `architecture/decisions.md` — harness auto-check safety decision
- KB `mistakes/workflow.md` — result verifier checks git diff up front

## Related history

- Registered KB card (redefined hostler = skills + cli)
- Auto-generated `UpdateBacklogMD` / `UpdateCurrentFocus`
- `DetectProjectKey` YAML auto-pinning
- Introduced `hstl-oss backlog rebuild-md`
- Introduced `hstl-oss project init-skeleton`
- Hotfix: added `--dry-run` to the two commands above
