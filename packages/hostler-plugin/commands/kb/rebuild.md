---
description: Rebuild the KB INDEX. Use when the user asks to "rebuild the KB index", "fix index counts", or after manually adding/removing cards — scans docs/07-knowledge/ and updates INDEX.md counts.
argument-hint: ""
allowed-tools: Bash(hstl-oss:kb:rebuild), Read
---

# /kb-rebuild — Recompute KB Index

Calls the `hstl-oss kb rebuild` CLI to recount the cards listed in
`docs/07-knowledge/INDEX.md`. Run when cards have been added/removed manually,
or when the INDEX is out of sync after `hstl-oss kb create`.

## Run

```bash
hstl-oss kb rebuild

# JSON output
hstl-oss kb rebuild
```

## Behavior

1. Scan all markdown files under `docs/07-knowledge/`
2. Recount cards per category
3. Update the category table in `INDEX.md` (count + last update)
4. Print the rollup (total cards + files scanned)

## Text output format

```
KB index rebuilt — 12 cards total, 14 files scanned
```

## JSON output format

```json
{
  "total_cards": 12,
  "files_scanned": 14,
  "by_category": {
    "mistakes": 5,
    "decisions": 3,
    "operations": 2,
    "architecture": 1,
    "api": 1,
    "domain": 0,
    "migration": 0
  }
}
```

## When to run

| Situation | Recommended? |
|------|---------|
| INDEX appears out of sync after `hstl-oss kb create` | Yes |
| Card file edited manually | Yes |
| Card file deleted/moved manually | Required |
| Right after a successful `hstl-oss kb create` | Unnecessary (auto-updated) |
| Periodic check at the end of a Sprint | Optional |

## Error handling

- Missing `docs/07-knowledge/` -> instruct user to run `/kb-init` first
- Files without read permission -> skip and warn
- No write permission on INDEX.md -> print error and abort

## Related

- `/kb-init` — initialize the KB structure (before first use)
- `/kb-create` — create a card
- `/kb-list` — list cards
- `hstl-oss kb rebuild --help` — full flag list
