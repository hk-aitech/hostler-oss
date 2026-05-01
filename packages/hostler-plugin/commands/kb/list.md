---
description: List Knowledge Base cards. Use when the user asks to "search the KB", "find a lesson", or "list KB cards" — calls `hstl-oss kb list` to query docs/07-knowledge/ with category/keyword/date filters.
argument-hint: "[--category <category>] [--keyword <keyword>] [--since <YYYY-MM-DD>]"
allowed-tools: Bash(hstl-oss:kb:list), Read
---

# /kb-list — List KB Cards

Calls the `hstl-oss kb list` CLI to query knowledge cards stored under `docs/07-knowledge/`.
Without flags it returns all cards.

## Flags

| Flag | Description | Example |
|--------|------|------|
| `--category` | Category filter | `--category mistakes` |
| `--keyword` | Title/content keyword filter | `--keyword "go build"` |
| `--since` | Date filter (created/updated since) | `--since 2026-01-01` |

All flags combine with AND.

## Run

```bash
# All cards
hstl-oss kb list

# Category filter
hstl-oss kb list --category mistakes

# Keyword search
hstl-oss kb list --keyword "vendor"

# Combined filter
hstl-oss kb list --category mistakes --keyword "build" --since 2026-01-01

# JSON output
hstl-oss kb list --category decisions
```

## Behavior

1. Scan all markdown files under `docs/07-knowledge/`
2. With `--category`, restrict to that subdirectory
3. With `--keyword`, do substring search across title/problem/solution
4. With `--since`, return only cards whose frontmatter `created` is on or after that date
5. Output sorted by ID ascending

## Text output format

```
KB cards (3)

M-001  mistakes/build.md         vendor cache error during go build     2026-03-10
M-002  mistakes/db-migration.md  ran migration without rollback         2026-03-15
A-001  decisions/arch.md         decision to introduce internal package 2026-04-01
```

## JSON output format

```json
{
  "cards": [
    {
      "id": "M-001",
      "title": "vendor cache error during go build",
      "category": "mistakes",
      "subcategory": "build",
      "created": "2026-03-10"
    }
  ],
  "total": 1
}
```

## Category list

| Category | Prefix | Description |
|---------|-------|------|
| `mistakes` | M | Mistakes / bugs / gotchas |
| `operations` | O | Operations / deployment / incidents |
| `architecture` | A | Architecture / design patterns |
| `api` | AP | API usage / gotchas |
| `domain` | D | Domain knowledge |
| `migration` | MG | Migration procedures |

## Related

- `/kb-create` — create a card
- `/kb-rebuild` — recompute the INDEX
- `hstl-oss kb list --help` — full flag list
