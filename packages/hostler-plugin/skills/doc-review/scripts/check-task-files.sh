#!/bin/bash
# ============================================================
# check-task-files.sh — Task file quality check (presence + content quality)
#
# Usage: check-task-files.sh [path]
# Example: check-task-files.sh works/sprints/active/
# ============================================================

# Do not use set -e — failures of individual commands like `grep` (exit 1)
# would abort the whole loop.
set -uo pipefail

TARGET="${1:-works/sprints}"
PASS=0
FAIL=0
WARN=0

echo "=== Task File Quality Check ==="
echo "Target: $TARGET"
echo ""

# --- helper: extract section content (from `## section` to just before the next `##`) ---
extract_section() {
    local file="$1" section="$2"
    sed -n "/^## ${section}/,/^## /p" "$file" 2>/dev/null | grep -v "^## " | grep -v "^$"
}

# --- helper: extract a frontmatter value ---
get_fm() {
    local file="$1" field="$2"
    grep -m1 "^${field}:" "$file" 2>/dev/null | sed "s/^${field}:[[:space:]]*//" | tr -d '[:space:]"'
}

while IFS= read -r -d '' f; do
    filename=$(basename "$f")
    errors=""
    warnings=""

    # ════════════════════════════════════════════
    # 1. Filename pattern (BLOCK)
    # Allow only \p{L} (unicode letter) + \p{N} (unicode number) + underscore +
    # hyphen. Spaces, special chars, and control chars remain rejected. Requires
    # Unicode support in `grep -P` (PCRE), compatible with GNU grep 2.5+.
    # ════════════════════════════════════════════
    if ! echo "$filename" | grep -qP '^T\d+-[\p{L}\p{N}_-]+\.md$'; then
        errors="${errors}\n  [X] filename pattern mismatch: $filename"
    fi

    # ════════════════════════════════════════════
    # 2. Frontmatter fields present (BLOCK)
    # ════════════════════════════════════════════
    for field in id title sprint status priority estimate depends_on created; do
        if ! grep -q "^${field}:" "$f" 2>/dev/null; then
            errors="${errors}\n  [X] frontmatter '${field}' missing"
        fi
    done

    # ════════════════════════════════════════════
    # 3. Frontmatter value validity (BLOCK)
    # ════════════════════════════════════════════
    fm_status=$(get_fm "$f" "status")
    fm_priority=$(get_fm "$f" "priority")
    fm_estimate=$(get_fm "$f" "estimate")

    case "$fm_status" in
        todo|in-progress|done|blocked|cancelled|deferred) ;;
        *) errors="${errors}\n  [X] status invalid: '$fm_status' (todo/in-progress/done/blocked/cancelled/deferred)";;
    esac

    # priority: P0/P1/P2/P3 or p0/p1/p2/p3
    if ! echo "$fm_priority" | grep -qiP '^p[0-3]$'; then
        errors="${errors}\n  [X] priority invalid: '$fm_priority' (P0/P1/P2/P3)"
    fi

    # estimate: XS/S/M/L/XL or hour units (4h, 8h, etc.)
    if ! echo "$fm_estimate" | grep -qiP '^(XS|S|M|L|XL|\d+h)$'; then
        errors="${errors}\n  [X] estimate invalid: '$fm_estimate' (XS/S/M/L/XL/Nh)"
    fi

    # ════════════════════════════════════════════
    # 4. Required sections present + content quality (BLOCK)
    # ════════════════════════════════════════════

    # 4a. ## Requirements or ## Purpose — present + word-count + placeholder detection.
    req_section=""
    if grep -q "^## Requirements" "$f" 2>/dev/null; then
        req_section="Requirements"
    elif grep -q "^## Purpose" "$f" 2>/dev/null; then
        req_section="Purpose"
    fi

    if [ -z "$req_section" ]; then
        errors="${errors}\n  [X] '## Requirements' or '## Purpose' section missing"
    else
        content=$(extract_section "$f" "$req_section")
        word_count=$(echo "$content" | wc -w) || true
        if [ "${word_count:-0}" -lt 10 ]; then
            errors="${errors}\n  [X] '## ${req_section}' section is thin (${word_count:-0} words, need at least 10)"
        fi
        if echo "$content" | grep -qiP '(TBD|TODO)' 2>/dev/null; then
            warnings="${warnings}\n  [!] placeholder found in '## ${req_section}' (TBD/TODO/draft pending)"
        fi
    fi

    # 4b. ## Done Criteria — present + 2+ checkboxes + each at least 10 chars
    if ! grep -qE "^## Done Criteria$|^## Completion" "$f" 2>/dev/null; then
        errors="${errors}\n  [X] '## Done Criteria' section missing"
    else
        criteria=$(extract_section "$f" "Done Criteria")
        cb_count=$(echo "$criteria" | grep -c "\- \[" 2>/dev/null) || true
        if [ "${cb_count:-0}" -lt 2 ]; then
            errors="${errors}\n  [X] only ${cb_count:-0} completion-criteria checkbox(es) (need at least 2)"
        fi
        # each checkbox item at least 10 chars
        short_items=$(echo "$criteria" | grep "\- \[" | awk '{$1=$2=""; print}' | awk 'length < 10' | wc -l) || true
        if [ "${short_items:-0}" -gt 0 ]; then
            warnings="${warnings}\n  [!] ${short_items} completion-criteria item(s) under 10 chars (thin)"
        fi
    fi

    # 4c. Optional implementation-guide / scope section
    if ! grep -qE "^## Scope( Limits)?$|^## Implementation" "$f" 2>/dev/null; then
        warnings="${warnings}\n  [!] '## Scope' or '## Implementation' section missing"
    fi

    # ════════════════════════════════════════════
    # 5. depends_on reference check (WARN)
    # ════════════════════════════════════════════
    deps=$(grep "^depends_on:" "$f" 2>/dev/null | grep -oP 'T\d+' | head -10) || true
    for dep in $deps; do
        dep_file=$(find works/sprints -name "${dep}-*.md" -path "*/tasks/*" 2>/dev/null | head -1) || true
        if [ -z "$dep_file" ]; then
            warnings="${warnings}\n  [!] depends_on '${dep}' references a non-existent Task file"
        fi
    done

    # ════════════════════════════════════════════
    # 6. done-task validation (WARN)
    # ════════════════════════════════════════════
    if [ "$fm_status" = "done" ]; then
        if ! grep -qE "^## Result$" "$f" 2>/dev/null; then
            warnings="${warnings}\n  [!] done but '## Result' section missing"
        else
            result_content=$(extract_section "$f" "Result")
            result_words=$(echo "$result_content" | wc -w) || true
            if [ "${result_words:-0}" -lt 5 ]; then
                warnings="${warnings}\n  [!] '## Result' section is thin (${result_words:-0} words)"
            fi
            if echo "$result_content" | grep -qiP '(TBD|TODO)' 2>/dev/null; then
                warnings="${warnings}\n  [!] placeholder in '## Result' (e.g. 'fill in after completion')"
            fi
        fi
        # unchecked checkboxes
        unchecked=$(grep -c "\- \[ \]" "$f" 2>/dev/null) || true
        if [ "${unchecked:-0}" -gt 0 ]; then
            warnings="${warnings}\n  [!] done but ${unchecked} unchecked checkbox(es)"
        fi
    fi

    # ════════════════════════════════════════════
    # 7. Body-wide placeholder detection (WARN)
    # ════════════════════════════════════════════
    # Detect placeholders in the body, excluding the frontmatter region
    body_placeholders=$(sed '1,/^---$/d' "$f" 2>/dev/null | sed '/^---$/,/^---$/d' | grep -ciP '(TBD|TODO|FIXME|HACK)' 2>/dev/null) || true
    if [ "${body_placeholders:-0}" -gt 2 ]; then
        warnings="${warnings}\n  [!] ${body_placeholders} placeholder(s) in body (TBD/TODO/draft pending)"
    fi

    # ════════════════════════════════════════════
    # Output
    # ════════════════════════════════════════════
    if [ -n "$errors" ]; then
        echo "FAIL: $f"
        echo -e "$errors"
        [ -n "$warnings" ] && echo -e "$warnings"
        FAIL=$((FAIL + 1))
        # Gate-log entry — via hostler CLI (silent fail).
        if command -v hstl-oss >/dev/null 2>&1 && [ "${HOSTLER_OSS_GATE_JUDGEMENT_LOG:-on}" != "off" ]; then
            printf '{"gate_type":"doc_review","item_id":%q,"verdict":"block","rule_id":"doc-review.task","input_snippet":%q}\n' \
                "$f" "$errors" | hstl-oss gate-log record --from-stdin >/dev/null 2>&1 || true
        fi
    elif [ -n "$warnings" ]; then
        echo "WARN: $f"
        echo -e "$warnings"
        WARN=$((WARN + 1))
        if command -v hstl-oss >/dev/null 2>&1 && [ "${HOSTLER_OSS_GATE_JUDGEMENT_LOG:-on}" != "off" ]; then
            printf '{"gate_type":"doc_review","item_id":%q,"verdict":"warn","rule_id":"doc-review.task","input_snippet":%q}\n' \
                "$f" "$warnings" | hstl-oss gate-log record --from-stdin >/dev/null 2>&1 || true
        fi
    else
        PASS=$((PASS + 1))
    fi

done < <(find "$TARGET" -name "T*.md" -path "*/tasks/*" -print0 2>/dev/null)

echo ""
echo "=== Summary ==="
echo "  PASS: $PASS | FAIL: $FAIL | WARN: $WARN"
echo "  Total: $((PASS + FAIL + WARN))"
echo "==============="

[ "$FAIL" -eq 0 ] && exit 0 || exit 1
