# Task File Quality Checklist

## Frontmatter (BLOCK)

Every Task file must include these 9 fields:

| Field | Type | Required | Example |
|------|------|:---:|------|
| id | string | ✅ | |
| title | string (quoted) | ✅ | "Implement IndicatorRegistry" |
| type | enum | ✅ | feature/bugfix/docs/refactor/infra/test/chore |
| sprint | string | ✅ | sprint-NN |
| status | enum | ✅ | todo/in-progress/done/cancelled/deferred |
| priority | enum | ✅ | P0/P1/P2/P3 |
| estimate | enum | ✅ | S/M/L/XL |
| depends_on | list | ✅ | [Tnnn, Tmmm] or [] |
| created | date | ✅ | 2026-03-20 |

### Frontmatter Example

```yaml
---
id:
title: "Conditional-search HTS setup guide"
type: feature
sprint: sprint-NN
status: todo
priority: P1
estimate: S
depends_on: []
created: 2026-03-20
---
```

## Filename (BLOCK)

Pattern: `TNN-kebab-case-title.md`
- TNN: T + number (e.g. T001, T042)
- kebab-case: lowercase ASCII + hyphens (no special chars)
- 50 chars maximum

Bad: `--5-hts---.md`, `------.md`
Good: `T001-condition-search-hts-setup.md`

## Required Sections (BLOCK)

### h1 title
- `# TNN: {title}` format

### `## Requirements`
- 3+ lines
- Core explanation of what / why / how

```markdown
## Requirements

Document the procedure for registering a conditional-search formula in Kiwoom HTS.
The conditional-search REST API does not work without prior HTS registration,
so a setup guide is required to reproduce the development environment.
```

### `## Done Criteria`
- Checkbox format: `- [ ]` or `- [x]`
- Minimum 1

```markdown
## Done Criteria

- [ ] Include screenshots of the HTS conditional-search registration procedure
- [ ] Verify a successful conditional-search API call from the test environment
- [ ] Save the guide file under docs/guides/
```

## Result Section (done tasks only, WARN)

Tasks with status=done must include:

```markdown
## Result

**Completed**: YYYY-MM-DD  |  **Session**: S#N

### Changed files
- `path/to/file` — description

### Key decisions
- decision content
```

## Optional Sections (INFO)

Recommended for M-and-larger Tasks:
- `## Work Details`: step-by-step implementation plan
- `## Technical Notes`: implementation pitfalls
- `## Related Files`: related source paths

## Additional Completion Criterion for Bulk-frontmatter Tasks (WARN)

For Tasks (estimate M and above) that bulk-apply frontmatter to many files via an
agent/script, the completion criteria must include the following item:

```markdown
- [ ] git-log-based `created` date verified (0 files with 7+ days difference)
```

**Verification**:
```bash
# Compare each file's first commit date with the frontmatter `created` date
git log --diff-filter=A --format=%ai -- {filepath}
```

If a file's difference is 7 days or more, correct it to the actual first commit date.

**Background**: When an agent fails to detect inline dates
(`> Authored: YYYY-MM-DD`), it inserts the working date by default. Completing
without this verification can produce follow-up correction Tasks across Sprints.

## Auto-fixable Items

| Issue | Fix |
|------|----------|
| done but missing `## Result` | Insert default stub |
| done but unchecked checkboxes | `- [ ]` → `- [x]` |
| Frontmatter field missing | Add with default (estimate: S, depends_on: []) |

## Not Auto-fixable

| Issue | Reason |
|------|------|
| `## Requirements` content missing | Must be authored by a human |
| `## Done Criteria` missing | Requires human judgement |
| Changed-files list under `## Result` | Requires reviewing the implementation |
