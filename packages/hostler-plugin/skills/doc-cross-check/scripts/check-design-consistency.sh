#!/usr/bin/env bash
# check-design-consistency.sh — automated consistency check for design docs
#
# Usage:
#   ./check-design-consistency.sh [DESIGN_DIR]
#
# Args:
#   DESIGN_DIR  directory to inspect (default: current directory)
#
# Checks:
#   1. Detect duplicate Feature IDs
#   2. Verify per-Context declared totals
#   3. Detect broken Markdown links
#
# Output:
#   Report on stdout. Exits with code 1 on any failure.

set -uo pipefail
# Do not use `set -e` — individual command failures (e.g., grep exit 1) would abort the whole script

DESIGN_DIR="${1:-.}"
ERRORS=0
WARNINGS=0

RED='\033[0;31m'
YELLOW='\033[1;33m'
GREEN='\033[0;32m'
RESET='\033[0m'

echo "═══════════════════════════════════════════════"
echo "  DESIGN CONSISTENCY CHECK"
echo "  Target: ${DESIGN_DIR}"
echo "  Time:   $(date '+%Y-%m-%d %H:%M:%S')"
echo "═══════════════════════════════════════════════"
echo ""

# ─────────────────────────────────────────────────
# Phase 1: Detect duplicate Feature IDs
# ─────────────────────────────────────────────────
echo "Phase 1: Detect duplicate Feature IDs"

# Feature ID pattern: {3 uppercase letters}-F{3 digits}
FEATURE_IDS=$(grep -rh --include="*.md" -oE '[A-Z]{3}-F[0-9]{3}' "${DESIGN_DIR}" 2>/dev/null | sort)

if [ -z "$FEATURE_IDS" ]; then
    echo "  ⚠️  No Feature IDs found (pattern: [A-Z]{3}-F[0-9]{3})"
    ((WARNINGS++)) || true
else
    TOTAL_COUNT=$(echo "$FEATURE_IDS" | wc -l | tr -d ' ')
    UNIQUE_COUNT=$(echo "$FEATURE_IDS" | sort -u | wc -l | tr -d ' ')

    DUPLICATES=$(echo "$FEATURE_IDS" | sort | uniq -d)
    if [ -n "$DUPLICATES" ]; then
        echo -e "  ${RED}❌ CRITICAL: duplicate Feature IDs found:${RESET}"
        echo "$DUPLICATES" | while read -r id; do
            FILES=$(grep -rl --include="*.md" "$id" "${DESIGN_DIR}" 2>/dev/null | tr '\n' ' ')
            echo "     ${id} → ${FILES}"
        done
        ((ERRORS++)) || true
    else
        echo -e "  ${GREEN}✅ ${UNIQUE_COUNT} Feature IDs — no duplicates${RESET}"
    fi
fi
echo ""

# ─────────────────────────────────────────────────
# Phase 2: Detect declared count vs actual total mismatches
# ─────────────────────────────────────────────────
echo "Phase 2: Feature count declaration consistency"

# Detect numeric declarations of the form "Feature count: N" or "N Features"
DECLARED_COUNTS=$(grep -rn --include="*.md" -iE 'Feature\s*count\s*:\s*[0-9]+|[0-9]+\s*Features?\b' \
    "${DESIGN_DIR}" 2>/dev/null | head -20)

if [ -z "$DECLARED_COUNTS" ]; then
    echo "  ℹ️  No numeric declaration patterns found (manual verification needed)"
else
    echo "  Numeric declarations found (manual verification recommended):"
    echo "$DECLARED_COUNTS" | while read -r line; do
        echo "     $line"
    done
    echo -e "  ${YELLOW}⚠️  Cannot auto-verify — recommend manual comparison against actual catalog row count${RESET}"
    ((WARNINGS++)) || true
fi
echo ""

# ─────────────────────────────────────────────────
# Phase 3: Detect broken Markdown links
# ─────────────────────────────────────────────────
echo "Phase 3: Markdown link health check"

BROKEN_LINKS=0
CHECKED_LINKS=0

while IFS= read -r -d '' md_file; do
    # Extract relative-path links from the [text](path) pattern
    # Skip http/https, anchors (#...), and mailto links
    while IFS= read -r link; do
        # Normalize the link
        link_path=$(echo "$link" | sed 's/#.*//')  # strip anchor
        [ -z "$link_path" ] && continue

        # Skip absolute paths and http links
        [[ "$link_path" =~ ^https?:// ]] && continue
        [[ "$link_path" =~ ^/ ]] && continue
        [[ "$link_path" =~ ^mailto: ]] && continue

        # Resolve against the base directory and check existence
        md_dir=$(dirname "$md_file")
        resolved="${md_dir}/${link_path}"

        ((CHECKED_LINKS++)) || true

        if [ ! -e "$resolved" ]; then
            if [ $BROKEN_LINKS -eq 0 ]; then
                echo -e "  ${RED}❌ Broken links found:${RESET}"
            fi
            echo "     ${md_file}: [${link_path}]"
            ((BROKEN_LINKS++)) || true
            ((ERRORS++)) || true
        fi
    done < <(grep -oP '\]\(\K[^)]+' "$md_file" 2>/dev/null)
done < <(find "${DESIGN_DIR}" -name "*.md" -print0 2>/dev/null)

if [ $BROKEN_LINKS -eq 0 ]; then
    echo -e "  ${GREEN}✅ Checked ${CHECKED_LINKS} links — none broken${RESET}"
else
    echo "  ${BROKEN_LINKS} broken links found (out of ${CHECKED_LINKS} inspected)"
fi
echo ""

# ─────────────────────────────────────────────────
# Summary
# ─────────────────────────────────────────────────
echo "═══════════════════════════════════════════════"
if [ $ERRORS -gt 0 ]; then
    echo -e "  ${RED}Result: FAILED — ${ERRORS} CRITICAL/ERROR, ${WARNINGS} WARNING${RESET}"
    echo ""
    echo "  Next steps:"
    echo "  1. Fix the errors above."
    echo "  2. Re-run this script to verify."
    echo "  3. Run the design-audit skill for the full 7-phase inspection."
    echo "═══════════════════════════════════════════════"
    exit 1
else
    echo -e "  ${GREEN}Result: PASSED — 0 ERROR, ${WARNINGS} WARNING${RESET}"
    echo "═══════════════════════════════════════════════"
    exit 0
fi
