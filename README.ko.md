# codex-troller MCP Local Server (Korean)

이 문서는 한국어 버전입니다. 영어 기본 문서는 `README.md`를 참고하세요.

본 프로젝트는 Codex CLI를 위한 **로컬 MCP 서버** PoC입니다.
목표는 “자동 실행 + 사용자 검토”를 강제하는 고객 미팅형 실행 흐름을 코드로 구성하는 것입니다.

## 핵심 컨셉

- 실행 이전에 상태를 저장하고, 검토 가능한 문서/요약을 우선 제시합니다.
- 상태는 세션 단위로 관리되며: `received -> intent_captured -> (council_briefing/discussion/finalized) -> plan_generated -> mockup_ready -> plan_approved -> action_executed -> verify_run -> visual_review -> record_user_feedback -> summarized`
- 기본 진행은 `목적 수집 -> council 1차 윤곽 -> 상담 재개(구체화+제안 반응) -> 계획/구현` 순서입니다.
- 주요 실패 경로에서 상태를 `failed`로 전환해 재실행/재회의 지점으로 복귀 가능합니다.
- 검증 실패/사용자 미승인 시 `FixLoopCount`를 올리고 `intent_captured`로 재진입해 반복 실행합니다.
- `clarify_intent`는 누락 정보가 남아 있으면 `next_step=clarify_intent`로 후속 질문을 계속 반환합니다.
- `clarify_intent`는 누락 정보 보완 후 상담가 윤곽(`consultant_message`)을 제시하고, 한 번에 한 질문으로 피드백을 반영해 반복합니다.
- 사용자 피드백에서 요구 충돌/트레이드오프 신호가 보이면 상담가 단독 판단 대신 `council_start_briefing`으로 재토론을 요청합니다.
- 사용자 이해도 프로필(`user_profile`)에 따라 질문 깊이/응답 필요도를 자동 조정합니다.
  - 익숙하지 않은 영역: 팀장 에이전트 자율권을 높이고 상담 전달은 추상화.
  - 익숙한 영역: 구조/리스크를 더 구체적으로 설명하고 구체화 질문도 기술적으로 확장.
- 사용자 전문성 추정은 `confidence`(0~1)와 `evidence` 로그를 함께 유지합니다.
  - 신뢰도 낮음(<0.55): 과도한 자동결정/강한 가정은 억제하고 보수적으로 진행.
- 인터뷰 질문은 고정 설문지가 아니라, 현재 누락된 정보 기준으로 매 턴 1개씩 동적으로 생성됩니다.
- 세부 구현 계획은 단일 에이전트가 즉시 확정하지 않고, 팀장 council 토론 단계를 선행합니다.
- council 팀장 구성은 세션별로 동적입니다. 프로젝트 특성에 맞게 역할을 추가/교체/제거할 수 있습니다.
- `verify_result` 이후에는 Visual Reviewer 절차를 거칩니다.
  - 렌더링 가능한 MCP가 감지되면(`available_mcps`, `available_mcp_tools`) Visual Reviewer가 구현 화면/상호작용을 검토합니다.
  - 검토 후 제작본을 기준으로 UX Director 회의 요약(`ux_director_summary`)을 남겨야 다음 단계로 진행됩니다.
- 세션 재개 시 `reconcile_session_state`로 저장 상태와 현재 코드 상태(git + footprint digest)를 비교합니다.
- Git 컨텍스트(브랜치, diff/복구/commit)를 함께 관리해 변경 추적을 쉽게 합니다.

## 빠른 시작

```bash
make agent-install   # 인터랙티브 설치(약관 동의 -> global/local 범위 -> Playwright MCP 선택 -> 전문성 설문)
make setup          # build + test + smoke + git hooks 설치
make build          # Go가 없으면 bootstrap-go.sh가 로컬에 Go를 설치합니다.
make test           # 단위 테스트 실행
make smoke          # JSON-RPC end-to-end 스모크 테스트(프로세스 관리 경로)
make install-hooks  # pre-commit/commit-msg hook 설치
make run-binary     # 빌드된 바이너리 실행 (.codex-mcp/bin/codex-mcp)
# 또는
make run-local      # 부트스트랩+빌드+실행
```

에이전트 자동 설치 절차는 `AGENT_INSTALL.md`를 참고하세요.
설치 후에는 스킬 `$codex-troller-autostart` 호출만으로 인터뷰를 바로 시작할 수 있습니다.
`make agent-install`은 `codex-troller` 등록 시 launcher를 통해 설치 설문에서 수집한 기본 `user_profile`을 자동 로드합니다.

`make build`는 내부에 Go가 없더라도 다음을 수행합니다.
- `./scripts/bootstrap-go.sh` 실행
- Go를 `.codex-mcp/.tools/go`에 설치
- `cmd/codex-mcp` 빌드

## 수동 실행 예시

```bash
echo '{"jsonrpc":"2.0","id":1,"method":"initialize"}' | .codex-mcp/bin/codex-mcp

echo '{"jsonrpc":"2.0","id":2,"method":"tools/list"}' | .codex-mcp/bin/codex-mcp

cat <<'EOF2' | .codex-mcp/bin/codex-mcp
{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"ingest_intent","arguments":{"raw_intent":"로그인 API 모듈에 에러 처리와 테스트를 추가해줘","session_id":""}}}
EOF2
```

## 프로세스 관리 테스트

- 단위 테스트: `make test`
- end-to-end 스모크: `make smoke`
- 스모크 테스트는 `ingest_intent -> council_start_briefing -> council_submit_brief(전 역할) -> council_summarize_briefs -> topic 토론 루프 -> council_finalize_consensus -> clarify_intent(요구사항) -> clarify_intent(proposal_feedback=\"이대로 진행\") -> generate_plan -> generate_mockup -> approve_plan -> run_action -> verify_result -> record_user_feedback -> summarize`를 JSON-RPC로 실제 호출하고, 최종 `step=summarized` 도달 여부를 검사합니다.
  - 스모크 기본 입력은 시각 렌더링 MCP를 전달하지 않으므로 `visual_review`는 자동 생략될 수 있습니다.
- 스모크 입력은 환경변수로 변경 가능:
  - `SMOKE_GOAL`, `SMOKE_SCOPE`, `SMOKE_CONSTRAINT`, `SMOKE_CRITERIA`, `SMOKE_TAGS_CSV`
  - `SMOKE_AVAILABLE_MCPS`, `SMOKE_AVAILABLE_MCP_TOOLS` (시각 렌더링 MCP를 넘기면 `visual_review`도 자동 실행)

## 현재 제공 도구

### 워크플로우 계열
- `ingest_intent`
- `start_interview`
- `clarify_intent`
- `generate_plan`
- `generate_mockup`
- `approve_plan`
- `reconcile_session_state`
- `set_agent_routing_policy`
- `get_agent_routing_policy`
- `council_start_briefing`
- `council_configure_team`
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

### Git 보조 계열
- `git_get_state`
- `git_diff_symbols`
- `git_commit_with_context`
- `git_resolve_conflict`
- `git_bisect_start`
- `git_recover_state`

## 운영 상태

- 상태 파일: `.codex-mcp/state/sessions.json`
- 토론 DB: `.codex-mcp/state/council.db` (SQLite, 이벤트 단위 즉시 반영)
- 상담 제안 이력(`proposal_history`)도 SQLite에 즉시 upsert되며, `council_get_status`에서 `proposals`로 조회됩니다.
- 세션 상태는 도구 호출 후 저장됩니다.
- 현재는 로컬 스토리지 기반 PoC이며, 추후 `git` 히스토리 연동과 분석 지표 강화로 확장 예정입니다.

## 인터뷰 시작(명령 최소화)

- 권장 시작 방식:
  - Codex 대화에서 `$codex-troller-autostart`를 호출
- 또는 MCP 엔트리포인트를 직접 호출:
  - `start_interview`
- 선택적으로 `start_interview`/`ingest_intent`에 `user_profile`을 넘길 수 있습니다.
  - `overall`: `beginner|intermediate|advanced`
  - `response_need`: `low|balanced|high`
  - `technical_depth`: `abstract|balanced|technical`
  - `domain_knowledge`: 예) `{ "backend": "advanced", "security": "beginner" }`
- 시각 검증 자동 활성화를 원하면 `available_mcps`/`available_mcp_tools`를 함께 전달합니다.
  - 예) `available_mcps:["playwright"]`, `available_mcp_tools:["playwright.screenshot"]`
- 이 시작점은 인터뷰 질문을 먼저 만들고, 이후 워크플로우를 순차 진행합니다.

## 동적 팀장 구성

- `council_configure_team`으로 세션별 팀장을 `append|replace|remove` 할 수 있습니다.
- `council_start_briefing` 호출 시 `managers`/`manager_mode`를 함께 전달해 one-shot으로 팀 구성을 바꿀 수도 있습니다.
- 팀장 식별자는 자동 정규화됩니다. 예: `Development Lead` -> `development_lead`.

## Git Hook 정책

- `make install-hooks`를 실행하면 `core.hooksPath=.githooks`가 설정됩니다.
- pre-commit:
  - `make test` 실행
- commit-msg:
  - 커밋 본문에 `goal_id: <id>` 메타데이터를 강제합니다.
  - 예외: `Merge`, `Revert` 커밋
  - 임시 우회: `SKIP_GOAL_ID_CHECK=1 git commit ...`

## 승인 게이트 정책

- `approve_plan`은 사용자가 계획 승인(`approved=true`)을 요청할 때 아래를 강제 검증합니다.
  - `requirement_tags`: 최소 1개 이상
  - `success_criteria`: 현재 세션의 의도에서 정의한 기준과 의미적으로 일치해야 함
- 조건 미충족 시 `StepFailed`로 이동하고, 미충족 항목과 다음 액션을 반환합니다.
- `clarify_intent`에서 `requirement_tags`/`success_criteria`를 미리 수집하면 승인 단계 부담이 줄어듭니다.

## Persistent Completion 정책

- `verify_result`가 통과해도 `record_user_feedback(approved=true)` 호출 전에는 완료로 보지 않습니다.
- 렌더링 MCP가 있는 UI/UX 성격 작업은 `visual_review` 완료 전까지 `record_user_feedback`이 차단됩니다.
- `approved=false` 또는 검증 실패가 발생하면 자동으로 재계획 루프에 진입합니다.
- 재시도 한도(`MaxFixLoops`, 기본 5회)를 넘기면 `manual_review`로 전환합니다.

## Agent Routing 기본값

- 사용자 인터뷰: `gpt-5.2`
- 오케스트레이터/리뷰: `gpt-5.3-codex`
- 함수/모듈 구현 워커: `gpt-5.3-codex-spark`

## Codex MCP 등록 예시

아래는 codex 실행 환경에서 MCP 서버를 등록할 때 넣을 예시 블록입니다.  
실제 설정 파일 포맷은 사용 중인 codex 버전에 따라 달라질 수 있으므로, 값은 경로만 환경에 맞춰 조정하세요.

```json
{
  "mcpServers": {
    "codex-troller": {
      "command": "/home/user/Workspace/codex-troller/.codex-mcp/bin/codex-mcp",
      "description": "Local Codex planning server"
    }
  }
}
```

권장: local-only 자동화이므로 `args`는 비워두고 필요 시 `WORKDIR` 환경변수로 작업 루트를 고정하세요.

## 설치 시 선택 항목

- `make agent-install` 실행 시 아래를 확인/수집합니다.
  - LLM 설치 동의(미동의 시 즉시 중단)
  - 동의 항목: 소프트웨어 검증 미완료, 문제 책임 사용자 본인, GNU GPL v3.0 라이선스 확인
  - 설치 범위: `global` 또는 `local`
  - Playwright MCP 설치 동의 여부 (`mcp_servers.playwright`, source: `https://github.com/microsoft/playwright-mcp`)
  - 사용자 전문성 초기값(`overall`, `response_need`, `technical_depth`, `domain hints`)
- 자동화가 필요하면 비대화형 모드를 사용하세요.
  - `AGENT_INSTALL_NON_INTERACTIVE=1 INSTALL_TERMS_AGREED=yes ... make agent-install`

## 다음 단계 제안

1. 코드 상태 인덱스(심볼 단위) 생성기 추가
2. 의도/계획/실행의 변경 추적을 위한 정적 검사 및 유효성 규칙 강화
3. 최소 단위 테스트(상태 전이, 입력 파싱, 허용 명령 실행) 추가
