#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
HOOK_DIR="$ROOT_DIR/.githooks"

if ! command -v git >/dev/null 2>&1; then
  echo "git command is required" >&2
  exit 1
fi

if ! git -C "$ROOT_DIR" rev-parse --is-inside-work-tree >/dev/null 2>&1; then
  echo "skip git hooks: $ROOT_DIR is not a git work tree"
  exit 0
fi

mkdir -p "$HOOK_DIR"
chmod +x "$HOOK_DIR/pre-commit" "$HOOK_DIR/commit-msg"

git -C "$ROOT_DIR" config core.hooksPath .githooks
echo "installed git hooks: core.hooksPath=.githooks"
