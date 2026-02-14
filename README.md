# codex-troller (Human README)

> If you are an LLM agent, ignore this file and use [README-LLM.md](README-LLM.md).

Human guides:
- [English](README.md)
- [한국어](README.ko.md)
- [日本語](README.ja.md)
- [中文](README.zh.md)

We recommend delegating both project explanation and installation to your LLM agent.

Use the prompts below.

Explanation prompt:

```text
Read README-LLM.md in https://github.com/Eunho-J/codex-troller and explain what codex-troller does, why it exists, and how the workflow works in the same language I am using in this chat.
```

Installation prompt:

```text
Read README-LLM.md in https://github.com/Eunho-J/codex-troller, ask me the required confirmations in the same language I am using in this chat, then install and configure codex-troller end-to-end.
```

## What This Project Is

`codex-troller` is a local Go-based MCP server for Codex CLI.
It is built to make AI-assisted development more reliable when user goals start vague.

## Core Purpose And Values

- Turn ambiguous user intent into structured, executable plans.
- Keep user intent aligned through interview -> planning -> execution -> verification.
- Preserve user control at sensitive boundaries (approval, permission, risk).
- Maintain resumable state so work can continue across sessions.

## High-Level Structure And Flow

- Intent capture and consultant-style clarification loop.
- Council-style planning with dynamic team composition.
- Small-scope execution and verification gates.
- Optional visual review for UI/UX tasks when render-capable MCP tools exist.
- Persistent state + git-aware reconciliation for long-running work.

## Design Notes

- [English Design Notes](mcp-server-discussion.md)
- [한국어 설계 노트](mcp-server-discussion.ko.md)
- [日本語設計ノート](mcp-server-discussion.ja.md)
- [中文设计说明](mcp-server-discussion.zh.md)
