# Modules インタラクション仕様書（itr-mod）

## ドキュメント管理情報

| 項目 | 値 |
|------|-----|
| Status | `draft` |
| Version | v2.1 |
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

| 相手                   | 方向        | やり取り            |
| -------------------- | --------- | --------------- |
| MCP Handler          | MOD ← HDL | プリミティブ操作リクエスト受信 |
| Token Vault          | MOD → TVL | トークン取得          |
| Data Store           | MOD → DST | クレジット消費         |
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

TVLは有効なトークンを1つ返す。トークン選択・リフレッシュはTVLの責務。

*Bearer Token形式（OAuth 2.0、長期トークン）:*
```json
{
  "user_id": "user-123",
  "service": "notion",
  "auth_type": "bearer",
  "credentials": {
    "access_token": "ntn_xxx"
  }
}
```

*OAuth 1.0a形式:*
```json
{
  "user_id": "user-123",
  "service": "zaim",
  "auth_type": "oauth1",
  "credentials": {
    "consumer_key": "xxx",
    "consumer_secret": "xxx",
    "access_token": "xxx",
    "access_token_secret": "xxx"
  }
}
```

**auth_type一覧（APIリクエスト時の認証方式）:**

| auth_type | 説明 | credentials | Authorizationヘッダー |
|-----------|------|-------------|----------------------|
| `bearer` | Bearer Token | `access_token` | `Authorization: Bearer {token}` |
| `oauth1` | OAuth 1.0a署名 | `consumer_key`, `consumer_secret`, `access_token`, `access_token_secret` | `Authorization: OAuth ...` |
| `basic` | Basic認証 | `username`, `password` | `Authorization: Basic {base64}` |
| `custom_header` | カスタムヘッダー | `token`, `header_name` | `{header_name}: {token}` |

**注:** OAuth 2.0で取得したトークンも長期トークン（APIキー）も、APIリクエスト時は `bearer` として扱う。

**トークン選択ロジック（TVL側で実行）:**
1. OAuth 2.0トークンが存在すれば優先使用（期限切れならリフレッシュ）
2. OAuth 2.0トークンがなければ長期トークンを使用
3. どちらもなければエラー

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
| トリガー | ツール実行成功時 |
| 操作 | クレジット残高の減算 |
| 対象 | 外部API呼び出しを伴うツール（メタツールを除く） |
| 除外 | get_module_schema, run, batch（メタツール）、リソース取得 |

**消費リクエスト:**
```json
{
  "user_id": "user-123",
  "module": "notion",
  "tool": "search",
  "amount": 1,
  "request_id": "req-456",
  "task_id": null
}
```

| フィールド | 必須 | 説明 |
|-----------|------|------|
| user_id | ✅ | ユーザーID |
| module | ✅ | モジュール名 |
| tool | ✅ | ツール名 |
| amount | ✅ | 消費量（現時点では固定1） |
| request_id | ✅ | リクエスト追跡用ID |
| task_id | ✅ | batch内タスクID（runの場合はnull） |

**run/batch の識別:**

| 呼び出し方式 | task_id | 説明 |
|-------------|---------|------|
| run | `null` | 単発ツール実行 |
| batch | `"task_name"` | batch内の各ツール実行 |

**設計意図:** task_idにより run/batch の利用状況を分析可能。

**消費レスポンス:**
```json
{
  "success": true
}
```

**注:** 監査・請求・分析の詳細設計は[dsn-adt.md](../../design/dsn-adt.md)を参照。

---

### MOD → EXT（リソースアクセス）

| 項目 | 内容 |
|------|------|
| プロトコル | HTTPS |
| 認証 | TVLから取得したauth_typeに応じた方式 |
| データ形式 | JSON（サービスにより異なる） |

**認証方式:** TVLから取得した`auth_type`に基づきAuthorizationヘッダーを構築。詳細は「MOD → TVL（トークン取得）」セクションのauth_type一覧を参照。

**API呼び出し例（Bearer Token - Notion）:**
```http
POST https://api.notion.com/v1/search
Authorization: Bearer ntn_xxx
Notion-Version: 2022-06-28
Content-Type: application/json

{
  "query": "設計ドキュメント"
}
```

**API呼び出し例（OAuth 1.0a - Zaim）:**
```http
GET https://api.zaim.net/v2/home/money
Authorization: OAuth oauth_consumer_key="xxx", oauth_token="xxx", oauth_signature_method="HMAC-SHA1", oauth_signature="xxx", oauth_timestamp="xxx", oauth_nonce="xxx", oauth_version="1.0"
```

**レスポンス処理:**
1. EXTからレスポンス受信
2. サービス固有のレスポンス形式を共通形式に変換
3. エラーの場合は適切なエラーコードにマッピング
4. HDLに結果を返却

---

## モジュール例

| モジュール | サービス | 主なツール |
|------------|----------|-----------|
| notion | Notion | search, create_page, update_page, get_database |
| google_calendar | Google Calendar | list_events, create_event, update_event, delete_event |
| microsoft_todo | Microsoft To Do | list_tasks, create_task, update_task, complete_task |
| zaim | Zaim | list_money, create_money |

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
| [itf-mod.md](../dtl-spc/itf-mod.md) | モジュールインターフェース仕様 |
| [dsn-adt.md](../../design/dsn-adt.md) | 監査・請求・分析設計書 |
| [dsn-err.md](../../design/dsn-err.md) | エラーハンドリング設計書 |
