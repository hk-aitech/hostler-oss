#!/usr/bin/env bash
# check-agents-drift.sh
# Verify that agent filenames in the agents/ directory are consistent with
# the agent references in skill bodies. Exits 1 when drift is detected.
#
# Background: combines three checks — agents hardcoding audit + Skill
# manifest auto-sync + dynamic agents lookup.
#
# Subjects:
#   packages/hostler-plugin/agents/*.md     → SSOT (filenames)
#   packages/hostler-plugin/skills/*/SKILL.md → agents referenced in bodies
#
# Operations: can be invoked via the pre-commit rule `precommit.agents.drift`.
set -uo pipefail

AGENTS_DIR="${AGENTS_DIR:-packages/hostler-plugin/agents}"

if [ ! -d "$AGENTS_DIR" ]; then
    echo "[check-agents-drift] $AGENTS_DIR absent — skip"
    exit 0
fi

mapfile -t ssot_agents < <(
    find "$AGENTS_DIR" -maxdepth 1 -name "*.md" -type f \
        | xargs -I{} basename {} .md \
        | sort -u
)

if [ ${#ssot_agents[@]} -eq 0 ]; then
    echo "[check-agents-drift] agents SSOT is empty"
    exit 0
fi

echo "[check-agents-drift] SSOT agents (${#ssot_agents[@]}): ${ssot_agents[*]}"

drift=0
while IFS= read -r name; do
    [ -z "$name" ] && continue
    matched=0
    for s in "${ssot_agents[@]}"; do
        if [ "$s" = "$name" ]; then matched=1; break; fi
    done
    if [ $matched -eq 0 ]; then
        echo "[DRIFT] agent reference not in SSOT: $name"
        drift=$((drift + 1))
    fi
done < <(grep -rhE 'subagent_type[= "]+"?[a-z-]+"' packages/hostler-plugin/skills/ 2>/dev/null \
    | grep -oE '"[a-z-]+"' \
    | tr -d '"' \
    | sort -u)

if [ $drift -gt 0 ]; then
    echo "[check-agents-drift] drift count ${drift}"
    exit 1
fi

echo "[check-agents-drift] OK"
exit 0
