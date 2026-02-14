---
name: troller-router
description: Check how user work requests should be handled first, then route to codex-troller workflow only when mode is on
argument-hint: "[user request]"
---

# troller-router

Purpose: keep codex-troller behavior switchable (`on/off`) without forcing every request into the workflow.

## Trigger

- Any user request that asks the assistant to do work.
- Explicit mode command such as `$troller`, `$troller-off`, or `$troller-router`.

## Required Behavior

1. Call `autostart_get_mode` before deciding execution style.
2. If user explicitly requests off/disable/stop:
   - call `autostart_set_mode` with `mode="off"` and `reason`.
   - this path should also be used when user invokes `$troller-off`.
   - continue in default Codex behavior (no codex-troller workflow tools).
3. If user explicitly requests on/enable/start interview:
   - call `autostart_set_mode` with `mode="on"` and `reason`.
   - this path should also be used when user invokes `$troller`.
   - if no active session exists, call `start_interview` with user request as `raw_intent`.
4. If current mode is `on`:
   - if `active_session_id` is empty, call `start_interview`.
   - if `active_session_id` exists, call `get_session_status` and follow `status.next`.
   - then execute the workflow rules in `skills/troller/SKILL.md`.
5. If current mode is `off` and user did not request enable:
   - do not call codex-troller workflow tools.
   - handle request with normal Codex behavior.

## Operating Rules

- Do not silently flip `off -> on` unless the user asked to start codex-troller workflow.
- Keep state checks cheap: one `autostart_get_mode` preflight per user turn.
- If `on` and session state is failed, follow recovery path (`continue_persistent_execution` / reconcile) from status guidance.
