---
description: Run code linting (formatting + static analysis) with auto-detected per-language linter (golangci-lint / ruff / dotnet format / npm run lint). Use when the user asks to "lint", "check style", or "run static analysis".
argument-hint: ""
allowed-tools: Bash(dotnet:format), Bash(npm:run), Bash(ruff:check), Bash(ruff:format), Bash(golangci-lint:run), Bash(mvn:checkstyle), Bash(gradle:check)
---

# Lint Code

## Auto-Detection

Auto-detects the project type and runs the appropriate lint command.

| Project type | Detection files | Lint command |
|--------------|-----------|-----------|
| .NET | `*.sln`, `*.csproj` | `dotnet format` |
| Node.js | `package.json` | `npm run lint` |
| Python | `pyproject.toml`, `ruff.toml` | `ruff check` |
| Go | `go.mod` | `golangci-lint run` |
| Java (Maven) | `pom.xml` | `mvn checkstyle:check` |
| Java (Gradle) | `build.gradle` | `gradle check` |

## Instructions

1. Detect project type
2. Run lint/format checks
3. Report violations
4. Suggest auto-fixes

## Lint Options

### .NET
```bash
# Format check (dry-run)
dotnet format --verify-no-changes

# Auto-fix
dotnet format

# Include analyzers
dotnet format analyzers
```

### Node.js
```bash
# ESLint check
npm run lint

# Auto-fix
npm run lint:fix

# Prettier check
npm run format:check
```

### Python
```bash
# Ruff check
ruff check .

# Auto-fix
ruff check --fix .

# Format check
ruff format --check .
```

## Output Format

```markdown
## Lint Results

| Item | Value |
|------|-----|
| Project type | {detected_type} |
| Lint command | {command} |
| Result | PASS / FAIL |
| Duration | {duration} |

### Summary
| Item | Count |
|------|-----|
| Files Checked | {n} |
| Errors | {n} |
| Warnings | {n} |
| Auto-fixable | {n} |

### Issues by Category
| Category | Count | Severity |
|----------|-------|----------|
| Formatting | {n} | Info |
| Style | {n} | Warning |
| Bug Risk | {n} | Error |

### Top Issues
1. **{rule_id}**: {description}
   - Files: {n}
   - Auto-fix: Yes/No

2. **{rule_id}**: {description}
   - Files: {n}
   - Auto-fix: Yes/No

### Auto-fix Available
```bash
# Auto-fix command
{auto_fix_command}
```

### Recommendations
{Mitigation for lint violations}
```

## Integration

Lint results are referenced during code review:
- `/review:start` — auto-includes lint issues
- Used as a code quality indicator

