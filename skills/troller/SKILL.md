---
name: troller
description: Start codex-troller with interview-first workflow automatically (troller-on)
argument-hint: "[task summary]"
---

# troller

Purpose: start a full task lifecycle with interview-first behavior by calling the `codex-troller` MCP tools directly, without asking the user to type workflow commands.

## Trigger

- User mentions `$troller` or asks to "start with interview" using codex-troller.

## Required Behavior

1. Call `autostart_set_mode` with `mode="on"` first.
2. Immediately call `start_interview`.
   - If user background is known from prior conversation, pass it as `user_profile` to adapt depth/autonomy.
3. Ask interview questions one by one from `interview_questions`.
   - Questions must be situational and concrete, never abstract policy prompts.
   - Ask like a client meeting for non-technical users (e.g., "when should info panel appear?" not "define approval policy").
4. Convert user answers into structured intent fields:
   - `goal: ...`
   - `scope: ...`
   - `constraints: ...`
   - `success_criteria: ...`
5. Call workflow tools in order:
   - Before each tool call after `start_interview`, call `get_session_status` and check `next`.
   - If your intended tool does not match `next`, do not force the call. Follow `next` first.
   - `ingest_intent`
   - Optional: `council_configure_team` when project needs role composition changes
   - `council_start_briefing`
   - `council_submit_brief` (all roles submit independently)
   - `council_summarize_briefs`
   - For each open topic:
     - `council_request_floor` (must include `topic_id`)
     - `council_grant_floor`
     - `council_publish_statement`
     - `council_respond_topic` (collect `pass/raise` from all roles)
     - `council_close_topic` (only when all roles are `pass`)
   - `council_finalize_consensus`
   - `clarify_intent` (repeat until `status=clarified`)
     - gather concrete constraints/success criteria
     - during proposal alignment, ask one question at a time and pass natural-language feedback in `answers.proposal_feedback`
     - use `answers.proposal_decision` only when the user explicitly states accept/refine/alternative
     - if user feedback suggests conflicting needs/tradeoffs, route back to `council_start_briefing` instead of resolving it in counselor voice
   - `generate_plan`
   - `generate_mockup`
   - discuss mockup with user
   - `approve_plan`
   - `run_action`
     - Must include worker execution metadata:
       - `executor_role`: worker-only role (e.g., `backend_worker`, `frontend_worker`, `implementation_worker`)
       - `executor_model`: must match `routing_policy.worker_model`
       - `delegated_by`: manager/consultant role that delegated the task
   - `verify_result`
   - if `verify_result.next_step` is `visual_review`, run `visual_review`:
     - provide render artifacts/findings
     - include `ux_director_summary` and `ux_decision`
   - `record_user_feedback`
   - `summarize`
6. Never run `run_action` before plan approval.
7. Consultant or manager roles must never implement directly.
   - Consultant/team leads produce intent/plan/review decisions only.
   - Actual code/command execution must be delegated to worker agents.
8. If approval/risk/permission ambiguity appears, pause execution and continue interview.
9. Keep looping until explicit user approval:
   - verification fail or `approved=false` -> continue from `generate_plan`
   - only treat done when `record_user_feedback(approved=true)` and step is `summarized`

## Operating Rules

- Keep user burden low: ask only what is missing.
- Use user familiarity signals to tune depth:
  - unfamiliar domain -> abstract phrasing + higher team autonomy
  - familiar domain -> more concrete technical tradeoff phrasing
- Respect profile confidence:
  - if confidence is low, avoid strong assumptions and keep decisions conservative until more evidence is gathered.
- If rendering-capable MCPs are known (e.g., playwright), pass them as `available_mcps`/`available_mcp_tools` so visual review can be auto-activated.
- Do not use a fixed questionnaire template. Decide whether to ask more based on current ambiguity/risk/authority boundaries.
- Never force menu-style choices (`A/B/C`, `accept/refine/alternative`) as the default UX; start with a concrete draft and one focused follow-up question.
- Preserve traceability: keep requirement tags and success criteria explicit.
- Use `get_session_status` to report progress when needed.
- Use `get_session_status` as a hard preflight gate before each workflow tool call.
- Treat `status.next` as authoritative for sequencing; never assume stage from memory.
- Use `council_get_status` to keep discussion state synchronized.
- If session is `failed` and retry budget remains, call `continue_persistent_execution` and resume the loop.
- On resume, call `reconcile_session_state(mode=\"check\")` first. If drift is high, ask user to choose `keep_context` or `restart_context`.
