# Sprint 001: 基盤構築

## 基本情報

| 項目 | 値 |
|------|-----|
| スプリント番号 | SPRINT-001 |
| 期間 | 2026-01-17 〜 2026-01-19 |
| マイルストーン | M1: 基盤構築 |
| 目標 | モノレポ構成、CI/CD、Supabase設定、ローカル開発環境 |
| 状態 | 完了 |

---

## スプリント目標

Phase 1（MVP）の基盤を構築する。

### 成果物

| タスク | 成果物 | 状態 |
|--------|--------|------|
| リポジトリ初期化 | モノレポ構成（apps/server, apps/console, apps/worker） | 完了 |
| CI/CD設定 | GitHub Actions（Lint, Test, Build） | 完了 |
| Supabase設定 | config.toml、マイグレーション | 完了 |
| ローカル開発環境 | Docker Compose、README | 完了 |

---

## タスク詳細

### T-001: リポジトリ構成の決定

**決定事項:**
- [x] 新規リポジトリを作成: `mcpist`
- [x] 現在のリポジトリ（go-mcp-dev）はプロトタイプとして保持
- [x] モノレポツール: Turborepo + pnpm

**新規リポジトリ構成:**
```
mcpist/
├── apps/
│   ├── server/         # MCP Server (Go)
│   │   ├── cmd/
│   │   ├── internal/
│   │   ├── go.mod
│   │   ├── go.sum
│   │   └── Dockerfile
│   ├── console/        # User Console (Next.js)
│   │   ├── src/
│   │   ├── package.json
│   │   └── ...
│   └── worker/         # API Gateway (Cloudflare Worker)
│       ├── src/
│       ├── wrangler.toml
│       └── package.json
├── packages/           # 共有パッケージ（将来用）
├── supabase/           # Supabase マイグレーション
├── docs/               # ドキュメント
├── .github/            # GitHub Actions
├── package.json        # ルート package.json (pnpm workspace)
├── pnpm-workspace.yaml
├── turbo.json          # Turborepo設定
└── docker-compose.yml  # ローカル開発用
```

**プロトタイプリポジトリ（go-mcp-dev）:**
- 設計ドキュメント（dev/DAY8/）
- ADR（docs/adr/）
- UIプロトタイプ（mcpist-ui*）
- 初期Goプロトタイプ（cmd/, internal/）

---

### T-002: CI/CD設定

**目的:** 自動テスト・リント・ビルドの設定

**対象ワークフロー:**

| ワークフロー | トリガー | 処理 |
|-------------|---------|------|
| lint.yml | PR, push to main | Go lint, ESLint, TypeScript check |
| test.yml | PR, push to main | Go test, Jest/Vitest |
| build.yml | PR, push to main | Go build, Next.js build, Worker build |

**前提:**
- GitHub Actions
- 各appごとのテスト・ビルド

---

### T-003: Supabase設定

**目的:** 認証・DB・Vaultの設定

**対象:**
- [x] プロジェクト作成（本番用）
- [x] Auth設定（メール/パスワード認証）
- [x] スキーマ作成（mcpistスキーマ）
- [x] RLS設定
- [x] Vault設定（トークン暗号化用）

**参考:** [spc-tbl.md](../DAY8/spc-tbl.md)

---

### T-004: ローカル開発環境

**目的:** 開発者がローカルで全スタックを動かせる環境

**構成:**
- Docker Compose
  - Supabase（supabase/cli によるローカル起動）
  - MCP Server（ホットリロード対応: Air）
- pnpm scripts
  - `pnpm dev` - 全体起動
  - `pnpm dev:server` - MCP Server のみ
  - `pnpm dev:console` - User Console のみ

---

## 作業ログ

### 2026-01-17

- [x] DAY9フォルダ作成
- [x] Sprint-001計画書作成
- [x] リポジトリ構成の決定（新規リポジトリ: mcpist）
- [x] GitHub上でmcpistリポジトリ作成（Private）
- [x] モノレポ構成セットアップ（Turborepo + pnpm）
- [x] apps/server雛形作成（Go）
- [x] apps/console雛形作成（Next.js）
- [x] apps/worker雛形作成（Cloudflare Worker）
- [x] CI/CD設定（GitHub Actions: ci.yml）
- [x] Supabase設定（config.toml、マイグレーション4件）
- [x] ローカル開発環境設定（Docker Compose、README、.env.example）

### 2026-01-18

- [x] 仕様書・テストドキュメント追加
- [x] Go MCP Server 最小実装
- [x] Notion モジュール実装
- [x] Console UI 最小実装（開発中）
- [x] サイドバー改善（リサイズ可能、スムーズなトランジション）
- [x] OAuth 2.1 + PKCE 認証フロー実装
  - 認可サーバーメタデータ（`/.well-known/oauth-authorization-server`）
  - 保護リソースメタデータ（`/.well-known/oauth-protected-resource`）
  - 認可エンドポイント（`/auth/authorize`）
  - トークンエンドポイント（`/api/auth/token`）
  - PKCEコードチャレンジ/検証

### 2026-01-19

- [x] API Key認証機能実装
  - `/api/apikey` エンドポイント（生成・取得・無効化）
  - `validate_api_key` RPC（サーバーサイド検証）
  - `get_masked_api_key` RPC（マスク表示：先頭6文字 + 末尾2文字）
- [x] トークン再生成時のVault重複問題修正
  - シークレット名にタイムスタンプを追加
- [x] トークン履歴/監査ログ機能実装
  - `oauth_token_history` テーブル追加
  - ローテーション/無効化時に履歴を記録
  - `get_my_token_history` RPC追加
- [x] MCP接続ページUI改善
  - API Keyと接続設定例を1つのセクションに統合
  - 折りたたみ可能なCollapsibleコンポーネント使用
- [x] Go認証ミドルウェア更新（API Key対応）

---

## 関連ドキュメント

| ドキュメント | 内容 |
|-------------|------|
| [spc-dev.md](../DAY8/spc-dev.md) | 開発計画書 |
| [spc-dsn.md](../DAY8/spc-dsn.md) | 設計仕様書 |
| [spc-inf.md](../DAY8/spc-inf.md) | インフラ仕様書 |
| [review.md](../DAY8/review.md) | DAY8レビュー |
