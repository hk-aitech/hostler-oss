#!/bin/bash
# =============================================================================
# Subagent Context Ack Hook
#
# Just before Claude Code spawns a subagent (Agent tool), record a context
# acknowledgement for the active task's work_ticket. Even when the AI forgets
# to call `hstl-oss context` manually, the ack is captured at subagent
# invocation time so the task-completion gate keeps passing.
#
# Design:
#   - matcher: PreToolUse + tool=Agent (subagent invocation)
#   - no active task / no work_ticket → no-op (fail-soft)
#   - `hstl-oss context --ticket` failure → warn only, do not block the call
#
# Defensive design: never assume the AI invokes context manually.
# =============================================================================

set +e  # fail-soft: never exit non-zero

readonly HOSTLER_OSS_HOOK="[subagent-context-ack]"

# Recursion guard — prevent the hstl-oss invocation inside this hook from
# triggering further hooks.
if [ "${HOSTLER_OSS_SUBAGENT_ACK_RUNNING:-}" = "1" ]; then
    exit 0
fi
export HOSTLER_OSS_SUBAGENT_ACK_RUNNING=1

# hstl-oss binary not installed → silent skip
if ! command -v hstl-oss >/dev/null 2>&1; then
    exit 0
fi

# jq required
if ! command -v jq >/dev/null 2>&1; then
    echo "$HOSTLER_OSS_HOOK jq not installed — skip" >&2
    exit 0
fi

# Look up the single in-progress task (response is .tasks array with count).
TASK_ID="$(hstl-oss task list --status in-progress -o json 2>/dev/null | jq -r '.tasks[0].task_id // empty' 2>/dev/null)"
if [ -z "$TASK_ID" ]; then
    exit 0
fi

# work_ticket lives in the task get frontmatter block.
# (the tasks list does not include work_ticket.)
TICKET="$(hstl-oss task get "$TASK_ID" -o json 2>/dev/null | jq -r '.frontmatter.work_ticket // empty' 2>/dev/null)"
if [ -z "$TICKET" ]; then
    # Active task exists but no ticket yet (task start not invoked, or legacy) → skip
    exit 0
fi

# Invoke context → ack auto-stored. Drop stderr for non-intrusive behaviour.
hstl-oss context --ticket "$TICKET" >/dev/null 2>&1 || {
    echo "$HOSTLER_OSS_HOOK context ack failed (fail-soft): ticket=$TICKET" >&2
}

exit 0
