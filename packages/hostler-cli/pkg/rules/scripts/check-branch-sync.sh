#!/usr/bin/env bash
# main/dev branch divergence guard script.
#
# Purpose: monitors how far main lags behind / leads dev so that a
# `/hstl:hotfix:start` flow remains viable under the "branch from main"
# assumption. Excessive divergence is surfaced (warn) or blocked.
#
# Logic:
#   1. If HSTL_BRANCH_SYNC=off, exit 0 immediately (opt-out).
#   2. Both origin/main and origin/dev must exist; otherwise skip.
#   3. Compute gap = `git rev-list --count origin/main..origin/dev`.
#   4. If gap > threshold (default 30), warn on stderr and exit 1.
#   5. On pass, print "✓ main/dev gap={N}" on stdout and exit 0.
#
# Env vars:
#   HSTL_BRANCH_SYNC_THRESHOLD  — divergence allowance (default 30)
#   HSTL_BRANCH_SYNC_WARN_RATIO — WARN ratio % (default 80 → threshold*0.8)  (T430 Sprint-36)
#   HSTL_BRANCH_SYNC            — off/0/false disables the check
#   HSTL_BRANCH_SYNC_STRICT     — 1 disables the sprint-*/hotfix- branch exemption
#
# Configuration sources (T430 precedence):
#   env vars > .hostler/project-config.yaml:precommit.branch_sync > defaults.
#   The hstl CLI's rules-exec wrapper injects config values as env vars.
#
# Output:
#   default: human-readable text
#   `-o json`: { "gap": N, "threshold": N, "ok": bool }
#
# Usage:
#   bash scripts/check-branch-sync.sh
#   bash scripts/check-branch-sync.sh -o json
#   HSTL_BRANCH_SYNC_THRESHOLD=50 bash scripts/check-branch-sync.sh

set -euo pipefail

# ── Constants ────────────────────────────────────────────────────────
readonly DEFAULT_THRESHOLD=30
# T827 (Sprint-97) — default WARN ratio. 30 × 80% = 24.
# When gap is in [24, 30] → stderr WARN (exit 0); gap > 30 → BLOCK
# (exit 1). Prevents recurrence of KB O002 — early warning during sprint
# planning.
# T430 (Sprint-36): overridable via the HSTL_BRANCH_SYNC_WARN_RATIO env var.
readonly DEFAULT_WARN_RATIO=80
readonly DRIFT_EXIT_CODE=1
readonly SETUP_SKIP_EXIT_CODE=0  # silently skip when git or branches are missing

# ── opt-out ──────────────────────────────────────────────────────────
sync_enabled="${HSTL_BRANCH_SYNC:-on}"
case "$sync_enabled" in
    off|0|false|no)
        # opt-out — silent pass.
        exit 0
        ;;
esac

# ── argument parsing ─────────────────────────────────────────────────
output_format="text"
while [ $# -gt 0 ]; do
    case "$1" in
        -o|--output)
            output_format="$2"
            shift 2
            ;;
        -o=*|--output=*)
            output_format="${1#*=}"
            shift
            ;;
        --json)
            output_format="json"
            shift
            ;;
        *)
            echo "unknown argument: $1" >&2
            exit 2
            ;;
    esac
done

# ── threshold resolution ─────────────────────────────────────────────
threshold="${HSTL_BRANCH_SYNC_THRESHOLD:-$DEFAULT_THRESHOLD}"
if ! [[ "$threshold" =~ ^[0-9]+$ ]] || [ "$threshold" -lt 1 ]; then
    echo "HSTL_BRANCH_SYNC_THRESHOLD must be a positive integer (got: '$threshold') — falling back to default $DEFAULT_THRESHOLD" >&2
    threshold=$DEFAULT_THRESHOLD
fi

# T430 (Sprint-36): WARN ratio env var support. Fall back to default if
# outside 1~100.
warn_ratio="${HSTL_BRANCH_SYNC_WARN_RATIO:-$DEFAULT_WARN_RATIO}"
if ! [[ "$warn_ratio" =~ ^[0-9]+$ ]] || [ "$warn_ratio" -lt 1 ] || [ "$warn_ratio" -gt 100 ]; then
    echo "HSTL_BRANCH_SYNC_WARN_RATIO must be an integer in 1~100 (got: '$warn_ratio') — falling back to default $DEFAULT_WARN_RATIO" >&2
    warn_ratio=$DEFAULT_WARN_RATIO
fi

# ── prerequisite checks ──────────────────────────────────────────────
if ! command -v git > /dev/null; then
    echo "[check-branch-sync] git not installed — skip" >&2
    exit $SETUP_SKIP_EXIT_CODE
fi

# Ensure both branches exist (prefer local refs, then origin).
# Skip silently if either ref is missing (consider personal forks /
# pre-origin environments).
resolve_ref() {
    local name="$1"
    if git show-ref --verify --quiet "refs/heads/$name"; then
        echo "$name"
        return 0
    fi
    if git show-ref --verify --quiet "refs/remotes/origin/$name"; then
        echo "origin/$name"
        return 0
    fi
    return 1
}

if ! main_ref="$(resolve_ref main)" || ! dev_ref="$(resolve_ref dev)"; then
    # Minimum requirements not met — skip without warning (CI may check
    # out a single branch).
    if [ "$output_format" = "json" ]; then
        echo '{"gap": null, "threshold": '"$threshold"', "ok": true, "skipped": "branches_unavailable"}'
    fi
    exit 0
fi

# ── compute gap ──────────────────────────────────────────────────────
# If rev-list fails (the two refs are unrelated), skip.
if ! gap="$(git rev-list --count "${main_ref}..${dev_ref}" 2>/dev/null)"; then
    if [ "$output_format" = "json" ]; then
        echo '{"gap": null, "threshold": '"$threshold"', "ok": true, "skipped": "rev_list_failed"}'
    fi
    exit 0
fi

# ── result ───────────────────────────────────────────────────────────
# T827 (Sprint-97): added WARN threshold. When gap >= warn_threshold,
# warn on stderr (exit 0).
# T430 (Sprint-36): warn_ratio overridable via env / config.
warn_threshold=$(( threshold * warn_ratio / 100 ))
ok=true
level="pass"
if [ "$gap" -gt "$threshold" ]; then
    ok=false
    level="block"
elif [ "$gap" -ge "$warn_threshold" ]; then
    level="warn"
fi

# ── T869 (Sprint-102): release-bound branches lift the BLOCK ─────────
# When the current branch is sprint-*/hotfix-*, it is in a
# pre-merge-to-main/dev state. Divergence beyond threshold here is
# "natural accumulation", so demote BLOCK → WARN. Use
# HSTL_BRANCH_SYNC_STRICT=1 to restore strict mode.
# Root-cause fix for the recurring branch-sync regression observed during
# late-stage sprints.
current_branch="$(git branch --show-current 2>/dev/null || echo '')"
strict_mode="${HSTL_BRANCH_SYNC_STRICT:-0}"
case "$current_branch" in
    sprint-*|hotfix-*|hotfix/*)
        if [ "$strict_mode" != "1" ] && [ "$level" = "block" ]; then
            level="warn"
            ok=true
        fi
        ;;
esac

if [ "$output_format" = "json" ]; then
    printf '{"gap": %s, "threshold": %s, "warn_threshold": %s, "level": "%s", "ok": %s, "main_ref": "%s", "dev_ref": "%s", "current_branch": "%s"}\n' \
        "$gap" "$threshold" "$warn_threshold" "$level" "$ok" "$main_ref" "$dev_ref" "$current_branch"
fi

if [ "$ok" = "false" ]; then
    if [ "$output_format" != "json" ]; then
        cat >&2 <<EOF
⚠️  main/dev divergence: ${gap} commits (threshold=${threshold}).
    ${main_ref}..${dev_ref} gap exceeds the threshold.
    Recommended: open a dev → main merge PR to bring main up to date.
    Temporary disable: HSTL_BRANCH_SYNC=off bash scripts/check-branch-sync.sh
EOF
    fi
    exit $DRIFT_EXIT_CODE
fi

# T827 — WARN advisory (exit 0).
if [ "$level" = "warn" ] && [ "$output_format" != "json" ]; then
    case "$current_branch" in
        sprint-*|hotfix-*|hotfix/*)
            # T869 — release-bound branches accumulate divergence by
            # design.
            cat >&2 <<EOF
⚠️  main/dev gap ${gap} — sprint/hotfix branch is pre-merge, so this is allowed.
    The warning will clear on dev/main promotion. Strict mode: HSTL_BRANCH_SYNC_STRICT=1
EOF
            ;;
        *)
            cat >&2 <<EOF
⚠️  main/dev gap WARN: ${gap} commits (warn=${warn_threshold}, block=${threshold}).
    KB O002 — consider merging dev → main before sprint planning (block imminent).
EOF
            ;;
    esac
fi

if [ "$output_format" != "json" ] && [ "$level" = "pass" ]; then
    echo "✓ main/dev gap=${gap} (threshold=${threshold}, warn=${warn_threshold})"
fi
exit 0
