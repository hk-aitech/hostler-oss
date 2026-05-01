# Contributing to hostler-oss

Thanks for considering a contribution! This document covers the quick reality of working on hostler-oss — branch policy, build / test expectations, code review, and release flow.

## Ground rules

- **License**: by submitting a contribution you agree it is licensed under [Apache License 2.0](LICENSE).
- **Language**: source code, comments, commit messages, ADRs, and skill bodies are written in English.
- **Author email**: include a real address in commits — the project uses `Co-Authored-By` trailers for collaborative work and they need to resolve.

## Workflow

### 1. Pick something to work on

- Open issues and pull requests live at <https://github.com/hk-aitech/hostler-oss>.
- For non-trivial changes, open an issue first with the proposal so we can align on scope before you write code.

### 2. Create a topic branch off `main`

```bash
git checkout main
git pull --ff-only
git checkout -b <kind>/<short-slug>          # kind ∈ feat, fix, refactor, docs, chore
```

### 3. Make the change

- Source layout follows the standard `packages/`, `docs/`, `works/`, `archive/` tree. Files belong where the [`project-structure`](packages/hostler-plugin/skills/project-structure) skill says they belong.
- Architecture decisions go into `docs/02-architecture/ADR-NNN-<slug>.md`. The [`project-adr`](packages/hostler-plugin/skills/project-adr) skill describes the conventions.
- Skill bodies, command bodies, and agent prompts are markdown — see existing examples for style.

### 4. Verify locally

The two gates you have to clear before opening an MR:

```bash
cd packages/hostler-cli
go build ./...
go test ./...
```

Both must pass. The CI pipeline in [`.gitlab-ci.yml`](.gitlab-ci.yml) re-runs them and adds a `go vet` lint stage.

### 5. Commit and push

- Use [Conventional Commits](https://www.conventionalcommits.org/) style:
  `feat(scope): summary`, `fix(scope): summary`, `docs(scope): summary`, etc.
- Keep each commit focused and self-explanatory in the body — the log is read by humans.
- One commit per logical change is the goal. Squashing tiny WIP commits before pushing is welcome.

```bash
git push -u origin <kind>/<short-slug>
```

### 6. Open a pull request

- Target branch: `main`.
- Fill in the description: what changed, why, how it was tested.
- Link related issues / ADRs.
- Mark as draft if you want early review on direction.

## Code review

- Reviewers focus on: correctness, scope discipline, test coverage of the change, layout / naming consistency, and whether new public surface is documented.
- Address review comments by pushing follow-up commits. Reviewers will squash on merge if useful, or you can rebase yourself.
- Avoid force-pushing once a reviewer has loaded the diff — leave the trail for them.

## Release

Releases follow [Semantic Versioning](https://semver.org/). Tags are created from `main` after the relevant changes have landed and CI is green.

```bash
git tag -a v0.X.Y -m "Release v0.X.Y"
git push origin v0.X.Y
```

Until `v1.0.0`, public API and command surface may shift. Document any breaking change in the MR description and the release notes.

## Reporting bugs

Open an issue with:

- Steps to reproduce (commands run, files created, expected vs. actual output).
- Environment (`hstl-oss version`, `go version`, OS).
- Whether you can reproduce with a clean workspace (`hstl-oss project init` in a fresh directory).

A reproduction fixture beats a long description; the [`troubleshooting`](packages/hostler-plugin/skills/troubleshooting) skill has a fixture template.

## Architecture decisions

If your change crosses a structural boundary — public CLI surface, plugin manifest, schema, dependency injection — write a short ADR and include it with the MR. Use one of the templates under [`packages/hostler-plugin/skills/project-adr/references/templates/`](packages/hostler-plugin/skills/project-adr/references/templates) (Nygard / MADR / Y-statement).

## Code of conduct

Be kind, focus on the work, assume good intent, and disagree on the merits. Project decisions are documented in ADRs so we can come back to them later.
