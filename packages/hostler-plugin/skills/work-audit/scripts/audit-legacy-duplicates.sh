#!/usr/bin/env bash
# audit-legacy-duplicates.sh — Q26 legacy duplicate file detection
#
# Detects schema/config-style files whose basename appears in 2+ places in the
# repo and reports them for manual review. Basenames listed in the allowlist
# are excluded.
#
# Usage:
#   bash skills/work-audit/scripts/audit-legacy-duplicates.sh [PROJECT_ROOT]
#
# Exit code:
#   0 — no duplicates (or all are in the allowlist)
#   1 — duplicates detected (manual review required — WARN level)
#
# References:
#   - skills/work-audit/SKILL.md §Q26
#   - skills/work-audit/references/legacy-duplicate-allowlist.json
#   - §1 (background on preventing recurrence)

set -euo pipefail

readonly SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
readonly ALLOWLIST_PATH="${SCRIPT_DIR}/../references/legacy-duplicate-allowlist.json"

readonly PROJECT_ROOT="${1:-$(git rev-parse --show-toplevel 2>/dev/null || pwd)}"
cd "${PROJECT_ROOT}"

# Detection target glob patterns (schemas/configs/harness/briefing focused)
readonly SCAN_PATTERNS=(
  "**/schemas/*.json"
  "**/*.config.yaml"
  "**/harness*.json"
  "**/harness*.yaml"
  "**/briefing*.yaml"
  "**/briefing*.yml"
  "**/project-config*.yaml"
)

# Read basename list from the allowlist file
read_allowlist_names() {
  if [[ ! -f "${ALLOWLIST_PATH}" ]]; then
    return 0
  fi
  # Works without jq — parse with python3
  python3 - <<PY 2>/dev/null || true
import json, sys
try:
    with open("${ALLOWLIST_PATH}") as f:
        data = json.load(f)
    for name in data.get("allowlist", {}):
        print(name)
except Exception:
    pass
PY
}

read_allowlist_paths() {
  if [[ ! -f "${ALLOWLIST_PATH}" ]]; then
    return 0
  fi
  python3 - <<PY 2>/dev/null || true
import json
try:
    with open("${ALLOWLIST_PATH}") as f:
        data = json.load(f)
    for hint in data.get("path_hints", []):
        print(hint)
except Exception:
    pass
PY
}

declare -A ALLOWLIST_BASENAMES=()
while IFS= read -r name; do
  [[ -n "${name}" ]] && ALLOWLIST_BASENAMES["${name}"]=1
done < <(read_allowlist_names)

declare -a ALLOWLIST_PATH_HINTS=()
while IFS= read -r hint; do
  [[ -n "${hint}" ]] && ALLOWLIST_PATH_HINTS+=("${hint}")
done < <(read_allowlist_paths)

# Collect file list — use git ls-files (respects .gitignore, committed files only)
declare -a CANDIDATE_FILES=()
while IFS= read -r f; do
  CANDIDATE_FILES+=("${f}")
done < <(
  for pattern in "${SCAN_PATTERNS[@]}"; do
    git ls-files "${pattern}" 2>/dev/null || true
  done | sort -u
)

# Filter by path hints
filter_by_hints() {
  local path="$1"
  for hint in "${ALLOWLIST_PATH_HINTS[@]}"; do
    if [[ "${path}" == *"${hint}"* ]]; then
      return 1
    fi
  done
  return 0
}

# Aggregate file list per basename
declare -A BASENAME_MAP=()
for f in "${CANDIDATE_FILES[@]}"; do
  if ! filter_by_hints "${f}"; then
    continue
  fi
  bn="$(basename "${f}")"
  if [[ -n "${ALLOWLIST_BASENAMES[${bn}]:-}" ]]; then
    continue
  fi
  BASENAME_MAP["${bn}"]="${BASENAME_MAP[${bn}]:-}${f}"$'\n'
done

# Report duplicates
found_duplicates=0
for bn in "${!BASENAME_MAP[@]}"; do
  count=$(printf '%s' "${BASENAME_MAP[${bn}]}" | grep -c . || true)
  if [[ "${count}" -ge 2 ]]; then
    if [[ "${found_duplicates}" -eq 0 ]]; then
      echo "[Q26] Legacy duplicate file detection"
      echo "========================================"
    fi
    found_duplicates=1
    echo ""
    echo "⚠️  basename: ${bn}  (${count} locations)"
    printf '%s' "${BASENAME_MAP[${bn}]}" | sed 's/^/    - /'
  fi
done

if [[ "${found_duplicates}" -eq 0 ]]; then
  echo "[Q26] No legacy duplicate files (after applying allowlist)"
  exit 0
fi

echo ""
echo "Recommendation: if the files above are intentional shadow copies,"
echo "                add them to ${ALLOWLIST_PATH}."
echo "                Otherwise, move the legacy copy to archive/ (skills/archive/SKILL.md)."
exit 1
