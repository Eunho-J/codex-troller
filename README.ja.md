# codex-troller（人間向け README）

> LLM エージェントの場合はこの文書を無視し、最新のリモートガイド（`https://raw.githubusercontent.com/Eunho-J/codex-troller/main/README-LLM.md`）を使用してください。

人間向け多言語ガイド:
- [English](README.md)
- [한국어](README.ko.md)
- [日本語](README.ja.md)
- [中文](README.zh.md)

プロジェクト説明とインストールは、LLM エージェントに任せることを推奨します。

以下のプロンプトを使ってください。

説明プロンプト:

```text
https://raw.githubusercontent.com/Eunho-J/codex-troller/main/README-LLM.md から最新ガイドを取得して読んでください（ローカルの README-LLM.md は既定では使わない）。ネットワーク権限がない場合は、先に私へアクセス許可を求めてください。その後、現在の会話言語（日本語）で codex-troller が何をするものか、なぜ必要か、ワークフローがどう動くかをわかりやすく説明してください。
```

インストールプロンプト:

```text
https://raw.githubusercontent.com/Eunho-J/codex-troller/main/README-LLM.md から最新ガイドを取得して読んでください（ローカルの README-LLM.md は既定では使わない）。ネットワーク権限がない場合は、先に私へアクセス許可を求めてください。その後、現在の会話言語（日本語）で必須確認事項を質問し、最新の GitHub リポジトリ基準で codex-troller のインストールと設定を最後まで実行してください。
```

## プロジェクト概要

`codex-troller` は Codex CLI 向けのローカル Go ベース MCP サーバーです。
ユーザー目標が曖昧な状態から始まっても、AI 開発作業の信頼性を高めるために作られています。

## コア目的と価値観

- 曖昧なユーザー意図を構造化された実行計画に変換する。
- インタビュー -> 計画 -> 実装 -> 検証まで意図整合性を維持する。
- 承認・権限・リスクの境界でユーザー制御を確保する。
- セッションをまたいでも再開可能な状態を保持する。

## 全体構造と動作方式

- 意図取得とコンサル型の明確化ループ。
- 動的にチーム編成できる council ベース計画。
- 小さな単位での実行と検証ゲート。
- UI/UX タスクでは描画可能 MCP がある場合にビジュアルレビューゲートを適用。
- 長期作業のための状態保存 + git 連携再調整。

## 設計ノート

- [English Design Notes](mcp-server-discussion.md)
- [한국어 설계 노트](mcp-server-discussion.ko.md)
- [日本語設計ノート](mcp-server-discussion.ja.md)
- [中文设计说明](mcp-server-discussion.zh.md)
