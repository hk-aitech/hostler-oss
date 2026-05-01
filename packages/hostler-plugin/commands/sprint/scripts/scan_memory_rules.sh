#!/usr/bin/env bash
#
# scan_memory_rules.sh — sprint:create pre-check helper script.
#
# Purpose: when the user memory rule
# `feedback_no_prod_dev_change_without_explicit.md` is active, scan the Sprint
# Task candidate strings for Dev/Prod/production/operations keywords. If any are
# found, render a WARN block + a "LocalDev-only redefinition" suggestion to
# stdout. The AI inserts that output into the sprint:create briefing and asks
# the user [y/edit/cancel].
#
# Background: when refactoring "Dev environment deployment verification"
# repeatedly clashed with the memory rule, manual re-edits became a recurring
# cost. Automatic detection removes the re-edit step.
#
# Usage:
#   scan_memory_rules.sh <memory-dir> "<text-1>" "<text-2>" ...   # texts: title/summary/criteria
#   scan_memory_rules.sh --self-test
#
# Inputs:
#   - memory-dir: e.g. ~/.claude/projects/-home-<user>-projects-<proj>/memory
#     If absent, silent skip (works in non-Claude-Code environments too).
#   - text args: strings to scan, such as the Sprint goal + Task candidate titles.
#
# Outputs:
#   - No detection or memory missing: empty stdout + exit 0.
#   - Detected: WARN block (markdown) + exit 0. The AI mixes the body into the briefing.
#
# shellcheck disable=SC2155

set -uo pipefail

# -----------------------------------------------------------------------------
# Constants
# -----------------------------------------------------------------------------

# Regex used to determine whether a feedback_*.md file represents an active rule.
readonly RULE_ACTIVITY_RE='no_prod_dev|prod.*forbidden|dev.*explicit|Prod/Dev|production.*change'
# Keywords to detect in Sprint Task candidates (case-insensitive).
readonly TARGET_KEYWORD_RE='Dev|Prod|production|operations environment'
# memory file pattern to search.
readonly MEMORY_PATTERN='feedback_*.md'

# -----------------------------------------------------------------------------
# Logging (stderr)
# -----------------------------------------------------------------------------

log_warn() {
    printf '[scan_memory_rules] WARN: %s\n' "$*" >&2
}

# -----------------------------------------------------------------------------
# Extract the list of active rule files from the memory directory
# -----------------------------------------------------------------------------

# find_active_rules MEMORY_DIR
#   Print, one per line, the basename of each feedback_*.md whose body matches
#   RULE_ACTIVITY_RE. Empty output if the directory is missing or has no matches.
find_active_rules() {
    local dir="$1"
    if [[ ! -d "$dir" ]]; then
        return 0
    fi
    local file
    # Use explicit find rather than relying on shopt nullglob, so that the loop
    # does not iterate over the literal wildcard when no files match.
    while IFS= read -r file; do
        [[ -z "$file" ]] && continue
        if grep -qiE "$RULE_ACTIVITY_RE" "$file" 2>/dev/null; then
            basename "$file"
        fi
    done < <(find "$dir" -maxdepth 1 -type f -name "$MEMORY_PATTERN" 2>/dev/null | sort)
}

# -----------------------------------------------------------------------------
# Detect keywords in candidate text
# -----------------------------------------------------------------------------

# detect_keywords TEXT
#   If TEXT matches TARGET_KEYWORD_RE, print the matched tokens one per line.
detect_keywords() {
    local text="$1"
    # -i (case-insensitive) + -E + -o (matched tokens only).
    # A 0-match grep is a normal flow -> suppress propagation with || true.
    printf '%s' "$text" | grep -oiE "$TARGET_KEYWORD_RE" | sort -u || true
}

# -----------------------------------------------------------------------------
# Render the WARN block
# -----------------------------------------------------------------------------

render_warning() {
    local rule_files="$1"   # newline-separated
    local matched_text="$2"
    local matched_keyword="$3"

    printf '\nWARN: possible user memory rule conflict\n'
    printf '- Active rules:\n'
    while IFS= read -r rf; do
        [[ -z "$rf" ]] && continue
        printf '    - %s\n' "$rf"
    done <<< "$rule_files"
    printf '- Detected keyword: %s\n' "$matched_keyword"
    printf '- Detected Task candidate: %s\n' "$matched_text"
    printf '- Suggestion: "LocalDev integrated verification — Dev/Prod deploy deferred to a separate stage with explicit user approval"\n'
    printf '\nProceed? [y/edit/cancel]\n'
}

# -----------------------------------------------------------------------------
# Main: scan + render
# -----------------------------------------------------------------------------

# scan_and_render MEMORY_DIR TEXTS...
scan_and_render() {
    local dir="$1"
    shift
    local texts=("$@")

    local rules
    rules=$(find_active_rules "$dir")
    if [[ -z "$rules" ]]; then
        # Rule inactive or memory missing — silent skip.
        return 0
    fi

    local t
    for t in "${texts[@]}"; do
        local hits
        hits=$(detect_keywords "$t")
        if [[ -n "$hits" ]]; then
            # Warn on the first match only — single WARN block even with multiple
            # hits. Persistent matches encourage the user to do a full review.
            local first
            first=$(printf '%s\n' "$hits" | head -1)
            render_warning "$rules" "$t" "$first"
            return 0
        fi
    done
}

# -----------------------------------------------------------------------------
# Built-in self-test
# -----------------------------------------------------------------------------

self_test() {
    _st_tmpdir=$(mktemp -d)
    local tmpdir="$_st_tmpdir"
    trap 'rm -rf "$_st_tmpdir"' EXIT

    local pass=0 fail=0
    assert_contains() {
        local name="$1" needle="$2" haystack="$3"
        if [[ "$haystack" == *"$needle"* ]]; then
            printf '  PASS: %s\n' "$name"
            pass=$((pass + 1))
        else
            printf '  FAIL: %s\n    needle: %s\n    haystack: %s\n' "$name" "$needle" "$haystack"
            fail=$((fail + 1))
        fi
    }
    assert_empty() {
        local name="$1" actual="$2"
        if [[ -z "$actual" ]]; then
            printf '  PASS: %s\n' "$name"
            pass=$((pass + 1))
        else
            printf '  FAIL: %s\n    actual: %s\n' "$name" "$actual"
            fail=$((fail + 1))
        fi
    }

    # Set up memory directory and an active rule file
    local memdir="$tmpdir/memory"
    mkdir -p "$memdir"
    printf 'name: no_prod_dev\nactive rule: prod forbidden\n' > "$memdir/feedback_no_prod_dev.md"
    # Inactive file (no rule)
    printf 'unrelated content\n' > "$memdir/feedback_other_topic.md"

    # Only the active rule file should be picked up
    local active
    active=$(find_active_rules "$memdir")
    assert_contains "active rule detected" "feedback_no_prod_dev.md" "$active"
    if [[ "$active" == *"feedback_other_topic.md"* ]]; then
        printf '  FAIL: inactive file false positive — %s\n' "$active"
        fail=$((fail + 1))
    else
        pass=$((pass + 1))
        printf '  PASS: inactive file excluded\n'
    fi

    # Keyword detection — multiple matches deduplicated
    assert_contains "Dev keyword detected" "Dev" "$(detect_keywords 'Dev environment deployment Task')"
    assert_contains "production detected" "production" "$(detect_keywords 'production deployment protection')"
    assert_empty "no keyword — unrelated text" "$(detect_keywords 'feature A implementation')"
    # Note: "LocalDev" partially matches Dev -> grep -o captures Dev.
    # Intended behavior — even if it produces a false positive, the user can pass
    # via "proceed" after the WARN, and catching more real conflicts is the
    # pragmatic safety choice.
    assert_contains "LocalDev -> Dev partial match (intended FP)" "Dev" "$(detect_keywords 'LocalDev')"

    # scan_and_render: active + matched -> WARN block
    local rendered
    rendered=$(scan_and_render "$memdir" "Dev environment deployment verification")
    assert_contains "WARN block rendered" "memory rule conflict" "$rendered"
    assert_contains "suggestion phrase included" "LocalDev integrated verification" "$rendered"

    # Active + no match -> empty output
    assert_empty "no match — skip" "$(scan_and_render "$memdir" "feature A implementation")"

    # memory missing -> empty output
    assert_empty "memory directory missing — skip" "$(scan_and_render "$tmpdir/nonexistent" "Dev")"

    # Only inactive rules -> empty output
    rm "$memdir/feedback_no_prod_dev.md"
    assert_empty "no active rules — skip" "$(scan_and_render "$memdir" "Dev environment deploy")"

    printf '\nself-test: %d pass / %d fail\n' "$pass" "$fail"
    [[ "$fail" -eq 0 ]]
}

# -----------------------------------------------------------------------------
# Entry point
# -----------------------------------------------------------------------------

main() {
    if [[ $# -eq 0 ]]; then
        cat <<USAGE
Usage: $(basename "$0") <memory-dir> "<text-1>" [...]
       $(basename "$0") --self-test
USAGE
        return 0
    fi
    case "$1" in
        --self-test) self_test ;;
        --help|-h) main ;;
        *)
            local dir="$1"
            shift
            scan_and_render "$dir" "$@"
            ;;
    esac
}

main "$@"
