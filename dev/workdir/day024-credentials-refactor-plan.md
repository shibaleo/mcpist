# Credentials リファクタリング統合計画書

## 基本情報

| 項目 | 値 |
|------|-----|
| 作成日 | 2026-02-05 |
| 関連タスク | S7-025 |
| 目的 | credentials 構造の仕様 v2.2 準拠、Console API リファクタリング |

---

## 変更概要

### 1. Console API リファクタリング

| Before | After | 備考 |
|--------|-------|------|
| `/api/validate-token` | `/api/credentials/validate` | REST スタイル |
| `/api/token-vault` | 削除 | 未使用 |
| 1ファイルに全 validator | validators/ に分割 | 保守性向上 |

### 2. Go Server credentials 構造変更

| Before | After | 備考 |
|--------|-------|------|
| `creds.Username` で api_key | `creds.APIKey` | 正しいフィールド使用 |
| `AuthTypeOAuth1` 定数 | 削除 | 不要 |
| `AuthType2`, `Metadata2` | 削除 | レガシーフィールド |

### 3. Console token-vault.ts 変更

| Before | After | 備考 |
|--------|-------|------|
| `username` に api_key 保存 | `api_key` フィールド | Trello 用 |

---

## 影響範囲

### Console

| ファイル | 変更内容 |
|---------|---------|
| `app/api/credentials/validate/route.ts` | **新規作成** |
| `app/api/credentials/validate/validators/*.ts` | **新規作成** |
| `app/api/validate-token/` | **削除** |
| `app/api/token-vault/` | **削除** |
| `lib/token-validator.ts` | API パス変更 |
| `lib/token-vault.ts` | `api_key` フィールド対応 |

### Go Server

| ファイル | 変更内容 |
|---------|---------|
| `internal/store/token.go` | `APIKey` フィールド追加、レガシー削除、`GetAuthType()`/`GetMetadata()` 削除 |
| `internal/store/token.go` | `GetModuleToken()` 内の正規化処理を簡略化 |
| `internal/modules/trello/module.go` | `APIKey` フィールド使用 |
| `internal/modules/notion/client.go` | `GetAuthType()` → `AuthType` 直接参照 |

---

## 作業手順

### Phase 1: Console validator 分割（独立して実施可能）

| # | 作業 | ファイル | 確認 |
|---|------|---------|------|
| 1-1 | `api/credentials/validate/` ディレクトリ作成 | - | |
| 1-2 | `validators/types.ts` 作成 | validators/types.ts | |
| 1-3 | 各 validator ファイル作成（既存ロジック移植） | validators/*.ts | |
| 1-4 | `validators/index.ts` 作成（registry） | validators/index.ts | |
| 1-5 | `route.ts` 作成 | credentials/validate/route.ts | |
| 1-6 | `lib/token-validator.ts` の API パス変更 | lib/token-validator.ts | |
| 1-7 | Console ビルド確認 | `pnpm build` | |
| 1-8 | 旧 API `/api/validate-token` 削除 | api/validate-token/ | |
| 1-9 | 未使用 API `/api/token-vault` 削除 | api/token-vault/ | |
| 1-10 | Console 再ビルド確認 | `pnpm build` | |

### Phase 2: Go Server credentials 構造変更

| # | 作業 | ファイル | 確認 |
|---|------|---------|------|
| 2-1 | `APIKey` フィールド追加 | store/token.go | |
| 2-2 | `AuthTypeOAuth1` 定数削除 | store/token.go | |
| 2-3 | レガシーフィールド削除（`AuthType2`, `Metadata2`） | store/token.go | |
| 2-4 | `GetAuthType()`, `GetMetadata()` ヘルパー削除 | store/token.go | |
| 2-5 | `GetModuleToken()` 内の正規化処理を簡略化 | store/token.go | |
| 2-6 | Notion モジュール修正（`GetAuthType()` → `AuthType`） | notion/client.go | |
| 2-7 | Trello モジュール修正（`APIKey` 使用） | trello/module.go | |
| 2-8 | Server ビルド確認 | `go build ./cmd/server` | |

### Phase 3: Console token-vault.ts 変更

| # | 作業 | ファイル | 確認 |
|---|------|---------|------|
| 3-1 | `buildCredentials()` で `api_key` 出力 | token-vault.ts | |
| 3-2 | Trello 設定フォーム確認 | 設定フォーム | |
| 3-3 | Console ビルド確認 | `pnpm build` | |

### Phase 4: 統合テスト

| # | モジュール | auth_type | 確認内容 |
|---|-----------|-----------|---------|
| 4-1 | notion | oauth2 | OAuth 連携・ツール実行 |
| 4-2 | github | oauth2/api_key | OAuth 連携・PAT 入力・ツール実行 |
| 4-3 | google_calendar | oauth2 | OAuth 連携・ツール実行 |
| 4-4 | google_docs | oauth2 | OAuth 連携・ツール実行 |
| 4-5 | google_drive | oauth2 | OAuth 連携・ツール実行 |
| 4-6 | google_sheets | oauth2 | OAuth 連携・ツール実行 |
| 4-7 | google_tasks | oauth2 | OAuth 連携・ツール実行 |
| 4-8 | google_apps_script | oauth2 | OAuth 連携・ツール実行 |
| 4-9 | microsoft_todo | oauth2 | OAuth 連携・ツール実行 |
| 4-10 | todoist | oauth2 | OAuth 連携・ツール実行 |
| 4-11 | asana | oauth2 | OAuth 連携・ツール実行 |
| 4-12 | airtable | oauth2 | OAuth 連携・ツール実行 |
| 4-13 | ticktick | oauth2 | OAuth 連携・ツール実行 |
| 4-14 | dropbox | oauth2 | OAuth 連携・ツール実行 |
| 4-15 | jira | basic | API Token 入力・ツール実行 |
| 4-16 | confluence | basic | API Token 入力・ツール実行 |
| 4-17 | trello | api_key | **再接続必須**・ツール実行 |
| 4-18 | supabase | api_key | PAT 入力・ツール実行 |
| 4-19 | grafana | api_key | Token 入力・ツール実行 |
| 4-20 | postgresql | basic | 接続情報入力・クエリ実行 |

---

## 詳細設計

### validators/types.ts

```typescript
// apps/console/src/app/api/credentials/validate/validators/types.ts

export interface ValidationResult {
  valid: boolean
  error?: string
  details?: Record<string, unknown>
}

export interface ValidationParams {
  token: string
  email?: string
  domain?: string
  api_key?: string
  base_url?: string
}

export type ValidatorFunction = (params: ValidationParams) => Promise<ValidationResult>
```

### validators/index.ts

```typescript
// apps/console/src/app/api/credentials/validate/validators/index.ts

import { ValidationParams, ValidationResult, ValidatorFunction } from './types'
import { validateNotionToken } from './notion'
import { validateGitHubToken } from './github'
import { validateSupabaseToken } from './supabase'
import { validateJiraToken } from './jira'
import { validateConfluenceToken } from './confluence'
import { validateTrelloToken } from './trello'
import { validateGrafanaToken } from './grafana'

export type { ValidationResult, ValidationParams }

export const validators: Record<string, ValidatorFunction> = {
  notion: validateNotionToken,
  github: validateGitHubToken,
  supabase: validateSupabaseToken,
  jira: validateJiraToken,
  confluence: validateConfluenceToken,
  trello: validateTrelloToken,
  grafana: validateGrafanaToken,
}

export const requiredParams: Record<string, string[]> = {
  jira: ['email', 'domain'],
  confluence: ['email', 'domain'],
  trello: ['api_key'],
  grafana: ['base_url'],
}
```

### store/token.go - Credentials 構造体

```go
// After
type Credentials struct {
    // auth_type は credentials 内に含まれる
    AuthType string `json:"auth_type,omitempty"`

    // OAuth 2.0
    AccessToken  string       `json:"access_token,omitempty"`
    RefreshToken string       `json:"refresh_token,omitempty"`
    ExpiresAt    FlexibleTime `json:"expires_at,omitempty"`

    // OAuth 1.0a (Zaim 等の将来サービス用)
    ConsumerKey       string `json:"consumer_key,omitempty"`
    ConsumerSecret    string `json:"consumer_secret,omitempty"`
    AccessTokenSecret string `json:"access_token_secret,omitempty"`

    // API Key (Trello, GitHub 等)
    APIKey string `json:"api_key,omitempty"`

    // Basic authentication (Jira, Confluence)
    Username string `json:"username,omitempty"`
    Password string `json:"password,omitempty"`

    // Custom header
    Token      string `json:"token,omitempty"`
    HeaderName string `json:"header_name,omitempty"`

    // Metadata (domain, workspace info 等)
    Metadata map[string]interface{} `json:"metadata,omitempty"`
}
```

### trello/module.go - addAuth 関数

```go
// After
func addAuth(endpoint string, ctx context.Context) string {
    creds := getCredentials(ctx)
    apiKey := creds.APIKey       // 新フィールド
    token := creds.AccessToken
    // ...
}
```

---

## Validator の設計方針

### Validator が必要なモジュール（7種）

手動入力されるトークン（API Key / Basic認証）は事前検証が必要：

| モジュール | auth_type | 検証方法 |
|-----------|-----------|----------|
| notion | api_key | Notion API `/v1/users/me` |
| github | api_key | GitHub API `/user` |
| supabase | api_key | Supabase Management API `/v1/projects` |
| jira | basic | Jira API `/rest/api/3/myself` |
| confluence | basic | Confluence API `/wiki/rest/api/user/current` |
| trello | api_key | Trello API `/1/members/me` |
| grafana | api_key | Grafana API `/api/org` |

### Validator が不要なモジュール（13種）

OAuth 2.0 フローで認証されるモジュールは、認証プロバイダがトークンを発行するため事前検証は不要：

| モジュール | auth_type | 理由 |
|-----------|-----------|------|
| notion | oauth2 | OAuth フロー |
| github | oauth2 | OAuth フロー |
| google_calendar | oauth2 | OAuth フロー |
| google_docs | oauth2 | OAuth フロー |
| google_drive | oauth2 | OAuth フロー |
| google_sheets | oauth2 | OAuth フロー |
| google_tasks | oauth2 | OAuth フロー |
| google_apps_script | oauth2 | OAuth フロー |
| microsoft_todo | oauth2 | OAuth フロー |
| todoist | oauth2 | OAuth フロー |
| asana | oauth2 | OAuth フロー |
| airtable | oauth2 | OAuth フロー |
| ticktick | oauth2 | OAuth フロー |
| dropbox | oauth2 | OAuth フロー |

### postgresql モジュール

接続情報（host, port, user, password, database）の検証は Console 側では行わない。
Server 側で接続時にエラーが返される設計。

---

## ディレクトリ構造（最終形）

```
apps/console/src/app/api/
├── credentials/
│   └── validate/
│       ├── route.ts            # エントリーポイント（50行以下）
│       └── validators/
│           ├── index.ts        # Registry（7種 + default）
│           ├── types.ts        # 共通型
│           ├── notion.ts       # api_key 検証
│           ├── github.ts       # api_key 検証
│           ├── supabase.ts     # api_key 検証
│           ├── jira.ts         # basic 検証
│           ├── confluence.ts   # basic 検証
│           ├── trello.ts       # api_key 検証
│           └── grafana.ts      # api_key 検証
├── oauth/                      # 維持（OAuth フロー用）
├── stripe/                     # 維持
├── user/                       # 維持
├── admin/                      # 維持
└── credits/                    # 維持

削除済み:
├── token-vault/                # 削除
└── validate-token/             # 削除
```

**注**: OAuth 2.0 モジュールは `/api/oauth/` 経由で認証するため、validators には含まれない。
未知のサービスが `validate` API を呼び出した場合は `{ valid: true }` を返す（バリデーションスキップ）。

---

## リスク・注意点

| リスク | 対策 |
|--------|------|
| 既存 Trello credentials が旧形式 | データマイグレーションで変換 |
| Console と Server の不整合 | Phase 2, 3 は同時デプロイ |
| インポートパスのミス | TypeScript/Go の型チェックで検出 |
| 既存動作の変更 | 各 validator のロジックは変更しない |

---

## データマイグレーション

### Trello credentials の変換

既存の Trello credentials は `username` フィールドに api_key が保存されている。
これを `api_key` フィールドに移動する。

**マイグレーションファイル**: `supabase/migrations/20260205000000_migrate_trello_credentials.sql`

```sql
-- Before
{ "auth_type": "api_key", "username": "abc123...", "access_token": "xyz789..." }

-- After
{ "auth_type": "api_key", "api_key": "abc123...", "access_token": "xyz789..." }
```

---

## デプロイ順序

```
1. Phase 1 (Console validator 分割)
   └── Console のみデプロイ可能（独立）

2. Phase 2 + Phase 3 (credentials 構造変更)
   └── 以下を同時に実行:
       ├── Supabase マイグレーション実行
       ├── Console デプロイ
       └── Server デプロイ
```

---

## 完了条件

### Phase 1
- [ ] `/api/credentials/validate` が動作する
- [ ] validators/ ディレクトリ構造が完成
- [ ] route.ts が 50行以下
- [ ] `/api/validate-token` が削除されている
- [ ] `/api/token-vault` が削除されている
- [ ] `pnpm build` が成功

### Phase 2
- [ ] `Credentials.APIKey` フィールドが追加されている
- [ ] `AuthTypeOAuth1` 定数が削除されている
- [ ] レガシーフィールドが削除されている
- [ ] Trello モジュールが `APIKey` を使用
- [ ] `go build ./cmd/server` が成功

### Phase 3
- [ ] `buildCredentials()` が `api_key` を出力
- [ ] `pnpm build` が成功

### Phase 4
- [ ] OAuth 2.0 モジュール（14種）接続テスト成功
  - notion, github, google_*, microsoft_todo, todoist, asana, airtable, ticktick, dropbox
- [ ] Basic 認証モジュール（3種）接続テスト成功
  - jira, confluence, postgresql
- [ ] API Key モジュール（3種）接続テスト成功
  - trello（再接続必須）, supabase, grafana

---

## 参考

- [dtl-itr-MOD-TVL.md](../../docs/002_specification/interaction/dtl-itr-MOD-TVL.md) - MOD→TVL 仕様 v2.2
- [dtl-itr-CON-TVL.md](../../docs/002_specification/interaction/dtl-itr-CON-TVL.md) - CON→TVL 仕様 v2.2
- [itr-TVL.md](../../docs/002_specification/interaction/itr-TVL.md) - TVL 仕様 v2.2
