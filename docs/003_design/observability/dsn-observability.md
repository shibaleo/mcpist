# Observability 設計書

| 項目 | 内容 |
|------|------|
| 文書ID | dsn-observability |
| ステータス | approved |
| バージョン | v2.0 |
| 更新日 | 2026-02-14 |

---

## 1. 概要

MCPist の Observability 基盤の設計判断を記述する。実装の SSoT は Go コード (`internal/observability/loki.go`) と Grafana Cloud。

### 1.1 設計原則

| 原則 | 説明 |
|------|------|
| **Best-effort** | ログ送信失敗がメイン処理に影響しない |
| **非同期送信** | Go Server: goroutine、Worker: `ctx.waitUntil()` |
| **PII 最小化** | リクエストボディ・トークン・API キーをログに含めない |
| **Information Hiding** | クライアントには曖昧なエラー、サーバーログに詳細 |
| **ゼロコスト運用** | Grafana Cloud Free Tier 内で運用 |

---

## 2. ログ基盤

### 2.1 技術スタック

- **ログ集約**: Grafana Loki (Cloud) — Free Tier: 50GB/月, 保持14日間
- **送信方式**: HTTP Push API (`POST /loki/api/v1/push`, Basic Auth)
- **リクエスト追跡**: Worker が `X-Request-ID` (UUID v4) を生成し Go Server に伝播

### 2.2 標準出力の方針

運用ログは Loki Push に一本化。標準出力 (`log.Printf`) は起動・初期化・Loki 送信失敗に限定。slog への移行は行わない。

---

## 3. ラベル設計

Loki はラベルでストリームを分割する。カーディナリティを低く保つことが重要。

### 3.1 共通ラベル（全エントリ自動付与）

| ラベル | 値 | 説明 |
|--------|-----|------|
| `app` | `mcpist-dev` / `mcpist-prd` | `APP_ENV` |
| `instance` | `worker`, `srv-xxx`, `local` | Instance ID (フォールバックチェーン) |
| `region` | `cloudflare`, `oregon`, `local` | Region (フォールバックチェーン) |

### 3.2 ログ種別ごとのラベル

| ログ種別 | 関数 | ラベル | 備考 |
|----------|------|--------|------|
| ツール実行 | `LogToolCall` | `module`, `status`, `level` | `tool` はカーディナリティ高(270+)のためデータフィールド |
| リクエスト | `LogRequest` | `type=request`, `method`, `path`, `level` | |
| セキュリティ | `LogSecurityEvent` | `type=security`, `level=warn` | `event` はデータフィールド |
| エラー | `LogError` | `type=error`, `level=error` | |

### 3.3 ログレベル

| レベル | 用途 | 例 |
|--------|------|-----|
| `error` | 内部エラー、復旧不可能な障害 | DB 接続失敗、ツール実行内部エラー |
| `warn` | セキュリティイベント | Gateway Secret 不一致、認証失敗 |
| `info` | 正常な操作ログ | ツール実行成功、リクエスト処理 |

---

## 4. セキュリティイベント

`LogSecurityEvent` で記録するイベント一覧。設定ミスやインフラの不整合を検知する目的。

| イベント | 発生条件 | 送信元 |
|----------|----------|--------|
| `invalid_gateway_secret` | X-Gateway-Secret の不一致 | Go Server |
| `batch_permission_denied` | バッチ実行時に許可されていないツールを要求 | Go Server |
| `run_permission_denied` | 単体実行時に許可されていないツールを要求 | Go Server |
| `credit_consume_rejected` | クレジット不足 (HTTP 402) | Go Server |
| `credit_consume_failed` | クレジット消費の DB エラー | Go Server |
| `batch_credit_consume_rejected` | バッチ実行時クレジット不足 | Go Server |
| `batch_credit_consume_failed` | バッチ実行時クレジット消費 DB エラー | Go Server |
| `auth_failed` | JWT/API Key 認証失敗 | Worker |

---

## 5. ヘルスチェック

`GET /health` — Supabase REST API への HEAD リクエストで DB 接続を確認。

| 状態 | HTTP | レスポンス |
|------|------|-----------|
| 正常 | 200 | `{"status":"ok"}` |
| DB 異常 | 503 | `{"status":"degraded","error":"..."}` |

---

## 6. Grafana ダッシュボード

フォルダ: MCPist (uid: `mcpist`)。MCP ツール (`create_update_dashboard`) で作成・更新。

| ダッシュボード | UID | 用途 |
|----------------|-----|------|
| MCPist Overview | `mcpist-observability` | リクエスト監視 + ツール実行サマリ + セキュリティ |
| MCPist Module Performance | `mcpist-module-performance` | モジュール・ツール別パフォーマンス |

---

## 7. アラート設計

Loki の `level` ラベルでフィルタし、Grafana Alerting で検知する。

| アラート | LogQL 条件 | 閾値 | 評価間隔 |
|----------|-----------|------|----------|
| Error Rate | `count_over_time({app=~"mcpist.*", level="error"} [5m])` | >= 5 | 5分 |
| Security Events | `count_over_time({app=~"mcpist.*", level="warn"} [5m])` | >= 10 | 5分 |
| Log Silence | `count_over_time({app=~"mcpist.*"} [15m])` | == 0 | 5分 |

通知先: Grafana OnCall (未設定)

### 7.1 外形監視 (Synthetic Monitoring)

Grafana Cloud Synthetic Monitoring で Worker (`mcp.mcpist.com/health`) と Console (`mcpist.com`) を外形監視する。未設定。

---

## 8. PII 考慮事項

**ログに含めないデータ**: リクエストボディ、Authorization ヘッダー、X-Gateway-Secret 値、OAuth トークン、外部 API レスポンスボディ

**ログに含めるデータ**: user_id (UUID)、request_id、module/tool、duration_ms、status/error message、remote_addr (セキュリティイベント時のみ)

---

## 参考文書

| 文書 | 内容 |
|------|------|
| [dsn-security.md](../security/dsn-security.md) | セキュリティ設計書 |
| [spc-ops.md](spec-operation.md) | 運用仕様書 |
