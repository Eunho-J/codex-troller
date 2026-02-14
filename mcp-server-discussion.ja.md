# Codex MCP サーバー設計ノート

このドキュメントは日本語版です。正本は `mcp-server-discussion.md`（英語）です。

## 要点

- 固定仕様ではなく、運用しながら更新する設計ログ。
- 目的は、あいまいな要求を構造化して実行信頼性を高めること。
- `generate_plan` 前に council 合意を必須化。
- UI/UX タスクでは条件付きで `visual_review` を必須化。
- 最終完了には `record_user_feedback(approved=true)` が必要。
- チーム構成はセッションごとに動的変更可能（`council_configure_team`）。

詳細は英語版を参照してください。
