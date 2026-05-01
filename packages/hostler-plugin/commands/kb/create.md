---
description: Create a new Knowledge Base card. Use when the user asks to "record a lesson", "add a KB card", or "capture a decision/mistake" — calls `hstl-oss kb create` to write a markdown card to docs/07-knowledge/ with category, subcategory, title, problem, and solution.
argument-hint: "--category <category> --subcategory <subcategory> --title \"<title>\" --problem \"<problem>\" --solution \"<solution>\""
allowed-tools: Bash(hstl-oss:kb:create), Read
---

# /kb-create — Create KB Card

Calls the `hstl-oss kb create` CLI to add a new knowledge card under `docs/07-knowledge/`.
Category, subcategory, title, problem, and solution are required; context, cause, prevention,
and references are optional.

## Required flags

| Flag | Description | Example |
|--------|------|------|
| `--category` | Category (subdirectory under docs/07-knowledge/) | `mistakes`, `decisions`, `operations` |
| `--subcategory` | Subcategory (used in the filename) | `code-review`, `db-migration` |
| `--title` | Card title | `"vendor cache error during go build"` |
| `--problem` | Problem description | `"build fails because vendor/ is stale"` |
| `--solution` | Solution | `"re-run go mod vendor"` |

## Optional flags

| Flag | Description |
|--------|------|
| `--context` | When/where it occurred |
| `--cause` | Root cause analysis |
| `--prevention` | Prevention measures |
| `--refs` | Reference link list (comma-separated) |

## Run

```bash
hstl-oss kb create \
  --category <category> \
  --subcategory <subcategory> \
  --title "<title>" \
  --problem "<problem>" \
  --solution "<solution>"
```

The default output is JSON. For human inspection use `-o text` or `-o console`:

```bash
hstl-oss kb create --category mistakes --subcategory build \
  --title "go build vendor error" \
  --problem "build fails due to vendor cache mismatch" \
  --solution "re-run go mod vendor" \
  -o console
```

## Behavior

1. Verify `--category`, `--subcategory`, `--title`, `--problem`, `--solution` are present (error otherwise)
2. Write the card file to `docs/07-knowledge/<category>/<subcategory>.md`
3. Determine the category prefix (`mistakes` -> `M`, `decisions` -> `A`, etc.)
4. Allocate a card ID and render the markdown template
5. Update counts in `docs/07-knowledge/INDEX.md`

## Completion report format

```
KB card created
   ID:       M-001
   Path:     docs/07-knowledge/mistakes/build.md
   Category: mistakes / build
```

## Error handling

- Missing `--category` / `--subcategory` / `--title` / `--problem` / `--solution` -> immediate error
- Missing `docs/07-knowledge/` directory -> instruct user to run `/kb-init` first
- File already exists at that path -> error without overwriting (preserves the existing card)

## Related

- `/kb-init` — initialize the KB structure (before first use)
- `/kb-list` — list cards
- `/kb-rebuild` — recompute the INDEX
- `hstl-oss kb create --help` — full flag list

## Output

JSON: `{status: ok, data: {file_path, card_id?}}` — returns the new card's file_path. `card_id` continues automatically from the existing prefix+number sequence in the sub-category (e.g. M008, A018).

Errors:
- Missing required flags (--category / --subcategory / --title / --problem / --solution): `error_category=INVALID_INPUT`
- Missing directory (KB not initialized): `error_category=NOT_FOUND` + recovery_hint (`hstl-oss kb init`)

