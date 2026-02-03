# Observability 設計書

| 項目 | 内容 |
|------|------|
| 文書ID | dsn-observability |
| ステータス | draft |
| バージョン | v1.0 |
| 作成日 | 2026-02-03 |
| Sprint | SPRINT-007 (S7-001, S7-002, S7-003, S7-004, S7-005, S7-006) |

---

## 1. 概要

MCPist の Observability 基盤の設計書。分散構成（Worker + Render/Koyeb + Vercel）の可用性監視と、18モジュール・250超ツールの運用可視化の仕組みを定義する。

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
| **非同期送信** | goroutine による非ブロッキング送信 |
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
└──────────┬──────────────┬────────────────┘
           │              │
           ▼              ▼ (failover)
┌────────────────┐  ┌────────────────┐
│ Render (Primary)│  │ Koyeb (Standby)│
│  Go API Server │  │  Go API Server │
│  /health       │  │  /health       │
└───────┬────────┘  └───────┬────────┘
        │                   │
        └─────────┬─────────┘
                  │ Async (goroutine, 5s timeout)
                  ▼
        ┌──────────────────┐
        │  Grafana Loki    │
        │  - ログ集約・検索 │
        └────────┬─────────┘
                 ▼
        ┌──────────────────┐
        │ Grafana Dashboard│
        │  - 可用性監視    │
        │  - 運用可視化    │
        └──────────────────┘
```

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
| リクエスト追跡 | X-Request-ID | UUID v4 (Worker生成) / 16-byte hex (Server生成) |

### 2.3 環境変数

| 変数 | 用途 | 必須 |
|------|------|------|
| `GRAFANA_LOKI_URL` | Loki エンドポイント | Yes (未設定時は無効化) |
| `GRAFANA_LOKI_USER` | Loki ユーザー ID | Yes |
| `GRAFANA_LOKI_API_KEY` | Loki API キー | Yes |

3つすべてが設定されていない場合、ログ送信は無効化される (`enabled: false`)。メイン処理への影響なし。

---

## 3. ログ構造

### 3.1 ストリームラベル設計

Loki はラベルでストリームを分割する。ラベルのカーディナリティを低く保つことが重要。

#### 共通ラベル

| ラベル | 値 | 説明 |
|--------|-----|------|
| `app` | `mcpist-dev` / `mcpist-prd` | アプリケーション識別子 |

#### ログ種別ごとのラベル

**ツール実行ログ** (`LogToolCall`):

| ラベル | 値例 | 説明 |
|--------|------|------|
| `module` | `notion`, `github` | モジュール名 |
| `status` | `success` / `error` | 実行結果 |

`tool` はカーディナリティが高い（250+）ため、データフィールドに含める。

**セキュリティイベント** (`LogSecurityEvent`):

| ラベル | 値例 | 説明 |
|--------|------|------|
| `type` | `security` | ログ種別 |
| `level` | `warn` | ログレベル |

`event` はカーディナリティ管理のためデータフィールドに含める。

**リクエストログ** (`LogRequest`):

| ラベル | 値例 | 説明 |
|--------|------|------|
| `type` | `request` | ログ種別 |
| `method` | `GET`, `POST` | HTTP メソッド |
| `path` | `/mcp` | リクエストパス |

**エラーログ** (`LogError`):

| ラベル | 値例 | 説明 |
|--------|------|------|
| `type` | `error` | ログ種別 |
| `level` | `error` | ログレベル |

### 3.2 ログデータフィールド

各ログエントリの JSON ボディに含まれるフィールド。

#### ツール実行ログ

| フィールド | 型 | 説明 | 必須 |
|------------|-----|------|------|
| `request_id` | string | リクエスト追跡 ID | Yes |
| `user_id` | string | 実行ユーザー ID | Yes (**S7-003 で追加**) |
| `module` | string | モジュール名 | Yes |
| `tool` | string | ツール名 | Yes |
| `duration_ms` | int64 | 実行時間 (ミリ秒) | Yes |
| `status` | string | `success` / `error` | Yes |
| `error` | string | エラーメッセージ | error 時のみ |

#### 異常イベント

| フィールド | 型 | 説明 | 必須 |
|------------|-----|------|------|
| `request_id` | string | リクエスト追跡 ID | Yes |
| `user_id` | string | ユーザー ID | Yes |
| `event` | string | イベント種別 | Yes |
| `details` | map | イベント詳細 | Yes |

#### リクエストログ

| フィールド | 型 | 説明 | 必須 |
|------------|-----|------|------|
| `method` | string | HTTP メソッド | Yes |
| `path` | string | リクエストパス | Yes |
| `status_code` | int | HTTP ステータスコード | Yes |
| `duration_ms` | int64 | レスポンス時間 (ミリ秒) | Yes |

---

## 4. 構造化ログ統一 (S7-002)

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

| ステップ | 内容 |
|----------|------|
| 1 | `observability.LogToolCall` 内の `log.Printf` を削除（Loki Push のみに） |
| 2 | `middleware/authz.go` の認証ログを `LogSecurityEvent` に統合 |
| 3 | 各モジュール内の `log.Printf` は起動・初期化以外を削除 |

---

## 5. user_id 追加 (S7-003)

### 5.1 現状

`LogToolCall` は `request_id` のみ記録しており、`user_id` がない:

```go
// 現状
func LogToolCall(requestID, module, tool string, durationMs int64, status, errMsg string)
```

`request_id` → `user_id` の紐付けは Authorize ミドルウェアのログに依存しており、ログ検索時に結合が必要。

### 5.2 変更内容

`LogToolCall` に `userID` パラメータを追加:

```go
// 変更後
func LogToolCall(requestID, userID, module, tool string, durationMs int64, status, errMsg string)
```

Loki Push のデータフィールドに `user_id` を追加:

```go
data := map[string]any{
    "request_id":  requestID,
    "user_id":     userID,     // 追加
    "module":      module,
    "tool":        tool,
    "duration_ms": durationMs,
    "status":      status,
}
```

### 5.3 呼び出し元の変更

`modules.Run()` から `user_id` を渡す:

```go
// apps/server/internal/modules/modules.go
func Run(ctx context.Context, moduleName, toolName string, params map[string]interface{}) (*ToolCallResult, error) {
    // ...
    requestID := middleware.GetRequestID(ctx)
    authCtx := middleware.GetAuthContext(ctx)
    userID := ""
    if authCtx != nil {
        userID = authCtx.UserID
    }

    observability.LogToolCall(requestID, userID, moduleName, toolName, durationMs, "success", "")
}
```

---

## 6. 異常イベント (S7-004)

個人プロジェクトが特定の攻撃対象になることはまれ。ここでの「異常イベント」は、設定ミスやインフラの不整合を検知するためのもの。

### 6.1 イベント一覧

| イベント | 発生条件 | レベル | 実装状態 |
|----------|----------|--------|----------|
| `batch_permission_denied` | バッチ実行時に許可されていないツールを要求 | WARN | ✅ 実装済み |
| `invalid_gateway_secret` | X-Gateway-Secret の不一致 | WARN | **S7-004 で追加** |

`invalid_gateway_secret` の主な発生原因は、デプロイ時の環境変数の不一致や、直接オリジンにアクセスした場合。

### 6.2 invalid_gateway_secret のログ仕様

Gateway Secret 検証失敗時に `LogSecurityEvent` で記録する。セキュリティ設計（検証ロジック・発生原因）は [dsn-security.md](../security/dsn-security.md) Section 2.4 を参照。

| フィールド | 値 |
|------------|-----|
| `event` | `invalid_gateway_secret` |
| `user_id` | 空文字列（ユーザー未特定） |
| `details.remote_addr` | リクエスト元 IP |

LogQL クエリ:
```logql
{app="mcpist-prd", type="security"} | json | event="invalid_gateway_secret"
```

---

## 7. ログレベル設計 (S7-005)

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

## 8. Grafana ダッシュボード設計 (S7-006)

### 8.1 ダッシュボード構成

2つのダッシュボードを作成する:

1. **Overview（可用性 + サマリ）**: インフラの稼働状況とツール実行サマリ
2. **Module Performance**: モジュール・ツール別パフォーマンス

個人プロジェクトで最も重要なのは「サーバーが動いているか」の確認。セキュリティ攻撃の対象になることはまれであり、セキュリティ専用ダッシュボードは不要。異常イベントは Overview のログパネルで確認する。

### 8.2 Overview ダッシュボード

**可用性セクション:**

Worker は 5分 cron で Render/Koyeb にヘルスチェックを行い、結果を Go Server 経由で Loki に送信する。ヘルスチェック結果をダッシュボードに表示する。

| パネル | 種別 | データソース | 説明 |
|--------|------|-------------|------|
| バックエンド稼働状態 | Stat | Worker /health API | Primary (Render) / Standby (Koyeb) の直近ステータス |
| フェイルオーバー発生履歴 | Logs | Loki | Worker が Primary → Standby に切り替えた記録 |

> **注**: Worker のヘルスチェックログは現在 Loki に送信されていない（`console.log` のみ）。Worker → Loki 連携は将来課題。現時点では `/health` エンドポイントを Grafana Synthetic Monitoring（Free Tier: 5 checks）で外形監視する。

**外形監視 (Synthetic Monitoring):**

| チェック対象 | URL | 間隔 | 期待値 |
|-------------|-----|------|--------|
| Worker (API Gateway) | `https://mcp.mcpist.com/health` | 5分 | HTTP 200 |
| Console (Vercel) | `https://mcpist.com` | 5分 | HTTP 200 |

Grafana Cloud Free Tier の Synthetic Monitoring で Worker と Console を外形監視する。Go Server の稼働は Worker のヘルスチェックでカバーされるため、個別の外形監視は不要。

**ツール実行サマリセクション:**

| パネル | 種別 | LogQL クエリ |
|--------|------|-------------|
| ツール実行数/時 | Time Series | `sum(count_over_time({app="mcpist-prd", module=~".+"} [1h])) by (status)` |
| エラー率 | Stat | `sum(count_over_time({app="mcpist-prd", status="error"} [24h])) / sum(count_over_time({app="mcpist-prd", module=~".+"} [24h])) * 100` |
| モジュール別実行数 | Bar Chart | `sum(count_over_time({app="mcpist-prd", module=~".+"} [24h])) by (module)` |
| 最新エラーログ | Logs | `{app="mcpist-prd", status="error"}` |
| 最新異常イベント | Logs | `{app="mcpist-prd", type="security"}` |

### 8.3 Module Performance ダッシュボード

| パネル | 種別 | LogQL クエリ |
|--------|------|-------------|
| ツール別実行時間 (p95) | Heatmap | `quantile_over_time(0.95, {app="mcpist-prd", module="$module"} \| json \| unwrap duration_ms [5m])` |
| モジュール別エラー率 | Table | `sum(count_over_time({app="mcpist-prd", status="error"} [24h])) by (module)` |
| ユーザー別実行数 | Table | `sum by (user_id) (count_over_time({app="mcpist-prd", module=~".+"} \| json \| user_id!="" [24h]))` |
| 遅延ツール Top 10 | Table | `topk(10, avg by (tool) (avg_over_time({app="mcpist-prd", module="$module"} \| json \| unwrap duration_ms [24h])))` |

### 8.4 アラート設定

| アラート | 条件 | 通知先 | 優先度 |
|----------|------|--------|--------|
| Worker ダウン | Synthetic Monitoring: mcp.mcpist.com が 2回連続失敗 | Grafana OnCall | Critical |
| Console ダウン | Synthetic Monitoring: mcpist.com が 2回連続失敗 | Grafana OnCall | High |
| エラー率急増 | 5分間のツール実行エラー率 > 50% | Grafana OnCall | High |

---

## 9. リクエスト追跡フロー

### 9.1 X-Request-ID ライフサイクル

```
Client → Worker → Go Server → External API
                     │
                     ├─ Loki (ToolCall ログ: request_id)
                     ├─ Loki (SecurityEvent: request_id)
                     └─ Supabase (ConsumeCredit: request_id)
```

1. **Worker**: `crypto.randomUUID()` で生成し `X-Request-ID` ヘッダーに設定
2. **Go Server**: ミドルウェアで受け取りコンテキストに保存。ヘッダーがない場合は `crypto/rand` で 16-byte hex を生成
3. **Loki**: すべてのログエントリに `request_id` を含める
4. **Supabase**: `ConsumeCredit` に `request_id` を渡し、usage_stats テーブルに記録

### 9.2 トレース検索

特定リクエストの全ログを検索:

```logql
{app="mcpist-prd"} | json | request_id="<target-request-id>"
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

| ID | タスク | ファイル | 優先度 |
|----|--------|----------|--------|
| S7-002-1 | `LogToolCall` 内の `log.Printf` を削除 | `internal/observability/loki.go` | High |
| S7-002-2 | `middleware/authz.go` の認証ログを `LogSecurityEvent` に統合 | `internal/middleware/authz.go` | High |
| S7-002-3 | 各モジュール内の不要な `log.Printf` を削除 | `internal/modules/*.go` | Medium |
| S7-003-1 | `LogToolCall` に `userID` パラメータ追加 | `internal/observability/loki.go` | High |
| S7-003-2 | `modules.Run` から `userID` を渡す | `internal/modules/modules.go` | High |
| S7-003-3 | `modules.Batch` から `userID` を渡す | `internal/modules/modules.go` | High |
| S7-004-1 | Gateway Secret 検証失敗時に異常イベントログ | `internal/middleware/authz.go` | High |
| S7-005-1 | `LogToolCall` ラベルから `tool` を削除、データフィールドに移動 | `internal/observability/loki.go` | High |
| S7-005-2 | `LogSecurityEvent` ラベルから `event` を削除、データフィールドに移動 | `internal/observability/loki.go` | High |
| S7-006-1 | Grafana Synthetic Monitoring 設定 (Worker + Console) | Grafana Cloud UI | **High** |
| S7-006-2 | Overview ダッシュボード作成 | Grafana Cloud UI | Medium |
| S7-006-3 | Module Performance ダッシュボード作成 | Grafana Cloud UI | Low |
| S7-006-4 | アラートルール設定 (ダウン検知 + エラー率) | Grafana Cloud UI | Medium |

---

## 12. Grafana Cloud Free Tier 制約

| リソース | 上限 | 現状の使用量 |
|----------|------|-------------|
| Loki ログ | 50 GB/月 | < 1 GB/月 (推定) |
| Synthetic Monitoring | 5 checks | 2 (Worker + Console) |
| Prometheus Series | 10,000 | 0 (未使用) |
| Dashboard | 無制限 | 0 |
| Alert Rules | 100 | 0 |
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
