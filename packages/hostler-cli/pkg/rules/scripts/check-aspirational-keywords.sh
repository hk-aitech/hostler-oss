#!/usr/bin/env bash
# check-aspirational-keywords.sh
# Warns when staged Task files contain aspirational (vague) keywords in
# the ## Done Criteria section. Recommends rewording into measurable terms.
#
# Background: the done-criteria section must contain "minimum measurable
# values", but future-tense vague phrases such as "soon", "later",
# "eventually" cannot be measured.
#
# Operations: callable from a pre-commit rule. By default prints warnings
# and exits 0. Strict mode is enabled via HOSTLER_ASPIRATIONAL_CHECK=strict.
set -uo pipefail

MODE="${HOSTLER_ASPIRATIONAL_CHECK:-warn}"

# staged Task files only — T*.md
staged=$(git diff --cached --name-only --diff-filter=AM 2>/dev/null | grep -E 'works/(tasks|sprints)/.*T[0-9]+.*\.md$' || true)
if [ -z "$staged" ]; then
    exit 0
fi

# aspirational keyword list
PATTERNS='\b(soon|later|eventually)\b'

found=0
while IFS= read -r f; do
    [ -z "$f" ] && continue
    # Extract the ## Done Criteria section and check for the patterns.
    matches=$(awk '/^## Done Criteria/,/^## /' "$f" 2>/dev/null | grep -cE "$PATTERNS" || true)
    if [ "${matches:-0}" -gt 0 ]; then
        echo "[aspirational] $f — ${matches} vague phrase(s) in ## Done Criteria"
        awk '/^## Done Criteria/,/^## /' "$f" 2>/dev/null | grep -nE "$PATTERNS" | head -3 | sed 's/^/    /'
        found=$((found + 1))
    fi
done <<< "$staged"

if [ $found -eq 0 ]; then
    exit 0
fi

echo ""
echo "[aspirational] vague keywords found in ${found} file(s) total"
echo "  Rationale: done criteria must be a minimum measurable value. Use concrete values instead of 'soon/later/eventually'."
echo "  To strict-block: HOSTLER_ASPIRATIONAL_CHECK=strict git commit"

if [ "$MODE" = "strict" ]; then
    exit 1
fi
exit 0
