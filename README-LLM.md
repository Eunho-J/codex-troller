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
   - Ask: "Do you want to provide domain hints now?"
     - `1)` no hints
     - `2)` yes, I will enter comma-separated hints
   - If `2`, ask for hints in user's language with short examples.

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
   - Ask with numbered options:
     - `1)` install/register Playwright MCP
     - `2)` skip Playwright MCP
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
