#!/bin/bash
# =============================================================================
# hostler-oss CLI build + install script
#
# Usage:
#   ./scripts/build-install.sh          # build + install (default)
#   ./scripts/build-install.sh build    # build only
#   ./scripts/build-install.sh install  # install only (reuse existing build)
#   ./scripts/build-install.sh test     # test + build + install
#   ./scripts/build-install.sh clean    # clean build + install
#
# Install target: ~/.local/bin/hstl-oss
# =============================================================================

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
BIN_DIR="$PROJECT_DIR/bin"
INSTALL_DIR="${HOSTLER_OSS_INSTALL_DIR:-$HOME/.local/bin}"
ACTION="${1:-all}"

# Colours
RED='\033[0;31m'
GREEN='\033[0;32m'
CYAN='\033[0;36m'
NC='\033[0m'

log_info()  { echo -e "${CYAN}[INFO]${NC} $1"; }
log_ok()    { echo -e "${GREEN}[OK]${NC} $1"; }
log_error() { echo -e "${RED}[ERROR]${NC} $1" >&2; }

# Version metadata
VERSION=$(cd "$PROJECT_DIR" && git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT=$(cd "$PROJECT_DIR" && git rev-parse --short HEAD 2>/dev/null || echo "unknown")
DATE=$(date -u '+%Y-%m-%dT%H:%M:%SZ')

build() {
    log_info "Building (version=$VERSION commit=$COMMIT)"
    cd "$PROJECT_DIR"

    mkdir -p "$BIN_DIR"

    go build -trimpath \
        -ldflags "-s -w \
            -X 'github.com/hk-aitech/hostler-oss/packages/hostler-cli/cmd/hstl-oss/cmd.Version=$VERSION' \
            -X 'github.com/hk-aitech/hostler-oss/packages/hostler-cli/cmd/hstl-oss/cmd.Commit=$COMMIT' \
            -X 'github.com/hk-aitech/hostler-oss/packages/hostler-cli/cmd/hstl-oss/cmd.Date=$DATE'" \
        -o "$BIN_DIR/hstl-oss" ./cmd/hstl-oss
    log_ok "hstl-oss CLI built ($(du -h "$BIN_DIR/hstl-oss" | cut -f1))"
}

install_bins() {
    mkdir -p "$INSTALL_DIR"

    if [ ! -f "$BIN_DIR/hstl-oss" ]; then
        log_error "$BIN_DIR/hstl-oss missing — run 'build' first"
        exit 1
    fi

    # Replace a running binary safely via rename.
    cp "$BIN_DIR/hstl-oss" "$INSTALL_DIR/hstl-oss.new"
    mv "$INSTALL_DIR/hstl-oss.new" "$INSTALL_DIR/hstl-oss"
    chmod +x "$INSTALL_DIR/hstl-oss"
    log_ok "hstl-oss -> $INSTALL_DIR/hstl-oss"

    echo ""
    log_info "Install result:"
    ls -lh "$INSTALL_DIR/hstl-oss"
    echo ""
    "$INSTALL_DIR/hstl-oss" version
}

run_tests() {
    log_info "Running tests"
    cd "$PROJECT_DIR"
    go test -count=1 ./...
    log_ok "All tests passed"
}

clean_build() {
    log_info "Clean build"
    cd "$PROJECT_DIR"
    rm -rf "$BIN_DIR"
    go clean -cache -testcache
    build
}

case "$ACTION" in
    build)
        build
        ;;
    install)
        install_bins
        ;;
    test)
        run_tests
        build
        install_bins
        ;;
    clean)
        clean_build
        install_bins
        ;;
    all)
        build
        install_bins
        ;;
    *)
        echo "Usage: $0 {build|install|test|clean|all}"
        exit 1
        ;;
esac

echo ""
log_ok "Done"
