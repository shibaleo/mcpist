# EXT - MOD インタラクション詳細（dtl-itr-EXT-MOD）

## ドキュメント管理情報

| 項目      | 値                                                 |
| ------- | ------------------------------------------------- |
| Status  | `reviewed`                                        |
| Version | v2.0                                              |
| Note    | External Service API - Modules Interaction Detail |

---

## 概要

| 項目 | 内容 |
|------|------|
| 連携元 | Modules (MOD) |
| 連携先 | External Service API (EXT) |
| 内容 | 外部サービスAPI呼び出し |
| プロトコル | HTTPS |

---

## 詳細

| 項目 | 内容 |
|------|------|
| プロトコル | HTTPS |
| 認証 | TVLから取得したauth_typeに応じた方式 |
| データ形式 | JSON |
| タイムアウト | 30秒 |

---

## 対応サービス

| サービス | 認証方式 | カスタムヘッダー | 備考 |
|----------|----------|----------------|------|
| Notion | OAuth2 / API Key | `Notion-Version: 2022-06-28` | Page, Database, Block, Comment, User操作 |
| GitHub | OAuth2 / API Key | `X-GitHub-Api-Version: 2022-11-28` | Repo, Issue, PR, Actions, Search |
| Jira | Basic / OAuth2 | - | Issue/Project検索、作成、更新、トランジション |
| Confluence | Basic / OAuth2 | - | Space, Page, Search, Comment, Label操作 |
| Supabase | API Key | - | DB操作、Migration、Logs、Storage |
| Google Calendar | OAuth2 | - | Calendar, Event管理（自動トークンリフレッシュ） |
| Microsoft To Do | OAuth2 | - | List, Task管理（自動トークンリフレッシュ） |

---

## 認証方式

### auth_type による認証ヘッダー構築

TVLから取得した`auth_type`に基づきAuthorizationヘッダーを構築。詳細は[dtl-itr-MOD-TVL.md](./dtl-itr-MOD-TVL.md)のauth_type一覧を参照。

| auth_type | Authorizationヘッダー |
|-----------|----------------------|
| `oauth2` | `Authorization: Bearer {access_token}` |
| `api_key` | `Authorization: Bearer {access_token}` |
| `basic` | `Authorization: Basic {base64(username:password)}` |
| `oauth1` | `Authorization: OAuth oauth_consumer_key=..., oauth_token=..., oauth_signature=...` |

### 動的ドメイン指定（Jira/Confluence）

Jira/Confluenceはユーザーごとにドメインが異なるため、TVLのmetadataから取得。

```
https://{metadata.domain}/rest/api/3/...
```

---

## API呼び出し例

### Bearer Token（Notion）

```http
POST https://api.notion.com/v1/search
Authorization: Bearer ntn_xxx
Notion-Version: 2022-06-28
Content-Type: application/json

{
  "query": "設計ドキュメント"
}
```

### Bearer Token（GitHub）

```http
GET https://api.github.com/repos/owner/repo/issues
Authorization: Bearer ghp_xxx
X-GitHub-Api-Version: 2022-11-28
Accept: application/vnd.github+json
```

### Basic認証（Jira）

```http
GET https://example.atlassian.net/rest/api/3/myself
Authorization: Basic dXNlckBleGFtcGxlLmNvbTpBVEFUVDN4RmZHRjA=
Content-Type: application/json
```

### OAuth 1.0a（将来対応予定）

```http
GET https://api.zaim.net/v2/home/money
Authorization: OAuth oauth_consumer_key="xxx", oauth_token="xxx", oauth_signature_method="HMAC-SHA1", oauth_signature="xxx", oauth_timestamp="xxx", oauth_nonce="xxx", oauth_version="1.0"
```

---

## トークンリフレッシュ

### 対象サービス

| サービス | 自動リフレッシュ | バッファ時間 |
|----------|----------------|-------------|
| Google Calendar | ✅ | 5分前 |
| Microsoft To Do | ✅ | 5分前 |
| その他OAuth2 | ❌ | - |

### リフレッシュフロー

```mermaid
sequenceDiagram
    participant MOD as Module
    participant TVL as Token Vault
    participant EAS as External Auth Server
    participant EXT as External API

    MOD->>TVL: トークン取得
    TVL-->>MOD: credentials (expires_at含む)
    MOD->>MOD: expires_at - 5分 < now?
    alt リフレッシュ必要
        MOD->>EAS: トークンリフレッシュリクエスト
        EAS-->>MOD: 新しいaccess_token, refresh_token
        MOD->>TVL: トークン更新保存
    end
    MOD->>EXT: API呼び出し（新トークン使用）
    EXT-->>MOD: レスポンス
```

---

## レスポンス処理

### 処理フロー

1. EXTからJSONレスポンス受信
2. ステータスコード検証（200-299 = 成功）
3. 成功時: JSONをそのまま返却
4. 失敗時: APIErrorとして返却

### 出力フォーマット変換

batchコマンドで`output: true`指定時、モジュール固有のToCompact関数でTOON/Markdown形式に変換。

| モジュール | 変換対象 |
|-----------|---------|
| Notion | Page, Database, Block |
| GitHub | Issue, PR, Actions |
| Jira | Issue, Comment |

---

## エラーハンドリング

### HTTPエラー

| ステータス | 処理 |
|-----------|------|
| 200-299 | 成功（JSONパース） |
| 204 | 成功（空レスポンス、DELETE操作時） |
| 400-499 | クライアントエラー（APIError返却） |
| 500-599 | サーバーエラー（APIError返却） |

### APIError構造

```go
type APIError struct {
    StatusCode int
    Body       string
}
```

---

## 期待する振る舞い

### API呼び出し

- MOD は TVL からサービス別のcredentialsを取得し、auth_typeに応じて認証ヘッダーを構築する
- HTTPS 経由で EXT にリクエストを送信し、30秒以内にレスポンスを待機する
- Jira/Confluence は metadata.domain から動的にベースURLを構築する
- Notion/GitHub は API バージョンヘッダーを追加する

### トークンリフレッシュ

- Google Calendar/Microsoft To Do は expires_at の5分前に自動リフレッシュを実行する
- リフレッシュ後、新しいトークンを TVL に保存し、API呼び出しを継続する
- Microsoft は新しい refresh_token を返す場合があり、その場合も TVL に保存する

### エラー処理

- HTTP ステータス 200-299 を成功として扱う
- DELETE 操作の 204 No Content は成功として扱う
- エラー時は APIError にステータスコードとボディを格納して返却する

---

## 将来実装予定

| 機能 | 説明 |
|------|------|
| リトライ処理 | 一時的なエラー時の exponential backoff |
| レート制限対応 | Retry-After ヘッダーの解釈 |
| エラー標準化 | サービス固有エラーの統一フォーマットへのマッピング |

---

## 関連ドキュメント

| ドキュメント | 内容 |
|-------------|------|
| [itr-MOD.md](./itr-MOD.md) | Modules 詳細仕様 |
| [itr-EXT.md](./itr-EXT.md) | External Service API 詳細仕様 |
| [dtl-itr-MOD-TVL.md](./dtl-itr-MOD-TVL.md) | MOD→TVL トークン取得・リフレッシュ |
