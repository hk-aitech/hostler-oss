#!/usr/bin/env bash
# ensure-cli.sh — SessionStart guard. The hstl-oss plugin needs the
# hstl-oss Go CLI on PATH; this hook detects when it is missing and
# prints platform-specific install instructions on stderr. It never
# auto-downloads a binary — the user runs the install step explicitly so
# trust boundaries stay clear.
#
# Exits 0 either way; the plugin is still useful (slash commands /
# skills load) even if the CLI happens to be missing.

set -uo pipefail

GREEN=$'\033[0;32m'
YELLOW=$'\033[0;33m'
CYAN=$'\033[0;36m'
DIM=$'\033[2m'
RESET=$'\033[0m'

if command -v hstl-oss >/dev/null 2>&1; then
    # CLI present — quietly announce the version so the user can spot
    # mismatches at a glance.
    version=$(hstl-oss version 2>/dev/null | sed -n 's/.*"version"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p' | head -1)
    if [ -n "$version" ]; then
        echo "${GREEN}[hstl-oss]${RESET} CLI present: ${CYAN}$(command -v hstl-oss)${RESET} ${DIM}(${version})${RESET}" >&2
    fi
    exit 0
fi

# CLI missing — show install instructions matching the user's platform.
os=$(uname -s 2>/dev/null || echo unknown)
arch=$(uname -m 2>/dev/null || echo unknown)

cat >&2 <<EOF
${YELLOW}[hstl-oss]${RESET} The ${CYAN}hstl-oss${RESET} CLI is not on PATH. Slash commands that
shell out to it (${DIM}/hstl-oss:task:*, /hstl-oss:sprint:*, etc.${RESET}) will fail
until you install it.

Detected platform: ${os}/${arch}

Pick whichever path matches your environment:

${CYAN}A. Pre-built binary (recommended — no Go required)${RESET}
    curl -fsSL https://github.com/hk-aitech/hostler-oss/releases/latest/download/install.sh | sh

${CYAN}B. With Go 1.24+${RESET}
    go install github.com/hk-aitech/hostler-oss/packages/hostler-cli/cmd/hstl-oss@latest

${CYAN}C. From a local clone${RESET}
    git clone https://github.com/hk-aitech/hostler-oss
    cd hostler-oss/packages/hostler-cli && make install

Or run ${CYAN}/hstl-oss:cli:install${RESET} inside Claude Code for a guided install.

After install, ensure ${DIM}\$HOME/.local/bin${RESET} (or your Go bin dir) is on PATH and
re-launch this Claude Code session so the plugin sees the new binary.

Full guide: ${DIM}docs/04-guides/installation.md${RESET}
EOF
exit 0
