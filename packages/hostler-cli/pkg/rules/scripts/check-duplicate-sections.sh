#!/usr/bin/env bash
# check-duplicate-sections.sh
# Detects duplicate ## sections with the same title in Task markdown files.
# This is common when a Task body is renamed or the result section is
# edited multiple times.
#
# Subjects: staged works/tasks/*.md + works/sprints/**/tasks/*.md.
# Operations: callable from pre-commit rule
# `precommit.task.duplicate_sections`. strict: HOSTLER_DUP_SECTIONS_CHECK=strict
# → exit 1 on duplicate.
set -uo pipefail

MODE="${HOSTLER_DUP_SECTIONS_CHECK:-warn}"

staged=$(git diff --cached --name-only --diff-filter=AM 2>/dev/null | grep -E 'works/(tasks|sprints)/.*T[0-9]+.*\.md$' || true)
if [ -z "$staged" ]; then
    exit 0
fi

# Section titles to inspect (## level — the ones that commonly duplicate).
TARGETS=(
    "## Result"
    "## Status Change History"
    "## Requirements"
    "## Done Criteria"
    "## References"
)

found=0
while IFS= read -r f; do
    [ -z "$f" ] && continue
    [ -f "$f" ] || continue
    for section in "${TARGETS[@]}"; do
        n=$(grep -cE "^${section}$" "$f" 2>/dev/null || echo 0)
        if [ "${n:-0}" -gt 1 ]; then
            echo "[duplicate-section] $f — '${section}' appears ${n} times (duplicate)"
            found=$((found + 1))
        fi
    done
done <<< "$staged"

if [ $found -eq 0 ]; then
    exit 0
fi

echo ""
echo "[duplicate-section] ${found} occurrence(s) total"
echo "  Cleanup: manually merge or remove duplicate sections."
echo "  Strict block: HOSTLER_DUP_SECTIONS_CHECK=strict git commit"

if [ "$MODE" = "strict" ]; then
    exit 1
fi
exit 0
