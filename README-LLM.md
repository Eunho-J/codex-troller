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

Use the following confirmation content in the user's language (semantic equivalent, not fixed English wording):

1. Terms consent
   - software is not sufficiently validated
   - user assumes responsibility for issues/damages
   - GNU GPL v3.0 license acknowledged
   - Ask for explicit agreement to all 3 terms (`yes/no`)
2. Install scope
   - Ask install scope: `global` (`~/.codex`) or `local` (`<repo>/.codex`)
3. Optional Playwright MCP
   - Ask whether to register Playwright MCP (`@playwright/mcp`) (`yes/no`)
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
