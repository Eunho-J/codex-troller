# codex-troller MCP ローカルサーバー

ガイド:
- [English Guide](README.md)
- [한국어 가이드](README.ko.md)
- [日本語ガイド](README.ja.md)
- [中文指南](README.zh.md)

設計ノート:
- [English Design Notes](mcp-server-discussion.md)
- [한국어 설계 노트](mcp-server-discussion.ko.md)
- [日本語設計ノート](mcp-server-discussion.ja.md)
- [中文设计说明](mcp-server-discussion.zh.md)

## インストール (MCP サーバー、スキルではない)

このリポジトリは Codex スキルパッケージではなく、**MCP サーバープロジェクト**です。
既定かつ推奨の方法は LLM 主導インストールです。
このリポジトリに `skill-installer` は使わないでください。

既定の Codex エージェントに次のプロンプトを貼り付けてください:

```text
Install `codex-troller` from `https://github.com/Eunho-J/codex-troller`.

Requirements:
- Treat this repository as an MCP server project, not a skill package.
- Do not use `skill-installer`.
- Read and follow this repository `README.md` install/setup steps.
- どのインストールコマンドを実行する前でも、以下をユーザーに確認し、明示的な回答を得てから進める:
  1) 規約同意（`十分に検証されていない`, `問題発生時はユーザー責任`, `GNU GPL v3.0 を認識`）;
  2) インストール範囲（`global` または `local`）;
  3) Playwright MCP 導入可否（`yes`/`no`）;
  4) 初期専門性プロファイル（`overall`, `response_need`, `technical_depth`, `domain_knowledge`）。
- 規約に同意しない場合はインストールを中止する。
- Run interactive `make agent-install` (terms consent, install scope, optional Playwright MCP registration, expertise survey).
- Run `make setup` and verify with `make smoke`.
- If a step fails, resolve it and continue until installation and smoke verification both pass.
```

`codex-troller` は Codex CLI 向けのローカル Go MCP サーバーです。
目的は、作業を構造化して信頼性と意図整合性を高めることです。

## コアコンセプト

- セッションワークフロー:
  - `received -> intent_captured -> (council_briefing/discussion/finalized) -> plan_generated -> mockup_ready -> plan_approved -> action_executed -> verify_run -> visual_review -> record_user_feedback -> summarized`
- 既定フロー:
  - 意図収集 -> council 下書き -> 相談ループで明確化 -> 計画/モックアップ -> 実行 -> 検証 -> ユーザー承認
- 動的 council チーム:
  - チームリードはセッション単位で、append/replace/remove が可能
- 永続実行:
  - 状態は JSON + SQLite に保存され、リポジトリ footprint と reconcile できる
- ビジュアルレビューゲート:
  - UI/UX タスクで描画可能 MCP ツールが検出された場合、最終承認前に `visual_review` が必須

## クイックスタート

```bash
make agent-install   # 対話型インストール: 規約 -> 範囲(global/local) -> Playwright MCP 同意 -> 専門性アンケート
make setup          # build + test + smoke + フック導入
make build          # Go が無い場合はローカル Go ツールチェーンを bootstrap
make test           # 単体テスト
make smoke          # JSON-RPC end-to-end スモークテスト
make install-hooks  # git フック導入
make run-binary     # ビルド済みバイナリ実行 (.codex-mcp/bin/codex-mcp)
# または
make run-local      # bootstrap + build + run
```

インストール後、Codex から `$codex-troller-autostart` を呼び出せます。

## インストーラーの挙動

`make agent-install` は次を強制します。

1. 規約同意ゲート (必須)
   - ソフトウェアは十分に検証されていない
   - 問題/損害の責任は利用者が負う
   - GNU GPL v3.0 ライセンスを認識
2. インストール範囲の選択
   - `global` (`~/.codex/...`) または `local` (`<repo>/.codex/...`)
3. Playwright MCP 登録（任意）
   - source: `https://github.com/microsoft/playwright-mcp`
4. 初期ユーザー専門性アンケート
   - `overall`, `response_need`, `technical_depth`, `domain_knowledge`
5. 既定プロファイル保存
   - ランタイムで launcher 環境変数を通して注入

非対話モード:

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

## 手動実行例

```bash
echo '{"jsonrpc":"2.0","id":1,"method":"initialize"}' | .codex-mcp/bin/codex-mcp

echo '{"jsonrpc":"2.0","id":2,"method":"tools/list"}' | .codex-mcp/bin/codex-mcp

cat <<'EOF2' | .codex-mcp/bin/codex-mcp
{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"ingest_intent","arguments":{"raw_intent":"goal: add error handling and tests to login API module","session_id":""}}}
EOF2
```

## プロセス検証

- 単体テスト: `make test`
- end-to-end スモーク: `make smoke`
- スモークは council 合意と最終ユーザー承認ゲートを含む全ルートを検証します。

スモーク環境変数:
- `SMOKE_GOAL`
- `SMOKE_SCOPE`
- `SMOKE_CONSTRAINT`
- `SMOKE_CRITERIA`
- `SMOKE_TAGS_CSV`
- `SMOKE_AVAILABLE_MCPS`
- `SMOKE_AVAILABLE_MCP_TOOLS`

## 利用可能ツール

### ワークフロー

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

### Git ヘルパー

- `git_get_state`
- `git_diff_symbols`
- `git_commit_with_context`
- `git_resolve_conflict`
- `git_bisect_start`
- `git_recover_state`

## ランタイム状態

- セッション状態: `.codex-mcp/state/sessions.json`
- council DB: `.codex-mcp/state/council.db`
- インストーラープロファイル（既定）:
  - global: `~/.codex/codex-troller/default_user_profile.json`
  - local: `<repo>/.codex/codex-troller/default_user_profile.json`

## 動的 council チーム例

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

## エージェントルーティング既定値

- クライアント面談: `gpt-5.2`
- オーケストレーター/レビュアー: `gpt-5.3-codex`
- ワーカー（関数/モジュール実装）: `gpt-5.3-codex-spark`
