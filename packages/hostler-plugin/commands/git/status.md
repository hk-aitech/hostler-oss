---
description: Inspect and summarize git repository state — git status / diff / log / branch / remote combined (Bash wrapper). Use when the user asks "what's the git status?", "show me the diff", or wants a snapshot of the working tree.
argument-hint: ""
allowed-tools: Bash(git:status), Bash(git:diff), Bash(git:log), Bash(git:branch), Bash(git:remote)
---

# Git Status

Inspect and summarize the current state of the git repository.

## Instructions

1. Collect git state

```bash
# Basic status
git status

# Short format
git status --short

# Branch info
git branch -vv

# Remote info
git remote -v
```

2. Analyze changes

```bash
# Staged changes
git diff --cached --stat

# Unstaged changes
git diff --stat

# Untracked files
git status --porcelain | grep "^??"
```

3. Inspect recent commits

```bash
# Last 5 commits
git log --oneline -5

# Detailed commit info
git log -1 --format=fuller
```

## Output Format

```markdown
## Git Status Summary

### Repository
| Item | Value |
|------|-----|
| Branch | {current_branch} |
| Tracking | {upstream_branch} |
| Ahead | {n} commits |
| Behind | {n} commits |

### Working Directory
| Category | Count |
|----------|-------|
| Staged | {n} |
| Modified | {n} |
| Untracked | {n} |
| Deleted | {n} |

### Staged Changes
```
{staged_files}
```

### Unstaged Changes
```
{unstaged_files}
```

### Untracked Files
```
{untracked_files}
```

### Recent Commits
| Hash | Message | Date |
|------|---------|------|
| {hash} | {message} | {date} |

### Recommendations
{State-dependent recommendations}
```

## Status Indicators

| Marker | Meaning |
|------|------|
| `M` | Modified |
| `A` | Added |
| `D` | Deleted |
| `R` | Renamed |
| `C` | Copied |
| `??` | Untracked |
| `!!` | Ignored |

## Common Scenarios

### 1. Clean Working Directory
```
Working directory clean
No staged or unstaged changes
```

### 2. Changes Ready to Commit
```
{n} files staged for commit
Run: git commit -m "message"
```

### 3. Uncommitted Changes
```
{n} files modified but not staged
Run: git add <files> or git add .
```

### 4. Behind Remote
```
{n} commits behind origin/{branch}
Run: git pull
```

### 5. Ahead of Remote
```
{n} commits ahead of origin/{branch}
Run: git push
```

### 6. Diverged
```
Diverged from origin/{branch}
Ahead: {n}, Behind: {n}
Consider: git pull --rebase
```

## Integration

- `/git:commit` — commit guide
- `/session:end` — session end (uncommitted change check)
- `/task:complete` — Task completion (commit state check)

