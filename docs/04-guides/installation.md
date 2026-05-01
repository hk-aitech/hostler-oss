# Installation

The two halves of hostler-oss install independently:

- **CLI** (`hstl-oss`) — Go binary, runs anywhere with Go 1.24+.
- **Plugin** — Claude Code plugin, installed into a Claude Code plugin directory.

## CLI

### One-line install (recommended)

```bash
curl -fsSL https://github.com/hk-aitech/hostler-oss/releases/latest/download/install.sh | sh
```

The installer script (published with every GitHub Release) auto-detects the user's OS and architecture, downloads the matching pre-built binary, verifies its SHA256 checksum, and drops `hstl-oss` into `$HOME/.local/bin`. Supported platforms: `linux-amd64`, `linux-arm64`, `darwin-amd64`, `darwin-arm64`, `windows-amd64`.

Environment overrides:

| Variable | Default | Purpose |
|---|---|---|
| `HSTL_VERSION` | `latest` | Pin to a specific release tag (e.g. `v0.1.0`). |
| `HSTL_INSTALL_DIR` | `$HOME/.local/bin` | Target directory. |
| `HSTL_FORCE` | `0` | Set to `1` to overwrite an existing install without prompting. |

Pin to a specific release:

```bash
curl -fsSL https://github.com/hk-aitech/hostler-oss/releases/download/v0.1.0/install.sh | sh
```

### `go install` (when Go is available)

```bash
GOTOOLCHAIN=auto go install github.com/hk-aitech/hostler-oss/packages/hostler-cli/cmd/hstl-oss@latest
```

Drops the binary into `$(go env GOPATH)/bin`. Make sure that directory is on `PATH`.

### Build from source

```bash
git clone https://github.com/hk-aitech/hostler-oss
cd hostler-oss/packages/hostler-cli
make install
```

`make install` produces `bin/hstl-oss`, copies it to `~/.local/bin/hstl-oss`, and prints the installed version.

### Cross-compile

```bash
cd hostler-oss/packages/hostler-cli
make cross
ls bin/hstl-oss-linux-{arm64,amd64}
```

### Verify

```bash
hstl-oss version
# {"data":{"commit":"...","date":"...","version":"..."},"status":"ok"}

hstl-oss --help
hstl-oss manifest --schema | jq '.commands | length'
```

## Plugin

The plugin lives under `packages/hostler-plugin/`. Three install paths, pick whichever matches your setup.

### Option A — marketplace install from GitHub (recommended for end users)

Inside Claude Code:

```
/plugin marketplace add https://github.com/hk-aitech/hostler-oss
/plugin install hstl-oss@hostler-oss
```

The first line registers the repo as a marketplace (it reads `.claude-plugin/marketplace.json` at the root). The second installs the `hstl-oss` plugin from that marketplace into your Claude Code session. Updates land via `/plugin marketplace update hostler-oss` followed by `/plugin install hstl-oss@hostler-oss` again.

CLI alternative (when you want to install without entering Claude Code):

```bash
claude plugin marketplace add https://github.com/hk-aitech/hostler-oss
claude plugin install hstl-oss@hostler-oss
```

Pin to a specific tag or commit:

```
/plugin marketplace add https://github.com/hk-aitech/hostler-oss@v0.1.0
/plugin marketplace add https://github.com/hk-aitech/hostler-oss@<commit-sha>
```

### Option B — local symlink (developer-friendly)

When you've cloned the repo and want Claude Code to load the working tree directly:

```bash
ln -s "$(pwd)/packages/hostler-plugin" ~/.claude/plugins/hstl-oss
```

Re-launch Claude Code. Edits to the plugin tree take effect on the next session start; no install step needed.

### Option C — direct CLI install from a local path

```bash
claude plugin install ./packages/hostler-plugin
```

Same effect as the symlink, except Claude Code copies the plugin into its plugin directory rather than referencing it in place.

### Verify

In Claude Code, type `/hstl-oss:` — autocompletion should list the command groups (`task`, `sprint`, `kb`, `dev`, `git`, `hotfix`, `project`, `cli`). Skills auto-trigger on relevant prompts; type a sentence like "create a sprint for refactoring" and the `sprint-management` skill should fire.

## Auto-checking that the CLI is present

The plugin ships a `SessionStart` hook (`hooks/ensure-cli.sh`) that runs once at the start of every Claude Code session. It only **inspects** PATH — it never auto-downloads binaries — and prints platform-specific install instructions on stderr if the CLI is missing. This way the plugin's slash commands and skills load even when the CLI isn't installed yet, but the user gets a clear nudge with copy-paste install commands.

If the CLI is missing, run the one-step installer command:

```
/hstl-oss:cli:install
```

The slash command walks through the install decision tree:

1. Detect whether `hstl-oss` is already present (and verifies the version).
2. Prefer `go install …@latest` when Go is on the user's machine.
3. Fall back to a `git clone … && make install` from source.
4. If neither path works, surface the GitHub Releases URL for a pre-built binary download.

Re-launch the Claude Code session after install so the SessionStart hook re-detects the binary and confirms the version.

## Environment variables

The CLI reads a small set of `HSTL_*` env vars. None are required — defaults work for most setups.

| Variable | Purpose | Default |
|---|---|---|
| `HSTL_PROJECT_ROOT` | Override project-root detection (skip git toplevel + cwd walk) | `git rev-parse --show-toplevel` or `cwd` |
| `HSTL_DB_PATH` | Override the SQLite index location | `~/.hostler/data/<project-key>/hstl.db` |
| `HSTL_SUMMARY_POLICY` | Strict heuristic on `task create --summary` (`strict` / `moderate` / `lenient` / `off`) | `strict` |
| `HSTL_SUMMARY_SUBAGENT` | Run an LLM judge after the heuristic (`on` / `off`) | `on` |
| `HSTL_SUMMARY_MIN_LEN` | Minimum `--summary` character count | `10` |
| `HSTL_OUTPUT_FORMAT` | Default output format (`json` / `text` / `console`) | `json` |
| `HSTL_TASK_AUTOCHECK_GO` | Auto-run `go build` / `go test` / lint on `task complete` (`on` / `off`) | `on` |
| `HSTL_INSTALL_DIR` | Override `make install` target directory | `~/.local/bin` |

Always pre-fixed with `HSTL_`. Setting any value to its default has no effect; the CLI emits no deprecation warnings on the canonical names.

## Project-config (`.hstl/project-config.yaml`)

Created by `hstl-oss project init-skeleton`. Schema highlights:

```yaml
version: "2.0.0"             # required, enforced via JSON Schema
project:
  key: <auto-generated>      # stable identifier; survives binary upgrades
  kind: go | dotnet          # platform; auto-detected from go.mod / *.csproj
sprints:
  size_mode: solo | pair | team   # tier; defaults follow tier-spec
precommit:
  branch_sync_max_behind: 50      # lift via HSTL_BRANCH_SYNC_MAX_BEHIND env
```

Run `hstl-oss config check` to validate; `hstl-oss config show-effective --field <key>` to see the resolved value with provenance.

## Database (SQLite index)

The CLI maintains a per-project SQLite index at `~/.hostler/data/<project-key>/hstl.db`. It is **always rebuildable** from files via `hstl-oss backlog sync` — losing it never loses data.

Override with `HSTL_DB_PATH` for in-tree storage during testing or when running multiple isolated instances.

## Audit log

Append-only event log with hash chain (tamper evidence). Same directory as the DB. Inspect with `hstl-oss audit list --entity-type task --entity-id T001`. The chain is verified on every read.

## Troubleshooting

### `hstl-oss: command not found`

`~/.local/bin` is not on `PATH`. Either add it (`export PATH="$HOME/.local/bin:$PATH"`) or symlink:

```bash
sudo ln -s ~/.local/bin/hstl-oss /usr/local/bin/hstl-oss
```

### `Sprint "sprint-NN" folder not found`

`task create --sprint sprint-NN` requires the Sprint folder to exist. Run `hstl-oss sprint create --id sprint-NN ...` first, or omit `--sprint` to land in the backlog and assign later (`task assign T001 sprint-NN`).

### `summary static heuristic failed`

The default `strict` summary policy rejects fixture-grade summaries. Two ways out:

1. Write a real summary covering **what / why / success criterion** in ≥10 chars.
2. Lift the policy: `HSTL_SUMMARY_POLICY=lenient hstl-oss task create …`. For CI fixture creation, `--skip-summary-validation` is the documented escape hatch.

### Build fails with `requires go1.25.0`

A transitive dependency (`modernc.org/sqlite`) demands Go 1.25. The toolchain directive in `go.mod` (`toolchain go1.25.9`) auto-fetches it under `GOTOOLCHAIN=auto`. If your shell pins `GOTOOLCHAIN=local`, unset it.

### Stale sprint state after manual edits

Files are SSOT. After hand-editing a Task / Sprint markdown:

```bash
hstl-oss backlog sync
```

reconciles the SQLite index back to the file truth. Listing returns the latest state immediately after.

## Uninstall

```bash
rm ~/.local/bin/hstl-oss
rm -rf ~/.hostler/   # all project DBs + audit logs
# remove the plugin symlink / unregister with Claude Code
```

Project-tree state (`.hstl/`, `works/`, `docs/`) belongs to the project — it stays.
