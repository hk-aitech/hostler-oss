#!/usr/bin/env bash
# 3-way version sync verification across plugin.json / marketplace.json / CLAUDE.md.
#
# The version SSOT for `hstl --manifest` is .claude-plugin/plugin.json.
# This script verifies that all three of the following match:
#   1. .claude-plugin/plugin.json → jq .version
#   2. .claude-plugin/marketplace.json → jq '.plugins[] | select(.name=="hstl-oss") | .version'
#   3. CLAUDE.md → grep "**Version**: X.Y.Z"
#
# On mismatch: exit 1 + per-file actuals + how to align.
#
# Intentional drift allowed: skip when CLAUDE.md contains the comment
# `<!-- version-sync: allow-drift -->`.
#
# Usage: bash scripts/check-version-sync.sh
# pre-commit hook: auto-invoked on changes to any of the 4 files
# integration-test hook: scripts/integration-test.sh check 8c

set -euo pipefail

# Embedded execution support (resolved dynamically via git rev-parse).
PROJECT_ROOT="$(git rev-parse --show-toplevel 2>/dev/null || echo "$PWD")"
cd "$PROJECT_ROOT"

PLUGIN_JSON=".claude-plugin/plugin.json"
MARKETPLACE_JSON=".claude-plugin/marketplace.json"
CLAUDE_MD="CLAUDE.md"

DRIFT_EXIT_CODE=1
SETUP_ERROR_EXIT_CODE=2
ALLOW_MARKER="version-sync: allow-drift"
PLUGIN_NAME="hstl-oss"

# Skip-marker check.
if grep -q "$ALLOW_MARKER" "$CLAUDE_MD" 2>/dev/null; then
    echo "Skipping check — version-sync: allow-drift marker found" >&2
    exit 0
fi

# File existence check.
for f in "$PLUGIN_JSON" "$MARKETPLACE_JSON" "$CLAUDE_MD"; do
    if [ ! -f "$f" ]; then
        echo "ERROR: file missing: $f" >&2
        exit $SETUP_ERROR_EXIT_CODE
    fi
done

# 1. plugin.json (SSOT)
PLUGIN_VER=$(jq -r '.version // empty' "$PLUGIN_JSON")
if [ -z "$PLUGIN_VER" ]; then
    echo "ERROR: $PLUGIN_JSON has no .version field" >&2
    exit $SETUP_ERROR_EXIT_CODE
fi

# 2. marketplace.json (plugin entry)
MARKET_VER=$(jq -r --arg n "$PLUGIN_NAME" '.plugins[] | select(.name==$n) | .version // empty' "$MARKETPLACE_JSON")
if [ -z "$MARKET_VER" ]; then
    echo "ERROR: $MARKETPLACE_JSON has no $PLUGIN_NAME plugin entry or version" >&2
    exit $SETUP_ERROR_EXIT_CODE
fi

# 3. CLAUDE.md "**Version**: X.Y.Z" line
CLAUDE_VER=$(grep -oE '\*\*Version\*\*:\s*[0-9]+\.[0-9]+\.[0-9]+' "$CLAUDE_MD" | grep -oE '[0-9]+\.[0-9]+\.[0-9]+' | head -1)
if [ -z "$CLAUDE_VER" ]; then
    echo "ERROR: $CLAUDE_MD has no '**Version**: X.Y.Z' line" >&2
    exit $SETUP_ERROR_EXIT_CODE
fi

# 3-way comparison.
if [ "$PLUGIN_VER" = "$MARKET_VER" ] && [ "$MARKET_VER" = "$CLAUDE_VER" ]; then
    echo "OK: version sync at $PLUGIN_VER"
    exit 0
fi

# Mismatch — emit details.
cat >&2 <<EOF
ERROR: version sync drift detected.

  plugin.json      : $PLUGIN_VER  (SSOT)
  marketplace.json : $MARKET_VER
  CLAUDE.md        : $CLAUDE_VER

How to align (the SSOT is plugin.json):

  1) Edit plugin.json to the desired version.
  2) Update the $PLUGIN_NAME entry's version in marketplace.json to match.
  3) Update the '**Version**: X.Y.Z' line in CLAUDE.md.
  4) Re-run: bash scripts/check-version-sync.sh

To intentionally allow drift, add this comment somewhere in CLAUDE.md:
  <!-- version-sync: allow-drift -->
EOF
exit $DRIFT_EXIT_CODE
