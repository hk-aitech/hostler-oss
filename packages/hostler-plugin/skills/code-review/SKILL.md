---
name: code-review
description: Automated code-quality review — static analysis, OWASP Top 10, banned patterns, complexity, and test-coverage signals, returned as a severity-graded report. Use this skill whenever the user mentions code review, "review my changes", security review, OWASP, banned patterns, complexity audit, or asks to vet code before merging — especially right before `task:complete` or as Phase 2 of `sprint:complete`. Do NOT use for documentation review (use `doc-review`) or for architecture review (use `architecture-design`).
compatibility:
  tools: [Read, Bash, Glob, Grep]
argument-hint: "[--session|--sprint|--repo] [path]"
paths: ["**/*.go", "**/*.ts", "**/*.py", "**/*.cs", "**/*.java", "**/*.md"]
---

# Code Review Skill

An automation Skill that analyzes code changes and identifies quality issues.

## Table of Contents

1. Scope
2. Capabilities
3. Determining the review scope
4. Automation Steps
4. Review Checklist
5. Severity classification
6. Output Format
7. Integration

---

## Scope

Specify the range of files to inspect. With no argument, `--session` is the default.

| Option | Target files | git command |
|--------|-------------|-------------|
| `--session` (default) | Current session changes | staged: `git diff --cached --name-only`, unstaged: `git diff --name-only HEAD~1` |
| `--sprint` | Active Sprint changes | Sprint-start commit onward: diff vs `git log --format=%H works/sprints/ \| tail -1` |
| `--repo` | Entire source tree | `find src/ tests/ -name "*.cs" -o -name "*.ts"` etc. |

### Scope interpretation rules
- Because `--session` is the default, calling `/code-review` alone reviews only the code you just changed
- `--sprint` is forwarded automatically when work-audit invokes this skill from inside `sprint:complete`
- Use `--repo` explicitly when you need to inspect the entire codebase
- Specifying a path directly (`/code-review src/MyApp.Collection/`) inspects only that path instead of the scope

### Per-scope file filters
Every scope excludes the following patterns:
- `*.lock`, `*.min.js`, `*.min.css` (generated files)
- `node_modules/`, `vendor/`, `dist/`, `obj/`, `bin/` (dependencies / build artifacts)
- `*.md` (documentation -- handled by doc-review)

---

## Capabilities

### 1. Static Analysis
- Coding-convention checks
- Detection of latent bugs
- Identification of security vulnerabilities
- Detection of performance issues

### 2. Change Analysis
- Git-diff-based change analysis
- Impact-range assessment
- Dependency-change tracking

### 3. Quality Metrics
- Complexity measurement
- Test-coverage check
- Duplicate-code detection

## Determining the review scope

The list of files to review is determined by the rules in the Scope section. If more than 20 files have changed, suggest narrowing the scope to the user. Show a `git diff --stat` summary first, then prioritize the core files for review.

## Automation Steps

### Step 1: Collect changed files
```bash
# Collect changed file list with git diff
git diff --name-only HEAD~1
git diff --stat HEAD~1

# Or compare against a specific branch
git diff --name-only main...HEAD
```

### Step 2: Per-file analysis

#### .NET analysis
```bash
dotnet format --verify-no-changes --verbosity diagnostic
dotnet build /p:TreatWarningsAsErrors=true
```

#### Node.js analysis
```bash
npm run lint -- --format json
npx tsc --noEmit
```

#### Python analysis
```bash
ruff check . --output-format json
mypy . --json-report
```

#### Go analysis
```bash
golangci-lint run --out-format json
go vet ./...
```

#### Java analysis
```bash
mvn checkstyle:check
mvn spotbugs:check
```

### Step 3: Security checks
```bash
# Hardcoded-secret patterns
grep -rn --include="*.{cs,ts,js,py,go,java}" \
  -E "(password|secret|api_key|token)\s*=\s*['\"][^'\"]+['\"]" \
  --exclude-dir=node_modules --exclude-dir=.git .

# SQL injection patterns
grep -rn --include="*.{cs,ts,js,py,go,java}" \
  -E "(SELECT|INSERT|UPDATE|DELETE).*\+.*\"|f\".*SELECT" .
```

### Step 3.5: Banned-pattern check

If the user's project docs file exists in the project root, search the changed files for the regex patterns it defines. Any match is reported at **CRITICAL** severity.

```bash
# Auto-check when banned-patterns.md is present.
# Use both the regex column of the pattern table and the exception column.
# Patterns / exceptions vary per project, so do NOT hardcode them in the hostler skill.
if [ -f "the user's project docs" ]; then
  # 1. Extract patterns (rows of the form | BP-NNN | `pattern` |)
  PATTERNS=$(grep -oP '`[^`]+`' the user's project docs \
    | head -6 | tr -d '`' | paste -sd'|')

  # 2. Extract exception patterns (the "exception" column of banned-patterns.md, or the grep -v patterns from its "manual checks" section)
  #    The project's banned-patterns.md "manual checks" section contributes grep -v lines that are applied here.
  EXCLUDES=$(grep -oP 'grep -v "\K[^"]+' the user's project docs 2>/dev/null \
    | paste -sd'\n' || true)

  echo "=== Banned-pattern check ==="
  FOUND=$(grep -rn -E "$PATTERNS" src/ \
    | grep -v "/obj/" | grep -v "/bin/" || true)

  # 3. Apply project-defined exception patterns
  if [ -n "$EXCLUDES" ]; then
    while IFS= read -r excl; do
      [ -n "$excl" ] && FOUND=$(echo "$FOUND" | grep -v "$excl" || true)
    done <<< "$EXCLUDES"
  fi

  if [ -n "$FOUND" ]; then
    echo "❌ CRITICAL: banned pattern found"
    echo "$FOUND"
  else
    echo "✅ No banned patterns"
  fi
fi
```

When a pattern is hit, also surface the "replacement code" column from `banned-patterns.md` so the user knows how to fix it.

> **Technology-neutral**: file extensions for `--include`, excluded filenames, etc., are defined inside each project's `banned-patterns.md`.
> Do not hardcode language- or framework-specific patterns into the hostler skill.

### Step 4: Complexity analysis

#### Function length check (over 50 lines)
```bash
# Python
grep -n "def " *.py | while read line; do
  # Extract the start line of each function and compute the line count up to the next function
  echo "$line"
done
```

## Review Checklist

### Code Quality
- [ ] Naming clarity -- variable/function names clearly describe purpose
- [ ] Function length appropriate (< 50 lines recommended)
- [ ] Complexity (Cyclomatic < 10 recommended)
- [ ] No duplicated code
- [ ] No unnecessary comments (code is self-explanatory)
- [ ] **No banned patterns** -- 0 hits on patterns defined in `banned-patterns.md` (Step 3.5)

### Security -- key OWASP Top 10 items

Security review uses OWASP Top 10 as the baseline; verify these items first:

- [ ] **A01 Access control**: every protected endpoint has authorization checks
- [ ] **A02 Cryptographic failures**: no hardcoded secrets or API keys
- [ ] **A03 Injection**: parameterized queries used; input validation performed
- [ ] **A05 Security misconfiguration**: error messages do not expose stack traces
- [ ] **A07 Authentication failures**: session timeout configured; brute-force defenses in place

For the detailed security checklist, see `references/security-checklist.md`.

### Performance
- [ ] **No N+1 queries** -- detect query execution inside loops
- [ ] **No memory-leak risk** -- unreleased event listeners, unbounded cache growth
- [ ] No unnecessary loops
- [ ] Appropriate caching
- [ ] No unnecessary API calls
- [ ] Streaming for large data

### Testability
- [ ] Unit tests exist
- [ ] Mockable design (dependency injection)
- [ ] Sufficient coverage (> 80% recommended)
- [ ] Edge cases tested

### Architecture
- [ ] Single Responsibility Principle
- [ ] Appropriate abstraction level
- [ ] Correct dependency direction
- [ ] Layer separation preserved

### UX → implementation consistency (when UI changes)

If the change includes dashboard API or frontend changes:
- [ ] Verify UI elements with the same label reference the same data source
- [ ] Verify a new API endpoint is added to the user's project docs mapping
- [ ] When changing an existing API, verify the DB/Redis paths in the mapping doc still match

## Severity Classification

Findings are graded into four severity levels. Required action depends on severity.

| Severity | Criteria | Action | Examples |
|----------|----------|--------|----------|
| **Critical** | Security vulnerability, data-loss risk | Block merge, fix immediately | SQL injection, hardcoded password, auth bypass |
| **High** | Functional bug, severe performance issue | Must fix before release | N+1 query, possible infinite loop, missing input validation |
| **Medium** | Code quality, maintainability degradation | Manage in backlog, fix next Sprint | Duplicated code, excessive complexity, low coverage |
| **Low** | Style or minor improvement | Optional | Naming improvement, unneeded comment, suggested cleaner pattern |

If even one Critical/High issue exists, mark the "Ready for merge" status as FAIL.

## Output Format

The review result is emitted as a structured report containing per-file findings and summary statistics.

```markdown
## Code Review Report

### Summary
| Item | Value |
|------|-----|
| Files Reviewed | {n} |
| Lines Changed | +{added} / -{removed} |
| Issues Found | {total} |
| Critical | {n} |
| High | {n} |
| Medium | {n} |
| Low | {n} |

### Critical Issues

#### [CRITICAL] Hardcoded API key
- **File**: `src/services/payment.ts:45`
- **Code**: `const API_KEY = "sk-live-..."`
- **Description**: An API key is embedded directly in source code.
- **Suggestion**: Move it to an environment variable and add a placeholder to .env.example
- **Reference**: OWASP A3:2017 - Sensitive Data Exposure

### High Issues

#### [HIGH] N+1 query pattern detected
- **File**: `src/repositories/order.py:78`
- **Code**:
  ```python
  for order in orders:
      items = db.query(OrderItem).filter_by(order_id=order.id).all()
  ```
- **Description**: Executes a query inside a loop -- N+1 problem
- **Suggestion**: Use JOIN or eager loading
  ```python
  orders = db.query(Order).options(joinedload(Order.items)).all()
  ```

### Medium / Low Issues

#### [MEDIUM] Function-name improvement recommended
- **File**: `src/utils/helper.go:23`
- **Current**: `func do()`
- **Suggestion**: `func processOrder()` -- clearer intent

### Statistics
| Metric | Value | Threshold | Status |
|--------|-------|-----------|--------|
| Cyclomatic Complexity (max) | {n} | < 10 | PASS/FAIL |
| Function Length (max lines) | {n} | < 50 | PASS/FAIL |
| Test Coverage | {n}% | > 80% | PASS/FAIL |
| Duplicated Lines | {n}% | < 3% | PASS/FAIL |

### Recommendations
1. {overall recommendation 1}
2. {overall recommendation 2}

### Approval Status
- [ ] All Critical issues resolved
- [ ] All High issues resolved or acknowledged
- [ ] Tests passing
- [ ] Ready for merge
```

For an output example, see `examples/sample-review.md`.

## Integration

This skill integrates with the following commands and skills:
- `/review:start` -- run analysis automatically when a review starts
- `/review:complete` -- generate the final report
- `/review:approve` -- approve
- `/task:complete` -- optional code review
- `references/quality-metrics.md` -- complexity thresholds, naming conventions, coverage targets in detail
- `references/security-checklist.md` -- full OWASP Top 10 checklist + detection patterns
- `examples/sample-review.md` -- a complete review-report example

## Rule Engine command review checklist

When reviewing a Rule command under `cli/pkg/rules/` or `.hostler/rules/`:

- [ ] No `|| true` -- failure swallowing is forbidden
- [ ] No `>/dev/null 2>&1` or `2>/dev/null` -- error hiding is forbidden
- [ ] Failure exit codes propagate to the Rule Engine

Details: the user's project docs

## Per-Project Extensions (auto-detected)

At runtime, this skill auto-detects `.hostler/project-config.md` in the project root.
If that file exists and contains a `## Per-skill extensions` → `### code-review` section,
the project-specific rules are applied **in addition to** the shared rules.

- Detection path: `{project_root}/.hostler/project-config.md`
- If absent, only the shared rules are applied (backward compatible)
