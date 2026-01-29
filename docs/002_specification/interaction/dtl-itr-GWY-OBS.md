# GWY - OBS インタラクション詳細（dtl-itr-GWY-OBS）

## ドキュメント管理情報

| 項目      | 値                                              |
| ------- | ---------------------------------------------- |
| Status  | `reviewed`                                     |
| Version | v2.0                                           |
| Note    | API Gateway - Observability Interaction Detail |

---

## 概要

| 項目 | 内容 |
|------|------|
| 連携元 | API Gateway (GWY) |
| 連携先 | Observability (OBS) |
| 内容 | リクエスト/認証/ルーティングのログ送信 |
| プロトコル | HTTP（Loki Push API） |

---

## 詳細

| 項目 | 内容 |
|------|------|
| トリガー | 全HTTPリクエスト処理時（認証・プロキシ完了後） |
| 操作 | リクエスト/レスポンス/認証/バックエンド選択のログ送信 |
| 送信タイミング | レスポンス送出後に非同期送信（失敗時はリクエスト処理に影響させない） |
| レート制御 | サンプリング/集約で送信量を制御（詳細は下記） |

### 送信方式（Loki Push API）

- エンドポイント: `POST /loki/api/v1/push`
- 認証: Basic Auth（Grafana Cloud 互換）
- Content-Type: `application/json`
- 送信単位: 1リクエスト=1ログ行（将来的にバッチ化可）

### ログ分類

- `request`: 通常リクエストログ
- `auth`: 認証結果ログ
- `routing`: バックエンド選択/フォールバックログ
- `error`: 例外/タイムアウト/通信失敗ログ

### 送信するログ情報（request）

| フィールド | 説明 |
|-----------|------|
| method | HTTPメソッド |
| path | リクエストパス |
| status | HTTPステータスコード |
| duration | レスポンス時間 |
| user_id | 認証済みユーザーID（認証成功時） |
| auth_type | 認証方式（jwt / api_key） |
| request_id | リクエスト追跡用ID |
| backend | 応答元バックエンド（primary / secondary / none） |
| cache | キャッシュ種別と結果（api_key_kv: hit / soft_expired / miss） |
| client_type | 推定クライアント種別（cli / web / unknown） |
| region | 実行リージョン（Workerのcolocation等） |

### 送信するログ情報（auth）

| フィールド | 説明 |
|-----------|------|
| request_id | リクエスト追跡用ID |
| auth_type | 認証方式（jwt / api_key） |
| result | 認証結果（success / failure） |
| reason | 失敗理由（missing_token / invalid_token / expired / revoked / unknown） |
| user_id | 認証済みユーザーID（成功時のみ） |

### 送信するログ情報（routing）

| フィールド | 説明 |
|-----------|------|
| request_id | リクエスト追跡用ID |
| selected_backend | 選択されたバックエンド（primary / secondary） |
| failover | フェイルオーバー有無（true / false） |
| error | フェイルオーバー理由（timeout / 5xx / network など） |
| backend_latency | バックエンド応答時間 |

### 送信するログ情報（error）

| フィールド | 説明 |
|-----------|------|
| request_id | リクエスト追跡用ID |
| context | エラー発生箇所（auth / proxy / backend / internal） |
| error | エラー概要（短いメッセージ） |
| status | 返却ステータス（可能な場合） |

### ログラベル（Loki stream labels）

| ラベル | 説明 |
|--------|------|
| app | `mcpist` |
| env | `dev` / `stg` / `prd` |
| type | `request` / `auth` / `routing` / `error` |
| auth_type | `jwt` / `api_key`（存在する場合のみ） |
| backend | `primary` / `secondary`（存在する場合のみ） |

### サンプリング/集約方針

- `request` は通常 100% 送信（負荷増大時は 10% までサンプリング可能）
- `error` / `auth` / `routing` は 100% 送信
- `health` や `OPTIONS` など低価値リクエストは省略可能

### セキュリティ/PII

- リクエストボディ/ヘッダの生値は送信しない
- APIキー/JWT等の機密値はログに含めない
- user_id は認証成功時のみ送信

---

## 関連ドキュメント

| ドキュメント                             | 内容               |
| ---------------------------------- | ---------------- |
| [itr-GWY.md](./itr-GWY.md)         | API Gateway 詳細仕様 |
