---
description: |
  Initializes the Knowledge Base directory structure (docs/07-knowledge/) for a new project.
  Creates 6 category subdirectories (mistakes, operations, architecture, api, domain, migration)
  and an INDEX.md template. Use on fresh project setup before the first /learned call. Do NOT use
  if docs/07-knowledge/ already exists (command will skip and report existing state).
allowed-tools: Bash(mkdir:-p), Bash(hstl-oss:*), Read, Write, Glob
argument-hint: ""
---

# /kb-init — Initialize Knowledge Base Structure

Auto-creates the `docs/07-knowledge/` KB directory in a new project, so the first
`/learned` call does not fail because `hstl-oss kb create` cannot find the directory.

## Behavior

1. **Existence check**: verify whether `docs/07-knowledge/` already exists
2. **If present**: report current structure/card counts and skip (no destruction)
3. **If absent**: create 6 category subdirectories + INDEX.md template
4. **Audit log**: report creation result

## Created structure

```
docs/07-knowledge/
├── INDEX.md              # Card count table per category (initial value 0)
├── mistakes/             # Mistakes / bugs / gotchas (M category)
│   └── README.md
├── operations/           # Operations / deployment / incidents (O category)
│   └── README.md
├── architecture/         # Architecture / design patterns (A category)
│   └── README.md
├── api/                  # API usage / gotchas (AP category)
│   └── README.md
├── domain/               # Domain knowledge (D category)
│   └── README.md
└── migration/            # Migration procedures (MG category)
    └── README.md
```

## Execution steps

### Step 1: Existence check

```bash
if [ -d "docs/07-knowledge" ]; then
    echo "[SKIP] docs/07-knowledge/ already exists"
    find docs/07-knowledge -type d -maxdepth 2 | sort
    exit 0
fi
```

### Step 2: Create directories and files

Create the 6 category directories and a `README.md` in each. Each README briefly
explains the category's purpose and its card ID prefix.

### Step 3: Generate INDEX.md template

```markdown
# Knowledge Base Index

> Index of cards added via `/learned` or the `hstl-oss kb create` CLI.

## Card counts per category

| Category | Subdirectory | Card count | Last updated |
|---------|:----------:|:-------:|:--------:|
| mistakes | mistakes/ | 0 | — |
| operations | operations/ | 0 | — |
| architecture | architecture/ | 0 | — |
| api | api/ | 0 | — |
| domain | domain/ | 0 | — |
| migration | migration/ | 0 | — |
| **Total** | | **0** | — |

## Usage

- Add a card: `/learned` skill or the `hstl-oss kb create` CLI
- List cards: `hstl-oss kb list` CLI
- Rebuild index: `hstl-oss kb rebuild` CLI
```

### Step 4: Completion report

Report the count of created directories/files and their paths to the user.

## Error handling

- Missing `docs/` directory -> auto-created via `mkdir -p docs/07-knowledge/...`
- Permission errors -> print error and abort (ask user to verify permissions)
- Existing directory -> skip + report current structure (never overwrite)

## Related

- `hstl-oss kb create` — KB card creation CLI
- `hstl-oss kb list` — KB card listing CLI
- `/learned` skill — record lessons during conversation
