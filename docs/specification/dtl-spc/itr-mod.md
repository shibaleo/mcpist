# Modules インタラクション仕様書（itr-mod）

## ドキュメント管理情報

| 項目 | 値 |
|------|-----|
| Status | `reviewed` |
| Version | v2.0 |
| Note | Modules Interaction Specification (MCP Server内部) |

---

## 概要

Modules（MOD）は、外部サービス（Notion, Google Calendar等）との連携を実装する個別モジュールの集合。

主な責務：
- 外部サービスAPIの呼び出し
- Token Vaultからのトークン取得
- サービス固有のビジネスロジック実装
- エラーハンドリングとリトライ

**位置づけ:** MCP Server内部コンポーネント

---

## 連携サマリー（spc-itrより）

| 相手 | 方向 | やり取り |
|------|------|----------|
| MCP Handler | MOD ← HDL | プリミティブ操作リクエスト受信 |
| Token Vault | MOD → TVL | トークン取得 |
| Data Store | MOD → DST | クレジット消費 |
| External Service API | MOD → EXT | リソースアクセス（HTTPS） |

---

## 連携詳細

### HDL → MOD（プリミティブ操作リクエスト受信）

| 項目 | 内容 |
|------|------|
| トリガー | HDLからのプリミティブ操作委譲 |
| 入力 | モジュール名、プリミティブ種別、プリミティブ名、パラメータ、ユーザーコンテキスト |

**実行コンテキスト（HDLから受け取る情報）:**

| フィールド | 説明 |
|-----------|------|
| user_id | 認証済みユーザーID |
| module | 対象モジュール名 |
| primitive_type | プリミティブ種別（tool/resource/prompt） |
| primitive_name | プリミティブ名 |
| params | パラメータ |
| request_id | リクエスト追跡用ID |

**MODの処理:**
1. HDLからプリミティブ操作リクエスト受信
2. TVLからユーザーのトークン取得
3. EXTにAPI呼び出し
4. 成功時：DSTにクレジット消費を記録
5. レスポンスを整形してHDLに返却

---

### MOD → TVL（トークン取得）

| 項目 | 内容 |
|------|------|
| トリガー | 外部API呼び出し前 |
| 操作 | ユーザーのサービス別トークン取得 |

**トークン取得リクエスト:**
```json
{
  "user_id": "user-123",
  "service": "notion"
}
```

**トークン取得レスポンス:**
```json
{
  "user_id": "user-123",
  "service": "notion",
  "long_term_token": "ntn_xxx",
  "oauth_token": null
}
```

**トークン優先度:**
1. `oauth_token` が存在すれば使用
2. `oauth_token` がなければ `long_term_token` を使用

**トークン未設定時:**
```json
{
  "error": "no token configured for user: user-123, service: notion"
}
```

---

### MOD → DST（クレジット消費）

| 項目 | 内容 |
|------|------|
| トリガー | ツール/プロンプト実行成功時 |
| 操作 | クレジット残高の減算 |

**消費リクエスト:**
```json
{
  "user_id": "user-123",
  "module": "notion",
  "primitive_type": "tool",
  "primitive_name": "search",
  "amount": 1,
  "request_id": "req-456"
}
```

**消費レスポンス:**
```json
{
  "success": true,
  "credit_balance": 999
}
```

**注:**
- 消費量（amount）は現時点では固定1
- 将来的にモジュール/ツールごとに異なるコスト設定を検討
- 消費記録は監査ログとして保存

---

### MOD → EXT（リソースアクセス）

| 項目 | 内容 |
|------|------|
| プロトコル | HTTPS |
| 認証 | Bearer Token（TVLから取得） |
| データ形式 | JSON（サービスにより異なる） |

**API呼び出し例（Notion）:**
```http
POST https://api.notion.com/v1/search
Authorization: Bearer ntn_xxx
Notion-Version: 2022-06-28
Content-Type: application/json

{
  "query": "設計ドキュメント"
}
```

**レスポンス処理:**
1. EXTからレスポンス受信
2. サービス固有のレスポンス形式を共通形式に変換
3. エラーの場合は適切なエラーコードにマッピング
4. REGに結果を返却

---

## 実装モジュール一覧

| モジュール | サービス | 主なツール |
|------------|----------|-----------|
| notion | Notion | search, create_page, update_page, get_database |
| google_calendar | Google Calendar | list_events, create_event, update_event, delete_event |
| microsoft_todo | Microsoft To Do | list_tasks, create_task, update_task, complete_task |

---

## モジュールインターフェース

各モジュールは以下の責務を実装する。

**モジュールが提供する機能:**

| 機能 | 説明 |
|------|------|
| Name | モジュール名を返却 |
| Description | モジュールの説明を返却 |
| GetSchema | 利用可能なプリミティブ一覧（スキーマ）を返却 |
| Execute | プリミティブを実行し結果を返却 |

**モジュールスキーマ:**

| フィールド | 型 | 説明 |
|-----------|-----|------|
| module | string | モジュール名 |
| description | string | モジュールの説明 |
| tools | array | ツールスキーマの配列 |
| resources | array | リソーススキーマの配列 |
| prompts | array | プロンプトスキーマの配列 |

**ツールスキーマ:**

| フィールド | 型 | 説明 |
|-----------|-----|------|
| name | string | ツール名 |
| description | string | ツールの説明 |
| inputSchema | object | 入力パラメータのJSONスキーマ |

**リソーススキーマ:**

| フィールド | 型 | 説明 |
|-----------|-----|------|
| uri | string | リソースURI |
| name | string | リソース名 |
| description | string | リソースの説明 |
| mimeType | string | MIMEタイプ（オプション） |

**プロンプトスキーマ:**

| フィールド | 型 | 説明 |
|-----------|-----|------|
| name | string | プロンプト名 |
| description | string | プロンプトの説明 |
| arguments | array | 引数定義（オプション） |

**実行結果:**

| フィールド | 型 | 説明 |
|-----------|-----|------|
| success | boolean | 実行成功/失敗 |
| result | object | 成功時の結果データ（オプション） |
| error | object | 失敗時のエラー情報（オプション） |

---

## エラーハンドリング

### トークン未設定

```json
{
  "success": false,
  "error": {
    "code": "TOKEN_NOT_CONFIGURED",
    "message": "Token not configured for service: notion"
  }
}
```

### API呼び出しエラー

```json
{
  "success": false,
  "error": {
    "code": "EXTERNAL_API_ERROR",
    "message": "Notion API returned 401: Invalid token"
  }
}
```

### レート制限

```json
{
  "success": false,
  "error": {
    "code": "RATE_LIMITED",
    "message": "Rate limit exceeded for Notion API",
    "retry_after": 60
  }
}
```

---

## リトライポリシー

| 条件 | リトライ | 最大回数 | バックオフ |
|------|----------|----------|-----------|
| 5xx エラー | する | 3回 | 指数バックオフ |
| 429 Rate Limit | する | 3回 | Retry-Afterに従う |
| 4xx エラー | しない | - | - |
| タイムアウト | する | 2回 | 固定1秒 |

---

## MODが直接やり取りしないコンポーネント

| コンポーネント | 理由 |
|----------------|------|
| MCP Client (CLO/CLK) | GWY/AMW/HDL経由 |
| API Gateway (GWY) | AMW経由 |
| Auth Server (AUS) | GWY経由 |
| Session Manager (SSM) | DST経由 |
| Auth Middleware (AMW) | HDL経由 |
| User Console (CON) | 別アプリケーション |
| Identity Provider (IDP) | SSM経由 |
| External Auth Server (EAS) | TVL経由（トークン取得のみ） |
| Payment Service Provider (PSP) | DST経由 |

---

## 関連ドキュメント

| ドキュメント | 内容 |
|-------------|------|
| [spc-sys.md](../spc-sys.md) | システム仕様書 |
| [spc-itr.md](../spc-itr.md) | インタラクション仕様書 |
| [itr-hdl.md](./itr-hdl.md) | MCP Handler詳細仕様 |
| [itr-dst.md](./itr-dst.md) | Data Store詳細仕様 |
| [itr-tvl.md](./itr-tvl.md) | Token Vault詳細仕様 |
| [itr-ext.md](./itr-ext.md) | External Service API詳細仕様 |
| [itr-srv.md](./itr-srv.md) | MCP Server詳細仕様 |
