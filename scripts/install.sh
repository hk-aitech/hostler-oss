#!/usr/bin/env sh
# install.sh — POSIX-shell installer for the hstl-oss CLI.
#
# Auto-detects OS / arch, downloads the matching pre-built tarball from
# GitHub Releases, verifies its SHA256, extracts the binary, and places
# it on PATH. Works without Go on the user's machine.
#
# Usage:
#   curl -fsSL https://github.com/hk-aitech/hostler-oss/releases/latest/download/install.sh | sh
#
# Or pin to a specific version:
#   curl -fsSL https://github.com/hk-aitech/hostler-oss/releases/download/v0.1.0/install.sh | sh
#
# Honour environment overrides:
#   HSTL_VERSION   = vX.Y.Z (default: latest)
#   HSTL_INSTALL_DIR = target dir (default: $HOME/.local/bin)
#   HSTL_FORCE     = 1 to overwrite existing install without prompt

set -eu

REPO="hk-aitech/hostler-oss"
VERSION="${HSTL_VERSION:-latest}"
INSTALL_DIR="${HSTL_INSTALL_DIR:-$HOME/.local/bin}"
FORCE="${HSTL_FORCE:-0}"

# ---- helpers --------------------------------------------------------------

green=''
yellow=''
red=''
dim=''
reset=''
if [ -t 1 ] && command -v tput >/dev/null 2>&1; then
    green=$(tput setaf 2 || true)
    yellow=$(tput setaf 3 || true)
    red=$(tput setaf 1 || true)
    dim=$(tput dim || true)
    reset=$(tput sgr0 || true)
fi

info()  { printf '%s[install]%s %s\n' "$green" "$reset" "$1"; }
warn()  { printf '%s[install]%s %s\n' "$yellow" "$reset" "$1" >&2; }
err()   { printf '%s[install]%s %s\n' "$red"    "$reset" "$1" >&2; exit 1; }

require() {
    command -v "$1" >/dev/null 2>&1 || err "missing required command: $1"
}

require curl
require tar
require uname

# ---- detect OS / arch -----------------------------------------------------

raw_os=$(uname -s | tr '[:upper:]' '[:lower:]')
raw_arch=$(uname -m)

case "$raw_os" in
    linux)  os=linux ;;
    darwin) os=darwin ;;
    msys*|mingw*|cygwin*) os=windows ;;
    *) err "unsupported OS: $raw_os" ;;
esac

case "$raw_arch" in
    x86_64|amd64) arch=amd64 ;;
    arm64|aarch64) arch=arm64 ;;
    *) err "unsupported architecture: $raw_arch" ;;
esac

ext=tar.gz
[ "$os" = "windows" ] && ext=zip

info "platform detected: ${os}/${arch}"

# ---- resolve version ------------------------------------------------------

if [ "$VERSION" = "latest" ]; then
    info "resolving latest release tag..."
    VERSION=$(curl -fsSL "https://api.github.com/repos/${REPO}/releases/latest" \
        | sed -n 's/.*"tag_name"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p' \
        | head -1)
    [ -n "$VERSION" ] || err "failed to resolve latest version (rate-limited? try setting HSTL_VERSION=vX.Y.Z)"
fi

case "$VERSION" in
    v*) ;;
    *)  VERSION="v${VERSION}" ;;
esac

info "installing ${VERSION}"

# ---- download asset + checksum -------------------------------------------

asset="hstl-oss-${VERSION}-${os}-${arch}.${ext}"
asset_url="https://github.com/${REPO}/releases/download/${VERSION}/${asset}"
sums_url="https://github.com/${REPO}/releases/download/${VERSION}/SHA256SUMS"

tmpdir=$(mktemp -d)
trap 'rm -rf "$tmpdir"' EXIT

info "downloading ${asset}..."
curl -fsSL --output "${tmpdir}/${asset}" "$asset_url" || err "download failed: $asset_url"

info "downloading checksums..."
curl -fsSL --output "${tmpdir}/SHA256SUMS" "$sums_url" || warn "no SHA256SUMS for ${VERSION} — skipping verification"

# ---- verify checksum ------------------------------------------------------

if [ -f "${tmpdir}/SHA256SUMS" ]; then
    expected=$(grep " ${asset}\$" "${tmpdir}/SHA256SUMS" | awk '{print $1}')
    if [ -z "$expected" ]; then
        warn "no entry for ${asset} in SHA256SUMS — skipping verification"
    else
        if command -v sha256sum >/dev/null 2>&1; then
            actual=$(sha256sum "${tmpdir}/${asset}" | awk '{print $1}')
        elif command -v shasum >/dev/null 2>&1; then
            actual=$(shasum -a 256 "${tmpdir}/${asset}" | awk '{print $1}')
        else
            warn "no sha256sum/shasum available — skipping verification"
            actual="$expected"
        fi
        if [ "$actual" != "$expected" ]; then
            err "checksum mismatch! expected=${expected} actual=${actual}"
        fi
        info "checksum OK"
    fi
fi

# ---- extract --------------------------------------------------------------

info "extracting..."
case "$ext" in
    tar.gz)
        tar -xzf "${tmpdir}/${asset}" -C "${tmpdir}"
        bin_src="${tmpdir}/hstl-oss-${VERSION}-${os}-${arch}/hstl-oss"
        ;;
    zip)
        require unzip
        unzip -q "${tmpdir}/${asset}" -d "${tmpdir}/extracted"
        bin_src="${tmpdir}/extracted/hstl-oss.exe"
        ;;
esac

[ -f "$bin_src" ] || err "extracted binary not found: $bin_src"

# ---- install --------------------------------------------------------------

mkdir -p "$INSTALL_DIR"
target="${INSTALL_DIR}/hstl-oss"
[ "$os" = "windows" ] && target="${target}.exe"

if [ -f "$target" ] && [ "$FORCE" != "1" ]; then
    if [ -t 0 ]; then
        existing=$("$target" version 2>/dev/null | sed -n 's/.*"version"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p' | head -1)
        printf '%s[install]%s replace existing %s? [y/N] ' "$yellow" "$reset" "${existing:-unknown}"
        read -r answer
        case "$answer" in y|Y|yes|YES) ;; *) err "aborted by user" ;; esac
    else
        warn "existing ${target} preserved (set HSTL_FORCE=1 to overwrite)"
        exit 0
    fi
fi

cp "$bin_src" "$target"
chmod +x "$target"

info "installed: ${target}"

# ---- PATH check -----------------------------------------------------------

case ":$PATH:" in
    *":${INSTALL_DIR}:"*) ;;
    *)
        warn "${INSTALL_DIR} is not on PATH"
        printf '%s     add this line to your shell rc:%s\n' "$dim" "$reset" >&2
        # shellcheck disable=SC2016 # the literal $PATH is part of the rc-line we want the user to copy
        printf '       export PATH="%s:$PATH"\n' "$INSTALL_DIR" >&2
        ;;
esac

# ---- verify ---------------------------------------------------------------

info "running version check..."
"$target" version || warn "version output looks off — please re-run manually"

info "done. start with:  hstl-oss --help"
