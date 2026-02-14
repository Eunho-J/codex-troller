# codex-troller (LLM Installation README)

This document is for LLM agents that install and configure `codex-troller`.
Do not use `skill-installer` for this repository. This repository is an MCP server project.

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

- Do not ask the user to manually execute installation commands.
- Execute installation commands yourself, one by one.
- For commands that may require elevated privileges, request execution approval first.
- After approval, run the command and report result briefly.
- If approval is denied, ask whether to skip or retry with a different method.

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
     - On Linux, OS-level dependency installation may require elevated privileges.
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

If terms are not accepted, stop installation immediately.

Recommended turn-by-turn order:
1. Ask terms consent only, wait for answer.
2. Ask install scope only, wait for answer.
3. Ask Playwright MCP consent only, wait for answer.
4. Ask expertise profile fields one by one, validating each answer.

## Installation Procedure (command-by-command)

After confirmations are complete, run commands one by one:

1. Prerequisite checks
```bash
command -v go
command -v cp
command -v awk
command -v rm
```

2. Choose install scope and paths
```bash
# global scope
export CODEX_HOME="$HOME/.codex"
# local scope
# export CODEX_HOME="$(pwd)/.codex"

export CONFIG_PATH="$CODEX_HOME/config.toml"
export STATE_DIR="$CODEX_HOME/.codex-troller"
export BIN_DIR="$STATE_DIR/bin"
export MCP_BIN_PATH="$BIN_DIR/codex-mcp"
export PROFILE_PATH="$STATE_DIR/default_user_profile.json"
export LAUNCHER_PATH="$BIN_DIR/codex-troller-launch"
```

3. Create temp fetch directory
```bash
export FETCH_DIR="$(mktemp -d)"
```

4. Fetch source (zip preferred, git fallback)
```bash
if command -v curl >/dev/null 2>&1 && command -v unzip >/dev/null 2>&1; then
  curl -fsSL https://github.com/Eunho-J/codex-troller/archive/refs/heads/main.zip -o "$FETCH_DIR/codex-troller.zip"
  unzip -q "$FETCH_DIR/codex-troller.zip" -d "$FETCH_DIR"
  export SRC_DIR="$FETCH_DIR/codex-troller-main"
elif command -v wget >/dev/null 2>&1 && command -v unzip >/dev/null 2>&1; then
  wget -qO "$FETCH_DIR/codex-troller.zip" https://github.com/Eunho-J/codex-troller/archive/refs/heads/main.zip
  unzip -q "$FETCH_DIR/codex-troller.zip" -d "$FETCH_DIR"
  export SRC_DIR="$FETCH_DIR/codex-troller-main"
else
  command -v git
  git clone --depth 1 https://github.com/Eunho-J/codex-troller.git "$FETCH_DIR/codex-troller-main"
  export SRC_DIR="$FETCH_DIR/codex-troller-main"
fi
```

5. Prepare target directories
```bash
mkdir -p "$STATE_DIR" "$BIN_DIR" "$CODEX_HOME/skills" "$(dirname "$CONFIG_PATH")"
touch "$CONFIG_PATH"
```

6. Build binary + run tests from fetched source
```bash
(cd "$SRC_DIR" && go build -o "$MCP_BIN_PATH" ./cmd/codex-mcp)
(cd "$SRC_DIR" && go test ./...)
```

7. Install skill files from fetched source
```bash
rm -rf "$CODEX_HOME/skills/codex-troller-autostart"
cp -R "$SRC_DIR/skills/codex-troller-autostart" "$CODEX_HOME/skills/codex-troller-autostart"
```

8. Write profile from interview answers
```bash
# Replace values below with captured interview answers.
export PROFILE_OVERALL="intermediate"
export PROFILE_RESPONSE_NEED="balanced"
export PROFILE_TECHNICAL_DEPTH="balanced"
export DOMAIN_KNOWLEDGE_JSON='{}'

cat > "$PROFILE_PATH" <<EOF
{
  "overall": "$PROFILE_OVERALL",
  "response_need": "$PROFILE_RESPONSE_NEED",
  "technical_depth": "$PROFILE_TECHNICAL_DEPTH",
  "domain_knowledge": $DOMAIN_KNOWLEDGE_JSON
}
EOF
```

9. Write launcher
```bash
cat > "$LAUNCHER_PATH" <<EOF
#!/usr/bin/env bash
set -euo pipefail
export CODEX_TROLLER_DEFAULT_PROFILE_PATH="$PROFILE_PATH"
exec "$MCP_BIN_PATH"
EOF
chmod +x "$LAUNCHER_PATH"
```

10. Register MCP server in config (idempotent section replace)
```bash
awk -v sec='[mcp_servers.codex-troller]' 'BEGIN{skip=0} $0==sec{skip=1;next} skip&&$0~/^\[/{skip=0} !skip{print}' "$CONFIG_PATH" > "$CONFIG_PATH.tmp" && mv "$CONFIG_PATH.tmp" "$CONFIG_PATH"
printf '\n[mcp_servers.codex-troller]\ncommand = "%s"\n' "$LAUNCHER_PATH" >> "$CONFIG_PATH"
```

11. If Playwright selected: register MCP + install dependencies command-by-command
```bash
command -v npx
awk -v sec='[mcp_servers.playwright]' 'BEGIN{skip=0} $0==sec{skip=1;next} skip&&$0~/^\[/{skip=0} !skip{print}' "$CONFIG_PATH" > "$CONFIG_PATH.tmp" && mv "$CONFIG_PATH.tmp" "$CONFIG_PATH"
printf '\n[mcp_servers.playwright]\ncommand = "npx"\nargs = ["-y", "@playwright/mcp@latest"]\n' >> "$CONFIG_PATH"
```
- Linux: request permission, then run:
```bash
npx -y playwright@latest install --with-deps chromium firefox webkit
```
- If denied or failed, run:
```bash
npx -y playwright@latest install chromium firefox webkit
```
- Verify runtime:
```bash
npx -y -p playwright node -e "const { chromium } = require('playwright'); (async()=>{ const b=await chromium.launch({headless:true}); await b.close(); })();"
```
- If verify fails, ask user:
  1) retry dependency install, or
  2) skip Playwright.
  Then execute choice and re-verify. Repeat until success or explicit skip.

12. Write consent log
```bash
export PLAYWRIGHT_MCP_VALUE="yes"   # or "no"
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

13. Verify installation (no shell wrapper)
```bash
test -x "$MCP_BIN_PATH"
test -x "$LAUNCHER_PATH"
test -f "$CODEX_HOME/skills/codex-troller-autostart/SKILL.md"
echo '{"jsonrpc":"2.0","id":1,"method":"initialize"}' | "$MCP_BIN_PATH"
echo '{"jsonrpc":"2.0","id":2,"method":"tools/list"}' | "$MCP_BIN_PATH"
```

14. Clean fetched files/folders
```bash
rm -rf "$FETCH_DIR"
```

## Verification Checklist

- `<CODEX_HOME>/.codex-troller/bin/codex-mcp` exists
- `<CODEX_HOME>/.codex-troller/bin/codex-troller-launch` exists
- Codex config has `[mcp_servers.codex-troller]`
- If requested, config also has `[mcp_servers.playwright]`
- Skill exists at `<CODEX_HOME>/skills/codex-troller-autostart/SKILL.md`
- `initialize` and `tools/list` JSON-RPC calls succeed via `<CODEX_HOME>/.codex-troller/bin/codex-mcp`

## Failure Handling

If any step fails:
- explain the failure in plain language,
- fix and retry automatically,
- ask user only when a permission/risk decision is required.
- do not redirect installation work to the user unless they explicitly choose manual mode.

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
