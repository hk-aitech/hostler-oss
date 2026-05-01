#!/bin/bash
# =============================================================================
# Session Context Hook (lightweight)
# Compact (<500 token) output — optimised for AI / subagent prompt injection.
#
# Output format:
#   [project] branch:main uncommitted:N ahead:N
#   [sprint] sprint-NN "title" N/N done (NN%)
#   [task] TNN "title" (in-progress) | next: TNN "title"
#   [rules] Conventional commits, const required, jq used, Command First
#   [issues] WARN: N | none
# =============================================================================

# ===== Recursion guard =====
if [ "${HSTL_OSS_HOOK_RUNNING:-}" = "1" ]; then
    exit 0
fi
export HSTL_OSS_HOOK_RUNNING=1

# ===== Constants =====
readonly HOOK_TIMEOUT_SEC="${HSTL_OSS_SESSION_TIMEOUT_SEC:-8}"

# ===== Timeout =====
(
    sleep "$HOOK_TIMEOUT_SEC"
    kill -9 $$ 2>/dev/null
) &
TIMEOUT_PID=$!
trap "kill $TIMEOUT_PID 2>/dev/null" EXIT

# ===== Error handler =====
trap 'exit 0' ERR

# ===== Project root =====
PROJECT_ROOT="$(git rev-parse --show-toplevel 2>/dev/null || true)"
if [ -z "$PROJECT_ROOT" ]; then
    exit 0
fi
cd "$PROJECT_ROOT" || exit 0

# ===== CLI path =====
HSTL_OSS_CLI="${HSTL_OSS_CLI:-}"
if [ -z "$HSTL_OSS_CLI" ]; then
    if [ -x "$PROJECT_ROOT/mcp-server/bin/hstl-oss" ]; then
        HSTL_OSS_CLI="$PROJECT_ROOT/mcp-server/bin/hstl-oss"
    elif command -v hstl-oss &>/dev/null; then
        HSTL_OSS_CLI="hstl-oss"
    fi
fi

_HAS_JQ=0
command -v jq &>/dev/null && _HAS_JQ=1

use_cli() {
    [ -n "$HSTL_OSS_CLI" ] && [ "$_HAS_JQ" -eq 1 ]
}

# ===== Colours =====
readonly CYAN='\033[0;36m'
readonly YELLOW='\033[0;33m'
readonly GREEN='\033[0;32m'
readonly BLUE='\033[0;34m'
readonly NC='\033[0m'

WARN_COUNT=0
INFO_COUNT=0

echo -e "${CYAN}=== hstl-oss Session Context ===${NC}"

# ===== works/ check =====
if [ ! -d "works" ]; then
    echo -e "${YELLOW}[WARN]${NC} works/ directory missing"
    ((WARN_COUNT++)) || true
else
    echo -e "${GREEN}[PASS]${NC} works/ directory present"
fi

# ===== [project] Git status =====
BRANCH=$(git branch --show-current 2>/dev/null || echo 'detached')
UNCOMMITTED=$(git status --short 2>/dev/null | wc -l | tr -d ' ')
AHEAD=$(git rev-list --count @{upstream}..HEAD 2>/dev/null || echo "0")
BEHIND=$(git rev-list --count HEAD..@{upstream} 2>/dev/null || echo "0")

echo "[project] branch:${BRANCH} uncommitted:${UNCOMMITTED} ahead:${AHEAD} behind:${BEHIND}"

if [ "$UNCOMMITTED" -gt 0 ]; then
    echo -e "${BLUE}[INFO]${NC} ${UNCOMMITTED} uncommitted change(s)"
    git status --short 2>/dev/null | head -5
    ((INFO_COUNT++)) || true
fi

if [ "$AHEAD" -gt 0 ] || [ "$BEHIND" -gt 0 ]; then
    echo -e "${BLUE}[INFO]${NC} remote sync needed (ahead: $AHEAD, behind: $BEHIND)"
    ((INFO_COUNT++)) || true
fi

# ===== [sprint] Active sprint =====
if use_cli; then
    _sprint_json=$("$HSTL_OSS_CLI" sprint list -o json -q --status active 2>/dev/null || echo "")
    if [ -n "$_sprint_json" ]; then
        _sprint_id=$(echo "$_sprint_json" | jq -r '.sprints[0].sprint_id // empty' 2>/dev/null)
        if [ -n "$_sprint_id" ]; then
            _progress=$("$HSTL_OSS_CLI" sprint progress "$_sprint_id" -o json -q 2>/dev/null || echo "")
            _title=$(echo "$_sprint_json" | jq -r '.sprints[0].title // ""' 2>/dev/null)
            _done=$(echo "$_progress" | jq -r '.done // 0' 2>/dev/null)
            _total=$(echo "$_progress" | jq -r '.total // 0' 2>/dev/null)
            _pct=$(echo "$_progress" | jq -r '.percent // 0' 2>/dev/null)
            echo "[sprint] ${_sprint_id} \"${_title}\" ${_done}/${_total} done (${_pct}%)"
        else
            echo "[sprint] none"
        fi
    else
        echo "[sprint] none"
    fi
else
    echo "[sprint] CLI unavailable"
fi

# ===== [task] Current task + next =====
if use_cli; then
    _task_json=$("$HSTL_OSS_CLI" task list -o json -q --status in-progress 2>/dev/null || echo "")
    _task_id=$(echo "$_task_json" | jq -r '.tasks[0].task_id // empty' 2>/dev/null)
    _task_title=$(echo "$_task_json" | jq -r '.tasks[0].title // empty' 2>/dev/null)

    _next_json=$("$HSTL_OSS_CLI" task next -o json -q 2>/dev/null || echo "")
    _next_id=$(echo "$_next_json" | jq -r '.task_id // empty' 2>/dev/null)
    _next_title=$(echo "$_next_json" | jq -r '.title // empty' 2>/dev/null)

    if [ -n "$_task_id" ]; then
        echo "[task] ${_task_id} \"${_task_title}\" (in-progress) | next: ${_next_id:-none}"
    elif [ -n "$_next_id" ]; then
        echo "[task] none | next: ${_next_id} \"${_next_title}\""
    else
        echo "[task] none"
    fi
else
    echo "[task] CLI unavailable"
fi

# ===== [rules] Core rules =====
echo "[rules] Conventional commits, const required, jq used, Command First, 1 Task = 1 commit"

# ===== [issues] Warnings =====

if [ "$WARN_COUNT" -gt 0 ]; then
    echo "[issues] WARN: ${WARN_COUNT}"
else
    echo "[issues] none"
fi

# ===== Workflow reminder =====
echo ""
echo -e "${CYAN}=== Workflow reminder ===${NC}"
echo "Load /hstl-oss:task-management before the first response in a session (reference guide)."
echo "Invoke procedure skills only when needed:"
echo "  Task start    → /hstl-oss:task:start"
echo "  Task complete → /hstl-oss:task:complete"
echo "  Sprint done   → /hstl-oss:sprint:complete"

echo -e "${GREEN}=== Ready ===${NC}"

exit 0
