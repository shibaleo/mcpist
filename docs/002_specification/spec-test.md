# MCPist テスト仕様書（spc-tst）

## ドキュメント管理情報

| 項目 | 値 |
|------|-----|
| Status | `draft` |
| Version | v3.1 (Sprint-012) |
| Note | Test Specification — テストファイル一覧・テスト方針を拡充 |

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

| ファイル | テスト対象 | 関数数 |
|---|---|---|
| `internal/db/encryption_test.go` | AES-256-GCM 暗号化・復号 | 4 |
| `internal/db/models_test.go` | JSONB Value/Scan/Marshal/Unmarshal | 4 |
| `internal/middleware/authz_test.go` | 認可 (WithinDailyLimit, CanAccessModule, CanAccessTool) | 4 |
| `internal/middleware/ratelimit_test.go` | Sliding window レート制限 | 3 |
| `internal/broker/user_test.go` | UserContext メソッド、キャッシュ TTL/stale | 6 |
| `internal/broker/token_test.go` | FlexibleTime パース、needsRefresh 判定 | 2 |
| `internal/mcp/handler_test.go` | JSON-RPC ルーティング、バッチ権限チェック、エラーコード変換 | 6 |
| `internal/modules/validate_test.go` | パラメータバリデーション (必須フィールド、型チェック) | 5 |
| `internal/modules/modules_test.go` | ツールフィルタ、循環依存検出、変数解決 | 5 |
| `internal/modules/helpers_test.go` | ToJSON、ToStringSlice | 2 |
| `internal/auth/keys_test.go` | Ed25519 JWT 生成・署名検証 | 3 |
| `pkg/supabaseapi/client_test.go` | Supabase API クライアント (環境変数必須) | — |
| `pkg/asanaapi/client_test.go` | Asana API クライアント (環境変数必須) | — |

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

外部依存（DB、HTTP、環境変数）なしで実行可能。

#### 暗号化 / 復号 (`internal/db/encryption_test.go`)

| テスト | 検証内容 |
|---|---|
| ラウンドトリップ | encrypt → decrypt で元の平文に一致 (空文字列、Unicode、10KB ペイロード) |
| バージョン付きフォーマット | `v1:` プレフィックス + 有効な base64 |
| ランダム性 | 同一平文の2回暗号化で異なる暗号文 (ランダム nonce) |
| 不正入力の拒否 | 不正 base64、短すぎるデータ、改ざんされた暗号文 |

#### JSONB 型 (`internal/db/models_test.go`)

| テスト | 検証内容 |
|---|---|
| Value | JSONB → driver.Value 変換 (空 → `"{}"`) |
| Scan | `[]byte`, `string`, `nil` からの復元 |
| MarshalJSON | JSON シリアライズ (空 → `{}`) |
| UnmarshalJSON | JSON デシリアライズのラウンドトリップ |

#### 認可コンテキスト (`internal/middleware/authz_test.go`)

| テスト | 検証内容 |
|---|---|
| WithinDailyLimit | 境界値 (ちょうど上限、1 超過、バッチ) |
| CanAccessModule | 有効/無効モジュール、空文字列 → `MODULE_NOT_ENABLED` |
| CanAccessTool | モジュール→ツール→上限の3段階チェック、エラーコード判定 |
| HTTP ステータス | 403 (モジュール/ツール拒否) と 429 (上限超過) の使い分け |

#### レート制限 (`internal/middleware/ratelimit_test.go`)

| テスト | 検証内容 |
|---|---|
| Allow 基本動作 | 上限以内 → true、超過 → false |
| ウィンドウ回復 | 時間経過後にリクエスト再許可 |
| ユーザー分離 | 異なるユーザーが独立したウィンドウを持つ |

#### UserContext / キャッシュ (`internal/broker/user_test.go`)

| テスト | 検証内容 |
|---|---|
| WithinDailyLimit | 上限判定 (境界値、バッチ) |
| IsModuleEnabled | モジュール有効/無効判定 |
| IsToolEnabled | ツールホワイトリスト判定 |
| キャッシュ TTL | set → get → 期限切れ → nil、getStale → 期限切れでも取得可 |
| キャッシュ削除 | delete 後に get/getStale ともに nil |
| キャッシュミス | 未登録キーは nil |

#### トークン管理 (`internal/broker/token_test.go`)

| テスト | 検証内容 |
|---|---|
| FlexibleTime | Unix タイムスタンプ (int64)、ISO 8601 文字列、空文字列、不正入力 |
| needsRefresh | 期限切れ、バッファ内、十分な残り時間、ExpiresAt = 0 |

#### JSON-RPC ハンドラ (`internal/mcp/handler_test.go`)

| テスト | 検証内容 |
|---|---|
| authErrorToRPC | AuthError → JSON-RPC エラーコード変換 (5パターン) |
| checkBatchPermissions | 全許可、拒否ツール混在、無効モジュール、空行/不正 JSON スキップ |
| バッチサイズ上限 | 10コマンド以内 OK、11 → エラー |
| バッチ日次上限 | toolCount 合計で WithinDailyLimit 判定 |
| handleInitialize | プロトコルバージョン、ケイパビリティ返却 |
| ProcessRequest ルーティング | 不明メソッド → MethodNotFound、initialized → nil |

#### パラメータバリデーション (`internal/modules/validate_test.go`)

| テスト | 検証内容 |
|---|---|
| 必須フィールド | 欠落、nil、空文字列 → エラーメッセージ |
| 型チェック | string, number, integer, boolean, array, object |
| スキーマなし | 空スキーマ → パススルー |
| findTool | ツール名検索 (存在/不存在) |

#### モジュールレジストリ (`internal/modules/modules_test.go`)

| テスト | 検証内容 |
|---|---|
| filterTools | ホワイトリストフィルタ (全許可、部分許可、nil) |
| detectCycle | 循環なし → 空文字列、A→B→A → 検出、自己参照 |
| resolveStringVariables | `${id.results[0].field}` → JSON 値抽出 |
| resolveVariables | ネストされたマップ/配列内の変数展開 |
| availableModuleNames | レジストリに存在するモジュールのみ返却 |

#### ヘルパー関数 (`internal/modules/helpers_test.go`)

| テスト | 検証内容 |
|---|---|
| ToJSON | struct, map, nil のシリアライズ |
| ToStringSlice | 文字列要素の抽出、非文字列のスキップ |

#### Ed25519 JWT (`internal/auth/keys_test.go`)

| テスト | 検証内容 |
|---|---|
| GenerateAPIKeyJWT | `mpt_` プレフィックス、claims (sub, kid, iat) |
| 有効期限 | expiresAt あり/なしの JWT 生成 |
| 署名検証 | 生成した JWT を Ed25519 公開鍵で検証 |

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
| [spec-operation.md](./spec-operation.md)           | 運用仕様書     |
| [spec-security.md](./spec-security.md)             | セキュリティ仕様書 |
