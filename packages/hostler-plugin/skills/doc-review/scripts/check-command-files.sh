#!/bin/bash
# ============================================================
# check-command-files.sh — quality validation for commands/ files
#
# Usage: check-command-files.sh [path]
# Example: check-command-files.sh commands/sprint/
#

# This script was added because doc-review's scan missed the CEREMONY.md
# stale reference, allowing it to recur. It scans every commands/**/*.md
# for stale patterns and basic frontmatter sanity.
# ============================================================

# Do not use set -e — a `grep` exit 1 would abort the whole loop
set -uo pipefail

TARGET="${1:-commands}"
PASS=0
FAIL=0
WARN=0

# Stale-pattern list — literal strings observed in project history as
# "deleted but still referenced". Use literal strings only (no regex)
# so that the rule remains "verifiable". Each entry is separated by "|<description>".
STALE_PATTERNS=(
    "CEREMONY.md|deprecated at this stage — harness_items is the SSOT"
    "SESSION-CONTEXT.md|replaced by CURRENT-FOCUS.md"
)

echo "=== Command Files Quality Check ==="
echo "Target: $TARGET"
echo ""

while IFS= read -r -d '' f; do
    filename=$(basename "$f")
    errors=""
    warnings=""

    # ════════════════════════════════════════════
    # 1. Frontmatter present (BLOCK)
    # By Claude Code slash-command convention, commands/*.md must start with
    # `---` frontmatter (description, argument-hint, etc.).
    # ════════════════════════════════════════════
    if ! head -n 1 "$f" | grep -q "^---$" 2>/dev/null; then
        errors="${errors}\n  [X] frontmatter missing — first line must be '---'"
    fi

    # ════════════════════════════════════════════
    # 2. description field present (BLOCK)
    # ════════════════════════════════════════════
    if ! grep -q "^description:" "$f" 2>/dev/null; then
        errors="${errors}\n  [X] frontmatter 'description' field missing"
    fi

    # ════════════════════════════════════════════
    # 3. Stale pattern detection (BLOCK)
    # ════════════════════════════════════════════
    for pattern_entry in "${STALE_PATTERNS[@]}"; do
        pattern="${pattern_entry%%|*}"
        reason="${pattern_entry#*|}"
        if grep -qF "$pattern" "$f" 2>/dev/null; then
            # If CEREMONY.md appears intentionally in a historical context
            # (e.g. "CEREMONY.md deprecated"), allow it — keyword-based check.
            if grep -qP "CEREMONY\.md.*(deprecat|removed)" "$f" 2>/dev/null; then
                continue
            fi
            errors="${errors}\n  [X] stale reference '${pattern}' — ${reason}"
        fi
    done

    # ════════════════════════════════════════════
    # 4. H1 title present (WARN)
    # ════════════════════════════════════════════
    if ! grep -qP "^# " "$f" 2>/dev/null; then
        warnings="${warnings}\n  [!] no H1 title (# ...)"
    fi

    # ════════════════════════════════════════════
    # Output
    # ════════════════════════════════════════════
    if [ -n "$errors" ]; then
        echo "FAIL: $f"
        echo -e "$errors"
        [ -n "$warnings" ] && echo -e "$warnings"
        FAIL=$((FAIL + 1))
    elif [ -n "$warnings" ]; then
        echo "WARN: $f"
        echo -e "$warnings"
        WARN=$((WARN + 1))
    else
        PASS=$((PASS + 1))
    fi

done < <(find "$TARGET" -name "*.md" -type f -print0 2>/dev/null)

echo ""
echo "=== Summary ==="
echo "  PASS: $PASS | FAIL: $FAIL | WARN: $WARN"
echo "  Total: $((PASS + FAIL + WARN))"
echo "==============="

[ "$FAIL" -eq 0 ] && exit 0 || exit 1
