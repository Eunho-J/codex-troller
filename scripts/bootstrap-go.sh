#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
GO_VERSION="${CODEX_MCP_GO_VERSION:-$(awk '/^go /{print $2; exit}' "$ROOT_DIR/go.mod")}"
if [ -n "${CODEX_MCP_GO_TARGET_DIR:-}" ]; then
  TARGET_DIR="$CODEX_MCP_GO_TARGET_DIR"
elif [ -n "${CODEX_HOME:-}" ]; then
  TARGET_DIR="$CODEX_HOME/.codex-troller/deps"
else
  TARGET_DIR="$ROOT_DIR/.codex-mcp/.tools"
fi
GO_ROOT="$TARGET_DIR/go"

normalize_version() {
  echo "$1" | sed 's/^go//'
}

version_ge() {
  local left right
  left="$(normalize_version "$1")"
  right="$(normalize_version "$2")"
  [ "$(printf '%s\n%s\n' "$left" "$right" | sort -V | tail -n 1)" = "$left" ]
}

if command -v go >/dev/null 2>&1; then
  system_go="$(command -v go)"
  system_version="$(go version | awk '{print $3}')"
  if version_ge "$system_version" "$GO_VERSION"; then
    echo "$system_go"
    exit 0
  fi
fi

if [ -x "$GO_ROOT/bin/go" ]; then
  echo "$GO_ROOT/bin/go"
  exit 0
fi

OS="$(uname -s | tr '[:upper:]' '[:lower:]')"
ARCH="$(uname -m)"
case "$ARCH" in
  x86_64) ARCH="amd64" ;;
  arm64) ARCH="arm64" ;;
  aarch64) ARCH="arm64" ;;
  *)
    echo "unsupported architecture: $ARCH" >&2
    exit 1
    ;;
esac

TMP="$(mktemp -d)"
cleanup() {
  rm -rf "$TMP"
}
trap cleanup EXIT

if command -v curl >/dev/null 2>&1; then
  curl -fsSL "https://go.dev/dl/go${GO_VERSION}.${OS}-${ARCH}.tar.gz" -o "$TMP/go.tar.gz"
elif command -v wget >/dev/null 2>&1; then
  wget -q "https://go.dev/dl/go${GO_VERSION}.${OS}-${ARCH}.tar.gz" -O "$TMP/go.tar.gz"
else
  echo "curl or wget is required to install Go automatically" >&2
  exit 1
fi

mkdir -p "$TARGET_DIR"
rm -rf "$GO_ROOT"
mkdir -p "$TARGET_DIR"
tar -C "$TARGET_DIR" -xzf "$TMP/go.tar.gz"

echo "$GO_ROOT/bin/go"
