#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
CONFIG_PATH="${CODEX_CONFIG_PATH:-$HOME/.codex/config.toml}"
BIN_PATH="$ROOT_DIR/.codex-mcp/bin/codex-mcp"
SKILL_SRC="$ROOT_DIR/skills/codex-troller-autostart"
SKILL_DST="$HOME/.codex/skills/codex-troller-autostart"

echo "[agent-install] start"
echo "[agent-install] root: $ROOT_DIR"
echo "[agent-install] config: $CONFIG_PATH"

make -C "$ROOT_DIR" setup >/dev/null

mkdir -p "$(dirname "$CONFIG_PATH")"
touch "$CONFIG_PATH"

TMP_FILE="$(mktemp)"
cleanup() {
  rm -f "$TMP_FILE"
}
trap cleanup EXIT

# Remove existing codex-troller MCP section for idempotent install.
awk '
BEGIN { skip = 0 }
{
  if ($0 ~ /^\[mcp_servers\.codex-troller\]$/) {
    skip = 1
    next
  }
  if (skip == 1 && $0 ~ /^\[/) {
    skip = 0
  }
  if (skip == 0) {
    print $0
  }
}
' "$CONFIG_PATH" >"$TMP_FILE"

mv "$TMP_FILE" "$CONFIG_PATH"

{
  echo
  echo "[mcp_servers.codex-troller]"
  echo "command = \"$BIN_PATH\""
} >>"$CONFIG_PATH"

if ! grep -q '^\[mcp_servers\.codex-troller\]$' "$CONFIG_PATH"; then
  echo "[agent-install] failed to write MCP section" >&2
  exit 1
fi

if ! grep -q "^command = \"$BIN_PATH\"$" "$CONFIG_PATH"; then
  echo "[agent-install] failed to write command path" >&2
  exit 1
fi

if [[ ! -f "$SKILL_SRC/SKILL.md" ]]; then
  echo "[agent-install] missing skill source: $SKILL_SRC/SKILL.md" >&2
  exit 1
fi

mkdir -p "$(dirname "$SKILL_DST")"
rm -rf "$SKILL_DST"
cp -R "$SKILL_SRC" "$SKILL_DST"

echo "[agent-install] done"
echo "[agent-install] registered mcp_servers.codex-troller"
echo "[agent-install] installed skill: codex-troller-autostart"
