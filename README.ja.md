# codex-troller MCP ローカルサーバー

このドキュメントは日本語版です。正本は `README.md`（英語）です。

関連ドキュメント:
- 英語: `README.md`
- 韓国語: `README.ko.md`
- 中国語: `README.zh.md`
- 設計ノート（英語）: `mcp-server-discussion.md`

## 概要

`codex-troller` は Codex CLI 向けのローカル Go 製 MCP サーバーです。
目的は、作業を構造化して信頼性を高め、ユーザー意図との整合性を向上させることです。

## 主要フロー

- セッション状態:
  - `received -> intent_captured -> (council_briefing/discussion/finalized) -> plan_generated -> mockup_ready -> plan_approved -> action_executed -> verify_run -> visual_review -> record_user_feedback -> summarized`
- 既定の進行:
  - 意図収集 -> council 下書き -> 相談ループで具体化 -> 計画/モックアップ -> 実行 -> 検証 -> 最終承認

## クイックスタート

```bash
make agent-install
make setup
make test
make smoke
```

## インストーラーの確認項目

`make agent-install` は次を確認します。

1. 利用規約への同意（必須）
   - ソフトウェアは十分に検証されていない
   - 問題発生時の責任は利用者本人
   - ライセンスは GNU GPL v3.0
2. インストール範囲
   - `global` または `local`
3. Playwright MCP の導入可否
   - 参照: `https://github.com/microsoft/playwright-mcp`
4. 利用者プロファイル初期値
   - `overall`, `response_need`, `technical_depth`, `domain_knowledge`

## 主なツール

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

詳細は `README.md` を参照してください。
