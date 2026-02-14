# codex-troller (사람용 README)

> LLM 에이전트라면 이 문서는 건너뛰고 최신 원격 가이드(`https://raw.githubusercontent.com/Eunho-J/codex-troller/main/README-LLM.md`)를 사용하세요.

사람용 다국어 가이드:
- [English](README.md)
- [한국어](README.ko.md)
- [日本語](README.ja.md)
- [中文](README.zh.md)

프로젝트 설명과 설치는 LLM 에이전트에게 맡기는 것을 권장합니다.

아래 프롬프트를 사용하세요.

설명 프롬프트:

```text
https://raw.githubusercontent.com/Eunho-J/codex-troller/main/README-LLM.md 에서 최신 가이드를 가져와 읽어줘(기본적으로 로컬 README-LLM.md는 사용하지 마). 인터넷 권한이 막혀 있으면 먼저 나에게 네트워크 접근 승인을 요청해줘. 그 다음 지금 대화 언어(한국어)로 codex-troller가 무엇을 하는지, 왜 필요한지, 워크플로우가 어떻게 동작하는지 쉽게 설명해줘.
```

설치 프롬프트:

```text
https://raw.githubusercontent.com/Eunho-J/codex-troller/main/README-LLM.md 에서 최신 가이드를 가져와 읽어줘(기본적으로 로컬 README-LLM.md는 사용하지 마). 인터넷 권한이 막혀 있으면 먼저 나에게 네트워크 접근 승인을 요청해줘. 그 다음 지금 대화 언어(한국어)로 필수 확인 항목을 질문하고, 최신 GitHub 레포지토리 기준으로 codex-troller 설치/설정을 끝까지 진행해줘.
```

## 프로젝트 개요

`codex-troller`는 Codex CLI를 위한 로컬 Go 기반 MCP 서버입니다.
사용자 목표가 두루뭉술하게 시작되는 상황에서도 AI 개발 작업의 신뢰도를 높이기 위해 만들어졌습니다.

## 핵심 목적과 가치관

- 모호한 사용자 의도를 구조화된 실행 계획으로 변환합니다.
- 인터뷰 -> 기획 -> 구현 -> 검증 전 과정에서 의도 정합성을 유지합니다.
- 승인/권한/리스크 같은 민감 경계에서 사용자 통제를 보장합니다.
- 세션이 끊겨도 재개 가능한 상태를 유지합니다.

## 전체 구조와 동작 방식

- 의도 수집 및 상담형 구체화 루프.
- 동적으로 팀 구성이 가능한 council 기반 기획.
- 작은 단위 실행과 검증 게이트.
- UI/UX 작업에서 렌더링 가능한 MCP가 있으면 시각 검토 게이트 적용.
- 장기 작업을 위한 상태 저장 + git 연동 재조정.

## 설계 노트

- [English Design Notes](mcp-server-discussion.md)
- [한국어 설계 노트](mcp-server-discussion.ko.md)
- [日本語設計ノート](mcp-server-discussion.ja.md)
- [中文设计说明](mcp-server-discussion.zh.md)
