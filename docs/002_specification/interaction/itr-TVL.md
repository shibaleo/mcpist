# Token Vault インタラクション仕様書（itr-TVL）

## ドキュメント管理情報

| 項目 | 値 |
|------|-----|
| Status | `draft` |
| Version | v2.2 |
| Note | Token Vault Interaction Specification |

---

## 概要

Token Vault（TVL）は、外部サービスのOAuthトークン・API KEYを安全に管理するデータストア。

主な責務：
- 外部サービスのOAuthトークン保存・取得
- APIシークレットの暗号化保存
- トークンの暗号化保存

**注意**: TVL 自体は credentials の保存・取得のみを行う「ストレージ」であり、認証処理やトークンリフレッシュは行わない。

---

## 連携サマリー（dtl-itrまとめ）

### CON → TVL（トークン登録）
- [dtl-itr-CON-TVL.md](./dtl-itr-CON-TVL.md)
  - 外部サービストークン登録・管理

### MOD → TVL（トークン取得・更新）
- [dtl-itr-MOD-TVL.md](./dtl-itr-MOD-TVL.md)
  - トークン取得・リフレッシュ後の保存

### DST
- [dtl-itr-DST-TVL.md](./dtl-itr-DST-TVL.md)
  - トークン管理のためのユーザー紐付け

---

## CON → TVL 連携詳細

### 処理の流れ

```
┌─────────────────────────────────────────────────────────────────────────┐
│ User Console (CON)                                                      │
│                                                                         │
│  ┌──────────────────┐    ┌──────────────────┐    ┌──────────────────┐   │
│  │ モジュール別       │    │ token-validator  │    │ token-vault.ts   │   │
│  │ 設定フォーム       │───>│ (検証)           │───>│ (保存)            │   │
│  │ (各モジュール固有)  │    │ (モジュール別)    │    │ (共通util)        │   │
│  └──────────────────┘    └──────────────────┘    └──────────────────┘   │
│                                   │                        │            │
└───────────────────────────────────│────────────────────────│────────────┘
                                    │                        │
                                    ▼                        ▼
                          ┌──────────────────┐    ┌──────────────────┐
                          │ External API     │    │ Supabase RPC     │
                          │ (検証リクエスト)   │    │ upsert_my_       │
                          │                  │    │ credential       │
                          └──────────────────┘    └──────────────────┘
```

### モジュール別処理とutil処理の分担

| 処理 | 担当 | 備考 |
|------|------|------|
| 設定フォームUI | **モジュール別** | 必要なフィールドがモジュールごとに異なる |
| トークン検証 | **モジュール別** | 検証エンドポイント・認証方式がモジュールごとに異なる |
| credentials 構築 | **共通util** | `buildCredentials()` で auth_type に応じて構築 |
| TVL への保存 | **共通util** | `upsert_my_credential` RPC 呼び出し |

### モジュール別の差異

| モジュール | 設定フォーム | 検証方式 | 備考 |
|-----------|------------|---------|------|
| notion | OAuth連携ボタン | `/v1/users/me` | OAuth 2.0 |
| github | OAuth連携 or PAT入力 | `/user` | 2種類の認証方式 |
| google_calendar | OAuth連携ボタン | `/calendar/v3/users/me/calendarList` | OAuth 2.0 |
| google_docs | OAuth連携ボタン | Google OAuth | OAuth 2.0 |
| google_drive | OAuth連携ボタン | `/drive/v3/about` | OAuth 2.0 |
| google_sheets | OAuth連携ボタン | Google OAuth | OAuth 2.0 |
| google_tasks | OAuth連携ボタン | `/tasks/v1/users/@me/lists` | OAuth 2.0 |
| google_apps_script | OAuth連携ボタン | Google OAuth | OAuth 2.0 |
| microsoft_todo | OAuth連携ボタン | `/me/todo/lists` | OAuth 2.0 (MS Graph) |
| todoist | OAuth連携ボタン | `/sync/v9/sync` | OAuth 2.0（refresh無し） |
| asana | OAuth連携ボタン | `/users/me` | OAuth 2.0 |
| airtable | OAuth連携ボタン | `/v0/meta/whoami` | OAuth 2.0 + PKCE |
| ticktick | OAuth連携ボタン | TickTick API | OAuth 2.0 |
| dropbox | OAuth連携ボタン | `/2/users/get_current_account` | OAuth 2.0 |
| jira | email + API Token + domain | `/rest/api/3/myself` | Basic認証 |
| confluence | email + API Token + domain | `/wiki/rest/api/user/current` | Basic認証 |
| trello | OAuth連携（OAuth 1.0a） | `/1/members/me` | クエリパラメータ方式 |
| supabase | PAT入力 | `/v1/projects` | Management API |
| grafana | Token + base_url | `/api/user` | base_url 必須 |
| postgresql | host/port/db/user/pass | 接続テスト | 直接接続 |

---

## MOD → TVL 連携詳細

### 処理の流れ

```
┌─────────────────────────────────────────────────────────────────────────┐
│ Go Server (MOD)                                                         │
│                                                                         │
│  ┌──────────────────┐    ┌──────────────────┐    ┌──────────────────┐   │
│  │ モジュール別       │    │ store/token.go   │    │ モジュール別       │   │
│  │ module.go        │<───│ (取得)            │───>│ client.go        │   │
│  │ (ツール実行)      │    │ (共通util)        │    │ (API呼び出し)      │  │
│  └──────────────────┘    └──────────────────┘    └──────────────────┘   │
│          │                        ▲                        │            │
│          │                        │                        │            │
│          │        refresh 後に更新 │                        │            │
│          │                        │                        ▼            │
│          │               ┌────────┴────────┐    ┌──────────────────┐    │
│          │               │ リフレッシュ処理  │<───│ External API      │   │
│          │               │ (モジュール別)    │    │ (Resource Server)│    │
│          │               └─────────────────┘    └──────────────────┘    │
│          │                                               │              │
│          └───────────────────────────────────────────────┘              │
└─────────────────────────────────────────────────────────────────────────┘
                                    │
                                    ▼
                          ┌──────────────────┐
                          │ Supabase RPC     │
                          │ get_user_        │
                          │ credential /     │
                          │ upsert_user_     │
                          │ credential       │
                          └──────────────────┘
```

### モジュール別処理とutil処理の分担

| 処理 | 担当 | 備考 |
|------|------|------|
| TVL からの取得 | **共通util** | `GetModuleToken()` |
| 認証ヘッダー生成 | **モジュール別** | auth_type に応じてモジュール側で生成 |
| 有効期限チェック | **モジュール別** | OAuth 2.0 モジュールのみ |
| トークンリフレッシュ | **モジュール別** | リフレッシュ用エンドポイントがモジュールごとに異なる |
| TVL への更新保存 | **共通util** | `UpdateModuleToken()` |

### モジュール別の差異

| モジュール | auth_type | リフレッシュ | 認証ヘッダー生成 |
|-----------|-----------|------------|----------------|
| notion | `oauth2` | あり | `Authorization: Bearer {token}` |
| github | `oauth2`/`api_key` | あり/なし | `Authorization: Bearer {token}` |
| google_calendar | `oauth2` | あり | `Authorization: Bearer {token}` |
| google_docs | `oauth2` | あり | `Authorization: Bearer {token}` |
| google_drive | `oauth2` | あり | `Authorization: Bearer {token}` |
| google_sheets | `oauth2` | あり | `Authorization: Bearer {token}` |
| google_tasks | `oauth2` | あり | `Authorization: Bearer {token}` |
| google_apps_script | `oauth2` | あり | `Authorization: Bearer {token}` |
| microsoft_todo | `oauth2` | あり | `Authorization: Bearer {token}` |
| todoist | `oauth2` | なし | `Authorization: Bearer {token}` |
| asana | `oauth2` | あり | `Authorization: Bearer {token}` |
| airtable | `oauth2` | あり | `Authorization: Bearer {token}` |
| ticktick | `oauth2` | あり | `Authorization: Bearer {token}` |
| dropbox | `oauth2` | あり | `Authorization: Bearer {token}` |
| jira | `basic` | なし | `Authorization: Basic {base64}` |
| confluence | `basic` | なし | `Authorization: Basic {base64}` |
| trello | `api_key` | なし | `?key={api_key}&token={token}` |
| supabase | `api_key` | なし | `Authorization: Bearer {token}` |
| grafana | `api_key` | なし | `Authorization: Bearer {token}` |
| postgresql | `basic` | なし | 接続文字列 |

### 設計理由

OAuth Auth Server と Resource Server は同じサービスのAPIであり、バージョン情報やエンドポイント情報を共有する。そのため、認証関連のスクリプト（リフレッシュ処理含む）はMOD（モジュール）側に配置する。

TVL は純粋なストレージとして、credentials の保存・取得のみを担当する。

---

## モジュール別トークン形式

| モジュール | auth_type | トークン形式 | 備考 |
|-----------|-----------|-------------|------|
| notion | `oauth2` | OAuth Access Token | Internal Integration Token も対応 |
| github | `oauth2`/`api_key` | OAuth Access Token / PAT | 2種類対応 |
| google_calendar | `oauth2` | OAuth Access Token | Google OAuth |
| google_docs | `oauth2` | OAuth Access Token | Google OAuth |
| google_drive | `oauth2` | OAuth Access Token | Google OAuth |
| google_sheets | `oauth2` | OAuth Access Token | Google OAuth |
| google_tasks | `oauth2` | OAuth Access Token | Google OAuth |
| google_apps_script | `oauth2` | OAuth Access Token | Google OAuth |
| microsoft_todo | `oauth2` | OAuth Access Token | MS Graph API |
| todoist | `oauth2` | OAuth Access Token | refresh_token なし |
| asana | `oauth2` | OAuth Access Token | - |
| airtable | `oauth2` | OAuth Access Token | PKCE 対応 |
| ticktick | `oauth2` | OAuth Access Token | - |
| dropbox | `oauth2` | OAuth Access Token | - |
| jira | `basic` | API Token | email + domain 必須 |
| confluence | `basic` | API Token | email + domain 必須 |
| trello | `api_key` | API Key + Token | OAuth 1.0a で取得 |
| supabase | `api_key` | PAT | Management API |
| grafana | `api_key` | Service Account Token | base_url 必須 |
| postgresql | `basic` | username/password | 直接接続 |

---

## TVLが直接やり取りしないコンポーネント

| コンポーネント | 理由 |
|----------------|------|
| MCP Client (CLO/CLK) | GWY経由 |
| API Gateway (GWY) | 直接連携なし |
| Auth Server (AUS) | OAuth2.0はAUS担当 |
| Session Manager (SSM) | DST経由 |
| Auth Middleware (AMW) | MCP Server内部 |
| MCP Handler (HDL) | MOD経由 |
| Observability (OBS) | 直接連携なし |
| Identity Provider (IDP) | SSM経由 |
| External Auth Server (EAS) | EXT経由 |
| External Service API (EXT) | MOD経由 |
| Payment Service Provider (PSP) | DST経由 |

---

## 関連ドキュメント

| ドキュメント | 内容 |
|-------------|------|
| [spc-sys.md](../spc-sys.md) | システム仕様書 |
| [spc-itr.md](spc-itr.md) | インタラクション仕様書 |
|[itf-tvl.md](itf-tvl.md)) | Token Vault API仕様 |
| [itr-CON.md](./itr-CON.md) | User Console詳細仕様 |
| [itr-MOD.md](./itr-MOD.md) | Modules詳細仕様 |
| [itr-DST.md](./itr-DST.md) | Data Store詳細仕様 |




