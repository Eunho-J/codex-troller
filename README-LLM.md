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

## Installation Procedure

After confirmations are complete:

1. Clone repository
```bash
git clone https://github.com/Eunho-J/codex-troller.git
cd codex-troller
```

2. Run installer (interactive, recommended)
```bash
make agent-install
```

3. Run setup/verification
```bash
make setup
make smoke
```

4. If Playwright was selected, run dependency setup command-by-command
   - Linux (ask approval first for elevated execution):
```bash
npx -y playwright@latest install --with-deps chromium firefox webkit
```
   - If Linux deps install fails or approval is denied, run browser-only install:
```bash
npx -y playwright@latest install chromium firefox webkit
```
   - Verify Playwright runtime:
```bash
npx -y -p playwright node -e "const { chromium } = require('playwright'); (async()=>{ const b=await chromium.launch({headless:true}); await b.close(); })();"
```
   - If verification fails:
     1) ask user whether to retry dependency install or skip Playwright,
     2) execute chosen action,
     3) verify again,
     4) repeat until success or explicit skip decision.

## Non-Interactive Alternative

Use only when the user explicitly asks for non-interactive mode and all answers are already confirmed.

```bash
AGENT_INSTALL_NON_INTERACTIVE=1 \
INSTALL_TERMS_AGREED=yes \
INSTALL_SCOPE=global \
INSTALL_PLAYWRIGHT_MCP=no \
INSTALL_PROFILE_OVERALL=intermediate \
INSTALL_PROFILE_RESPONSE_NEED=balanced \
INSTALL_PROFILE_TECHNICAL_DEPTH=balanced \
INSTALL_PROFILE_DOMAIN_HINTS="backend,security" \
make agent-install
```

## Verification Checklist

- `.codex-mcp/bin/codex-mcp` exists
- `.codex-mcp/bin/codex-troller-launch` exists
- Codex config has `[mcp_servers.codex-troller]`
- If requested, config also has `[mcp_servers.playwright]`
- Skill exists at `<CODEX_HOME>/skills/codex-troller-autostart/SKILL.md`
- `make smoke` passes

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
1. Status: install + setup + smoke result (plain language).
2. Immediate next action (one sentence).
3. Optional: brief "what was configured" summary without deep internals.
4. Optional offer for detailed technical report on request.
