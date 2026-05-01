---
description: Run tests with auto-detected per-language test runner (go test / dotnet test / pytest / npm test) and summarize results. Use when the user asks to "run tests", "execute tests", or to verify changes before committing.
argument-hint: ""
allowed-tools: Bash(dotnet:test), Bash(npm:test), Bash(npm:run), Bash(pytest), Bash(go:test), Bash(mvn:test), Bash(gradle:test)
---

# Run Tests

## Auto-Detection

Auto-detects the project type and runs the appropriate test command.

| Project type | Detection files | Test command |
|--------------|-----------|-------------|
| .NET | `*.sln`, `*.csproj` | `dotnet test` |
| Node.js | `package.json` | `npm test` |
| Python | `pytest.ini`, `pyproject.toml` | `pytest` |
| Go | `go.mod` | `go test ./...` |
| Java (Maven) | `pom.xml` | `mvn test` |
| Java (Gradle) | `build.gradle` | `gradle test` |

## Instructions

1. Detect project type
2. Run tests
3. Summarize results and coverage

## Test Options

### .NET
```bash
# All tests
dotnet test

# Including coverage
dotnet test --collect:"XPlat Code Coverage"

# Specific project
dotnet test tests/MyProject.Tests/

# Filtered
dotnet test --filter "Category=Unit"
```

### Node.js
```bash
# Default test run
npm test

# Including coverage
npm run test:coverage

# Watch mode
npm run test:watch
```

### Python
```bash
# Default test run
pytest

# Including coverage
pytest --cov=src --cov-report=html

# Specific test
pytest tests/test_module.py
```

## Output Format

```markdown
## Test Results

| Item | Value |
|------|-----|
| Project type | {detected_type} |
| Test command | {command} |
| Result | PASS / FAIL |
| Duration | {duration} |

### Summary
| Item | Count |
|------|-----|
| Total | {n} |
| Passed | {n} |
| Failed | {n} |
| Skipped | {n} |

### Coverage (if available)
| Module | Line | Branch |
|--------|------|--------|
| {module} | {n}% | {n}% |
| **Total** | {n}% | {n}% |

### Failed Tests (if any)
1. **{test_name}**
   - Location: `file:line`
   - Error: {error_message}

### Recommendations
{Mitigation when tests fail}
```

## Validation Integration

Test results are consumed for verification by:
- `/task:complete` — BLOCK-level check
- `/session:end` — BLOCK-level check
- `/review:complete` — test coverage check

