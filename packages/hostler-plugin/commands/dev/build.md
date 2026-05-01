---
description: Run a project build with auto-detected language tooling (go build / dotnet build / npm run build / pytest, etc.) and summarize the result. Use when the user asks to "build the project", "run the build", or after code changes that need verification.
argument-hint: ""
allowed-tools: Bash(dotnet:build), Bash(npm:run), Bash(go:build), Bash(mvn:package), Bash(gradle:build)
---

# Build Project

## Auto-Detection

Auto-detects the project type and runs the appropriate build command.

| Project type | Detection files | Build command |
|--------------|-----------|-----------|
| .NET | `*.sln`, `*.csproj` | `dotnet build` |
| Node.js | `package.json` | `npm run build` |
| Python | `pyproject.toml`, `setup.py` | `python -m build` |
| Go | `go.mod` | `go build ./...` |
| Java (Maven) | `pom.xml` | `mvn package` |
| Java (Gradle) | `build.gradle` | `gradle build` |

## Instructions

1. Detect project type from the project root
2. Run the appropriate build command
3. Provide a summary of the result

## Build Options

### .NET
```bash
# Debug build (default)
dotnet build

# Release build
dotnet build --configuration Release

# Specific project
dotnet build src/MyProject/MyProject.csproj
```

### Node.js
```bash
# Default build
npm run build

# Production build
npm run build:prod
```

### Python
```bash
# Package build
python -m build

# Editable install
pip install -e .
```

## Output Format

```markdown
## Build Result

| Item | Status |
|------|------|
| Project type | {detected_type} |
| Build command | {command} |
| Result | PASS / FAIL |
| Duration | {duration} |

### Build Output
```
{build_output}
```

### Summary
| Item | Count |
|------|-----|
| Warnings | {n} |
| Errors | {n} |

### Warnings (if any)
1. {warning_1}
2. {warning_2}

### Errors (if any)
1. {error_1}
2. {error_2}

### Recommendation
{Mitigation if the build fails}
```

## Validation Integration

The build result is consumed for verification by:
- `/task:complete` — BLOCK-level check
- `/session:end` — BLOCK-level check

