# codex-troller MCP 本地服务器

指南:
- [English Guide](README.md)
- [한국어 가이드](README.ko.md)
- [日本語ガイド](README.ja.md)
- [中文指南](README.zh.md)

设计说明:
- [English Design Notes](mcp-server-discussion.md)
- [한국어 설계 노트](mcp-server-discussion.ko.md)
- [日本語設計ノート](mcp-server-discussion.ja.md)
- [中文设计说明](mcp-server-discussion.zh.md)

## 安装（MCP 服务器，不是 Skill）

本仓库是 **MCP 服务器项目**，不是 Codex skill 包。
默认且推荐路径是 LLM 驱动安装。
请不要对本仓库使用 `skill-installer`。

请将下列提示词复制到你的默认 Codex agent：

```text
Install `codex-troller` from `https://github.com/Eunho-J/codex-troller`.

Requirements:
- Treat this repository as an MCP server project, not a skill package.
- Do not use `skill-installer`.
- Read and follow this repository `README.md` install/setup steps.
- 在执行任何安装命令前，先向用户提问并获得明确答复后再继续：
  1) 条款同意（`尚未充分验证`、`问题责任由用户承担`、`已知悉 GNU GPL v3.0`）；
  2) 安装范围（`global` 或 `local`）；
  3) 是否安装 Playwright MCP（`yes`/`no`）；
  4) 初始专业度画像（`overall`, `response_need`, `technical_depth`, `domain_knowledge`）。
- 若用户不同意条款，必须中止安装。
- Run interactive `make agent-install` (terms consent, install scope, optional Playwright MCP registration, expertise survey).
- Run `make setup` and verify with `make smoke`.
- If a step fails, resolve it and continue until installation and smoke verification both pass.
```

`codex-troller` 是面向 Codex CLI 的本地 Go MCP 服务器。
目标是通过结构化流程提升可靠性与用户意图对齐程度。

## 核心概念

- 会话流程:
  - `received -> intent_captured -> (council_briefing/discussion/finalized) -> plan_generated -> mockup_ready -> plan_approved -> action_executed -> verify_run -> visual_review -> record_user_feedback -> summarized`
- 默认路径:
  - 意图收集 -> council 草案 -> 咨询细化循环 -> 计划/原型 -> 执行 -> 验证 -> 用户确认
- 动态 council 团队:
  - team lead 按会话配置，可 append/replace/remove
- 持久执行:
  - 状态保存为 JSON + SQLite，并可与仓库 footprint 进行 reconcile
- 可视化评审门禁:
  - 在 UI/UX 任务中检测到可渲染 MCP 工具时，最终批准前必须完成 `visual_review`

## 快速开始

```bash
make agent-install   # 交互式安装: 条款 -> 范围(global/local) -> Playwright MCP 同意 -> 专业度问卷
make setup          # build + test + smoke + 安装 hooks
make build          # 若缺少 Go，会自动 bootstrap 本地 Go 工具链
make test           # 单元测试
make smoke          # JSON-RPC 端到端 smoke 测试
make install-hooks  # 安装 git hooks
make run-binary     # 运行已构建二进制 (.codex-mcp/bin/codex-mcp)
# 或
make run-local      # bootstrap + build + run
```

安装后可在 Codex 中调用 `$codex-troller-autostart`。

## 安装器行为

`make agent-install` 会强制执行：

1. 条款同意门禁（必需）
   - 软件尚未充分验证
   - 问题/损失责任由用户承担
   - 已确认 GNU GPL v3.0 许可证
2. 安装范围选择
   - `global`（`~/.codex/...`）或 `local`（`<repo>/.codex/...`）
3. Playwright MCP 注册（可选）
   - source: `https://github.com/microsoft/playwright-mcp`
4. 初始用户专业度问卷
   - `overall`, `response_need`, `technical_depth`, `domain_knowledge`
5. 默认用户画像持久化
   - 运行时通过 launcher 环境变量注入画像文件

非交互模式：

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

## 手动运行示例

```bash
echo '{"jsonrpc":"2.0","id":1,"method":"initialize"}' | .codex-mcp/bin/codex-mcp

echo '{"jsonrpc":"2.0","id":2,"method":"tools/list"}' | .codex-mcp/bin/codex-mcp

cat <<'EOF2' | .codex-mcp/bin/codex-mcp
{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"ingest_intent","arguments":{"raw_intent":"goal: add error handling and tests to login API module","session_id":""}}}
EOF2
```

## 流程验证

- 单元测试: `make test`
- 端到端 smoke: `make smoke`
- smoke 会验证包含 council 共识与最终用户批准门禁在内的完整路径。

smoke 环境变量：
- `SMOKE_GOAL`
- `SMOKE_SCOPE`
- `SMOKE_CONSTRAINT`
- `SMOKE_CRITERIA`
- `SMOKE_TAGS_CSV`
- `SMOKE_AVAILABLE_MCPS`
- `SMOKE_AVAILABLE_MCP_TOOLS`

## 可用工具

### 工作流

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

### Git 辅助工具

- `git_get_state`
- `git_diff_symbols`
- `git_commit_with_context`
- `git_resolve_conflict`
- `git_bisect_start`
- `git_recover_state`

## 运行时状态

- 会话状态: `.codex-mcp/state/sessions.json`
- council 数据库: `.codex-mcp/state/council.db`
- 安装器默认画像:
  - global: `~/.codex/codex-troller/default_user_profile.json`
  - local: `<repo>/.codex/codex-troller/default_user_profile.json`

## 动态 council 团队示例

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

## Agent 路由默认值

- 客户访谈: `gpt-5.2`
- 编排/评审: `gpt-5.3-codex`
- 工作者（函数/模块实现）: `gpt-5.3-codex-spark`
