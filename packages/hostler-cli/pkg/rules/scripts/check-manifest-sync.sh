#!/usr/bin/env bash
# hstl --manifest <-> Skill/Command consistency check.
#
# An executable extraction of the doc-cross-check SKILL.md Phase 11
# pseudocode. Auto-invoked from pre-commit / Makefile / integration-test.
#
# Checks:
#   1. manifest commands[].file_path exist
#   2. ceremony 6-command files exist (commands/task/*.md, commands/sprint/*.md)
#   3. ceremony=true commands expose the `--with-ceremony` flag in CLI help
#   4. Skill files exist (manifest.skills[]) when present
#
# Usage:
#   bash scripts/check-manifest-sync.sh
#
# Dependencies: jq, hstl CLI

set -euo pipefail

# Embedded execution support (resolved dynamically via git rev-parse).
PROJECT_ROOT="$(git rev-parse --show-toplevel 2>/dev/null || echo "$PWD")"
cd "$PROJECT_ROOT"

DRIFT_EXIT_CODE=1
SETUP_ERROR_EXIT_CODE=2

if ! command -v jq >/dev/null 2>&1; then
    echo "ERROR: jq not installed — install jq to run this check" >&2
    exit $SETUP_ERROR_EXIT_CODE
fi

if ! command -v hstl >/dev/null 2>&1; then
    echo "ERROR: hstl not installed — run 'cd cli && make install' first" >&2
    exit $SETUP_ERROR_EXIT_CODE
fi

MANIFEST_JSON="$(hstl --manifest -o json 2>/dev/null)"
if [ -z "$MANIFEST_JSON" ]; then
    echo "ERROR: hstl --manifest -o json returned an empty response" >&2
    exit $SETUP_ERROR_EXIT_CODE
fi

drift_count=0
drift_log=""

add_drift() {
    drift_count=$((drift_count + 1))
    drift_log="${drift_log}  - $1\n"
}

# 1. commands[].file_path exist (only when the file_path field is present)
while IFS= read -r fp; do
    [ -z "$fp" ] && continue
    if [ ! -f "$fp" ]; then
        add_drift "command file missing: $fp"
    fi
done < <(echo "$MANIFEST_JSON" | jq -r '.commands[]? | .file_path // empty')

# 2. ceremony 6-command files exist
declare -A CEREMONY_CMD_FILES=(
    ["hstl task create"]="commands/task/create.md"
    ["hstl task start"]="commands/task/start.md"
    ["hstl task complete"]="commands/task/complete.md"
    ["hstl sprint create"]="commands/sprint/create.md"
    ["hstl sprint start"]="commands/sprint/start.md"
    ["hstl sprint complete"]="commands/sprint/complete.md"
)

ceremony_cmds=$(echo "$MANIFEST_JSON" | jq -r '[.commands[] | select(.ceremony == true) | .name]')
for cmd in "${!CEREMONY_CMD_FILES[@]}"; do
    file="${CEREMONY_CMD_FILES[$cmd]}"
    declared=$(echo "$ceremony_cmds" | jq -r --arg c "$cmd" 'any(. == $c)')
    if [ "$declared" != "true" ]; then
        add_drift "manifest is missing ceremony=true: $cmd"
    fi
    if [ ! -f "$file" ]; then
        add_drift "ceremony command file missing: $cmd → $file"
    fi
done

# 3. ceremony commands expose --with-ceremony in CLI help
while IFS= read -r cmd; do
    [ -z "$cmd" ] && continue
    # "hstl task create" → "task create"
    args="${cmd#hstl }"
    if ! hstl $args --help 2>&1 | grep -q -- "--with-ceremony"; then
        add_drift "CLI flag missing: $cmd lacks --with-ceremony"
    fi
done < <(echo "$MANIFEST_JSON" | jq -r '.commands[]? | select(.ceremony == true) | .name')

# 4. Skill files exist (when present)
while IFS= read -r fp; do
    [ -z "$fp" ] && continue
    if [ ! -f "$fp" ]; then
        add_drift "skill file missing: $fp"
    fi
done < <(echo "$MANIFEST_JSON" | jq -r '.skills[]? | .file_path // empty')

# Result
if [ "$drift_count" -eq 0 ]; then
    echo "OK: manifest <-> Skill/Command consistent"
    exit 0
fi

echo "ERROR: manifest drift detected (${drift_count}):" >&2
printf "$drift_log" >&2
echo "" >&2
echo "How to resolve:" >&2
echo "  - Missing command file → create the file or fix the manifest schema" >&2
echo "  - Missing --with-ceremony flag → add it under cli/cmd/cli/cmd/<cmd>.go" >&2
echo "  - Regenerate manifest: cd cli && make generate-manifest" >&2
exit $DRIFT_EXIT_CODE
