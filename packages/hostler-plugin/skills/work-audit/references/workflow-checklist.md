# hostler Workflow Compliance Checklist

## W1: task:start was invoked

Verify the status-transition history in done-task frontmatter:
- Whether the git log contains a commit changing "status: todo → status: in-progress"
- Evidence of `hstl-oss task start` CLI invocation (audit_events `task.transitioned`, or "task:start" in commit message)

```bash
# Check in-progress transition history on recent done tasks
for f in $(grep -rl "^status:.*done" works/sprints/active/*/tasks/*.md); do
  id=$(grep -oP "^id:\s*\K.*" "$f")
  has_transition=$(git log --all -p -- "$f" | grep -c "status:.*in-progress\|status:.*in_progress")
  echo "$id: transition=$has_transition"
done
```

## W2: task:complete + Results section

Verify done tasks contain a "## Result" section:

```bash
for f in $(grep -rl "^status:.*done" works/sprints/active/*/tasks/*.md); do
  id=$(grep -oP "^id:\s*\K.*" "$f")
  has_result=$(grep -c "^## Result" "$f")
  echo "$id: result_section=$has_result"
done
```

## W3: Commit message format

```bash
# Check format on the latest 20 commits
git log --oneline -20 | grep -cP "^[a-f0-9]+ (feat|fix|docs|chore|refactor|test):"
```

Format: `{type}: {description}` or `{type}: {TaskID} — {description}`

## W4: Sprint completion uses git mv

If a Sprint where every Task is done still lives under active/, WARN:

```bash
for d in works/sprints/active/*/; do
  total=$(find "$d/tasks" -name "*.md" 2>/dev/null | wc -l)
  done=$(grep -rl "^status:.*done" "$d/tasks/"*.md 2>/dev/null | wc -l)
  if [ "$total" -eq "$done" ] && [ "$total" -gt 0 ]; then
    echo "WARN: $(basename $d) all Tasks done — needs to move to completed/"
  fi
done
```

## W5: CURRENT-FOCUS.md up to date

Verify Sprint progress matches the actual Task statuses.

## W6: Sprint retrospective recorded (most recent 2-3)

Verify that the SPRINT.md of the **most recent 2-3 completed Sprints**
contains a `## Retro` section. If missing, prompt to run `/retro {sprint-id}`.

```bash
# Check the retro section on the latest 3 completed sprints
for d in $(ls -d works/sprints/completed/sprint-* 2>/dev/null | sort -t'-' -k2 -n | tail -3); do
  sprint=$(basename "$d")
  has_retro=$(grep -c "## Retro\|## Retrospective" "$d/SPRINT.md" 2>/dev/null || echo 0)
  if [ "$has_retro" -eq 0 ]; then
    echo "⚠️ W6 $sprint: retro missing → run /retro $sprint"
  else
    echo "✅ W6 $sprint: retro present"
  fi
done
```

### Verdict
- Latest 3 Sprints all have a retro → PASS
- 1+ missing → WARN (list missing Sprints + suggest `/retro`)

### Inspection range
Inspect only the **2-3** most-recent completed Sprints immediately before the
active Sprint. Do not warn about missing retros on older Sprints (6+ months).

## W7: Worklog commit coverage

Verify that every commit from yesterday (or the most recent session) is recorded in the worklog:

```bash
# Yesterday's commit count
yesterday_commits=$(git log --oneline --after="yesterday 00:00" --before="today 00:00" | wc -l)

# Sum of commit counts recorded in yesterday's worklogs
worklog_commits=$(grep -oP 'commits?.*?(\d+)' works/worklogs/yesterday-*.md | grep -oP '\d+' | paste -sd+ | bc)

# Compare
echo "Yesterday commits: $yesterday_commits, recorded in worklog: $worklog_commits"
```

### Verdict
- Counts match (±1 for the worklog's own commit) → PASS
- Mismatch → WARN (provide list of missing commits)
