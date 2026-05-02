#!/usr/bin/env bash
# Ensures the conclave binary exists, downloading it if needed,
# then runs the session-start hook.

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]:-$0}")" && pwd)"
PLUGIN_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
BINARY="${PLUGIN_ROOT}/conclave"

# Detect OS
case "$(uname -s)" in
    Linux)  OS="linux" ;;
    Darwin) OS="darwin" ;;
    *)
        # Unsupported OS, fall back to bash script
        exec "${SCRIPT_DIR}/session-start.sh"
        ;;
esac

# Detect architecture
case "$(uname -m)" in
    x86_64)  ARCH="amd64" ;;
    aarch64) ARCH="arm64" ;;
    arm64)   ARCH="arm64" ;;
    *)
        exec "${SCRIPT_DIR}/session-start.sh"
        ;;
esac

# Probe binary: an unsigned cross-compiled darwin/arm64 binary will be killed
# by the macOS kernel. A broken binary cached from a prior run would make
# `exec` abort the script (exit 127) before the fallback could run, so we
# verify it works first and remove it if not.
binary_works() {
    [ -x "$1" ] && "$1" version >/dev/null 2>&1
}

if binary_works "$BINARY"; then
    exec "$BINARY" hook session-start
fi
rm -f "$BINARY"

# Read version from plugin.json (no jq dependency)
VERSION=$(grep -o '"version"[[:space:]]*:[[:space:]]*"[^"]*"' "${PLUGIN_ROOT}/.claude-plugin/plugin.json" | grep -o '"[^"]*"$' | tr -d '"')

if [ -z "$VERSION" ]; then
    exec "${SCRIPT_DIR}/session-start.sh"
fi

# Download binary
URL="https://github.com/signalnine/conclave/releases/download/v${VERSION}/conclave-${OS}-${ARCH}"

if curl -fsSL -o "$BINARY" "$URL" 2>/dev/null; then
    chmod +x "$BINARY"
    # Ad-hoc sign on macOS: cross-compiled Mach-O binaries built on Linux
    # runners have no signature, and arm64 macOS rejects unsigned binaries.
    if [ "$OS" = "darwin" ] && command -v codesign >/dev/null 2>&1; then
        codesign --sign - --force "$BINARY" >/dev/null 2>&1 || true
    fi
    if binary_works "$BINARY"; then
        exec "$BINARY" hook session-start
    fi
    rm -f "$BINARY"
fi

# Download failed or binary still doesn't run — fall back to bash script
exec "${SCRIPT_DIR}/session-start.sh"
