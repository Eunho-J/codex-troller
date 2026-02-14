#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
GO_BIN="$("$ROOT_DIR/scripts/bootstrap-go.sh")"
OUT_BIN="$ROOT_DIR/.codex-mcp/bin/codex-mcp"

"$GO_BIN" build -o "$OUT_BIN" "$ROOT_DIR/cmd/codex-mcp"
echo "CODEx MCP server: $OUT_BIN"
"$OUT_BIN"
