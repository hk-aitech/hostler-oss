#!/usr/bin/env bash
# Detect drift across the three Sprint Phase definitions.
#
# SSOT: cli/pkg/db/schemas/harness_defaults.json sprint:default[].name
# Subjects:
#   1. Go loader (cli/pkg/db/db.go LoadSprintPhaseNames) — auto-consistent
#      because it parses the embedded JSON
#   2. SPRINT_PHASE_TABLE block in commands/sprint/complete.md — kept
#      manually, drift risk
#
# This script compares the 10 sprint:default[].name entries from the JSON
# against the 10 "Phase N:" entries from the complete.md block. On
# mismatch, prints a detailed diff to stderr and exits 1.
#
# Usage: bash scripts/check-sprint-phase-drift.sh

set -euo pipefail

# Embedded execution support (resolved dynamically via git rev-parse).
PROJECT_ROOT="$(git rev-parse --show-toplevel 2>/dev/null || echo "$PWD")"
cd "$PROJECT_ROOT"

HARNESS_JSON="cli/pkg/db/schemas/harness_defaults.json"
COMMAND_MD="commands/sprint/complete.md"

if [ ! -f "$HARNESS_JSON" ]; then
    echo "ERROR: harness_defaults.json missing: $HARNESS_JSON" >&2
    exit 2
fi
if [ ! -f "$COMMAND_MD" ]; then
    echo "ERROR: command md missing: $COMMAND_MD" >&2
    exit 2
fi

DRIFT_EXIT_CODE=1

# ── JSON SSOT: sprint:default[].name (order preserved) ──
json_phases=$(jq -r '."sprint:default"[].name' "$HARNESS_JSON")

# ── Command md: extract "Phase N: ..." entries inside the SPRINT_PHASE_TABLE block ──
# Parse `| N | content | condition |` rows between the markers and pull
# only the second column.
md_phases=$(
    awk '
        /SPRINT_PHASE_TABLE:BEGIN/ { inblock = 1; next }
        /SPRINT_PHASE_TABLE:END/   { inblock = 0; next }
        inblock && /^\| *[0-9]+ *\|/ {
            # Extract the second column: | N | content | condition |
            n = split($0, fields, "|")
            if (n >= 4) {
                gsub(/^ +| +$/, "", fields[3])
                print fields[3]
            }
        }
    ' "$COMMAND_MD"
)

echo "=== Sprint Phase 3-way drift check ==="
echo "SSOT: $HARNESS_JSON (sprint:default[].name)"
echo ""

json_count=$(printf "%s\n" "$json_phases" | wc -l | tr -d ' ')
md_count=$(printf "%s\n" "$md_phases" | wc -l | tr -d ' ')

if [ "$json_count" != "$md_count" ]; then
    echo "ERROR: phase count mismatch: JSON=$json_count, Command md=$md_count" >&2
    echo "" >&2
    echo "-- JSON --" >&2
    printf "%s\n" "$json_phases" >&2
    echo "" >&2
    echo "-- Command md --" >&2
    printf "%s\n" "$md_phases" >&2
    exit "$DRIFT_EXIT_CODE"
fi

drift_count=0
i=1
while IFS= read -r json_line && IFS= read -r md_line <&3; do
    if [ "$json_line" != "$md_line" ]; then
        echo "FAIL  Phase $i drift:" >&2
        echo "      JSON   : $json_line" >&2
        echo "      Command: $md_line" >&2
        drift_count=$((drift_count + 1))
    else
        echo "OK    Phase $i: $json_line"
    fi
    i=$((i + 1))
done < <(printf "%s\n" "$json_phases") 3< <(printf "%s\n" "$md_phases")

echo "==================================="
if [ "$drift_count" -gt 0 ]; then
    echo "ERROR: drift count $drift_count — update $COMMAND_MD to match the SSOT ($HARNESS_JSON)." >&2
    exit "$DRIFT_EXIT_CODE"
fi
echo "OK: no Sprint Phase drift (JSON <-> Command md)"
