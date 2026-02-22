# MCPist テスト仕様書（spc-tst）

## ドキュメント管理情報

| 項目 | 値 |
|------|-----|
| Status | `draft` |
| Version | v3.0 (Sprint-012) |
| Note | Test Specification — 現行実装に基づく全面改訂 |

---

## 概要

本ドキュメントは、MCPist のテスト戦略と現行テスト構成を定義する。

---

## テスト構成

### Server (Go)

| 項目 | 値 |
|---|---|
| フレームワーク | Go 標準 `testing` パッケージ |
| 実行コマンド | `go test -v -race ./...` |
| パターン | テーブル駆動テスト (`t.Run()`) |
| 外部依存 | 環境変数がない場合 `t.Skip()` でスキップ |

**テストファイル:**

| ファイル | テスト対象 |
|---|---|
| `internal/modules/validate_test.go` | パラメータバリデーション |
| `pkg/supabaseapi/client_test.go` | Supabase API クライアント |
| `pkg/asanaapi/client_test.go` | Asana API クライアント |

### Console (TypeScript)

| 項目 | 値 |
|---|---|
| フレームワーク | Vitest |
| 実行コマンド | `vitest` |
| 状態 | 設定済み、テストファイルは未作成 |

### Worker (TypeScript)

| 項目 | 値 |
|---|---|
| テスト | なし |
| 型チェック | `tsc --noEmit` |

---

## CI パイプライン

GitHub Actions (`ci.yml`)。トリガー: workflow_dispatch (手動)。

| ジョブ | 内容 |
|---|---|
| lint-server | golangci-lint |
| test-server | `go test -v -race ./...` |
| build-server | `go build -v ./...` |
| lint-console | ESLint |
| build-console | `pnpm build` |
| lint-worker | `tsc --noEmit` |

Console と Worker のテスト実行ジョブは未実装。

---

## テスト実行方法

### ローカル

```bash
# 全アプリのテスト (Turborepo)
pnpm test

# Server のみ
cd apps/server && go test -v ./...

# Server (環境変数付き)
cd apps/server && dotenv -e ../../.env.dev -- go test -v ./...

# Console のみ
cd apps/console && pnpm test
```

### CI

手動トリガー (`workflow_dispatch`) で全ジョブを実行。

---

## テスト方針

### 単体テスト

| 対象 | テスト内容 |
|---|---|
| パラメータバリデーション | 必須フィールド、型チェック |
| 暗号化 / 復号 | AES-256-GCM のラウンドトリップ |
| JSON-RPC パーサー | メソッド・パラメータ解析 |
| レート制限 | sliding window の動作 |

### 結合テスト

| 対象 | テスト内容 |
|---|---|
| 外部 API クライアント | 実 API への疎通 (環境変数必須) |
| batch 実行 | DAG 解決、並列/依存実行 |
| 認可フロー | Gateway JWT → ユーザー解決 → 権限チェック |

結合テストは環境変数がない場合スキップされる。

### セキュリティテスト

| テスト項目 | 検証内容 |
|---|---|
| JWT 偽造 | 不正署名の拒否 |
| Gateway トークン欠如 | 401 返却 |
| 無効化モジュール呼び出し | -32001 PermissionDenied |
| 日次上限超過 | -32002 UsageLimitExceeded |
| Stripe Webhook 署名偽造 | 拒否 |

---

## 関連ドキュメント

| ドキュメント                                             | 内容        |
| -------------------------------------------------- | --------- |
| [spec-systems.md](./spec-systems.md)               | システム仕様書   |
| [spec-infrastructure.md](./spec-infrastructure.md) | インフラ仕様書   |
| [spc-ops.md](spec-operation.md)                    | 運用仕様書     |
| [spec-security.md](./spec-security.md)             | セキュリティ仕様書 |
