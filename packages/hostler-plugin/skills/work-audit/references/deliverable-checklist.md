# Implementation Completeness Checklist

> A general-purpose checklist that verifies the actual implementation against the Task plan document.
> Not tied to any specific project language or framework.

## D1: Verify changed files exist (BLOCK)

Extract the paths recorded in the `## Result` → `### Changed files` section of done tasks and verify they actually exist.

### How to check

```bash
# Extract changed-file paths from the Task file
grep -A 20 "### Changed files" $TASK_FILE | grep "^- \`" | sed "s/.*\`\([^\\`]*\)\`.*/\1/"

# Verify each path exists
for path in $PATHS; do
  [ -f "$path" ] && echo "✅ $path" || echo "❌ $path (missing)"
done
```

### Verdict
- Any missing → BLOCK
- All present → PASS

## D2: Completion-criteria checkboxes (WARN)

Compute the checkbox ratio inside the `## Done Criteria` section of done tasks.

### How to check

```bash
# `grep -c` prints "0" on stdout and exits 1 when there are 0 hits.
# Do NOT use `|| echo 0` — use `) || true` instead.
TOTAL=$(grep -c "\- \[" "$TASK_FILE" 2>/dev/null) || true
CHECKED=$(grep -c "\- \[x\]" "$TASK_FILE" 2>/dev/null) || true
[ "${TOTAL:-0}" -gt 0 ] && RATIO=$(( ${CHECKED:-0} * 100 / ${TOTAL:-1} )) || RATIO=0
```

### Verdict
- 100% checked → PASS
- 50-99% → WARN
- 0-49% (despite being done) → WARN (high priority)

## D3: Sprint Task count match (WARN)

Verify the number of rows in SPRINT.md's Task list matches the actual file count under `tasks/`.

### How to check

```bash
# Row count in SPRINT.md table (rows starting with |T)
TABLE_COUNT=$(grep -c "^| T" $SPRINT_MD)

# File count in directory
DIR_COUNT=$(find $TASKS_DIR -name "T*.md" | wc -l)
```

### Verdict
- Match → PASS
- Mismatch → WARN (state which side is larger)

## D4: Sprint-Task status match (WARN)

Verify the status text in SPRINT.md matches each Task frontmatter's status.

### How to check

Extract each Task ID's status from SPRINT.md and compare against the status field in that Task file's frontmatter.

### Verdict
- All match → PASS
- Any mismatch → WARN (concrete mismatch list)

## D5: Tests pass (BLOCK)

Run the project's test suite and verify everything passes.

### Auto-detection (general)

```bash
# Python
[ -f "pytest.ini" -o -f "pyproject.toml" ] && .venv/bin/python -m pytest tests/ -q --tb=no

# .NET (slnx or sln)
SLN=$(find . -maxdepth 1 -name "*.slnx" -o -name "*.sln" 2>/dev/null | head -1)
[ -n "$SLN" ] && dotnet test "$SLN" --verbosity normal

# Node.js
[ -f "package.json" ] && npm test

# Go
[ -f "go.mod" ] && go test ./...

# Rust
[ -f "Cargo.toml" ] && cargo test
```

> **Note**: `.NET` with `--verbosity quiet` does not print pass counts. Use `normal`.

### Verdict
- All pass → PASS
- Any failure → BLOCK

## D6: Build / lint succeeds (WARN)

Run the project's lint tool and verify static analysis passes.

### Auto-detection (general)

```bash
# Python
command -v ruff && ruff check src/

# Node.js
[ -f ".eslintrc*" ] && npx eslint .

# Go
go vet ./...
```

### Verdict
- 0 errors → PASS
- Warnings only → INFO
- Errors present → WARN

---

## Fix Guidance (performed outside the audit)

The audit only inspects. The user requests a fix separately as needed.

| Check | Fix owner | How to fix |
|-------|----------|-----------|
| D1 missing file | developer | Create the file or fix the path in the Task results section |
| D2 unchecked | tech-writer | Check off all boxes on done tasks |
| D3 count mismatch | tech-writer | Sync the SPRINT.md table or the files |
| D4 status mismatch | tech-writer | Sync SPRINT.md or the Task frontmatter |
| D5 test failure | developer | Fix the failing tests |
| D6 lint error | developer | Preview with `ruff check --diff`, then fix |
| D7 UX mapping not refreshed | tech-writer | Update the matching row in `dashboard-ui-api-mapping.md` |
