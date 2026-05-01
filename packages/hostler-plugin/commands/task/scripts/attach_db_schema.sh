#!/usr/bin/env bash
#
# attach_db_schema.sh — task:start pre-check helper script.
#
# Purpose: when the Task body contains DB-schema change keywords
# (ALTER/CREATE/DROP TABLE, ADD/DROP COLUMN), extract the affected table names
# and print the LocalDev psql `\d` output as a `### DB Schema (LocalDev)` section
# to stdout. The task:start briefing appends this output below "Related files".
#
# Background: SQL written against an assumed schema rather than the real one
# previously caused a LocalDev 500 + a wasted image rebuild. Pre-checking the
# schema at task start prevents the same class of mistake from recurring.
#
# Usage:
#   attach_db_schema.sh <task-file-path>       # normal run
#   attach_db_schema.sh --self-test            # built-in unit tests
#
# Opt-in: only runs when HOSTLER_OSS_TASK_START_DB_SCHEMA_ATTACH=true (default off).
#   DSN comes from HOSTLER_OSS_LOCALDEV_DSN env var, or from `localdev.psql_dsn` in
#   `.hostler/project-config.yaml` (this script reads env vars only — YAML
#   parsing is the calling Command document's responsibility via export).
#
# Security rules:
#   - Read-only SQL only. Whitelist of `\d` / `\dt` / `\dv`.
#   - Arbitrary SQL is blocked at the code level (see `run_psql_meta`).
#   - Graceful degradation: psql missing / DSN unset / connection failure ->
#     single stderr WARN + normal exit 0 (briefing flow preserved).
#
# Constraints:
#   - PostgreSQL psql only (current scope — see "Future Sprints").
#   - Table names are restricted to ASCII alphanumerics + underscore.
#
# shellcheck disable=SC2155


set -uo pipefail

# -----------------------------------------------------------------------------
# Constants
# -----------------------------------------------------------------------------

readonly OPTIN_ENV="HOSTLER_OSS_TASK_START_DB_SCHEMA_ATTACH"
readonly DSN_ENV="HOSTLER_OSS_LOCALDEV_DSN"
# Safe table-name regex (ASCII alphanumerics/underscore, up to 63 chars — PostgreSQL identifier limit).
readonly SAFE_TABLE_RE='^[A-Za-z_][A-Za-z0-9_]{0,62}$'
# Allowed psql meta-command whitelist.
readonly PSQL_META_WHITELIST='\\d|\\dt|\\dv'

# -----------------------------------------------------------------------------
# Logging (stderr)
# -----------------------------------------------------------------------------

log_warn() {
    printf '[attach_db_schema] WARN: %s\n' "$*" >&2
}

# -----------------------------------------------------------------------------
# Extract a set of table names from the Task body
# -----------------------------------------------------------------------------

# extract_table_names TASK_FILE
#   Detect DB-schema change keywords in the body and print deduplicated table
#   names one per line. Empty output + exit 0 if the body cannot be read (warn only).
extract_table_names() {
    local file="$1"
    if [[ ! -f "$file" ]]; then
        log_warn "task file missing: $file"
        return 0
    fi

    # Pattern 1: (ALTER|CREATE|DROP) TABLE <name>
    local tables1
    tables1=$(grep -oiE '(ALTER|CREATE|DROP)[[:space:]]+TABLE[[:space:]]+[A-Za-z_][A-Za-z0-9_]*' "$file" \
        | awk '{print tolower($NF)}' || true)

    # Pattern 2: (ADD|DROP) COLUMN — the table name comes from the preceding
    # ALTER TABLE, so here we just confirm the keyword exists. Used as a hint
    # when no table is extracted.
    local has_col
    has_col=$(grep -oiE '(ADD|DROP)[[:space:]]+COLUMN' "$file" | head -1 || true)

    local merged
    merged=$(printf '%s\n' "$tables1" | grep -v '^$' | sort -u || true)

    if [[ -z "$merged" && -n "$has_col" ]]; then
        # Column keyword present but table extraction failed — guidance only.
        log_warn "ADD/DROP COLUMN detected but could not extract a target table (check body format)"
        return 0
    fi

    printf '%s\n' "$merged"
}

# -----------------------------------------------------------------------------
# Run psql (read-only whitelist)
# -----------------------------------------------------------------------------

# run_psql_meta META_CMD TARGET
#   META_CMD must be one of the whitelist (\d / \dt / \dv). TARGET must match SAFE_TABLE_RE.
#   On failure or missing tooling: stderr WARN + exit 0 (graceful).
run_psql_meta() {
    local meta="$1"
    local target="$2"

    if ! printf '%s' "$meta" | grep -qE "^($PSQL_META_WHITELIST)\$"; then
        log_warn "blocked psql meta-command outside whitelist: $meta"
        return 0
    fi
    if ! printf '%s' "$target" | grep -qE "$SAFE_TABLE_RE"; then
        log_warn "blocked unsafe table name: $target"
        return 0
    fi
    if ! command -v psql >/dev/null 2>&1; then
        log_warn "psql not installed — skip"
        return 0
    fi
    if [[ -z "${!DSN_ENV:-}" ]]; then
        log_warn "$DSN_ENV not set — skip"
        return 0
    fi

    # -X: no .psqlrc, -A: unaligned, -t: tuples-only removes header — keep \d output.
    # Connection failures yield non-zero exit codes; absorb gracefully.
    local out
    if ! out=$(psql "${!DSN_ENV}" -X -c "$meta $target" 2>&1); then
        log_warn "psql connection/query failed (target=$target): ${out:-unknown}"
        return 0
    fi
    printf '%s\n' "$out"
}

# -----------------------------------------------------------------------------
# Main: render the schema section from the Task file
# -----------------------------------------------------------------------------

render_schema_section() {
    local task_file="$1"
    if [[ "${!OPTIN_ENV:-}" != "true" ]]; then
        # Opt-in not set — output nothing (caller's briefing flow is unchanged).
        return 0
    fi

    local tables
    tables=$(extract_table_names "$task_file")
    if [[ -z "$tables" ]]; then
        return 0
    fi

    printf '\n### DB Schema (LocalDev)\n\n'
    printf 'DB-schema change keywords detected in the Task body. LocalDev psql `\\d` results:\n\n'
    local t
    while IFS= read -r t; do
        [[ -z "$t" ]] && continue
        printf '#### \\d %s\n\n' "$t"
        printf '```\n'
        run_psql_meta '\d' "$t"
        printf '```\n\n'
    done <<< "$tables"
}

# -----------------------------------------------------------------------------
# Built-in self-test
# -----------------------------------------------------------------------------

self_test() {
    # Keep tmpdir global so the EXIT trap is visible after the function returns.
    _self_test_tmpdir=$(mktemp -d)
    local tmpdir="$_self_test_tmpdir"
    trap 'rm -rf "$_self_test_tmpdir"' EXIT

    local pass=0 fail=0
    assert_eq() {
        local name="$1" expected="$2" actual="$3"
        if [[ "$expected" == "$actual" ]]; then
            printf '  PASS: %s\n' "$name"
            pass=$((pass + 1))
        else
            printf '  FAIL: %s\n    expected: %s\n    actual:   %s\n' "$name" "$expected" "$actual"
            fail=$((fail + 1))
        fi
    }

    # Pattern 1: ALTER TABLE
    local f1="$tmpdir/t1.md"
    printf 'ALTER TABLE stock_master ADD COLUMN is_active BOOLEAN\n' > "$f1"
    assert_eq "ALTER TABLE extraction" "stock_master" "$(extract_table_names "$f1")"

    # Pattern 2: CREATE TABLE + DROP TABLE multiple
    local f2="$tmpdir/t2.md"
    printf 'CREATE TABLE orders (id int);\nDROP TABLE legacy_cache;\n' > "$f2"
    # After sort -u, alphabetical — legacy_cache before orders.
    assert_eq "multiple tables (CREATE + DROP)" $'legacy_cache\norders' "$(extract_table_names "$f2")"

    # Pattern 3: dedup
    local f3="$tmpdir/t3.md"
    printf 'ALTER TABLE foo ADD COLUMN x int;\nALTER TABLE foo ADD COLUMN y text;\n' > "$f3"
    assert_eq "dedup" "foo" "$(extract_table_names "$f3")"

    # Pattern 4: no keyword
    local f4="$tmpdir/t4.md"
    printf 'SELECT * FROM foo WHERE bar=1;\n' > "$f4"
    assert_eq "no keyword — empty output" "" "$(extract_table_names "$f4")"

    # Pattern 5: missing file — graceful
    assert_eq "missing file — empty output" "" "$(extract_table_names "$tmpdir/does-not-exist.md")"

    # Pattern 6: ADD COLUMN only — warn then empty
    local f6="$tmpdir/t6.md"
    printf 'ADD COLUMN foo int\n' > "$f6"
    assert_eq "ADD COLUMN only — empty output" "" "$(extract_table_names "$f6" 2>/dev/null)"

    # render_schema_section: opt-in not set — empty output
    unset "${OPTIN_ENV:-}" 2>/dev/null || true
    assert_eq "opt-in off — empty output" "" "$(HOSTLER_OSS_TASK_START_DB_SCHEMA_ATTACH="" render_schema_section "$f1")"

    # run_psql_meta: command outside whitelist blocked
    assert_eq "non-whitelist blocked" "" "$(run_psql_meta 'SELECT *' foo 2>/dev/null)"
    # run_psql_meta: unsafe table name blocked
    assert_eq "unsafe table blocked" "" "$(run_psql_meta '\d' 'foo; DROP TABLE' 2>/dev/null)"
    # run_psql_meta: DSN not set — graceful empty
    unset "${DSN_ENV:-}" 2>/dev/null || true
    assert_eq "DSN not set — skip" "" "$(HOSTLER_OSS_LOCALDEV_DSN="" run_psql_meta '\d' foo 2>/dev/null)"

    printf '\nself-test: %d pass / %d fail\n' "$pass" "$fail"
    [[ "$fail" -eq 0 ]]
}

# -----------------------------------------------------------------------------
# Entry point
# -----------------------------------------------------------------------------

main() {
    case "${1:-}" in
        --self-test)
            self_test
            ;;
        --help|-h|"")
            cat <<USAGE
Usage: $(basename "$0") <task-file-path>
       $(basename "$0") --self-test

Opt-in: schema section is rendered only when $OPTIN_ENV=true.
DSN:    $DSN_ENV environment variable.
USAGE
            ;;
        *)
            render_schema_section "$1"
            ;;
    esac
}

main "$@"
