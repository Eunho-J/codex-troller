---
name: troller-off
description: Turn off codex-troller workflow mode and return to default Codex behavior
argument-hint: "[optional reason]"
---

# troller-off

Purpose: stop codex-troller workflow mode cleanly.

## Trigger

- User mentions `$troller-off`.
- User explicitly asks to disable/stop troller workflow.

## Required Behavior

1. Call `autostart_set_mode` with:
   - `mode="off"`
   - `reason`: short natural-language reason from user request
2. Confirm mode is off by calling `autostart_get_mode`.
3. Report completion briefly and continue with normal Codex behavior.

## Operating Rules

- Do not call `start_interview` in this skill.
- Do not continue codex-troller workflow tools after off is confirmed.
