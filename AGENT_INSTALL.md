# Agent Install Guide

이 문서는 사용자가 "설치해줘"라고 요청했을 때, 기본 상태의 Codex agent가 추가 질문 없이 실행할 설치 절차를 정의한다.

## 목표

- 사용자가 직접 설정 파일을 만지지 않아도 설치가 끝나야 한다.
- 설치는 재실행(idempotent) 가능해야 한다.
- 설치 후 즉시 동작 검증까지 자동으로 수행해야 한다.

## Agent Standard Procedure

1. 저장소 루트에서 설치 명령 실행:
   - `make agent-install`
2. 설치 완료 후 검증:
   - `make smoke`
   - smoke는 council 토론 합의(`council_finalize_consensus`)와 사용자 승인 게이트(`record_user_feedback`)까지 포함한 완료 경로를 검사해야 한다.
3. 설치 결과 확인:
   - 바이너리 존재: `.codex-mcp/bin/codex-mcp`
   - 토론 DB 존재: `.codex-mcp/state/council.db`
   - Codex 설정 반영: `~/.codex/config.toml`에 `[mcp_servers.codex-troller]` 섹션 존재
   - 스킬 반영: `~/.codex/skills/codex-troller-autostart/SKILL.md` 존재

## What `make agent-install` Does

- `make setup` 실행:
  - 빌드: `make build`
  - 테스트: `make test`
  - 워크플로우 스모크: `make smoke`
  - Git hooks 설치: `make install-hooks`
- Codex 설정 파일 자동 반영:
  - 기본 경로: `~/.codex/config.toml`
  - 환경변수 override: `CODEX_CONFIG_PATH=/path/to/config.toml`
  - 추가되는 설정:
    - `[mcp_servers.codex-troller]`
    - `command = "<repo>/.codex-mcp/bin/codex-mcp"`
- 스킬 자동 설치:
  - `skills/codex-troller-autostart/SKILL.md`를 `~/.codex/skills/codex-troller-autostart/`로 복사
- 기본 역할-모델 라우팅 정책 탑재:
  - 인터뷰: `gpt-5.2`
  - 오케스트레이터/리뷰: `gpt-5.3-codex`
  - 구현 워커: `gpt-5.3-codex-spark`

## Idempotency Rules

- 동일 환경에서 `make agent-install`을 반복 실행해도 설정 중복이 생기지 않아야 한다.
- 기존 `[mcp_servers.codex-troller]` 섹션이 있으면 삭제 후 최신 경로로 재등록한다.
- 스킬 디렉터리는 덮어써서 항상 최신 스킬 정의를 유지한다.

## Failure Handling

- Go 부재: `scripts/bootstrap-go.sh`가 `.codex-mcp/.tools/go`에 자동 설치.
- 네트워크 실패: 설치 중단 후 에러 반환(재시도 가능).
- 권한 문제: config 파일 경로 권한 확인 후 재실행.
- Git 저장소가 아닌 폴더: hook 설치를 자동 skip하고 나머지 설치/검증은 계속 진행.
