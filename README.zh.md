# codex-troller MCP 本地服务器

本文件为中文版本，规范文档以英文 `README.md` 为准。

相关文档:
- 英文: `README.md`
- 韩文: `README.ko.md`
- 日文: `README.ja.md`
- 设计讨论（英文）: `mcp-server-discussion.md`

## 简介

`codex-troller` 是面向 Codex CLI 的本地 Go MCP 服务器。
核心目标是通过结构化流程提升可靠性，并更准确地对齐用户意图。

## 主要流程

- 会话状态机:
  - `received -> intent_captured -> (council_briefing/discussion/finalized) -> plan_generated -> mockup_ready -> plan_approved -> action_executed -> verify_run -> visual_review -> record_user_feedback -> summarized`
- 默认路径:
  - 意图收集 -> council 初稿 -> 咨询细化循环 -> 计划/原型 -> 执行 -> 验证 -> 用户最终确认

## 快速开始

```bash
make agent-install
make setup
make test
make smoke
```

## 安装器确认项

`make agent-install` 会依次确认：

1. 条款同意（必需）
   - 软件尚未充分验证
   - 问题/损失责任由用户自行承担
   - 已确认许可证为 GNU GPL v3.0
2. 安装范围
   - `global` 或 `local`
3. 是否安装 Playwright MCP
   - 参考: `https://github.com/microsoft/playwright-mcp`
4. 用户专业度初始配置
   - `overall`, `response_need`, `technical_depth`, `domain_knowledge`

## 主要工具

- `start_interview`
- `ingest_intent`
- `clarify_intent`
- `council_configure_team`
- `council_start_briefing`
- `generate_plan`
- `run_action`
- `verify_result`
- `visual_review`
- `summarize`

完整说明请查看 `README.md`。
