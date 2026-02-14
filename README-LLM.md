# codex-troller (LLM Installation README)

This document is for LLM agents that install and configure `codex-troller`.
Do not use `skill-installer` for this repository. This repository is an MCP server project.

## Source-of-Truth Policy (mandatory)

- Always fetch the latest installation guide from:
  - `https://github.com/Eunho-J/codex-troller/blob/main/README-LLM.md`
- Before any remote/network command, request permission first via the platform approval tool-call.
- Do not run probe commands first (for example, do not try `curl` once and ask permission only after failure).
- If internet access is blocked/unavailable after approval, report the failure and ask whether to retry.
- If network permission is denied, ask the user whether to:
  1) stop installation, or
  2) continue with local fallback (explicitly warn it may be stale).
- Use latest GitHub repository content as installation source (`main.zip` or `git clone --depth 1`).

## Project Overview

`codex-troller` is a local Go MCP server for Codex CLI.
Its purpose is to increase reliability by turning ambiguous user intent into a structured workflow:
- intent capture,
- council planning,
- staged execution,
- verification + user approval,
- persistent resumability.

## Required User Confirmations (must ask before install)

Before running any install command, ask the user and wait for explicit answers.

## Language Policy (mandatory)

- Detect the user's working language from their latest request.
- All user-facing questions, consent prompts, and summaries must be in that language.
- Do not default to English when the user started in another language.
- Keep field keys/enum values in canonical form where needed (`global|local`, `yes|no`, profile keys).
- At install completion, auto-detect the predominant user language used during the install conversation and save it to default profile as `consultant_lang`.
- Current canonical `consultant_lang` values are `ko|en` (map Korean-dominant installs to `ko`, otherwise use `en`).

## Interview Policy (mandatory)

- Ask exactly one confirmation question at a time.
- Wait for the user's answer before asking the next question.
- Do not present the whole checklist in a single message.
- If an answer is ambiguous, ask a short follow-up for that single item.
- Do not proceed to installation until all required items are confirmed.

## Question Style Policy (mandatory)

- Do not ask with raw field names like `overall`, `response_need`, `technical_depth`, `domain_knowledge`.
- Do not ask with enum-only forms like "choose beginner/intermediate/advanced" without context.
- Ask in user-goal language so the user can answer immediately without extra thinking.
- Keep each question short, concrete, and action-oriented.
- After each answer, silently map it to canonical values for installer input.
- Prefer numeric responses for speed: present numbered options and ask user to reply with a number.
- Option labels must be translated into the user's language (do not show English-only options to non-English users).

## Input Format Policy (mandatory)

- For each question, show numbered choices (`1`, `2`, `3`...) in the user's language.
- Ask: "Reply with the number."
- Accept both number and text answers, but normalize to canonical values internally.
- If the user gives text, map it and continue without forcing re-entry.
- For optional free-text fields, provide numbered branching first:
  - `1) none`
  - `2) I will enter details`
- For optional integrations/tools, explain value first in 1-2 short sentences:
  - what it enables,
  - when it is useful.
- Then present numbered options and ask for a number.

## Command Execution Policy (mandatory)

- Execute installation commands yourself, one by one.
- For commands that may require elevated privileges, request execution approval first.
- After approval, run the command and report result briefly.
- If approval is denied, ask whether to skip or retry with a different method.
- For remote fetch commands (`curl`, `wget`, `git clone` to internet), always request approval before first execution.
- Do not request approval for local non-privileged commands (for example local build/test/verify commands that only read/write inside workspace or `<CODEX_HOME>`).
- Exception for Linux OS package dependencies (apt):
  - Do not run `sudo apt*` automatically.
  - Ask the user to run required apt commands directly.
  - Wait for user confirmation, then continue verification.
- Install toolchain/runtime dependencies inside `<CODEX_HOME>/.codex-troller/deps` by default (Go, Node, Playwright assets).
- Do not require persistent global environment variable setup for installation.

## Go Installation Policy (mandatory)

- If Go is missing, install Go locally from official tarball source (`https://go.dev/dl/`) into `<CODEX_HOME>/.codex-troller/deps/go`.
- Never install Go via system package managers in default flow (`apt`, `apt-get`, `yum`, `dnf`, `brew`, `choco`, `snap`, etc.).
- Never ask the user to choose between "root install Go" and "stop".
- Default behavior is: local Go install -> continue build/test/install.
- Ask only for normal command approval when needed, not for root package-manager installation for Go.

Recommended expertise-question style (ask in user's language):

1. `overall` mapping
   - Ask: "Which guidance style do you want for this setup?"
     - `1)` simple guidance + safe defaults
     - `2)` normal technical guidance
     - `3)` detailed technical trade-offs
2. `response_need` mapping
   - Ask: "How often should I check with you during work?"
     - `1)` only when necessary
     - `2)` balanced checkpoints
     - `3)` frequent confirmations
3. `technical_depth` mapping
   - Ask: "How should I explain decisions?"
     - `1)` high-level outcomes
     - `2)` balanced summary + key reason
     - `3)` technical details and implications
4. `domain_knowledge` mapping
   - Before asking, explain briefly in the user's language:
     - Domain hints tell the system where the user has stronger/weaker familiarity.
     - This controls where to ask more detail vs where to move faster with less explanation.
   - Ask: "Do you want to provide domain hints now?"
     - `1)` no hints
     - `2)` yes, I will enter comma-separated hints
   - If `2`, ask for hints in user's language with short examples.
   - Example hint categories (translate for user):
     - `frontend`, `backend`, `db`, `security`, `infra`, `ai_ml`, `asset`, `game`
   - Example input:
     - `frontend,testing`
     - `backend,security,db`
   - Optional level form:
     - `backend=advanced,frontend=beginner`
   - Keep prompt short and practical; avoid internal schema details.

Use the following confirmation content in the user's language (semantic equivalent, not fixed English wording):

1. Terms consent
   - software is not sufficiently validated
   - user assumes responsibility for issues/damages
   - GNU GPL v3.0 license acknowledged
   - Ask with numbered options:
     - `1)` agree to all terms
     - `2)` do not agree
2. Install scope
   - Ask with numbered options:
     - `1)` global (`~/.codex`)
     - `2)` local (`<repo>/.codex`)
3. Optional Playwright MCP
   - Before asking, briefly explain in the user's language:
     - It enables browser automation and UI verification.
     - It is useful for web projects, E2E checks, and visual flow testing.
     - If selected, agent will run Playwright dependency installation commands step by step.
     - On Linux, OS-level dependency installation may require apt with sudo, which the user may need to run directly.
   - Ask with numbered options:
     - `1)` install/register Playwright MCP
     - `2)` skip Playwright MCP
   - Recommendation rule:
     - If task includes web UI, browser interactions, E2E, or visual review -> recommend `1`.
     - Otherwise -> recommend `2`.
   - Keep the explanation short; do not over-explain tooling internals.
4. Initial expertise profile
   - Ask and capture (questions in user's language):
     - `overall` (`beginner|intermediate|advanced`)
     - `response_need` (`low|balanced|high`)
     - `technical_depth` (`abstract|balanced|technical`)
     - `domain_knowledge` (comma-separated optional hints)
   - Auto-capture (do not ask unless ambiguous):
     - `consultant_lang` from predominant language during install conversation (`ko|en`)

If terms are not accepted, stop installation immediately.

Recommended turn-by-turn order:
1. Ask terms consent only, wait for answer.
2. Ask install scope only, wait for answer.
3. Ask Playwright MCP consent only, wait for answer.
4. Ask expertise profile fields one by one, validating each answer.
5. Determine `consultant_lang` automatically from the install conversation language and persist it in profile.

## Installation Procedure (command-by-command)

After confirmations are complete, run commands one by one:

1. Prerequisite checks
```bash
command -v cp
command -v awk
command -v rm
command -v tar
```

2. Choose install scope and paths
```bash
# global scope
CODEX_HOME="$HOME/.codex"
# local scope
# CODEX_HOME="$(pwd)/.codex"

CONFIG_PATH="$CODEX_HOME/config.toml"
STATE_DIR="$CODEX_HOME/.codex-troller"
BIN_DIR="$STATE_DIR/bin"
DEPS_DIR="$STATE_DIR/deps"
MCP_BIN_PATH="$BIN_DIR/codex-mcp"
PROFILE_PATH="$STATE_DIR/default_user_profile.json"
CACHE_DIR="$STATE_DIR/cache"
NPM_CACHE_DIR="$DEPS_DIR/npm-cache"
PLAYWRIGHT_BROWSERS_DIR="$DEPS_DIR/playwright-browsers"
```

3. Create temp fetch directory
```bash
FETCH_DIR="$(mktemp -d)"
```

4. Fetch source (zip preferred, git fallback)
   - First request approval for network commands, then execute.
```bash
if command -v curl >/dev/null 2>&1 && command -v unzip >/dev/null 2>&1; then
  curl -fsSL https://github.com/Eunho-J/codex-troller/archive/refs/heads/main.zip -o "$FETCH_DIR/codex-troller.zip"
  unzip -q "$FETCH_DIR/codex-troller.zip" -d "$FETCH_DIR"
  SRC_DIR="$FETCH_DIR/codex-troller-main"
elif command -v wget >/dev/null 2>&1 && command -v unzip >/dev/null 2>&1; then
  wget -qO "$FETCH_DIR/codex-troller.zip" https://github.com/Eunho-J/codex-troller/archive/refs/heads/main.zip
  unzip -q "$FETCH_DIR/codex-troller.zip" -d "$FETCH_DIR"
  SRC_DIR="$FETCH_DIR/codex-troller-main"
else
  command -v git
  git clone --depth 1 https://github.com/Eunho-J/codex-troller.git "$FETCH_DIR/codex-troller-main"
  SRC_DIR="$FETCH_DIR/codex-troller-main"
fi
```

5. Prepare target directories
```bash
mkdir -p "$STATE_DIR" "$BIN_DIR" "$DEPS_DIR" "$CACHE_DIR" "$NPM_CACHE_DIR" "$PLAYWRIGHT_BROWSERS_DIR" "$CODEX_HOME/skills" "$(dirname "$CONFIG_PATH")"
touch "$CONFIG_PATH"
```

6. Install/reuse local Go toolchain inside state directory
```bash
GO_VERSION="$(awk '/^go /{print $2; exit}' "$SRC_DIR/go.mod")"
GO_ROOT="$DEPS_DIR/go"
GO_BIN="$GO_ROOT/bin/go"

# Do NOT use apt/brew/choco for Go in this flow.

if [ ! -x "$GO_BIN" ]; then
  OS="$(uname -s | tr '[:upper:]' '[:lower:]')"
  ARCH="$(uname -m)"
  case "$ARCH" in
    x86_64) ARCH="amd64" ;;
    arm64|aarch64) ARCH="arm64" ;;
    *) echo "unsupported architecture: $ARCH" >&2; exit 1 ;;
  esac

  if command -v curl >/dev/null 2>&1; then
    curl -fsSL "https://go.dev/dl/go${GO_VERSION}.${OS}-${ARCH}.tar.gz" -o "$FETCH_DIR/go.tar.gz"
  elif command -v wget >/dev/null 2>&1; then
    wget -qO "$FETCH_DIR/go.tar.gz" "https://go.dev/dl/go${GO_VERSION}.${OS}-${ARCH}.tar.gz"
  else
    echo "curl or wget is required to install local Go toolchain" >&2
    exit 1
  fi

  rm -rf "$GO_ROOT"
  tar -C "$DEPS_DIR" -xzf "$FETCH_DIR/go.tar.gz"
fi

"$GO_BIN" version
```

7. Install/reuse local Node.js toolchain inside state directory
```bash
NODE_VERSION="${NODE_VERSION:-22.13.1}"
NODE_ROOT="$DEPS_DIR/node"
NODE_BIN_DIR="$NODE_ROOT/bin"
NODE_BIN="$NODE_BIN_DIR/node"
NPM_BIN="$NODE_BIN_DIR/npm"
NPX_BIN="$NODE_BIN_DIR/npx"

if [ ! -x "$NPX_BIN" ]; then
  OS="$(uname -s | tr '[:upper:]' '[:lower:]')"
  ARCH="$(uname -m)"
  case "$ARCH" in
    x86_64) ARCH="x64" ;;
    arm64|aarch64) ARCH="arm64" ;;
    *) echo "unsupported architecture: $ARCH" >&2; exit 1 ;;
  esac
  case "$OS" in
    linux|darwin) ;;
    *) echo "unsupported OS for local Node bootstrap: $OS" >&2; exit 1 ;;
  esac

  if command -v curl >/dev/null 2>&1; then
    curl -fsSL "https://nodejs.org/dist/v${NODE_VERSION}/node-v${NODE_VERSION}-${OS}-${ARCH}.tar.gz" -o "$FETCH_DIR/node.tar.gz"
  elif command -v wget >/dev/null 2>&1; then
    wget -qO "$FETCH_DIR/node.tar.gz" "https://nodejs.org/dist/v${NODE_VERSION}/node-v${NODE_VERSION}-${OS}-${ARCH}.tar.gz"
  else
    echo "curl or wget is required to install local Node.js toolchain" >&2
    exit 1
  fi

  rm -rf "$NODE_ROOT"
  mkdir -p "$NODE_ROOT"
  tar -C "$NODE_ROOT" --strip-components=1 -xzf "$FETCH_DIR/node.tar.gz"
fi

PATH="$NODE_BIN_DIR:$PATH" "$NODE_BIN" --version
PATH="$NODE_BIN_DIR:$PATH" "$NPM_BIN" --version
PATH="$NODE_BIN_DIR:$PATH" "$NPX_BIN" --version
```

8. Build binary + run tests from fetched source (using local Go)
```bash
(cd "$SRC_DIR" && "$GO_BIN" build -o "$MCP_BIN_PATH" ./cmd/codex-mcp)
(cd "$SRC_DIR" && "$GO_BIN" test ./...)
```

9. Install skill files from fetched source
```bash
rm -rf "$CODEX_HOME/skills/codex-troller-autostart"
cp -R "$SRC_DIR/skills/codex-troller-autostart" "$CODEX_HOME/skills/codex-troller-autostart"
```

10. Write profile from interview answers
```bash
# Replace values below with captured interview answers.
PROFILE_OVERALL="intermediate"
PROFILE_RESPONSE_NEED="balanced"
PROFILE_TECHNICAL_DEPTH="balanced"
PROFILE_CONSULTANT_LANG="en"
DOMAIN_KNOWLEDGE_JSON='{}'

cat > "$PROFILE_PATH" <<EOF
{
  "overall": "$PROFILE_OVERALL",
  "response_need": "$PROFILE_RESPONSE_NEED",
  "technical_depth": "$PROFILE_TECHNICAL_DEPTH",
  "consultant_lang": "$PROFILE_CONSULTANT_LANG",
  "domain_knowledge": $DOMAIN_KNOWLEDGE_JSON
}
EOF
```

11. Register MCP server in config (idempotent section replace)
```bash
awk -v sec='[mcp_servers.codex-troller]' 'BEGIN{skip=0} $0==sec{skip=1;next} skip&&$0~/^\[/{skip=0} !skip{print}' "$CONFIG_PATH" > "$CONFIG_PATH.tmp" && mv "$CONFIG_PATH.tmp" "$CONFIG_PATH"
printf '\n[mcp_servers.codex-troller]\ncommand = "%s"\n' "$MCP_BIN_PATH" >> "$CONFIG_PATH"
```

12. If Playwright selected: register MCP + install dependencies command-by-command
```bash
awk -v sec='[mcp_servers.playwright]' 'BEGIN{skip=0} $0==sec{skip=1;next} skip&&$0~/^\[/{skip=0} !skip{print}' "$CONFIG_PATH" > "$CONFIG_PATH.tmp" && mv "$CONFIG_PATH.tmp" "$CONFIG_PATH"
cat > "$BIN_DIR/playwright-mcp-launch" <<EOF
#!/usr/bin/env bash
set -euo pipefail
export npm_config_cache="$NPM_CACHE_DIR"
export XDG_CACHE_HOME="$CACHE_DIR"
export PLAYWRIGHT_BROWSERS_PATH="$PLAYWRIGHT_BROWSERS_DIR"
exec "$NPX_BIN" -y @playwright/mcp@latest
EOF
chmod +x "$BIN_DIR/playwright-mcp-launch"
printf '\n[mcp_servers.playwright]\ncommand = "%s"\n' "$BIN_DIR/playwright-mcp-launch" >> "$CONFIG_PATH"
```
- Install Playwright browser binaries first:
```bash
PATH="$NODE_BIN_DIR:$PATH" npm_config_cache="$NPM_CACHE_DIR" XDG_CACHE_HOME="$CACHE_DIR" PLAYWRIGHT_BROWSERS_PATH="$PLAYWRIGHT_BROWSERS_DIR" "$NPX_BIN" -y playwright@latest install chromium firefox webkit
```
- Optional diagnostics (recommended when the user wants immediate browser-run validation in this environment):
```bash
PATH="$NODE_BIN_DIR:$PATH" npm_config_cache="$NPM_CACHE_DIR" XDG_CACHE_HOME="$CACHE_DIR" PLAYWRIGHT_BROWSERS_PATH="$PLAYWRIGHT_BROWSERS_DIR" "$NPX_BIN" -y -p playwright node -e "const { chromium } = require('playwright'); (async()=>{ const b=await chromium.launch({headless:true, args:['--no-sandbox','--disable-setuid-sandbox']}); await b.close(); })();"
```
- If that fails due module-resolution issues in this environment, use deterministic local verify fallback:
```bash
PLAYWRIGHT_VERIFY_DIR="$STATE_DIR/playwright-verify"
mkdir -p "$PLAYWRIGHT_VERIFY_DIR"
cd "$PLAYWRIGHT_VERIFY_DIR"
if [ ! -f package.json ]; then
  PATH="$NODE_BIN_DIR:$PATH" npm_config_cache="$NPM_CACHE_DIR" "$NPM_BIN" init -y
fi
PATH="$NODE_BIN_DIR:$PATH" npm_config_cache="$NPM_CACHE_DIR" XDG_CACHE_HOME="$CACHE_DIR" PLAYWRIGHT_BROWSERS_PATH="$PLAYWRIGHT_BROWSERS_DIR" "$NPM_BIN" install --no-audit --no-fund --save-dev playwright@latest
PATH="$NODE_BIN_DIR:$PATH" XDG_CACHE_HOME="$CACHE_DIR" PLAYWRIGHT_BROWSERS_PATH="$PLAYWRIGHT_BROWSERS_DIR" "$NODE_BIN" -e "const { chromium } = require('playwright'); (async()=>{ const b=await chromium.launch({headless:true, args:['--no-sandbox','--disable-setuid-sandbox']}); await b.close(); })();"
```
- If diagnostics report missing Linux shared libraries, ask user to run apt commands directly (do not run `sudo apt*` automatically):
```bash
sudo apt-get update
sudo env "PATH=$NODE_BIN_DIR:$PATH" "$NPX_BIN" -y playwright@latest install-deps chromium firefox webkit
```
- After user confirms apt step is done, rerun diagnostics.
- If diagnostics still fail, continue installation with a warning note:
  - registration/config is complete,
  - runtime browser launch may still require host environment fixes.

13. Write consent log
```bash
PLAYWRIGHT_MCP_VALUE="yes"   # or "no"
cat > "$STATE_DIR/install-consent.log" <<EOF
timestamp: $(date -u +"%Y-%m-%dT%H:%M:%SZ")
terms_version: 2026-02-14
terms_assertions:
  - unverified_software=true
  - user_assumes_responsibility=true
  - license_gnu_gpl_v3_ack=true
scope: $( [ "$CODEX_HOME" = "$HOME/.codex" ] && echo global || echo local )
terms_accepted: yes
playwright_mcp: $PLAYWRIGHT_MCP_VALUE
profile_path: $PROFILE_PATH
EOF
```

14. Verify installation (no shell wrapper)
```bash
test -x "$GO_BIN"
test -x "$NODE_BIN"
test -x "$NPM_BIN"
test -x "$NPX_BIN"
test -x "$MCP_BIN_PATH"
test -f "$CODEX_HOME/skills/codex-troller-autostart/SKILL.md"
echo '{"jsonrpc":"2.0","id":1,"method":"initialize"}' | "$MCP_BIN_PATH"
echo '{"jsonrpc":"2.0","id":2,"method":"tools/list"}' | "$MCP_BIN_PATH"
MCP_LIST_OUTPUT="$(CODEX_HOME="$CODEX_HOME" codex mcp list)"
echo "$MCP_LIST_OUTPUT" | grep -Eq '^codex-troller[[:space:]]'
if [ "$PLAYWRIGHT_MCP_VALUE" = "yes" ]; then
  echo "$MCP_LIST_OUTPUT" | grep -Eq '^playwright[[:space:]]'
fi
```

15. Clean fetched files/folders
```bash
rm -rf "$FETCH_DIR"
```

## Verification Checklist

- `<CODEX_HOME>/.codex-troller/deps/go/bin/go` exists
- `<CODEX_HOME>/.codex-troller/deps/node/bin/npx` exists
- `<CODEX_HOME>/.codex-troller/default_user_profile.json` includes `consultant_lang`
- If Playwright selected: `<CODEX_HOME>/.codex-troller/deps/playwright-browsers` has browser files
- `<CODEX_HOME>/.codex-troller/bin/codex-mcp` exists
- Codex config has `[mcp_servers.codex-troller]`
- If requested, config also has `[mcp_servers.playwright]`
- Skill exists at `<CODEX_HOME>/skills/codex-troller-autostart/SKILL.md`
- `initialize` and `tools/list` JSON-RPC calls succeed via `<CODEX_HOME>/.codex-troller/bin/codex-mcp`
- `CODEX_HOME="<target>" codex mcp list` includes `codex-troller` (and `playwright` when selected)

## Failure Handling

If any step fails:
- explain the failure in plain language,
- fix and retry automatically,
- ask user only when a permission/risk decision is required.
- for Linux apt OS dependencies, explicitly ask the user to run the required apt commands and confirm completion before retry.
- if Playwright optional diagnostics fail but MCP registration/integration checks pass, finish install with warning instead of marking overall install failed.

## Completion Message Policy (mandatory)

- Final completion message must be user-friendly first, operator-friendly second.
- Use the user's language.
- First line must state outcome in plain words (installed/not installed).
- Next line must give exactly one immediate action.
  - Local install: "Restart Codex in this folder."
  - Global install: "Restart Codex. It will be available from any folder."
- Do not dump full paths/config blocks first.
- Do not lead with MCP section names, binary paths, or profile JSON values.
- Do not show `CODEX_HOME=...` guidance in default completion messaging.
- Do not suggest cleanup/deletion of old directories unless the user asks.
- Provide technical details only as an optional follow-up:
  - "If you want, I can show detailed paths/logs."

Recommended completion template:
1. Status: install + verification result (plain language).
2. Immediate next action (one sentence).
3. Optional: brief "what was configured" summary without deep internals.
4. Optional offer for detailed technical report on request.
