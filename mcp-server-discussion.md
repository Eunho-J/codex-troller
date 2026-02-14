# Codex MCP Server Discussion

English is the canonical language for active design notes.
Translated archives are preserved at:
- `mcp-server-discussion.ko.md`
- `mcp-server-discussion.ja.md`
- `mcp-server-discussion.zh.md`

## Operating rule

- This is a living design log, not a fixed final spec.
- Update this file immediately when decisions are made.
- Do not batch-update after long sessions; persist incremental decisions.

## Problem statement

- Users often start with ambiguous goals.
- Direct execution without structured clarification causes mismatch and rework.
- Reliability drops when intent, plan, and execution are weakly connected.

## Product goal

Build a local Go MCP server that improves Codex reliability by enforcing:
- structured intent capture,
- staged planning/execution/verification,
- explicit user approval gates,
- persistent resumability.

## Core principles

- Intent alignment over output speed.
- Small commitments with fast feedback.
- Human control at sensitive boundaries.
- Reproducible workflow with traceable state.
- Recoverable failure loops.
- Least-privilege execution defaults.

## Current architecture summary

- Workflow state machine is enforced.
- Session persistence uses JSON + SQLite.
- Council-based planning is mandatory before `generate_plan`.
- Consultant loop refines requirements with one focused question per turn.
- Visual review gate is conditionally required for UI/UX tasks.
- Final completion requires explicit user approval.

## Dynamic council team

- Council managers are session-scoped (`council_managers`).
- Team can be configured by:
  - `council_configure_team` (`append|replace|remove`)
  - one-shot update via `council_start_briefing` (`manager_mode`, `managers`)
- Topic participation/closure checks use active session managers, not a hardcoded role list.

## Installer policy

`make agent-install` includes mandatory consent/interview gates:

1. Terms consent (required)
   - software is not sufficiently validated
   - user assumes responsibility for issues/damages
   - GNU GPL v3.0 acknowledged
2. Install scope selection (`global` or `local`)
3. Optional Playwright MCP registration
4. Initial expertise survey profile capture

The installer writes a default user profile and runtime launcher wiring.

## Model routing baseline

- Client interview: `gpt-5.2`
- Orchestrator/reviewer: `gpt-5.3-codex`
- Worker: `gpt-5.3-codex-spark`

## Validation baseline

- `make test` must pass.
- `make smoke` must pass.
- Installer changes must support both interactive and non-interactive paths.

## Next updates

- Keep this file English-only.
- If Korean documentation is needed, update `*.ko.md` counterparts in parallel.
