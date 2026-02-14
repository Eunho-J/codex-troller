# Agent Install Guide

This document defines the installation flow for a "just install it for me" request.

## Goals

- Require explicit consent before modifying config or installing integrations.
- Keep setup idempotent and repeatable.
- Capture initial user expertise profile at install time and reuse it as default.

## Interactive flow (`make agent-install`)

`scripts/install-agent.sh` runs an interview-style installer with these gates:

1. LLM consent gate
   - User must explicitly confirm all three terms before any build/config change:
     - software is not sufficiently validated
     - user assumes responsibility for issues/damages
     - license acknowledged as GNU GPL v3.0
   - If not accepted, installation stops immediately.
2. Install scope selection
   - `global` or `local`.
   - `global`: writes under `~/.codex/...`
   - `local`: writes under `<repo>/.codex/...`
3. Optional Playwright MCP integration
   - Ask whether to register Playwright MCP (`mcp_servers.playwright`).
   - Source: `https://github.com/microsoft/playwright-mcp`
   - If accepted, config appends:
     - `command = "npx"`
     - `args = ["-y", "@playwright/mcp@latest"]`
4. Short expertise survey
   - overall: `beginner|intermediate|advanced`
   - response_need: `low|balanced|high`
   - technical_depth: `abstract|balanced|technical`
   - optional domain hints: comma list (`backend,frontend,security`, ...)
5. Setup execution
   - runs `make setup` (build/test/smoke/hooks)
6. Config + skill install
   - registers `mcp_servers.codex-troller`
   - installs `skills/codex-troller-autostart`
7. Default profile persistence
   - writes JSON profile to `<CODEX_HOME>/codex-troller/default_user_profile.json`
   - launcher exports `CODEX_TROLLER_DEFAULT_PROFILE_PATH`

## Automation mode (no prompt)

Use:

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

## Idempotency rules

- Existing `[mcp_servers.codex-troller]` and `[mcp_servers.playwright]` sections are removed then re-written.
- Skill directory is replaced with the latest copy.
- Launcher script is overwritten with current profile path.

## Verification checklist

- Binary exists: `.codex-mcp/bin/codex-mcp`
- Launcher exists: `.codex-mcp/bin/codex-troller-launch`
- Config has `[mcp_servers.codex-troller]`
- Optional config has `[mcp_servers.playwright]` (if accepted)
- Skill exists at `<CODEX_HOME>/skills/codex-troller-autostart/SKILL.md`
- `make smoke` passes
