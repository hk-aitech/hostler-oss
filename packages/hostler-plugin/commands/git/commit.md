---
description: Git commit creation guide — Conventional Commits format with automatic Task ID inclusion and Co-Authored-By trailer. Use when the user asks to "make a commit", "write a commit message", or wants help structuring a commit.
argument-hint: "[<message>]"
allowed-tools: Bash(git:add), Bash(git:commit), Bash(git:diff), Bash(git:status)
---

# Git Commit Guide

## Instructions

1. Review the changes
2. Provide commit message guidance
3. Execute the commit (after user confirmation)

## Pre-commit Checklist

### Required (BLOCK)
- [ ] Build succeeds
- [ ] Tests pass
- [ ] No sensitive data (.env, secrets, etc.)

### Recommended (WARN)
- [ ] Lint passes
- [ ] Unrelated changes separated

## Commit Message Format

### Conventional Commits

```
<type>[optional scope]: <description>

[optional body]

[optional footer(s)]
```

### Types

| Type | Description | Example |
|------|------|------|
| `feat` | New feature | `feat: add user authentication` |
| `fix` | Bug fix | `fix: handle login failure` |
| `docs` | Documentation | `docs: update API documentation` |
| `style` | Formatting, semicolons, etc. | `style: format code` |
| `refactor` | Refactoring | `refactor: split service` |
| `test` | Add/modify tests | `test: add unit tests` |
| `chore` | Build, tooling, etc. | `chore: update dependencies` |
| `perf` | Performance improvement | `perf: optimize query` |

### Scope Examples

- `(auth)`: authentication
- `(api)`: API
- `(ui)`: UI
- `(db)`: database

### Task Reference

Include the Task ID in the commit message:

```
feat(auth): add social login

- Google OAuth integration
- Token refresh logic

Refs: TASK-123
```

## Session/Sprint Reference

Including sprint/session info is recommended:

```
[S#N] feat: feature description

or

[p1-s1] feat: feature description
```

## Output Format

```markdown
## Git Commit Guide

### Current Changes
```
{git status output}
```

### Staged Files
```
{git diff --staged --stat}
```

### Suggested Commit Message

```
{type}({scope}): {description}

{body}

Refs: {task_id}
```

### Commit Command
```bash
git add {files}
git commit -m "{message}"
```

### Validation
| Check | Status |
|-------|--------|
| Build | PASS/FAIL |
| Tests | PASS/FAIL |
| No Secrets | PASS/FAIL |
```

## Auto-generated Footer

Footer added automatically on commit:

```
Generated with [Claude Code](https://claude.com/claude-code)

Co-Authored-By: Claude <noreply@anthropic.com>
```

## Examples

### Feature Commit
```bash
git commit -m "$(cat <<'EOF'
[S#24] feat(order): add order cancellation

- Implement order status validation
- Record cancellation reason
- Trigger refund processing

Refs: TASK-145

Generated with [Claude Code](https://claude.com/claude-code)

Co-Authored-By: Claude <noreply@anthropic.com>
EOF
)"
```

### Bug Fix Commit
```bash
git commit -m "$(cat <<'EOF'
[S#24] fix(auth): fix infinite loop on token expiry

Add retry count limit when token refresh fails.

Refs: TASK-150

Generated with [Claude Code](https://claude.com/claude-code)

Co-Authored-By: Claude <noreply@anthropic.com>
EOF
)"
```

