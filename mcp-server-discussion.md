# Codex MCP 서버 설계 토론

## 진행 방식(중요)
- 이 문서는 최종 합의본이 아니라, 변경되는 운영 기준을 누적 반영하는 살아있는 설계 문서다.
- 논의가 진행될 때마다 `현재 합의 정리`를 갱신하고, 하위 항목에 새로운 제약/규칙을 덧붙인다.
- 목표는 한번에 완성하는 것이 아니라, 사용 중에도 기준이 일관되게 진화하도록 만드는 것이다.

## 문제 정의
- 사용자가 Codex에 애매한 요구를 넣으면, 실제 실행은 기대와 다르게 흘러가거나 맥락이 손실됨
- 작업 결과물이 실행 가능하지만, 사용자의 의도와 “왜 그 작업을 했는지”가 불명확한 경우가 있음
- 반복 질문/수정으로 인해 신뢰성(재현성/예측성) 저하

## 목표
- Go 기반 로컬 MCP server를 두고 Codex 요청을 구조화된 단계로 변환
- 의도 모호성 감지 → 정제 → 실행 → 검증 → 보고 루프를 기본화
- 작업 결과를 코드 실행 로그, 산출물, 판단 근거와 함께 저장해 추적성 확보

## 핵심 가치관 및 전략
- `의도 정합성 우선`: 결과 자체보다 사용자의 실제 목표와 일치하는가를 우선 판단한다.
  - 각 단계의 산출물은 “무엇을 했는가”보다 “왜 그렇게 했는가”를 기록한다.
- `작은 확약, 빠른 피드백`: 큰 변경은 작은 action으로 쪼개고, 각 단계에서 실패/불일치 원인을 즉시 노출한다.
- `인간 중심 제어권`: 자동화 수준을 높이되, 중요한 전환(범위 확대/파괴적 변경/외부작업)은 승인 지점으로 묶는다.
- `재현 가능한 신뢰성`: 동일 입력이면 동일 상태에서 동일한 실행 흐름/결과가 나오도록 로그와 메타데이터를 보존한다.
- `회복 가능한 실패`: 실패를 막는 대신, 실패를 예측·분류·복구할 수 있게 설계한다.
  - 실패는 “중단”이 아니라 다음 단계로의 정보를 제공하는 이벤트로 다룬다.
- `최소 권한 실행`: 실행 범위는 제한된 작업 디렉터리, 허용 명령, 타임아웃/자원 제한으로 격리한다.

## 구현 전략(핵심 원칙)
- 아웃라인 중심 설계: 초기에는 완성형 기능보다 흐름과 경계(스키마, 체크리스트, 에러 모델)를 먼저 고정한다.
- 단계별 확장: V1에서는 `의도 정제 -> 계획 생성 -> 승인/실행 -> 검증 -> 요약`의 폐회로를 만들고, 이후 확장한다.
- 규칙 우선, 모델 보조: 임계값/규칙 기반 정규화로 기본 동작을 고정하고, LLM은 예측 불가 영역만 보완한다.
- 메트릭 기반 개선: 매 작업마다 “의도 일치도, 계획 충실도, 검증 통과율, 재요청 횟수”를 추적해 전략을 조정한다.

## 핵심 아이디어
- Codex가 단일 텍스트 응답을 받는 대신 MCP를 통해 `workflow` 형태로 상호작용
- “작업 의도 해석기 → 실행 계획 수립기 → 실행기 → 검증기 → 사후요약기”로 분리
- 각 단계는 독립 도구(tool)로 노출되어 실패 시 명확한 복구 전략 적용

## 시스템 구성(안)
- MCP 서버(Go)
  - JSON-RPC 2.0 over stdio
  - `Tools`:
    - `ingest_intent`: 사용자 요청을 내부 정규화 모델로 정리
    - `clarify_intent`: 누락 항목(목표/범위/제약/성공 기준) 체크리스트 생성
    - `generate_plan`: 실행 단계(명령어, 파일, 우선순위, 리스크) 초안 생성
    - `approve_plan`: 사용자 승인/수정 반영
    - `run_action`: 실제 명령 실행(샌드박스 옵션 포함)
    - `verify_result`: 테스트/형식/정적검사 실행 후 합격 기준 산출
    - `summarize`: 결과 요약, 위험, 다음 액션 제안
  - 상태 저장소: 워크플로우 상태 JSON(세션ID, 단계, 산출물, 마지막 오류)
- Codex 통합 가이드
  - 하나의 요청마다 “의도 정제 + 계획 승인” 과정을 거치게 하는 프롬프트/툴 시퀀스
  - 실패 시 자동 재질문 템플릿 제공

## 구현 단계(1차)
1. 공통 스키마 설계
   - 요청/계획/작업/검증 결과 스키마 (JSON Schema)
   - 공통 에러 코드와 재시도 정책 정의
2. Go MCP 최소 서버 골격
   - stdio transport, tool 등록, 핸들러 라우팅
   - 상태 저장체계(메모리 + 로컬 JSON 파일)
3. 핵심 3개 Tool부터 구현
   - `ingest_intent`, `generate_plan`, `verify_result`
4. Codex와의 통신 시나리오 테스트
   - 의도 모호성 케이스, 실행 실패 케이스, 테스트 실패 케이스
5. 단계 확장
   - `clarify_intent`, `approve_plan`, `run_action`, `summarize`

## 초기 기술 선택(안)
- Go 1.22+, 표준 라이브러리 `encoding/json`, `os/exec`, context
- MCP 구현: 공식 Go MCP SDK 사용(우선 `go`) 또는 최소 자체 라우팅으로 시작
- 로그: `slog` + 실행 메타데이터 JSON 라인
- 신뢰성 보강: 타임아웃, 최대 재시도 횟수, 실행 허용 디렉터리 화이트리스트

## 핵심 규칙 엔진 설계(제안)
- 워크플로우 상태 전이를 항상 검증하는 `validate_workflow_transition` 룰을 둔다.
- 입력은 `current_step`, `next_step`, `session_context`, `artifacts`, `risk_flags`를 받아 다음 4가지를 판별한다.
  - `allowed` (bool): 전이 허용 여부
  - `required_human_checks` (string[]): 사람이 개입해 재확인해야 할 항목
  - `auto_fixes` (string[]): 즉시 자동 보정 가능한 액션
  - `blocking_reasons` (string[]): 불가능한 경우 중단 사유
- 거부 규칙 예시:
  - 의도 정리 미완료 상태에서 `generate_plan`/`run_action`로 전환 시 차단
  - 계획 승인 미완료 상태에서 `run_action` 실행 시 차단
  - `verify_result`를 통과하지 못한 상태에서 `summarize`로 바로 이동 시 경고 또는 차단
  - 사용자 제약(파일 경로, 명령 허용 목록, 네트워크 제한)을 위반하면 차단
- 결과를 상태 로그에 JSON으로 남겨 추적성과 사후 분석을 보장한다.

## validate_workflow_transition API 초안
- `Tool`: `validate_workflow_transition`
- `Input`:
  - `session_id` (string): 워크플로우 세션 식별자
  - `current_step` (enum): `received | intent_captured | plan_generated | plan_approved | action_executed | verify_run | summarized | failed`
  - `next_step` (enum): 다음 실행 단계
  - `context` (object): 목표, 범위, 제약, 성공기준, 위험도
  - `artifacts` (array): 기존 산출물 메타데이터(파일/명령/테스트 결과)
  - `dry_run` (bool): 실제 강제 전이 없이 진단만 수행할지 여부
- `Output`:
  - `allowed` (bool)
  - `blocking_reasons` (array)
  - `required_checks` (array)
  - `suggested_next_actions` (array)
  - `next_step` (string): 규칙 기반 조정된 권고 단계
  - `confidence` (0.0~1.0): 판정 신뢰도

## 실패 정책(안전 기본값)
- `blocked` + `dry_run=true`: 현재 단계 유지, 다음 추천 action 제시
- `blocked` + `dry_run=false`: 오류 이벤트 기록 후 사용자 승인 루프로 되돌림
- `warn` + `allowed`: 진행은 허용하되 세션 요약에 경고를 남김
- 3회 연속 `blocked` 시: 자동으로 `failed`로 전이하고 원인 요약 생성

## 토론 포인트
- Codex가 매 단계마다 어느 정도 개입해야 하는가? (사람 승인 비율)
- `plan` 수립을 코드 생성 직후 바로 실행할지, 사용자가 승인 후 실행할지
- 실행 도구의 범위를 로컬 파일작업/테스트까지로 제한할지, 네트워크 호출 허용 여부
- “의도 요약” 품질을 높이기 위해 어떤 체크리스트를 기본 적용할지

## 다음 액션(권장)
- 1차: 위 4개 핵심 Tool 기반 PoC 구현 (`ingest_intent`, `clarify_intent`, `generate_plan`, `approve_plan`)
- 2차: `run_action` + `verify_result` 추가 후 실제 작업흐름 자동화
- 3차: 실패 시나리오/재시도 정책 정교화, 사용자 피드백 점수 기반 튜닝

## 코드 자산 관리 브릿지(Execution Agent용)
- 목표: 실행 에이전트가 전체 코드를 모두 읽지 않고, 계약(인터페이스/시그니처/테스트) 기준으로만 수정 범위를 탐색하게 한다.
- 접근: 처음부터 완전한 Graph DB를 강제하지 않고, **경량 지식 레이어 + 점진적 그래프화**로 시작한다.
- 1단계 지식 레이어(필수)
  - `module_contract`: 모듈 단위 계약 정의
    - `모듈명`, `역할`, `공개 API`, `입력`, `출력`, `부작용`, `의존 모듈`, `관련 테스트`, `갱신일`
  - `symbol_index`: 파일/함수/타입/변수별 인덱스와 해시
  - `requirement_index`: 요구사항 키워드/테마/비즈니스 규칙 태그
  - `dependency_index`: import/호출/테스트/설정 의존성 그래프(먼저는 정적 텍스트/AST 기반)
  - `change_log`: 변경 이벤트(누가/무엇을/왜/무엇이 영향받는지)
- 1차 저장소 형태
  - 로컬 단일 저장소(예: SQLite + JSON 컬럼 + FTS)로 빠르게 운영
  - 필요 시 추후 Graph DB 전환: 노드(요구사항/모듈/심볼/테스트), 엣지(의존/영향/요구-구현 매칭)
- 업데이트 파이프라인
  - 파일 변경 감지 → AST/임포트 파싱 → 인덱스 갱신 → 영향 범위 재산정 → 실행 에이전트 대상 컨텍스트 생성
  - 변경 전후 심볼 해시 비교로 리스크 있는 노드만 갱신
- 검색/조회 방식
  - 요구사항으로 조회: `요구사항 태그 + 시그니처 + 호출 경로`로 상위 N 후보 선정
  - 실행 전 조회: 필요한 컨텍스트를 “계약 단위”만 주입(관련 코드 조각 + 호출자 체인 + 테스트)
  - 수정 후 검증: 영향받는 상위/하위 노드 재검증 후 사용자 승인 로그에 반영
- Execution Agent 동작 규칙
  - 에이전트 초기 상태는 `module_contract` + `관련 테스트` + `영향도` 만으로 시작
  - 컨텍스트 미달이면 추가 조회 2회 허용, 초과 시 사용자/오케스트레이터에게 보류
  - 최종 반영 시 변경 요약은 코드 diff + 계약 변경 영향(테스트/의존)에 대해 기록

## 제품 기준점으로의 적용
- 이 문서는 기능 스펙이 아니라 운영 철학의 기준선이다.
- Codex 및 MCP의 기본 동작은 아래 4개 질문을 항상 통과해야 한다.
  - 요청의 목표, 범위, 제약, 성공 기준이 충분히 정리되었는가?
  - 현재 단계의 실행/결정이 추적 가능한 상태 모델에 기록되는가?
  - 사람이 개입해야 할 승인 포인트를 의도적으로 생략하지 않았는가?
  - 실패 시 사용자에게 다음 액션을 명확히 제시했는가?
- 설계 원칙 우선순위: `신뢰성` > `재현성` > `자동화 편의성`
- 각 세션 종료 시, 위 기준 위반 여부는 로그/요약에서 점검 가능한 항목으로 남긴다.

## Git 연동 상태관리 제안
- Git은 코드의 최종 진실 소스로 사용한다.
- DB는 코드 검색·요구사항 매핑·의존성·작업 이력의 인덱스 레이어로 운용한다.
- 핵심 일치성 검증 규칙:
  - Function/Class 단위 `signature_hash`(AST 기반)와 `tracked_file_hash`를 저장
  - 현재 HEAD/워크트리와 비교해 stale/orphan 상태를 판별
  - DB의 변경 상태가 Git 상태와 다르면 실행 전 게이트에서 경고 또는 차단
- 추적성(Traceability) 규칙:
  - `change_event`를 커밋 단위로 기록한다.
  - 각 변경 이벤트는 `commit_id`, `agent_id`, `goal_id`, `수정 심볼`, `영향 모듈`, `요구사항 태그`, `테스트 결과`를 포함한다.
- 구현 우선순위:
  - 1차: `git diff` 기반 증분 동기화
  - 2차: pre/post-commit hook 연동
  - 3차: Graph DB 전환(필요 시) 또는 Neo4j/유사 저장소로 확장

## Git 기능 최대 활용 전략
- Git 정보는 단순 버전관리 수단이 아니라 오케스트레이션의 신뢰 계층으로 사용한다.
- 핵심 연동 포인트:
  - commit anchor: 모든 세션 상태 변화는 특정 `commit_id` 또는 `commit_range`와 연결
  - branch 전략:
    - `main`(안정), `agent/<domain>`(에이전트 단위 작업), `incident/<id>`(실패 원인 분리), `release/*`(검증 완료 후)
    - 에이전트는 브랜치 단위로 분리 작업 후 오케스트레이터가 병합
  - staging/working tree:
    - `git status`로 실행 전 정합성 검사, 실행 후 변경 범위를 스냅샷으로 기록
    - 실패 후 `git restore`/`git checkout --`로 안전 복구
  - commit 규약:
    - 제목 형식 예시: `feat(agent-id): <목표 요약>`
    - 바디에 `goal_id`, `requirement_tags`, `test_ids`, `risk` 필수 기재
    - 세션 종료 시 자동 생성 요약과 함께 `commit message`에 정책 준수 태그 반영
  - hooks:
    - pre-commit: 포맷/정적 검사, 테스트 스모크, 워크플로우 상태 유효성 검사
    - commit-msg: 커밋 규약 검증 및 `goal_id` 존재 검사
    - post-commit: 변경된 심볼/테스트 영향도 계산 큐 적재
  - git notes:
    - 코드 변경을 넘어서 의도/권한 판단 근거를 `git notes`에 저장해 audit trail 확보
  - blame / log / grep:
    - `git blame`으로 해당 라인의 책임 주체 추적
    - `git log -S`로 특정 심볼/함수 변경 이력 추적
    - `git grep`으로 요구사항 태그 연계 텍스트 검색
  - bisect:
    - 실패율 급증 시 자동 bisect 모드로 회귀 커밋 후보를 좁혀 수정 범위를 축소
  - stash / worktree:
    - 다중 에이전트 동시 작업은 worktree로 분리해 충돌 최소화
    - 임시 실험은 stash로 숨기고 재현 가능한 상태에서만 커밋
  - tags / reflog:
    - 안정 상태를 태그로 고정해 실행 기준점 롤백 가능성 확보
    - reflog로 마지막 안전 상태로 빠르게 복구
  - MCP에 반영할 Git 툴 확장:
  - `git_get_state`, `git_diff_symbols`, `git_resolve_conflict`, `git_commit_with_context`, `git_bisect_start`, `git_recover_state`
  - 모든 Git 툴은 현재 브랜치와 변경 범위를 상태 머신에 바인딩해 traceability를 보장

## Git 기능 실행 우선순위(실무 로드맵)
- 필수(MVP):
  - `git_get_state` (working tree + HEAD + branch + status)
  - `git_diff_symbols` (변경 심볼 추출 및 커밋/기준 비교)
  - `git_commit_with_context` (goal/요구사항 태그를 포함한 규격 커밋)
  - `git status`/`git diff` 기반 상태 불일치 경고
- 권장(운영 안정화):
  - pre-commit 훅(포맷/테스트/워크플로우 상태 유효성)
  - commit-msg 훅(커밋 규약 + goal_id 검사)
  - `git notes`를 통한 근거 기록
  - branch 전략 (`agent/<domain>`, `incident/<id>`)
- 확장(회복력 강화):
  - `git_resolve_conflict`(단계별 충돌 템플릿 + 수동 재질의 라우팅)
  - `git_recover_state`(안전 지점/태그 기반 복구)
  - `git_bisect_start`(실패 증감 구간 탐지)
  - worktree 기반 병렬 에이전트 실행
- 실행 규칙:
  - 필수는 바로 다음 버전부터, 권장은 운영이 안정될수록 순차 적용
  - 확장은 실패율/충돌률이 임계치(예: 회귀 반복 2회 이상) 넘을 때 우선 적용

## MCP Git Tool Contract (초안)
- `git_get_state`
  - Input
    - `scope`: `"repo" | "branch" | "file"` (기본: `"repo"`)
    - `path`: string? (scope가 file일 때 필수)
  - Output
    - `branch`: string
    - `head_commit`: string
    - `clean`: bool
    - `status_summary`: object (`modified`, `added`, `deleted`, `untracked` counts)
    - `stale_symbols`: array (symbol id + reason)
- `git_diff_symbols`
  - Input
    - `base`: string (commit id 또는 ref)
    - `target`: string (commit id 또는 `HEAD`)
    - `include_untracked`: bool
  - Output
    - `changed_symbols`: array (symbol id, type, before_hash, after_hash, file)
    - `deleted_symbols`: array
    - `renamed_symbols`: array
    - `tests_affected`: array
    - `confidence`: float
- `git_commit_with_context`
  - Input
    - `goal_id`: string
    - `goal_summary`: string
    - `requirement_tags`: array[string]
    - `agent_id`: string
    - `risk_level`: `"low" | "medium" | "high"`
    - `auto_push`: bool (현재는 기본 false)
  - Output
    - `commit_id`: string
    - `commit_message`: string
    - `created_files`: array
    - `modified_files`: array
    - `post_checks`: object
- `git_resolve_conflict`
  - Input
    - `files`: array[string]
    - `strategy`: `"abort" | "manual_review" | "ours" | "theirs" | "skip"`
    - `notes`: string
  - Output
    - `resolved`: bool
    - `remaining_conflicts`: array
    - `next_action`: string
- `git_bisect_start`
  - Input
    - `good_commit`: string
    - `bad_commit`: string
    - `test_command`: string
  - Output
    - `bisect_session_id`: string
    - `status`: string
- `git_recover_state`
  - Input
    - `mode`: `"checkout_safe_point" | "undo_uncommitted" | "restore_branch"`
    - `safe_point`: string
    - `branch`: string?
  - Output
    - `restored`: bool
    - `new_head`: string
    - `remaining_changes`: array[string]

## 현재 구현 진행
- 로컬 PoC 코드 스켈레톤 작성 완료(입출력 파싱, 세션 상태, 워크플로우 툴, Git 연동 툴)
- 생성 파일:
  - `go.mod`
  - `cmd/codex-mcp/main.go`
  - `internal/server/server.go`
  - `internal/server/types.go`
  - `internal/server/tools.go`
  - `README.md`
- 구현 범위:
  - JSON-RPC stdio 처리
  - `tools/list`, `tools/call` 라우팅
  - Workflow 도구: `ingest_intent`, `clarify_intent`, `generate_plan`, `approve_plan`, `validate_workflow_transition`, `run_action`, `verify_result`, `summarize`
  - Git 도구: `git_get_state`, `git_diff_symbols`, `git_commit_with_context`, `git_resolve_conflict`, `git_bisect_start`, `git_recover_state`
- 다음 보완 포인트:
  - 세션 상태 영속화
  - plan/verify 결과 기반 재시도 정책 강화
  - 권한/안전 정책 상세화
  - 테스트 시나리오 작성

## 외부 레퍼런스 반영(oh-my-claudecode 핵심 차용)
- 반영 기준: "중간 성공"이 아니라 "사용자 명시적 OK"까지 지속 수행.
- 도입한 패턴:
  - Persistent loop: 검증 실패/사용자 미승인 시 자동으로 `intent_captured`로 재진입 후 재계획.
  - Explicit sign-off gate: `record_user_feedback(approved=true)` 호출 전에는 완료 상태로 간주하지 않음.
  - Retry budget: `FixLoopCount`/`MaxFixLoops`로 재시도 상한을 두고, 초과 시 `manual_review`로 전환.
  - Failure recovery action: `continue_persistent_execution`로 실패 세션을 반복 경로에 재연결.
- 적용 목적:
  - 사용자 만족 기준을 단계 종결 조건으로 강제
  - 자동화와 안전 제어권의 균형 유지(무한 루프 방지 + 수동 개입 지점 보장)

## 현재 합의 정리(회의록)
- 운영 모드: 기본 자동 실행 + 대화형 검토
  - 실행 전 전체 계획을 문서/요약으로 보여주고 확인받음
  - 사용자는 고객사 미팅처럼 대화로 핵심 요구를 점진적으로 정리
  - 사용자 의도가 모호하면 에이전트가 가정 제안 + 확인 질문을 추가한다.
- 자동/수동 경계:
  - 요구사항이 충분히 정합되면 에이전트가 실행을 진행
  - 권한·리스크·기준 이탈처럼 판단이 애매한 항목은 재회의(추가 확인)로 전환
- Execution Agent 설계:
  - 에이전트는 코드 전체를 읽지 않고 계약 기반 컨텍스트로 시작
  - 우선 `module_contract`, `symbol_index`, `dependency_index`, `관련 테스트`, `변경 영향도`만 참조
  - 컨텍스트가 부족하면 제한 횟수 내에서 추가 조회, 초과 시 중단 후 사용자 재확인
- 구현 전략:
  - 테스트 중심으로 신뢰성 확보(단위 테스트를 전면 실행)
  - 에이전트는 모듈/기능 단위로 분할해 최소 단위 작업 수행
  - 변경 추적은 코드 diff + 계약 변경 영향 + 테스트 결과 기반으로 재사용성 높은 자산 관리
- 배포 범위:
  - 현재는 로컬 MCP 사용을 1차 목표(본인 환경의 codex cli 연동)
  - 추후 Linear/Slack 등 외부 연동은 선택 확장 기능으로만 고려

## 1인 SI 운영 원칙(당사 기준)
- 자원 제약을 반영한 운영 모드:
  - 한 번에 처리하는 범위를 작게 쪼개고, 각 단계 종료 후 재동기화한다.
  - 에이전트 간 핸드오프는 “입출력 계약 + 영향 범위 + 테스트 계획”으로만 한다.
- 비용 대비 안정성 우선:
  - 새 기능보다 실패 복구 경로(rollback/recover/retry/rollback note)를 먼저 완성한다.
  - 실수 가능성이 큰 영역은 수동 승인 게이트를 둔다.
- 인간 개입의 위치 고정:
  - 범위 확대, 파괴적 변경, 권한 경계 진입, 외부 호출 유발 지점은 항상 승인 요구.
- 커뮤니케이션 규칙:
  - 매 세션마다 “무엇을 왜 했는지”가 아니라 “다음 액션에서 어떤 값을 보장하는지”를 선명하게 기록한다.
- 품질 게이트:
  - 단위 테스트 + 정적 체크 없이 변경은 승인되지 않는다.
  - 실행 로그/요약에서 실패 원인과 재시도 권고를 분리해 제공한다.

## Phase 기반 실행 전략(자동화+안정성)
- Phase 1: 기반 정합성
  - 진입 조건: 요구사항-모듈 구조, 워크플로우 상태 모델, 기본 승인 규칙이 정해짐
  - 완료 조건: `workflow_state`, `clarify_intent`, `generate_plan`, `approve_plan`, `validate_workflow_transition`가 동작하고, 변경 추적이 `git_get_state`와 연결됨
- Phase 2: 제한 실행 안정화
  - 진입 조건: Phase 1 완료 및 기본 테스트 통과
  - 완료 조건: Execution Agent가 `module_contract` 기반으로 동작하고, `run_action`가 계약 범위를 준수하며, `git_diff_symbols`가 변경 심볼을 추적
- Phase 3: 회복력 강화
  - 진입 조건: Phase 2에서 충돌/실패 패턴이 누적되기 시작
  - 완료 조건: `git_recover_state`, `git_resolve_conflict`, pre-commit/commit-msg hook이 기본적으로 적용되어 자동 복구/안전 복구 경로가 작동
- Phase 4: 운영 정착
  - 진입 조건: Phase 3에서 재현성 높은 복구가 검증됨
  - 완료 조건: 고객 미팅형 대화 템플릿 표준화, 요구사항 태그 검색으로 자산 추적 리포트가 생성되고, 로컬-only 승인 플로우가 유지됨

## 오늘 바로 실행할 운영 규칙(작업 지침)
- 오늘부터 적용되는 기본 규칙:
  - 어떤 작업도 `요구사항 태그 + 성공 기준` 없이 실행하지 않는다.
  - 한 번에 1개 모듈 변경 원칙.
  - 테스트 통과 없이 summarize 단계로 넘어가지 않는다.
  - Git 상태가 dirty이면 핵심 실행 전 사전 동기화(필요 시 임시 보류).

## 지금까지 구현 반영(2026-02-14)

- 코드 골격 확정:
  - 로컬 stdio MCP 서버 진입점: `cmd/codex-mcp/main.go`
  - 워크플로우 상태/전이 모델: `internal/server/types.go`
  - JSON-RPC 처리 + 세션 라우팅: `internal/server/server.go`
  - 도구 정의/핵심 액션: `internal/server/tools.go`
- 실행 환경 내재화:
  - Go 부재 시 `.codex-mcp/.tools/go`에 자동 설치 (`scripts/bootstrap-go.sh`)
  - 빌드/실행 자동화 (`Makefile`, `scripts/run-local.sh`)
  - 세션 상태 영속화: `.codex-mcp/state/sessions.json`
- 1차 운영 결정을 코드로 고정:
  - 사용자 확인이 필요한 단계는 명시적 도구 호출(`approve_plan`, `clarify_intent`)로 분리
  - 실패 전이(`failed`)를 허용해 재회의/재실행 경로가 남도록 유지
  - Git 관련 보조 액션을 별도 도구로 분리해 “자동 실행 + 수동 판단” 경계 유지
- 다음 단계 후보(자동으로 이어서 진행):
  - 세션 상태 스키마를 "계약 기준(입력/출력 명세)" 중심으로 단축
  - `run_action` 결과를 기반으로 최소 단위 unit test 추가
  - `git` 상태와 세션 상태의 일관성 체크 규칙 강화 (예: 변경 후 커밋 메타데이터 태깅)

## 현재 진행 업데이트(2026-02-14)

- 문서/실행성 강화 반영:
  - Codex MCP 연결 예시(JSON) 섹션 추가.
  - Go bootstrap 동작 명세 보강(설치 버전 조건부 사용).
  - 스펙 테스트/CI 대비 최소 단위 테스트 4개 추가:
    - `tools/list` 결과 누락/중복 체크
    - initialize/unknown method 처리
    - command 허용 목록 판정
    - 상태 영속/복원 회귀
- MCP 실행 안정성 개선:
  - `run_action`, `verify_result`에서 명령 허용 판정을 공통화.
  - 환경변수/상대경로가 붙은 command에서 허용 바이너리 판정이 되도록 보정.
- 다음 바로 할 일:
  - `approve_plan`/`run_action` 사이 게이트를 더 엄격하게 바인딩(요청 요약-승인 기록-리스크 태그 일치 검사)
  - `git_get_state`와 세션 상태의 상호 일관성 체크 규칙 자동화

## 진행 업데이트(2026-02-14, 계속 반영)

- 승인 게이트 고정(현재 반영):
  - `approve_plan`에 `requirement_tags`와 `success_criteria` 입력을 허용.
  - 사용자가 `approve=true`를 보냈을 때:
    - 태그 누락이면 즉시 `failed` 전이.
    - 의도에 `success_criteria`가 명시된 경우, 승인 입력이 그 기준과 의미적으로 매칭되어야 함.
    - 미충족 사유와 다음 액션(`blocking_reasons`, `required_actions`)을 구조화해서 반환.
  - `clarify_intent`에서 `requirement_tags`/`success_criteria`를 미리 수집해 승인 부담을 줄이는 동작도 지원.
- 상태/요청 모델 보강:
  - 세션에 `requirement_tags`, `approved_criteria` 저장 추가.
  - intent 파서에서 명시적 `성공기준:`이 있으면 기본 성공 기준을 대체해 “암묵적 기준” 혼재를 방지.
- 테스트/검증 상태:
  - `go test ./...` 통과
  - `make build` 통과
  - 승인 게이트 실패/성공 케이스 단위테스트가 현재 동작 검증 범위에 포함됨
- 프로세스 관리 강화(추가 반영):
  - `summarize` 호출 시 `verify_run -> summarized` 상태 전이를 실제로 반영하도록 수정.
  - JSON-RPC 경유 워크플로우 통합 테스트 추가(`tools/call` 연쇄 호출로 `summarized` 도달 검증).
  - 운영용 smoke 스크립트 추가: `scripts/smoke-workflow.sh`
  - 실행 편의 타깃 추가: `make test`, `make smoke`
- 운영 자동화 내재화(추가 반영):
  - 세션 상태 이력 `step_history`를 저장해 상태 전이 추적성 강화.
  - 세션 관제 도구 `get_session_status` 추가(현재 단계/다음 액션/이력 조회).
  - `ingest_intent` 호출 시 기존 세션 워크플로우 상태를 초기화해 재실행 누적 부작용 제거.
  - Git 훅 자동 설치 스크립트 추가: `scripts/install-hooks.sh`
  - 훅 정책:
    - pre-commit: `make test`
    - commit-msg: `goal_id:` 메타데이터 강제
  - 원클릭 초기화 타깃 추가: `make setup` (build+test+smoke+install-hooks)
  - 에이전트 전용 설치 타깃 추가: `make agent-install`
  - 설치 문서 추가: `AGENT_INSTALL.md`
  - 설치 스크립트에서 Codex 설정(`~/.codex/config.toml`)에 `mcp_servers.codex-troller` 자동 등록
  - 인터뷰 시작용 엔트리포인트 도구 추가: `start_interview`
  - 스킬 기반 시작점 추가: `codex-troller-autostart` (스킬 호출만으로 인터뷰 시작)
  - 설치 시 스킬 파일을 `~/.codex/skills/codex-troller-autostart/`에 자동 배포
  - Codex MCP stdio framing(`Content-Length`) 호환 처리 추가로 startup `Transport closed` 오류 해소
  - 스킬 파일 YAML frontmatter 보강으로 skill 로딩 오류 해소
  - 인터뷰 질문을 정책형 문구에서 상황형 문구로 전환(비기술 사용자도 바로 답변 가능한 질문 세트)
- 문서 반영 규칙 준수:
  - "합의본" 대신 문서를 실행 상태별로 계속 갱신(Phase 기준), 매 변경 직후 바로 추적 항목 추가.

## 정정 반영(2026-02-14, 사용자 피드백)

- 범위 정정:
  - 과거 테스트 대화에서 사용된 "작품 감상 웹" 문맥은 제품 요구사항이 아니라 테스트 입력 사례임.
  - 서버 기본 동작/질문/문서는 특정 도메인(웹 전시, 게임 등)에 종속되지 않아야 함.
- 구현 정정:
  - `start_interview`는 raw intent가 이미 충분하면 바로 `generate_plan`으로 이동하고, 부족하면 `clarify_intent` 질문만 반환.
  - `clarify_intent`는 단발성이 아니라 반복 루프(`needs_more_info -> clarify_intent`)를 지원해, 누락 정보가 채워질 때까지 후속 질문을 생성.
  - `clarify_intent` 응답을 `intent` 구조체(목표/범위/제약/성공기준)에 직접 반영하도록 수정.
  - 쉘 명령 허용 로직에서 체인 실행(`;`, `&&`, `||`, `|`)을 차단해 신뢰성과 안전성을 강화.
  - smoke 스크립트 입력은 고정값이 아니라 환경변수(`SMOKE_GOAL`, `SMOKE_SCOPE`, `SMOKE_CONSTRAINT`, `SMOKE_CRITERIA`, `SMOKE_TAGS_CSV`)로 오버라이드 가능하게 변경.
- 운영 기준 정리:
  - 완료 조건은 "테스트 성공" + "사용자 명시적 승인(`record_user_feedback`)".
  - 인터뷰 질문은 언제나 구체적 상황 질문으로 유지하고, 추상 정책 질문은 금지.

## 정정 반영(2026-02-14, 인터뷰 방식)

- 문제점:
  - 인터뷰 질문이 "정해진 패턴"처럼 보이고, Codex의 맥락 적응성이 약하게 느껴짐.
- 조치:
  - 인터뷰를 고정 설문지에서 "적응형 단일 질문 루프"로 전환.
  - 각 턴에서 누락된 정보(목표/범위/제약/완료기준) 중 우선순위 1개만 질문.
  - 도메인 신호(프론트/백엔드/게임/유지보수/자동화)를 감지해 질문 문구와 예시를 상황에 맞게 변경.
  - 정보가 충분해지면 추가 질문 없이 `generate_plan`으로 즉시 진행.

## 정정 반영(2026-02-14, 반복 루프/재개/모델 라우팅)

- 핵심 루프:
  - 두루뭉술한 아이디어 -> 대화로 구체화 -> 빠른 mockup -> 재논의 -> 구현/검증 -> 사용자 승인.
  - 사용자가 "이 정도면 완료"라고 승인할 때까지 루프를 반복.
- 재개/복구:
  - 세션이 종료되어도 상태를 유지하고, 재호출 시 저장 상태와 현재 코드 상태를 `git + footprint digest`로 비교.
  - 변경량이 큰 경우 이전 문맥 유지(`keep_context`) / 새 시작(`restart_context`)을 사용자에게 선택받음.
- 에이전트 모델 역할 분리:
  - 고객 대화 담당: `gpt-5.2`
  - 오케스트레이터/리뷰: `gpt-5.3-codex`
  - 함수/모듈 구현 워커: `gpt-5.3-codex-spark`

## 최신 반영(2026-02-14, 팀장 토론 프로토콜 + 즉시 저장)

- 인터뷰/질문 기준 정정:
  - 고정 스크립트 질문을 폐기하고, 누락/위험/권한 경계 기준으로 "지금 꼭 필요한 질문 1개"만 생성.
  - 사용자 답변이 `알아서/무관`이면 해당 항목을 자동결정 후보로 전환하고, 다른 핵심 항목으로 이동.
  - 구현 단위 질문으로 점프하지 않고, 얕은 확인 -> 필요 시만 깊게 파고드는 규칙으로 유지.
- 팀장 토론(council) 선행 게이트 추가:
  - `generate_plan` 진입 전, 반드시 `council_finalize_consensus`가 완료되어야 함.
  - 역할: `ux_director`, `frontend_lead`, `backend_lead`, `db_lead`, `asset_manager`, `security_manager`.
  - 절차:
    - 병렬 발제 시작(`council_start_briefing`)
    - 역할별 독립 발제 제출(`council_submit_brief`)
    - 진행자 요약/안건화(`council_summarize_briefs`)
    - 안건별 발언권 요청/부여/발언(`request -> grant -> publish`)
    - 전 역할 `pass/raise` 수집(`council_respond_topic`)
    - 전 역할 `pass`일 때만 안건 종결(`council_close_topic`)
    - 모든 안건 종결 후 전체 합의(`council_finalize_consensus`)
- 즉시 로컬 저장 규칙:
  - 토론 이벤트는 SQLite(`.codex-mcp/state/council.db`)에 즉시 기록(메시지/투표/발언권/안건 상태).
  - 상담 제안 이력(`version`, `options`, `user_decision`, `user_feedback`)도 SQLite에 즉시 기록.
  - compact 이후에도 `council_get_status`로 현재 상태를 복구 가능.
  - "모아서 나중에 쓰기" 방식은 금지하고 이벤트 단위로 저장.
- 테스트/검증 반영:
  - 단위 테스트에 council 합의 경로를 포함해 `generate_plan` 선행 조건 검증.
  - smoke 테스트를 council 전 과정을 포함한 end-to-end 경로로 교체.
  - 상담가 제안 루프를 필수 단계로 고정:
    - council 1차 초안 이후 `clarify_intent`에서 제안안을 제시.
    - 고정 메뉴 선택 강요 없이, 초안 + 한 번에 한 질문 방식으로 사용자 반응을 받아 즉시 이력 갱신 후 반복.
    - 충돌/트레이드오프 신호가 감지되면 상담가 단독 결정 대신 council 재토론으로 회귀.

## 최신 반영(2026-02-14, council-first 재배치)

- 사용자 피드백 반영:
  - 초기 목적 수집 후 바로 1차 팀장 토론(council)으로 진입.
  - council 초안/안건을 확보한 뒤 상담을 재개해 세부 요구사항과 제안 정렬을 진행.
- 현재 기본 순서:
  - `ingest_intent -> council_* -> clarify_intent(구체화+제안 반응 루프) -> generate_plan`
- 이유:
  - 초기 초안(발제/안건) 없이 상담만 길어지는 문제를 줄이고, 사용자가 빠르게 선택/수정 포인트를 잡도록 하기 위함.

## 최신 반영(2026-02-14, 사용자 이해도 기반 적응형 상담/자율권)

- 사용자 이해도 프로필 도입:
  - 세션에 `user_profile`(`overall`, `domain_knowledge`, `response_need`, `technical_depth`)을 저장.
  - `start_interview`/`ingest_intent`에서 사전 프로필 입력을 받고, 미입력 시 intent 문장 기반으로 1차 추론.
- 응답 부담/질문 깊이 자동 조절:
  - 이해도 낮음 + `response_need=low`인 경우, 고위험이 아니면 범위/제약/완료기준을 자동결정 후보로 확장.
  - 이해도 높음인 경우, 동일 안건에서도 질문 문구를 구조/리스크/검증 기준 중심으로 구체화.
- 팀장 에이전트 자율권 조절:
  - `council_start_briefing`의 역할별 발제 프롬프트에 `autonomy_level`, `consult_depth`, `policy_note`를 포함.
  - 익숙하지 않은 도메인은 팀장 자율권을 높이고 상담가 전달은 추상 레벨로, 익숙한 도메인은 근거/트레이드오프를 구체 전달.
- 상담 초안 표현 차등화:
  - 초안 설명에서 사용자 이해도에 따라 추상/균형/기술 중심 서술을 분기.
  - 후속 질문도 이해도 기반으로 문구를 조절하되, 여전히 "한 번에 한 질문" 원칙 유지.
- 전문성 추정 신뢰도 관리:
  - `user_profile.confidence`(0~1) + `user_profile.evidence`를 세션에 유지.
  - confidence가 낮으면(기본 임계: 0.55 미만) 자동결정 확장을 억제하고 team autonomy를 보수적으로 제한.

## 최신 반영(2026-02-14, 구현 검증에 Visual Reviewer + UX Director 회의 게이트 추가)

- 검증 절차 확장:
  - `verify_result` 이후 `visual_review` 단계를 추가.
  - 기존 흐름: `... -> verify_run -> record_user_feedback`
  - 변경 흐름: `... -> verify_run -> visual_review -> record_user_feedback`
- 렌더링 MCP 자동 감지:
  - 세션 입력(`start_interview`/`ingest_intent`/`verify_result`)으로 `available_mcps`, `available_mcp_tools`를 받음.
  - 목록에 렌더링 키워드(`playwright`, `screenshot`, `browser` 등)가 있으면 렌더링 가능 상태로 판단.
  - UI/UX 성격 작업(frontend/game/화면·페이지·반응형 키워드 포함)에서만 Visual Reviewer 게이트를 강제.
- Visual Reviewer 완료 조건:
  - 최소 1개 이상의 시각 검토 증거(artifact 또는 findings/reviewer_notes).
  - 제작본 기반 `ux_director_summary` 필수.
  - `ux_decision`(`pass|raise`) 기록.
- 상태/재개:
  - `visual_review` 상태(`required`, `status`, `artifacts`, `findings`, `ux_director_summary`)를 세션에 저장.
  - `record_user_feedback`는 Visual Reviewer 미완료 시 차단.

## 최신 반영(2026-02-14, 동적 팀장 구성/참여)

- 세션별 팀장 roster(`council_managers`) 도입:
  - 기본 팀장을 유지하되, 프로젝트 특성에 따라 역할을 동적으로 추가/교체/제거 가능.
  - 역할/도메인/모델을 함께 저장하고, 세션 재개 시 동일 구성을 유지.
- 신규 도구:
  - `council_configure_team` (`append|replace|remove`)로 팀장을 사전 구성.
  - `council_start_briefing`에서도 `manager_mode` + `managers`를 one-shot으로 받아 즉시 구성 변경 가능.
- 참여/합의 로직 동적화:
  - 발제 제출, 발언권 요청, 안건 응답/종결은 고정 역할 목록이 아니라 현재 세션의 활성 팀장 목록 기준으로 판정.
  - 따라서 프로젝트마다 필요한 팀장만 참여시키는 토론 운영이 가능.

## 최신 반영(2026-02-14, 설치 인터뷰/동의 게이트 강화)

- `make agent-install` 설치 흐름에 동의/설문 절차를 추가.
  - LLM 설치 동의(약관 미동의 시 즉시 중단)
  - 설치 범위 선택(global/local)
  - Playwright MCP 설치 동의(`mcp_servers.playwright` 등록 여부)
  - 사용자 전문성 간략 설문(overall/response_need/technical_depth/domain hints)
- 설치 설문 결과를 기본 사용자 프로필 JSON으로 저장하고 launcher가 환경변수로 주입.
  - `CODEX_TROLLER_DEFAULT_PROFILE_PATH`
  - `start_interview`/`ingest_intent`에서 명시적 프로필이 없으면 기본값 자동 적용.
