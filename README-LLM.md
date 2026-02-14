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

Use this exact confirmation block:

1. Terms consent
   - "This software is not sufficiently validated."
   - "You assume responsibility for issues/damages."
   - "You acknowledge the GNU GPL v3.0 license."
   - Ask: "Do you agree to all three terms? (yes/no)"
2. Install scope
   - Ask: "Install scope: global (`~/.codex`) or local (`<repo>/.codex`)?"
3. Optional Playwright MCP
   - Ask: "Do you want to register Playwright MCP (`@playwright/mcp`)? (yes/no)"
4. Initial expertise profile
   - Ask and capture:
     - `overall` (`beginner|intermediate|advanced`)
     - `response_need` (`low|balanced|high`)
     - `technical_depth` (`abstract|balanced|technical`)
     - `domain_knowledge` (comma-separated optional hints)

If terms are not accepted, stop installation immediately.

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
