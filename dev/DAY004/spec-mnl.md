---
title: MCPist マニュアル仕様書（spec-mnl）
aliases:
  - spec-mnl
  - MCPist-manual-specification
tags:
  - MCPist
  - specification
  - manual
document-type:
  - specification
document-class: specification
created: 2026-01-14T00:00:00+09:00
updated: 2026-01-14T00:00:00+09:00
---
# MCPist マニュアル仕様書（spec-mnl）

## ドキュメント管理情報

| 項目 | 値 |
|------|-----|
| Status | `current` |
| Version | v1.0 (DAY4) |
| Note | DAY4で新規追加 |

---

本ドキュメントは、MCPistの各種マニュアルの構成と内容を定義する。

---

## 1. マニュアル一覧

| マニュアル | 対象者 | 目的 |
|-----------|--------|------|
| 管理者ガイド | 情シス担当 | 初期セットアップ、日常管理 |
| ユーザーガイド | 一般社員 | ログイン、アカウント連携、利用方法 |
| 開発者ガイド | 開発者 | ローカル環境、開発フロー、モジュール追加 |

---

## 2. 管理者ガイド（admin-guide）

### 2.1 初期セットアップチェックリスト

- [ ] Supabase プロジェクト作成
- [ ] Supabase Auth 設定（ソーシャルログイン有効化）
  - [ ] Google
  - [ ] Microsoft
  - [ ] GitHub
- [ ] DBスキーマ適用（マイグレーション実行）
- [ ] Edge Functions デプロイ
- [ ] 外部サービス OAuth アプリ登録
  - [ ] Google Calendar
  - [ ] Microsoft Graph
  - [ ] GitHub
  - [ ] Notion
  - [ ] Jira / Confluence
- [ ] Koyeb デプロイ
  - [ ] 環境変数設定
  - [ ] ヘルスチェック確認
- [ ] Vercel デプロイ
  - [ ] 環境変数設定
- [ ] Grafana Cloud 設定
  - [ ] Loki（ログ）
  - [ ] アラート設定
- [ ] 許可リスト（allowed_users）登録
- [ ] 初回ログイン → admin 化確認

### 2.2 日常運用チェックリスト

- [ ] アラート確認（Grafana）
- [ ] ログ確認（エラー有無）
- [ ] 外部サービス障害確認（ステータスページ）

### 2.3 月次チェックリスト

- [ ] 可用性 KPI 確認
- [ ] セキュリティアップデート確認（Dependabot）
- [ ] コスト確認（各サービス無料枠内か）

### 2.4 ユーザー管理

- ロール作成手順
- ユーザーへのロール割当手順
- 権限変更手順

### 2.5 トラブルシューティング

- spec-ops §9 Runbook への参照

---

## 3. ユーザーガイド（user-guide）

### 3.1 はじめに

- MCPist とは
- 利用できる機能一覧

### 3.2 ログイン手順

- [ ] 管理者から招待を受ける（許可リスト登録）
- [ ] ログイン画面にアクセス
- [ ] ソーシャルログイン（Google / Microsoft / GitHub）
- [ ] ダッシュボード確認

### 3.3 アカウント連携手順

- [ ] `/tools` ページにアクセス
- [ ] 連携したいサービスの「連携する」をクリック
- [ ] OAuth 認可画面で許可
- [ ] 連携完了確認（緑バッジ）

### 3.4 LLM クライアントからの利用

- Claude Code での設定方法
- Cursor での設定方法
- 基本的な使い方（ツール呼び出し例）

### 3.5 よくある質問

- 連携が切れた場合
- 権限が足りない場合
- 問い合わせ先

---

## 4. 開発者ガイド（dev-guide）

### 4.1 ローカル環境セットアップ

- [ ] リポジトリクローン
- [ ] Go 1.22 インストール
- [ ] Node.js 20 インストール
- [ ] `.env.local` 作成
- [ ] Supabase CLI インストール
- [ ] ローカル Supabase 起動
- [ ] マイグレーション適用
- [ ] テストデータ投入（Seed）

### 4.2 開発フロー

```
1. dev から feat/xxx ブランチ作成
2. 開発・コミット
3. ローカルテスト実行
4. PR 作成（feat → dev）
5. CI パス確認
6. レビュー・マージ
```

### 4.3 テスト実行

- 単体テスト: `go test ./...`
- 結合テスト: `go test -tags=integration ./...`
- E2E テスト: `npx playwright test`

### 4.4 モジュール追加手順

- [ ] `modules/` 配下にディレクトリ作成
- [ ] Module インターフェース実装
- [ ] ツール定義（inputSchema, outputSchema）
- [ ] 単体テスト作成
- [ ] 結合テスト作成
- [ ] module_registry に登録

### 4.5 デプロイ手順

- dev → main PR 作成
- Squash merge
- deploy.yml 手動実行
- Smoke Test 確認

### 4.6 トラブルシューティング

- ローカル環境の問題
- CI 失敗時の対応
- デプロイ失敗時の対応

---

## 5. 関連ドキュメント

| ドキュメント | 内容 |
|-------------|------|
| [要件仕様書](spec-req.md) | 要件定義 |
| [システム仕様書](spec-sys.md) | システム全体像 |
| [設計仕様書](spec-dsn.md) | 詳細設計 |
| [インフラ仕様書](spec-inf.md) | インフラ構成 |
| [運用仕様書](spec-ops.md) | 運用設計 |
| [テスト仕様書](spec-tst.md) | テスト設計 |
