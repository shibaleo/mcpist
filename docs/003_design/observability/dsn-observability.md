# Observability 設計書

| 項目 | 内容 |
|------|------|
| 文書ID | dsn-observability |
| ステータス | approved |
| バージョン | v1.1 |
| 作成日 | 2026-02-03 |
| 更新日 | 2026-02-04 |
| Sprint | SPRINT-007 (S7-001, S7-002, S7-003, S7-004, S7-005, S7-006) |

---

## 1. 概要

MCPist の Observability 基盤の設計書。分散構成（Worker + Render/Koyeb + Vercel）の可用性監視と、20モジュール・270超ツールの運用可視化の仕組みを定義する。

### 1.1 目的

1. **可用性監視**: サーバー（Render/Koyeb）・Worker・Vercel が正常稼働しているかの確認
2. **障害検知**: ツール実行エラー、外部 API 障害の検知
3. **運用可視化**: モジュール別・ツール別の利用状況把握
4. **トレーサビリティ**: リクエスト単位でのエンドツーエンド追跡
5. **異常検知**: Gateway Secret 不一致などの異常リクエストの記録（補助的）

### 1.2 設計原則

| 原則 | 説明 |
|------|------|
| **Best-effort** | ログ送信失敗がメイン処理に影響しない |
| **非同期送信** | Go Server: goroutine、Worker: `ctx.waitUntil()` による非ブロッキング送信 |
| **PII 最小化** | ログにリクエストボディ・トークン・API キーを含めない |
| **Information Hiding** | クライアントには曖昧なエラー、サーバーログに詳細 |
| **ゼロコスト運用** | Grafana Cloud Free Tier 内で運用 |

---

## 2. アーキテクチャ

### 2.1 全体構成

```
                    ┌──────────────────┐
                    │  Vercel (Console) │
                    │  - Next.js UI    │
                    │  - OAuth routes  │
                    └──────────────────┘

┌──────────────────────────────────────────┐
│         Cloudflare Worker (GWY)          │
│  - X-Request-ID 生成 (crypto.randomUUID) │
│  - JWT/API Key 検証                      │
│  - LB + フェイルオーバー                 │
│  - ヘルスチェック (5分 cron)             │
│  - Loki Push (ctx.waitUntil)             │
└──────────┬──────────────┬───────┬────────┘
           │              │       │
           ▼              ▼       │ Async (waitUntil)
┌────────────────┐  ┌──────────┐  │
│ Render (Primary)│  │ Koyeb    │  │
│  Go API Server │  │ (Standby)│  │
│  /health       │  │          │  │
└───────┬────────┘  └────┬─────┘  │
        │                │        │
        └──────┬─────────┘        │
               │ Async (goroutine)│
               ▼                  ▼
        ┌──────────────────────────┐
        │      Grafana Loki        │
        │  - ログ集約・検索         │
        └────────┬─────────────────┘
                 ▼
        ┌──────────────────────────┐
        │   Grafana Dashboard      │
        │  - Overview (可用性+運用) │
        │  - Module Performance    │
        └──────────────────────────┘
```

**ログ送信元:**

| コンポーネント | 送信方式 | ラベル: instance | ラベル: region |
|----------------|----------|-----------------|----------------|
| Cloudflare Worker | `ctx.waitUntil(fetch(...))` | `worker` | `cloudflare` |
| Render (Go Server) | `go client.push(...)` (goroutine) | `RENDER_INSTANCE_ID`（自動） | `RENDER_REGION`（手動 = `oregon`） |
| Koyeb (Go Server) | `go client.push(...)` (goroutine) | `KOYEB_INSTANCE_ID`（自動） | `KOYEB_REGION`（自動） |

**監視対象コンポーネント:**

| コンポーネント | ホスト | 役割 | ダウン時の影響 |
|----------------|--------|------|----------------|
| Cloudflare Worker | mcp.mcpist.com | API Gateway, LB | 全 MCP リクエスト不可 |
| Render (Primary) | mcpist-api-dev.onrender.com | ツール実行 | Koyeb にフェイルオーバー |
| Koyeb (Standby) | mcpist.koyeb.app | ツール実行（待機） | Primary ダウン時のみ影響 |
| Vercel (Console) | mcpist.com | UI, OAuth | コンソール操作不可 |
| Supabase | - | DB, Auth | 全機能停止 |

### 2.2 技術スタック

| コンポーネント | 技術 | 備考 |
|----------------|------|------|
| ログ集約 | Grafana Loki (Cloud) | Free Tier: 50GB/月 |
| ダッシュボード | Grafana Cloud | Free Tier: 10,000 series |
| ログ送信 | HTTP Push API v1 | `POST /loki/api/v1/push` |
| 認証 | Basic Auth | `GRAFANA_LOKI_USER:GRAFANA_LOKI_API_KEY` |
| リクエスト追跡 | X-Request-ID | UUID v4 (Worker生成) → Go Server に伝播 |

### 2.3 環境変数

**Loki 接続（Go Server / Worker 共通）:**

| 変数 | 用途 | 必須 |
|------|------|------|
| `GRAFANA_LOKI_URL` | Loki エンドポイント | Yes (未設定時は無効化) |
| `GRAFANA_LOKI_USER` | Loki ユーザー ID | Yes |
| `GRAFANA_LOKI_API_KEY` | Loki API キー | Yes |
| `APP_ENV` | `app` ラベル値 (`mcpist-dev` / `mcpist-prd`) | No (デフォルト: `mcpist-dev`) |

3つすべてが設定されていない場合、ログ送信は無効化される (`enabled: false`)。メイン処理への影響なし。

**Go Server のみ — Instance/Region 自動検出:**

| 変数 | 用途 | 優先順位 |
|------|------|----------|
| `INSTANCE_ID` | インスタンス識別子（手動） | 1（最優先） |
| `RENDER_INSTANCE_ID` | Render 自動提供 | 2 |
| `KOYEB_INSTANCE_ID` | Koyeb 自動提供 | 3 |
| (fallback) | `"local"` | 4 |
| `INSTANCE_REGION` | リージョン（手動） | 1（最優先） |
| `RENDER_REGION` | Render 手動設定 | 2 |
| `KOYEB_REGION` | Koyeb 自動提供 | 3 |
| (fallback) | `"local"` | 4 |

`firstNonEmpty()` ヘルパーで上から順に非空の値を採用するフォールバックチェーン。

**Worker — Instance/Region:**

Worker は固定値: `instance="worker"`, `region="cloudflare"`。

---

## 3. ログ構造

### 3.1 ストリームラベル設計

Loki はラベルでストリームを分割する。ラベルのカーディナリティを低く保つことが重要。

#### 共通ラベル（全ログエントリに自動付与）

| ラベル | 値 | 説明 |
|--------|-----|------|
| `app` | `mcpist-dev` / `mcpist-prd` | アプリケーション識別子 (`APP_ENV`) |
| `instance` | `worker`, `srv-xxx-xxx`, `local` | インスタンス識別子 |
| `region` | `cloudflare`, `oregon`, `local` | リージョン |

#### ログ種別ごとのラベル

**ツール実行ログ** (`LogToolCall` / Go Server):

| ラベル | 値例 | 説明 |
|--------|------|------|
| `module` | `notion`, `github` | モジュール名 |
| `status` | `success` / `error` | 実行結果 |

`tool` はカーディナリティが高い（270+）ため、データフィールドに含める。

**セキュリティイベント** (`LogSecurityEvent` / Go Server + Worker):

| ラベル | 値例 | 説明 |
|--------|------|------|
| `type` | `security` | ログ種別 |
| `level` | `warn` | ログレベル |

`event` はカーディナリティ管理のためデータフィールドに含める。

**リクエストログ** (`LogRequest` / Go Server, `logRequest` / Worker):

| ラベル | 値例 | 説明 |
|--------|------|------|
| `type` | `request` | ログ種別 |
| `method` | `GET`, `POST` | HTTP メソッド |

Go Server では `path` もラベルに含める。Worker ではデータフィールドのみ。

**エラーログ** (`LogError` / Go Server):

| ラベル | 値例 | 説明 |
|--------|------|------|
| `type` | `error` | ログ種別 |
| `level` | `error` | ログレベル |

### 3.2 ログデータフィールド

各ログエントリの JSON ボディに含まれるフィールド。

#### ツール実行ログ (Go Server)

| フィールド | 型 | 説明 | 必須 |
|------------|-----|------|------|
| `request_id` | string | リクエスト追跡 ID | Yes |
| `user_id` | string | 実行ユーザー ID | Yes |
| `module` | string | モジュール名 | Yes |
| `tool` | string | ツール名 | Yes |
| `duration_ms` | int64 | 実行時間 (ミリ秒) | Yes |
| `status` | string | `success` / `error` | Yes |
| `error` | string | エラーメッセージ | error 時のみ |

#### リクエストログ (Worker)

| フィールド | 型 | 説明 | 必須 |
|------------|-----|------|------|
| `request_id` | string | リクエスト追跡 ID (Worker生成) | Yes |
| `method` | string | HTTP メソッド | Yes |
| `path` | string | リクエストパス | Yes |
| `status_code` | int | HTTP ステータスコード | Yes |
| `duration_ms` | int64 | レスポンス時間 (ミリ秒) | Yes |
| `user_id` | string | ユーザー ID | 認証成功時 |
| `auth_type` | string | `jwt` / `api_key` | 認証成功時 |
| `backend` | string | `primary` / `secondary` | Yes |

#### リクエストログ (Go Server)

| フィールド | 型 | 説明 | 必須 |
|------------|-----|------|------|
| `method` | string | HTTP メソッド | Yes |
| `path` | string | リクエストパス | Yes |
| `status_code` | int | HTTP ステータスコード | Yes |
| `duration_ms` | int64 | レスポンス時間 (ミリ秒) | Yes |

#### 異常イベント

| フィールド | 型 | 説明 | 必須 |
|------------|-----|------|------|
| `request_id` | string | リクエスト追跡 ID | Yes |
| `user_id` | string | ユーザー ID | Yes |
| `event` | string | イベント種別 | Yes |
| `details` | map | イベント詳細 | Yes |

---

## 4. 構造化ログ統一 (S7-002) ✅

### 4.1 現状の問題

現在、Go Server のログ出力は2系統に分かれている:

1. **Loki Push**: `observability.LogToolCall()` 等で非同期送信 → 構造化されている
2. **標準出力**: `log.Printf()` で出力 → 非構造化、フォーマットがバラバラ

```go
// 現状: バラバラな log.Printf
log.Println("Loki client initialized")
log.Printf("Registered modules: %v", moduleNames)
log.Printf("Authorization: user=%s credits=free:%d+paid:%d modules=%d", ...)
log.Printf("Tool call: request=%s module=%s tool=%s duration=%dms status=%s", ...)
```

### 4.2 統一方針

**方針: Loki Push を唯一の運用ログ基盤とする**

運用ログは Loki Push (`observability.Push()`) に一本化する。標準出力は起動・初期化・致命的エラーなど Loki 送信前のログに限定し、運用時の日次確認は Grafana (Loki) のみで行う。

標準出力への二重送信は行わない。理由:
- サーバーの標準出力ログは日常的に参照しない
- Loki に送信済みのログを標準出力にも出すのは冗長
- Loki 未設定時（ローカル開発等）は `log.Printf` のフォールバックが残る

### 4.3 標準出力の用途

標準出力 (`log.Printf`) は以下に限定する:

| 用途 | 例 |
|------|-----|
| サーバー起動・初期化 | `log.Println("Server started on :8080")` |
| Loki 送信失敗 | `log.Printf("Loki push failed: %v", err)` |
| モジュール登録 | `log.Printf("Registered modules: %v", names)` |

slog への全面移行は行わない。Loki Push が構造化ログを担い、標準出力は補助的な役割のみ。

### 4.4 移行計画

| ステップ | 内容 | 状態 |
|----------|------|------|
| 1 | `observability.LogToolCall` 内の `log.Printf` を削除（Loki Push のみに） | ✅ |
| 2 | `middleware/authz.go` の認証ログを `LogSecurityEvent` に統合 | ✅ |
| 3 | 各モジュール内の `log.Printf` は起動・初期化以外を削除 | ✅ |

---

## 5. user_id 追加 (S7-003) ✅

### 5.1 変更内容

`LogToolCall` に `userID` パラメータを追加:

```go
func LogToolCall(requestID, userID, module, tool string, durationMs int64, status, errMsg string)
```

Loki Push のデータフィールドに `user_id` を追加:

```go
data := map[string]any{
    "request_id":  requestID,
    "user_id":     userID,
    "module":      module,
    "tool":        tool,
    "duration_ms": durationMs,
    "status":      status,
}
```

### 5.2 呼び出し元の変更

`modules.Run()` から `user_id` を渡す:

```go
// apps/server/internal/modules/modules.go
requestID := middleware.GetRequestID(ctx)
authCtx := middleware.GetAuthContext(ctx)
userID := ""
if authCtx != nil {
    userID = authCtx.UserID
}
observability.LogToolCall(requestID, userID, moduleName, toolName, durationMs, "success", "")
```

---

## 6. 異常イベント (S7-004) ✅

個人プロジェクトが特定の攻撃対象になることはまれ。ここでの「異常イベント」は、設定ミスやインフラの不整合を検知するためのもの。

### 6.1 イベント一覧

| イベント | 発生条件 | レベル | 送信元 | 実装状態 |
|----------|----------|--------|--------|----------|
| `batch_permission_denied` | バッチ実行時に許可されていないツールを要求 | WARN | Go Server | ✅ |
| `invalid_gateway_secret` | X-Gateway-Secret の不一致 | WARN | Go Server | ✅ |
| `auth_failed` | JWT/API Key 認証失敗 | WARN | Worker | ✅ |

### 6.2 invalid_gateway_secret のログ仕様

Gateway Secret 検証失敗時に `LogSecurityEvent` で記録する。セキュリティ設計（検証ロジック・発生原因）は [dsn-security.md](../security/dsn-security.md) Section 2.4 を参照。

| フィールド | 値 |
|------------|-----|
| `event` | `invalid_gateway_secret` |
| `user_id` | 空文字列（ユーザー未特定） |
| `details.remote_addr` | リクエスト元 IP |

### 6.3 auth_failed のログ仕様

Worker で JWT/API Key 認証が失敗した場合に `logSecurityEvent` で記録する。

| フィールド | 値 |
|------------|-----|
| `event` | `auth_failed` |
| `details.method` | HTTP メソッド |
| `details.path` | リクエストパス |
| `details.duration_ms` | レスポンス時間 |

LogQL クエリ:
```logql
{app="mcpist-dev", type="security"} | json | event="auth_failed"
```

---

## 7. ログレベル設計 (S7-005) ✅

### 7.1 レベル定義

| レベル | 用途 | Loki ラベル | 例 |
|--------|------|-------------|-----|
| **ERROR** | 内部エラー、復旧不可能な障害 | `level=error` | DB 接続失敗、Loki 送信失敗 |
| **WARN** | 異常イベント、外部 API エラー | `level=warn` | Gateway Secret 不一致、API 4xx/5xx |
| **INFO** | 正常な操作ログ | `level=info` | ツール実行成功、起動完了 |
| **DEBUG** | 開発用の詳細ログ | (出力しない) | パラメータダンプ等 |

### 7.2 分類基準

| シナリオ | レベル | 理由 |
|----------|--------|------|
| ツール実行成功 | INFO | 正常動作 |
| ツール実行エラー (外部 API) | WARN | 外部要因、対応不要の場合が多い |
| ツール実行エラー (内部) | ERROR | 調査・修正が必要 |
| invalid_gateway_secret | WARN | 異常イベント（設定ミスの可能性） |
| auth_failed | WARN | 認証失敗（トークンなし/無効） |
| batch_permission_denied | WARN | 異常イベント |
| DB 接続エラー | ERROR | 運用影響あり |
| サーバー起動完了 | INFO | 正常動作 |
| モジュール登録 | INFO | 正常動作 |

### 7.3 外部 API エラーの分類

ツール実行時の外部 API エラーは、ユーザーの入力ミスか外部サービスの障害かを区別する:

| HTTP ステータス | 分類 | ログレベル |
|-----------------|------|------------|
| 400 Bad Request | ユーザー入力エラー | INFO (ツールの status=error) |
| 401/403 | 認証・認可エラー | WARN (トークン期限切れ等) |
| 404 Not Found | リソースなし | INFO |
| 429 Too Many Requests | レートリミット | WARN |
| 500+ | 外部サービス障害 | WARN |

---

## 8. Grafana ダッシュボード (S7-006)

### 8.1 ダッシュボード構成

| ダッシュボード | UID | 用途 | 実装状態 |
|----------------|-----|------|----------|
| **MCPist Overview** | `mcpist-observability` | リクエスト監視 + ツール実行サマリ + セキュリティ | ✅ |
| **MCPist Module Performance** | `mcpist-module-performance` | モジュール・ツール別パフォーマンス | ✅ |

フォルダ: MCPist (uid: `mcpist`)

作成方法: Grafana API (`create_update_dashboard`) 経由。Grafana Cloud UI ではなく MCP ツールで作成・更新する。

### 8.2 Overview ダッシュボード

| 設定 | 値 |
|------|-----|
| 自動更新 | 30秒 |
| デフォルト範囲 | 1時間 |
| テンプレート変数 | `$instance`（インスタンスフィルター） |

**パネル一覧:**

| # | パネル | 種別 | LogQL クエリ |
|---|--------|------|-------------|
| 1 | Request Rate by Instance | Time Series (bars) | `sum by (instance) (count_over_time({app="mcpist-dev", type="request"} [1m]))` |
| 2 | Avg Response Time by Instance | Time Series (line) | `avg_over_time({app="mcpist-dev", type="request"} \| json \| unwrap duration_ms [5m]) by (instance)` |
| 3 | Tool Executions by Status | Time Series (bars) | `sum by (status) (count_over_time({app="mcpist-dev", module!=""} [1h]))` |
| 4 | Error Rate (%) | Stat | `sum(count_over_time({app="mcpist-dev", status="error"} [$__range])) / sum(count_over_time({app="mcpist-dev", module!=""} [$__range])) * 100` |
| 5 | Status Code Distribution | Piechart | `sum by (status_code) (count_over_time({app="mcpist-dev", type="request"} \| json \| status_code != "" [$__range]))` |
| 6 | Module Usage | Bar Chart | `topk(10, sum by (module) (count_over_time({app="mcpist-dev", module!=""} [$__range])))` |
| 7 | Security Events | Table | `{app="mcpist-dev", type="security"}` |
| 8 | Latest Error Logs | Logs | `{app="mcpist-dev", status="error"}` |
| 9 | Recent Logs | Logs | `{app="mcpist-dev"}` |

### 8.3 Module Performance ダッシュボード

| 設定 | 値 |
|------|-----|
| 自動更新 | 30秒 |
| デフォルト範囲 | 24時間 |
| テンプレート変数 | `$module`（モジュールフィルター） |

**パネル一覧:**

| # | パネル | 種別 | LogQL クエリ |
|---|--------|------|-------------|
| 1 | Tool Execution Time (p95/p50) | Time Series (line) | `quantile_over_time(0.95, {app="mcpist-dev", module=~"$module"} \| json \| unwrap duration_ms [5m])` |
| 2 | Module Error Count | Table | `sum by (module) (count_over_time({app="mcpist-dev", status="error", module=~"$module"} [$__range]))` |
| 3 | User Activity | Table | `sum by (user_id) (count_over_time({app="mcpist-dev", module!=""} \| json \| user_id!="" [$__range]))` |
| 4 | Slowest Tools Top 10 | Table | `topk(10, avg by (tool) (avg_over_time({app="mcpist-dev", module=~"$module"} \| json \| unwrap duration_ms [$__range])))` |
| 5 | Tool Executions by Module | Time Series (bars) | `sum by (module) (count_over_time({app="mcpist-dev", module=~"$module"} [1h]))` |

### 8.4 外形監視 (Synthetic Monitoring)

| チェック対象 | URL | 間隔 | 期待値 | 実装状態 |
|-------------|-----|------|--------|----------|
| Worker (API Gateway) | `https://mcp.mcpist.com/health` | 5分 | HTTP 200 | 未設定 |
| Console (Vercel) | `https://mcpist.com` | 5分 | HTTP 200 | 未設定 |

Grafana Cloud Free Tier の Synthetic Monitoring で Worker と Console を外形監視する。Go Server の稼働は Worker のヘルスチェックでカバーされるため、個別の外形監視は不要。

### 8.5 アラート設定

| アラート | 条件 | 通知先 | 優先度 | 実装状態 |
|----------|------|--------|--------|----------|
| Worker ダウン | Synthetic Monitoring: mcp.mcpist.com が 2回連続失敗 | Grafana OnCall | Critical | 未設定 |
| Console ダウン | Synthetic Monitoring: mcpist.com が 2回連続失敗 | Grafana OnCall | High | 未設定 |
| エラー率急増 | 5分間のツール実行エラー率 > 50% | Grafana OnCall | High | 未設定 |

---

## 9. リクエスト追跡フロー

### 9.1 X-Request-ID ライフサイクル

```
Client → Worker ──────────────→ Go Server → External API
           │                        │
           ├─ Loki (リクエストログ)   ├─ Loki (ToolCall ログ: request_id)
           ├─ Loki (SecurityEvent)  ├─ Loki (SecurityEvent: request_id)
           │                        └─ Supabase (ConsumeCredit: request_id)
           │
           └─ X-Request-ID ヘッダーで Go Server に伝播
```

1. **Worker**: `crypto.randomUUID()` で生成し `X-Request-ID` ヘッダーに設定。同時に Worker 自身も `request_id` を含めて Loki に送信
2. **Go Server**: ミドルウェアで受け取りコンテキストに保存。ヘッダーがない場合は `crypto/rand` で 16-byte hex を生成
3. **Loki**: Worker と Go Server の両方のログエントリに同一の `request_id` を含める（E2E トレーシング）
4. **Supabase**: `ConsumeCredit` に `request_id` を渡し、usage_stats テーブルに記録

### 9.2 トレース検索

特定リクエストの全ログを検索（Worker + Go Server の両方がヒット）:

```logql
{app="mcpist-dev"} | json | request_id="<target-request-id>"
```

---

## 10. PII・セキュリティ考慮事項

### 10.1 ログに含めないデータ

| データ | 理由 |
|--------|------|
| リクエストボディ | ユーザーデータを含む可能性 |
| Authorization ヘッダー | JWT / API Key の漏洩防止 |
| X-Gateway-Secret 値 | インフラシークレット |
| OAuth トークン | アクセストークン・リフレッシュトークン |
| 外部 API レスポンスボディ | ユーザーデータを含む |

### 10.2 ログに含めるデータ

| データ | 理由 |
|--------|------|
| user_id | ユーザー特定（UUID、可逆ではない） |
| request_id | リクエスト追跡 |
| module / tool | 操作内容の特定 |
| duration_ms | パフォーマンス分析 |
| status / error message | エラー分析 |
| remote_addr | セキュリティイベント時のみ |

---

## 11. 実装タスク一覧

| ID | タスク | ファイル | 状態 |
|----|--------|----------|------|
| S7-002-1 | `LogToolCall` 内の `log.Printf` を削除 | `internal/observability/loki.go` | ✅ |
| S7-002-2 | `middleware/authz.go` の認証ログを `LogSecurityEvent` に統合 | `internal/middleware/authz.go` | ✅ |
| S7-002-3 | 各モジュール内の不要な `log.Printf` を削除 | `internal/modules/*.go` | ✅ |
| S7-003-1 | `LogToolCall` に `userID` パラメータ追加 | `internal/observability/loki.go` | ✅ |
| S7-003-2 | `modules.Run` から `userID` を渡す | `internal/modules/modules.go` | ✅ |
| S7-003-3 | `modules.Batch` から `userID` を渡す | `internal/modules/modules.go` | ✅ |
| S7-004-1 | Go Server: Gateway Secret 検証失敗時に異常イベントログ | `internal/middleware/authz.go` | ✅ |
| S7-004-2 | Worker: 認証失敗時に異常イベントログ | `apps/worker/src/index.ts` | ✅ |
| S7-005-1 | `LogToolCall` ラベルから `tool` を削除 | `internal/observability/loki.go` | ✅ |
| S7-005-2 | `LogSecurityEvent` ラベルから `event` を削除 | `internal/observability/loki.go` | ✅ |
| S7-006-1 | Grafana Synthetic Monitoring 設定 | Grafana Cloud UI | 未着手 |
| S7-006-2 | Overview ダッシュボード作成 | Grafana API | ✅ |
| S7-006-3 | Module Performance ダッシュボード作成 | Grafana API | ✅ |
| S7-006-4 | アラートルール設定 | Grafana API | 未着手 |
| (追加) | Worker Loki Push 実装 | `apps/worker/src/index.ts` | ✅ |
| (追加) | Worker request_id 統一 | `apps/worker/src/index.ts` | ✅ |
| (追加) | Instance/Region 自動検出 | `internal/observability/loki.go` | ✅ |
| (追加) | Grafana `query_datasource` ツール追加 | `internal/modules/grafana/module.go` | ✅ |

---

## 12. Grafana Cloud Free Tier 制約

| リソース | 上限 | 現状の使用量 |
|----------|------|-------------|
| Loki ログ | 50 GB/月 | < 1 GB/月 (推定) |
| Synthetic Monitoring | 5 checks | 0 (未設定) |
| Prometheus Series | 10,000 | 0 (未使用) |
| Dashboard | 無制限 | 2 (Overview + Module Performance) |
| Alert Rules | 100 | 0 (未設定) |
| Data Retention | 14日間 | - |

現在の規模 (5-10 ユーザー) では Free Tier で十分。ユーザー数増加時はサンプリング戦略を導入する。

---

## 参考文書

| 文書 | 内容 |
|------|------|
| [dtl-itr-HDL-OBS.md](../../002_specification/interaction/dtl-itr-HDL-OBS.md) | Handler → Observability インタラクション仕様 |
| [dtl-itr-GWY-OBS.md](../../002_specification/interaction/dtl-itr-GWY-OBS.md) | Gateway → Observability インタラクション仕様 |
| [dsn-infrastructure.md](../system/dsn-infrastructure.md) | インフラ設計書 |
| [spc-ops.md](../../002_specification/spc-ops.md) | 運用仕様書 |
