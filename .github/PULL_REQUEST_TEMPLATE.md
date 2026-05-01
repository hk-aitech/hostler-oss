<!--
Thanks for the contribution! A few asks before review:

- Keep the PR focused. Bundle large, unrelated changes into separate PRs.
- Match the existing code style. Run `make test` and `make lint` from
  packages/hostler-cli/ before pushing.
- For docs-only PRs, skip the "Tests" section and explain the doc shift.
-->

## Summary

<!-- What does this PR change, in two sentences? -->

## Why

<!-- The problem this solves or the use case it enables. Link related
     issues with `Fixes #N` / `Refs #N`. -->

## Changes

- [ ] CLI surface change (mention new flag / subcommand)
- [ ] Plugin surface change (skill / command / hook / agent)
- [ ] File-state shape change (frontmatter / directory layout)
- [ ] Documentation only
- [ ] Internal refactor — no observable behavior change
- [ ] CI / build / release pipeline

## Tests

<!-- Which test files were added or updated? Did `go test -count=1 -race
     ./...` pass locally? Include a short transcript snippet if the PR
     fixes a regression. -->

## Breaking change?

<!-- If yes:
     - What breaks?
     - What's the migration path?
     - Did you update CHANGELOG.md under [Unreleased] → ### Changed (BREAKING)?
-->

## Checklist

- [ ] `make build` / `make test` pass locally
- [ ] `go vet ./...` clean
- [ ] `CHANGELOG.md` updated (under `[Unreleased]`)
- [ ] Docs updated (if behaviour or surface changed)
- [ ] No internal-only references reintroduced (project codenames, private hostnames, internal task IDs, internal env-var prefixes, etc.)
- [ ] Commit message follows Conventional Commits (`feat:` / `fix:` / `chore:` / `docs:` / …)
