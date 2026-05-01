#!/bin/bash
# =============================================================================
# pre-commit-check.sh — Git pre-commit hook script
# hstl-oss Plugin
#
# Validation steps:
#   1. Commit message format: `{type}: ` prefix required
#      Allowed types: feat / fix / refactor / docs / test / chore
#   2. ruff check (only when installed)
#
# Installation (run from project root):
#   cp /path/to/hstl-oss-plugin/hooks/pre-commit-check.sh .git/hooks/pre-commit
#   chmod +x .git/hooks/pre-commit
#
# Or via symlink:
#   ln -sf /path/to/hstl-oss-plugin/hooks/pre-commit-check.sh .git/hooks/pre-commit
#
# Note: not registered in hooks.json — install per-project as a git pre-commit hook.
# =============================================================================

set -euo pipefail

# ===== RECURSION GUARD =====
if [ "${HOSTLER_OSS_PRECOMMIT_RUNNING:-}" = "1" ]; then
    exit 0
fi
export HOSTLER_OSS_PRECOMMIT_RUNNING=1

# ===== Colour output =====
RED='\033[0;31m'
# shellcheck disable=SC2034  # palette completeness — kept for consistency with other hook scripts and future messages
YELLOW='\033[1;33m'
GREEN='\033[0;32m'
NC='\033[0m' # No Color

ERRORS=0

# ===== 1. Commit message format check =====
check_commit_message() {
    # COMMIT_EDITMSG file path
    local msg_file="${GIT_DIR:-.git}/COMMIT_EDITMSG"

    # No file → cannot validate, treat as pass
    if [ ! -f "$msg_file" ]; then
        return 0
    fi

    local first_line
    first_line=$(head -1 "$msg_file")

    # Allowed types
    local allowed_types="feat|fix|refactor|docs|test|chore"
    local pattern="^(${allowed_types}): .+"

    if ! echo "$first_line" | grep -qE "$pattern"; then
        echo -e "${RED}[ERROR] Commit message format error${NC}"
        echo ""
        echo "  current: $first_line"
        echo ""
        echo "  format:  {type}: {description}"
        echo "  types:   feat | fix | refactor | docs | test | chore"
        echo ""
        echo "  examples:"
        echo "    feat: implement user authentication"
        echo "    fix: resolve login token expiry bug"
        echo "    docs: update API specification"
        echo ""
        ERRORS=$((ERRORS + 1))
    else
        echo -e "${GREEN}[OK] Commit message format ok${NC}"
        echo "     $first_line"
    fi
}

# ===== 2. ruff lint check =====
check_ruff() {
    # Skip silently if ruff is not installed
    if ! command -v ruff &>/dev/null; then
        return 0
    fi

    # Staged Python files
    local staged_py_files
    staged_py_files=$(git diff --cached --name-only --diff-filter=ACM | grep '\.py$' || true)

    if [ -z "$staged_py_files" ]; then
        # No Python files → skip
        return 0
    fi

    echo ""
    echo "=== ruff check ==="

    local ruff_output
    local ruff_exit=0
    # shellcheck disable=SC2086  # $staged_py_files is a whitespace-separated list — splitting is intentional
    ruff_output=$(ruff check $staged_py_files 2>&1) || ruff_exit=$?

    if [ $ruff_exit -ne 0 ]; then
        echo -e "${RED}[ERROR] ruff lint errors found${NC}"
        echo "$ruff_output"
        ERRORS=$((ERRORS + 1))
    else
        echo -e "${GREEN}[OK] ruff lint passed${NC}"
        if [ -n "$ruff_output" ]; then
            echo "$ruff_output"
        fi
    fi
}

# ===== MAIN =====

echo "=== hstl-oss pre-commit checks ==="
echo ""

check_commit_message
check_ruff

echo ""

if [ $ERRORS -gt 0 ]; then
    echo -e "${RED}[FAIL] ${ERRORS} error(s) — commit blocked${NC}"
    echo ""
    echo "Fix the errors above and commit again."
    echo "To skip the checks (not recommended): git commit --no-verify"
    exit 1
fi

echo -e "${GREEN}=== Checks passed — proceeding with commit ===${NC}"
exit 0
