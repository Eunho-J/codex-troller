# codex-troller（面向人类的 README）

> 如果你是 LLM agent，请忽略本文件并使用 [README-LLM.md](README-LLM.md)。

面向人类的多语言指南:
- [English](README.md)
- [한국어](README.ko.md)
- [日本語](README.ja.md)
- [中文](README.zh.md)

建议将项目说明和安装都交给 LLM agent 执行。

请使用下面的提示词。

说明提示词:

```text
请阅读 https://github.com/Eunho-J/codex-troller 的 README-LLM.md，并用通俗语言说明 codex-troller 是做什么的、为什么需要它、以及工作流如何运行。
```

安装提示词:

```text
请阅读 https://github.com/Eunho-J/codex-troller 的 README-LLM.md，先向我确认必需的确认项，然后把 codex-troller 的安装与配置完整执行到结束。
```

## 项目概览

`codex-troller` 是面向 Codex CLI 的本地 Go MCP 服务器。
它用于在用户目标尚不清晰时，提高 AI 辅助开发的可靠性。

## 核心目标与价值观

- 将模糊意图转化为结构化、可执行的计划。
- 在访谈 -> 规划 -> 执行 -> 验证全流程保持意图对齐。
- 在批准、权限、风险等敏感边界保留用户控制。
- 持久化状态，支持跨会话续跑。

## 整体结构与工作方式

- 意图采集与顾问式澄清循环。
- 支持动态团队编排的 council 规划。
- 小粒度执行与验证门禁。
- UI/UX 任务在检测到可渲染 MCP 时启用视觉评审门禁。
- 面向长周期任务的状态存储 + git 感知重对齐。

## 设计说明

- [English Design Notes](mcp-server-discussion.md)
- [한국어 설계 노트](mcp-server-discussion.ko.md)
- [日本語設計ノート](mcp-server-discussion.ja.md)
- [中文设计说明](mcp-server-discussion.zh.md)
