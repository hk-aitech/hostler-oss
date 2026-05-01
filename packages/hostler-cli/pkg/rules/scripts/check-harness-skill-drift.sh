#!/usr/bin/env bash
# Detect drift between harness_defaults.json and the task-management Skill table.
#
# Parses the set of required=true items per task:* entry in
# harness_defaults.json against the Required Items column of the
# "Harness items per Task type" table in skills/task-management/SKILL.md
# and compares the two. On mismatch, prints a detailed diff to stderr +
# exits 1.
#
# Usage: bash scripts/check-harness-skill-drift.sh
# Make integration: cli/Makefile → check-harness-skill-drift target

set -euo pipefail

# Embedded execution support.
# When bundled into the binary via go:embed and run from a temp file,
# SCRIPT_DIR is /tmp. Resolving via git rev-parse handles both rule-engine
# execution (cmd.Dir=root) and direct developer execution
# (cd plugin root).
PROJECT_ROOT="$(git rev-parse --show-toplevel 2>/dev/null || echo "$PWD")"
cd "$PROJECT_ROOT"

HARNESS_JSON="cli/pkg/db/schemas/harness_defaults.json"
SKILL_MD="skills/task-management/SKILL.md"

if [ ! -f "$HARNESS_JSON" ]; then
    echo "ERROR: harness_defaults.json missing: $HARNESS_JSON" >&2
    exit 2
fi
if [ ! -f "$SKILL_MD" ]; then
    echo "ERROR: skill file missing: $SKILL_MD" >&2
    exit 2
fi

# ── constants ──
DRIFT_EXIT_CODE=1
SETUP_ERROR_EXIT_CODE=2

# ── JSON source: set of required=true item_id values per task:* entry ──
# Output: "feature:build_passed,code_review,criteria_checked,tests_passed"
json_required_items() {
    local task_type="$1"
    jq -r --arg k "task:$task_type" '
        .[$k]
        | map(select(.required == true) | .id)
        | sort
        | join(",")
    ' "$HARNESS_JSON"
}

# ── Skill source: extract backtick items from the Required Items column ──
# Example skill table row:
#   | `feature` | `criteria_checked`, `build_passed`, ... | `lint_passed` |
# Extraction rules:
#   1. The task type is the first backtick token after the second pipe.
#   2. Required Items lives between the third and fourth pipes.
#   3. Collect only backtick-wrapped identifiers (ignore decorations such
#      as superscripts after `lint_passed`).
skill_required_items() {
    local task_type="$1"
    awk -v want="$task_type" '
        /^\| `[a-z]+` \|/ {
            # Extract the task type from the first backtick field.
            match($0, /\|[[:space:]]*`([a-z]+)`/, m)
            if (m[1] != want) next
            # Third column = Required Items.
            n = split($0, fields, "|")
            if (n < 4) next
            required_col = fields[3]
            # Extract backtick identifiers only.
            while (match(required_col, /`[a-z_]+`/)) {
                item = substr(required_col, RSTART+1, RLENGTH-2)
                items[item] = 1
                required_col = substr(required_col, RSTART+RLENGTH)
            }
            # Sort and join with commas.
            n2 = 0
            for (k in items) { sorted[++n2] = k }
            # Simple bubble sort (item count < 10 — this is enough).
            for (i = 1; i <= n2; i++) {
                for (j = i+1; j <= n2; j++) {
                    if (sorted[i] > sorted[j]) { t = sorted[i]; sorted[i] = sorted[j]; sorted[j] = t }
                }
            }
            out = ""
            for (i = 1; i <= n2; i++) { out = out (i>1?",":"") sorted[i] }
            print out
            exit 0
        }
    ' "$SKILL_MD"
}

# ── task types to compare ──
# Extract task:* keys from harness_defaults.json (includes hotfix).
task_types=$(jq -r 'keys[] | select(startswith("task:")) | sub("^task:"; "")' "$HARNESS_JSON")

drift_count=0
echo "=== harness <-> Skill drift check ==="
for t in $task_types; do
    json_items=$(json_required_items "$t")
    skill_items=$(skill_required_items "$t" || true)

    if [ -z "$skill_items" ]; then
        echo "WARN  $t: skill table has no entry (parse failure or missing)"
        drift_count=$((drift_count + 1))
        continue
    fi

    if [ "$json_items" != "$skill_items" ]; then
        echo "FAIL  $t drift:"
        echo "      JSON : $json_items"
        echo "      Skill: $skill_items"
        drift_count=$((drift_count + 1))
    else
        echo "OK    $t: $json_items"
    fi
done

echo "==================================="
if [ "$drift_count" -gt 0 ]; then
    echo "ERROR: drift count $drift_count — update either harness_defaults.json or the Skill table." >&2
    exit "$DRIFT_EXIT_CODE"
fi
echo "OK: no drift"
