# codex-troller MCP 로컬 서버

이 문서는 한국어 가이드입니다.
아래에 `README.md`와 동기화된 전체 본문(영문)을 포함합니다.

---

# codex-troller MCP Local Server

Guides:
- [English Guide](README.md)
- [한국어 가이드](README.ko.md)
- [日本語ガイド](README.ja.md)
- [中文指南](README.zh.md)

Design notes:
- [English Design Notes](mcp-server-discussion.md)
- [한국어 설계 노트](mcp-server-discussion.ko.md)
- [日本語設計ノート](mcp-server-discussion.ja.md)
- [中文设计说明](mcp-server-discussion.zh.md)

## Install (MCP Server, Not Skill)

This repository is an MCP server project, **not** a Codex skill package.
Do **not** use `skill-installer` for this repository.

Copy/paste install command:

```bash
git clone https://github.com/Eunho-J/codex-troller.git
cd codex-troller
AGENT_INSTALL_NON_INTERACTIVE=1 INSTALL_TERMS_AGREED=yes INSTALL_SCOPE=global INSTALL_PLAYWRIGHT_MCP=no make agent-install
```

`codex-troller` is a local Go MCP server for Codex CLI.
Its purpose is to structure work for higher reliability and better intent alignment.

## Core concept

- Session workflow:
  - `received -> intent_captured -> (council_briefing/discussion/finalized) -> plan_generated -> mockup_ready -> plan_approved -> action_executed -> verify_run -> visual_review -> record_user_feedback -> summarized`
- Default flow:
  - intent capture -> council draft -> consultant clarification loop -> plan/mockup -> execution -> verification -> user approval
- Dynamic council team:
  - team leads are session-scoped and can be appended/replaced/removed
- Persistent execution:
  - state is stored in JSON + SQLite and can be reconciled with repo footprint
- Visual review gate:
  - when render-capable MCP tools are detected for UI/UX tasks, `visual_review` is required before final user approval

## Quick start

```bash
make agent-install   # interactive install: terms -> scope(global/local) -> Playwright MCP consent -> expertise survey
make setup          # build + test + smoke + install hooks
make build          # bootstraps local Go toolchain if missing
make test           # unit tests
make smoke          # JSON-RPC end-to-end smoke test
make install-hooks  # install git hooks
make run-binary     # run built binary (.codex-mcp/bin/codex-mcp)
# or
make run-local      # bootstrap + build + run
```

After install, you can trigger `$codex-troller-autostart` from Codex.

## Installer behavior

`make agent-install` enforces:

1. Terms consent gate (required)
   - software is not sufficiently validated
   - user assumes responsibility for issues/damages
   - user acknowledges GNU GPL v3.0 license
2. Install scope selection
   - `global` (`~/.codex/...`) or `local` (`<repo>/.codex/...`)
3. Optional Playwright MCP registration
   - source: `https://github.com/microsoft/playwright-mcp`
4. Initial user expertise survey
   - `overall`, `response_need`, `technical_depth`, `domain_knowledge`
5. Default profile persistence
   - profile file is injected at runtime via launcher env var

Non-interactive mode:

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

## Manual run examples

```bash
echo '{"jsonrpc":"2.0","id":1,"method":"initialize"}' | .codex-mcp/bin/codex-mcp

echo '{"jsonrpc":"2.0","id":2,"method":"tools/list"}' | .codex-mcp/bin/codex-mcp

cat <<'EOF2' | .codex-mcp/bin/codex-mcp
{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"ingest_intent","arguments":{"raw_intent":"goal: add error handling and tests to login API module","session_id":""}}}
EOF2
```

## Process verification

- Unit tests: `make test`
- End-to-end smoke: `make smoke`
- Smoke validates the full route including council consensus and final user approval gate.

Smoke environment variables:
- `SMOKE_GOAL`
- `SMOKE_SCOPE`
- `SMOKE_CONSTRAINT`
- `SMOKE_CRITERIA`
- `SMOKE_TAGS_CSV`
- `SMOKE_AVAILABLE_MCPS`
- `SMOKE_AVAILABLE_MCP_TOOLS`

## Available tools

### Workflow

- `start_interview`
- `ingest_intent`
- `clarify_intent`
- `generate_plan`
- `generate_mockup`
- `approve_plan`
- `reconcile_session_state`
- `set_agent_routing_policy`
- `get_agent_routing_policy`
- `council_configure_team`
- `council_start_briefing`
- `council_submit_brief`
- `council_summarize_briefs`
- `council_request_floor`
- `council_grant_floor`
- `council_publish_statement`
- `council_respond_topic`
- `council_close_topic`
- `council_finalize_consensus`
- `council_get_status`
- `validate_workflow_transition`
- `run_action`
- `verify_result`
- `visual_review`
- `record_user_feedback`
- `continue_persistent_execution`
- `summarize`
- `get_session_status`

### Git helpers

- `git_get_state`
- `git_diff_symbols`
- `git_commit_with_context`
- `git_resolve_conflict`
- `git_bisect_start`
- `git_recover_state`

## Runtime state

- Session state: `.codex-mcp/state/sessions.json`
- Council DB: `.codex-mcp/state/council.db`
- Installer profile (default):
  - global: `~/.codex/codex-troller/default_user_profile.json`
  - local: `<repo>/.codex/codex-troller/default_user_profile.json`

## Dynamic council team example

```json
{
  "name": "council_configure_team",
  "arguments": {
    "session_id": "s1",
    "mode": "replace",
    "managers": [
      { "role": "development lead", "domain": "backend" },
      { "role": "ux director", "domain": "frontend" },
      { "role": "llm lead", "domain": "ai_ml" }
    ]
  }
}
```

## Agent routing defaults

- Client interview: `gpt-5.2`
- Orchestrator/reviewer: `gpt-5.3-codex`
- Worker (function/module implementation): `gpt-5.3-codex-spark`
