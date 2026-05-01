---
description: Install the hstl-oss Go CLI binary on the user's machine. Use when `hstl-oss` is not on PATH, the SessionStart hook reported the binary missing, or the user explicitly asks to install / re-install / upgrade the CLI.
allowed-tools: Bash(sh:*), Bash(curl:*), Bash(go:install), Bash(git:clone), Bash(make:install), Bash(command:*), Bash(hstl-oss:*), Bash(uname:*), Read
argument-hint: "[--from-source | --go-install]"
---

# CLI install — bring `hstl-oss` onto the user's PATH

The plugin needs the `hstl-oss` binary to run slash commands like `/hstl-oss:task:start`. This command picks the right install path for the user's environment, runs it, and verifies the result.

## Decision tree

1. **Already installed?** Run `command -v hstl-oss && hstl-oss version`. If both succeed and the version looks current, report and exit.

2. **Default — pre-built binary** (works without Go on the user's machine):

    ```bash
    curl -fsSL https://github.com/hk-aitech/hostler-oss/releases/latest/download/install.sh | sh
    ```

    The script auto-detects OS / arch (linux-amd64, linux-arm64, darwin-amd64, darwin-arm64, windows-amd64), downloads the matching tarball from the latest GitHub Release, verifies its SHA256 against `SHA256SUMS`, extracts the binary, and drops it into `$HOME/.local/bin/hstl-oss`. Honours environment overrides:

    - `HSTL_VERSION=vX.Y.Z` — pin to a specific release.
    - `HSTL_INSTALL_DIR=/usr/local/bin` — alternate target directory.
    - `HSTL_FORCE=1` — overwrite an existing install without prompting.

3. **`--go-install` requested or pre-built download fails**: when Go 1.24+ is on the user's machine:

    ```bash
    GOTOOLCHAIN=auto go install github.com/hk-aitech/hostler-oss/packages/hostler-cli/cmd/hstl-oss@latest
    ```

    Drops the binary into `$(go env GOBIN)` (or `$(go env GOPATH)/bin`). Verify the directory is on PATH; if not, add a one-line export and tell the user.

4. **`--from-source` requested**: when the user wants to compile locally:

    ```bash
    tmpdir=$(mktemp -d)
    git clone --depth 1 https://github.com/hk-aitech/hostler-oss "$tmpdir/hostler-oss"
    cd "$tmpdir/hostler-oss/packages/hostler-cli" && make install
    ```

    Cleanup: leave the clone in place if the user wants to develop, otherwise `rm -rf "$tmpdir"`.

## After install

Always verify:

```bash
hstl-oss version
```

Expected output is a JSON envelope `{"data":{"commit":"…","date":"…","version":"…"},"status":"ok"}`. If `command -v hstl-oss` still fails, the install dir isn't on PATH — surface the exact `export PATH=…` line the user should add to their shell rc.

Re-launch the Claude Code session at the end so the SessionStart hook re-detects the binary and confirms the version.

## Don'ts

- Don't fabricate binary URLs. The installer script + checksums file are published as part of the GitHub Release; if a release doesn't exist yet for the current platform, fall back to `go install` or `from-source` rather than hand-rolling a download URL.
- Don't sudo. `~/.local/bin` and `$GOPATH/bin` cover the supported targets; if neither is on PATH, fix PATH, not permissions.
- Don't symlink, mv, or overwrite an existing `hstl-oss` without asking — the user may have a custom install. The installer script prompts when stdin is a TTY; outside a TTY it preserves the existing binary.
