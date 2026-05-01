# hstl-oss Plugin Hooks

## Overview

This directory contains the Claude Code hooks shipped with the hstl-oss plugin.

## Registered hooks

| Hook | Event | Type | Matcher | Timeout | Purpose |
|------|-------|------|---------|---------|---------|
| `session-context.sh` | SessionStart | command | `*` | 10s | Session context briefing (Git status, current task, sprint progress) |
| `subagent-context-ack.sh` | PreToolUse | command | `Agent` | 5s | Auto-record context acknowledgement for the active task's work ticket before each subagent call |

## Standalone install scripts

| Script | Purpose | Installation |
|--------|---------|--------------|
| `pre-commit-check.sh` | git pre-commit hook (commit message format + ruff lint) | Per-project: copy to `.git/hooks/pre-commit` or symlink it there |

## SessionStart

Shows the project status at a glance when a session starts.

- Verifies that the `works/` directory exists.
- Reports the Git branch, uncommitted changes, and remote sync status.
- Reports active sprint progress (via the CLI).
- Reports the current task and the suggested next task (via the CLI).
- Prints a workflow reminder.

## Safety mechanisms

### Recursion guard

```bash
if [ "${HSTL_OSS_HOOK_RUNNING:-}" = "1" ]; then
    exit 0
fi
export HSTL_OSS_HOOK_RUNNING=1
```

### Timeout protection

```bash
TIMEOUT_SECONDS=8
(
    sleep $TIMEOUT_SECONDS
    kill -9 $$ 2>/dev/null
) &
```

### Non-blocking exit

- All hooks exit with code 0 (success).
- On error they exit silently.

## Manual test

```bash
bash hooks/session-context.sh
```
