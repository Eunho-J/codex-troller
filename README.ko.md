# codex-troller MCP 로컬 서버

가이드:
- [English Guide](README.md)
- [한국어 가이드](README.ko.md)
- [日本語ガイド](README.ja.md)
- [中文指南](README.zh.md)

설계 노트:
- [English Design Notes](mcp-server-discussion.md)
- [한국어 설계 노트](mcp-server-discussion.ko.md)
- [日本語設計ノート](mcp-server-discussion.ja.md)
- [中文设计说明](mcp-server-discussion.zh.md)

## 설치 (MCP 서버, 스킬 아님)

이 저장소는 Codex 스킬 패키지가 아니라 **MCP 서버 프로젝트**입니다.
기본이자 권장 경로는 LLM 기반 설치입니다.
이 저장소에는 `skill-installer`를 사용하지 마세요.

기본 Codex 에이전트에 아래 프롬프트를 복사/붙여넣기 하세요:

```text
Install `codex-troller` from `https://github.com/Eunho-J/codex-troller`.

Requirements:
- Treat this repository as an MCP server project, not a skill package.
- Do not use `skill-installer`.
- Read and follow this repository `README.md` install/setup steps.
- Run interactive `make agent-install` (terms consent, install scope, optional Playwright MCP registration, expertise survey).
- Run `make setup` and verify with `make smoke`.
- If a step fails, resolve it and continue until installation and smoke verification both pass.
```

`codex-troller`는 Codex CLI용 로컬 Go MCP 서버입니다.
핵심 목적은 작업을 구조화해 신뢰성과 사용자 의도 정합성을 높이는 것입니다.

## 핵심 개념

- 세션 워크플로우:
  - `received -> intent_captured -> (council_briefing/discussion/finalized) -> plan_generated -> mockup_ready -> plan_approved -> action_executed -> verify_run -> visual_review -> record_user_feedback -> summarized`
- 기본 흐름:
  - 의도 수집 -> council 초안 -> 상담 구체화 루프 -> 계획/목업 -> 실행 -> 검증 -> 사용자 승인
- 동적 council 팀:
  - 팀장 역할은 세션 단위이며 append/replace/remove가 가능합니다.
- 지속 실행:
  - 상태는 JSON + SQLite에 저장되며 저장소 footprint와 reconcile할 수 있습니다.
- 시각 검토 게이트:
  - UI/UX 작업에서 렌더링 가능한 MCP 툴이 감지되면 최종 승인 전에 `visual_review`가 필요합니다.

## 빠른 시작

```bash
make agent-install   # 대화형 설치: 약관 -> 범위(global/local) -> Playwright MCP 동의 -> 전문성 설문
make setup          # build + test + smoke + 훅 설치
make build          # Go가 없으면 로컬 Go 툴체인을 부트스트랩
make test           # 단위 테스트
make smoke          # JSON-RPC end-to-end 스모크 테스트
make install-hooks  # git 훅 설치
make run-binary     # 빌드된 바이너리 실행 (.codex-mcp/bin/codex-mcp)
# 또는
make run-local      # bootstrap + build + run
```

설치 후 Codex에서 `$codex-troller-autostart`를 호출할 수 있습니다.

## 설치 동작

`make agent-install`은 아래를 강제합니다.

1. 약관 동의 게이트 (필수)
   - 소프트웨어가 충분히 검증되지 않음
   - 문제/손해에 대한 책임은 사용자 본인
   - GNU GPL v3.0 라이선스 인지
2. 설치 범위 선택
   - `global` (`~/.codex/...`) 또는 `local` (`<repo>/.codex/...`)
3. Playwright MCP 등록(선택)
   - source: `https://github.com/microsoft/playwright-mcp`
4. 사용자 전문성 초기 설문
   - `overall`, `response_need`, `technical_depth`, `domain_knowledge`
5. 기본 프로필 저장
   - 런타임에서 launcher 환경변수로 프로필 파일 주입

비대화형 모드:

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

## 수동 실행 예시

```bash
echo '{"jsonrpc":"2.0","id":1,"method":"initialize"}' | .codex-mcp/bin/codex-mcp

echo '{"jsonrpc":"2.0","id":2,"method":"tools/list"}' | .codex-mcp/bin/codex-mcp

cat <<'EOF2' | .codex-mcp/bin/codex-mcp
{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"ingest_intent","arguments":{"raw_intent":"goal: add error handling and tests to login API module","session_id":""}}}
EOF2
```

## 프로세스 검증

- 단위 테스트: `make test`
- end-to-end 스모크: `make smoke`
- 스모크는 council 합의와 최종 사용자 승인 게이트를 포함한 전체 경로를 검증합니다.

스모크 환경변수:
- `SMOKE_GOAL`
- `SMOKE_SCOPE`
- `SMOKE_CONSTRAINT`
- `SMOKE_CRITERIA`
- `SMOKE_TAGS_CSV`
- `SMOKE_AVAILABLE_MCPS`
- `SMOKE_AVAILABLE_MCP_TOOLS`

## 제공 도구

### 워크플로우

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

### Git 헬퍼

- `git_get_state`
- `git_diff_symbols`
- `git_commit_with_context`
- `git_resolve_conflict`
- `git_bisect_start`
- `git_recover_state`

## 런타임 상태

- 세션 상태: `.codex-mcp/state/sessions.json`
- council DB: `.codex-mcp/state/council.db`
- 설치 프로필(기본):
  - global: `~/.codex/codex-troller/default_user_profile.json`
  - local: `<repo>/.codex/codex-troller/default_user_profile.json`

## 동적 council 팀 예시

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

## 에이전트 라우팅 기본값

- 사용자 인터뷰: `gpt-5.2`
- 오케스트레이터/리뷰어: `gpt-5.3-codex`
- 워커(함수/모듈 구현): `gpt-5.3-codex-spark`
