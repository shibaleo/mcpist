# DAY022 作業ログ

## 日付

2026-02-02

---

## 完了タスク

### PKCE 認証エラー修正 ✅

| ID | タスク | 状態 | 備考 |
|----|--------|------|------|
| D22-001 | 初回ログイン時の PKCE エラー調査 | ✅ | `AuthPKCECodeVerifierMissingError` の原因特定 |
| D22-002 | auth/callback Route Handler 修正 | ✅ | cookie を明示的にレスポンスに設定 |
| D22-003 | ブラウザクライアントの cookieOptions 設定 | ✅ | sameSite: 'lax' を明示的に指定 |
| D22-004 | 本番環境でのログイン確認 | ✅ | dev.mcpist.app で正常動作 |

---

## 作業詳細

### 1. 問題の症状

初回ログイン時に以下のエラーが発生：

```
[Auth Callback] exchangeCodeForSession error: PKCE code verifier not found in storage.
This can happen if the auth flow was initiated in a different browser or device,
or if the storage was cleared. For SSR frameworks (Next.js, SvelteKit, etc.),
use @supabase/ssr on both the server and client to store the code verifier in cookies.
```

### 2. 原因分析

**根本原因:** `/auth/callback` Route Handler で使用していた `createClient()` の実装に問題があった。

| 問題点 | 説明 |
|--------|------|
| cookie がレスポンスに反映されない | `cookies()` から取得した cookieStore への `set()` は NextResponse に自動反映されない |
| セッション cookie が保存されない | `exchangeCodeForSession()` が内部で設定する cookie がブラウザに送信されない |

### 3. 修正内容

#### auth/callback/route.ts

**Before:** `createClient()` を使用（server.ts からインポート）

**After:** Route Handler 内で直接 `createServerClient` を作成し、cookie を明示的にレスポンスに設定

```typescript
// cookie を追跡
const cookiesToSet: { name: string; value: string; options?: Record<string, unknown> }[] = []

const supabase = createServerClient(url, key, {
  cookies: {
    getAll() { return cookieStore.getAll() },
    setAll(cookies) {
      cookies.forEach((cookie) => cookiesToSet.push(cookie))
    },
  },
})

// レスポンス時に cookie を設定
const response = NextResponse.redirect(redirectUrl)
cookiesToSet.forEach(({ name, value, options }) => {
  response.cookies.set(name, value, options)
})
return response
```

#### client.ts

ブラウザクライアントに `cookieOptions` を追加：

```typescript
return createBrowserClient<Database>(supabaseUrl, supabaseKey, {
  cookieOptions: {
    sameSite: 'lax',
    secure: true,
    path: '/',
  },
})
```

### 4. デバッグログ追加

トラブルシューティング用にログを残した：

```typescript
const allCookies = cookieStore.getAll()
console.log("[Auth Callback] Received cookies:", allCookies.map(c => c.name))
const pkceVerifier = allCookies.find(c => c.name.includes('code-verifier'))
console.log("[Auth Callback] PKCE code_verifier cookie present:", !!pkceVerifier)
```

### 5. 途中で遭遇したエラー

| エラー | 原因 | 対応 |
|--------|------|------|
| `Flow state not found` | Supabase メンテナンス中 | 待機後に再試行 |
| `code_verifier cookie present: false` | 検索文字列の間違い（`code_verifier` vs `code-verifier`） | ハイフンに修正 |

---

## 変更ファイル

| ファイル | 変更内容 |
|----------|----------|
| `apps/console/src/app/auth/callback/route.ts` | Route Handler 内で createServerClient を作成、cookie を明示的に設定 |
| `apps/console/src/lib/supabase/client.ts` | cookieOptions 追加 |

---

## コミット履歴

| コミット | 内容 |
|----------|------|
| 60b079c | fix(auth): set session cookies explicitly in auth callback response |
| (未コミット) | client.ts の cookieOptions 追加、デバッグログ追加 |

---

---

## Todoist モジュール実装 ✅

### 完了タスク

| ID | タスク | 状態 | 備考 |
|----|--------|------|------|
| D22-001 | `modules/todoist/module.go` 作成 | ✅ | 12ツール実装 |
| D22-002 | OAuth アプリ設定を Supabase に登録 | ✅ | Todoist Developer Console で登録 |
| D22-003 | `main.go` に RegisterModule 追加 | ✅ | server と tools-export 両方 |
| D22-004 | Console に Todoist OAuth 連携 UI 追加 | ✅ | authorize/callback ルート、oauth-apps.ts、services/page.tsx |
| D22-005 | 動作確認 | ✅ | 全12ツールの動作確認完了 |

### 実装したツール

| ツール | 説明 | readOnlyHint | destructiveHint |
|--------|------|--------------|-----------------|
| list_projects | プロジェクト一覧 | true | - |
| get_project | プロジェクト詳細 | true | - |
| list_tasks | タスク一覧（フィルター対応） | true | - |
| get_task | タスク詳細 | true | - |
| create_task | タスク作成 | false | false |
| update_task | タスク更新 | false | false |
| complete_task | タスク完了 | false | false |
| reopen_task | タスク再開 | false | false |
| delete_task | タスク削除 | false | **true** |
| quick_add | 自然言語でタスク追加 | false | false |
| list_sections | セクション一覧 | true | - |
| list_labels | ラベル一覧 | true | - |

### Todoist OAuth の特徴

- **リフレッシュトークンなし**: アクセストークンは長期間有効（revoke されるまで）
- **Sync API**: quick_add のみ Sync API を使用（自然言語パース）
- **スコープ**: `data:read_write`, `data:delete`

---

## OAuth callback リダイレクト修正 ✅

| プロバイダー | 修正内容 |
|--------------|----------|
| Google | デフォルト `/connections` → `/tools` |
| Microsoft | デフォルト `/connections` → `/tools` |
| Todoist | デフォルト `/connections` → `/tools` |

**動作**: `returnTo` パラメータで指定されたページに戻る。フォールバックは `/tools`

---

## ローカル開発環境の Cookie 修正 ✅

`client.ts` で localhost 判定を追加:

```typescript
const isLocalhost = typeof window !== 'undefined' &&
  (window.location.hostname === 'localhost' || window.location.hostname === '127.0.0.1')

return createBrowserClient<Database>(supabaseUrl, supabaseKey, {
  cookieOptions: {
    sameSite: 'lax',
    secure: !isLocalhost,  // HTTP では secure: false
    path: '/',
  },
})
```

---

## 変更ファイル

| ファイル | 変更内容 |
|----------|----------|
| `apps/server/internal/modules/todoist/module.go` | 新規作成（12ツール） |
| `apps/server/cmd/server/main.go` | RegisterModule(todoist.New()) 追加 |
| `apps/server/cmd/tools-export/main.go` | RegisterModule(todoist.New()) 追加 |
| `apps/console/src/app/api/oauth/todoist/authorize/route.ts` | 新規作成 |
| `apps/console/src/app/api/oauth/todoist/callback/route.ts` | 新規作成 |
| `apps/console/src/lib/oauth-apps.ts` | Todoist プロバイダー追加 |
| `apps/console/src/app/(console)/services/page.tsx` | Todoist authConfig 追加 |
| `apps/console/src/lib/supabase/client.ts` | localhost 判定追加 |
| `apps/console/src/app/api/oauth/google/callback/route.ts` | returnTo デフォルト変更 |
| `apps/console/src/app/api/oauth/microsoft/callback/route.ts` | returnTo デフォルト変更 |
| `apps/console/src/lib/tools.json` | Todoist ツール追加 |

---

## コミット履歴

| コミット | 内容 |
|----------|------|
| 2fac647 | feat(todoist): add Todoist MCP module with OAuth support |
| (未コミット) | fix(oauth): change default redirect to /tools and fix cookie secure for localhost |

---

---

## Notion OAuth 対応 ✅

### 完了タスク

| ID | タスク | 状態 | 備考 |
|----|--------|------|------|
| D22-010 | Notion OAuth authorize/callback ルート作成 | ✅ | HTTP Basic Auth でトークン交換 |
| D22-011 | oauth-apps.ts に Notion 追加 | ✅ | スコープは URL パラメータで指定しない |
| D22-012 | services/page.tsx に OAuth/API Key 選択 UI 追加 | ✅ | alternativeAuth パターン |
| D22-013 | metadata の map[string]interface{} 対応 | ✅ | ネストした owner オブジェクト対応 |
| D22-014 | トークンリフレッシュロジック実装 | ✅ | expires_at ベースの事前リフレッシュ |
| D22-015 | 動作確認 | ✅ | OAuth 認証、API 呼び出し、リフレッシュ |

### Notion OAuth の特徴

| 項目 | 内容 |
|------|------|
| トークン形式 | `ntn_xxx`（アクセス）、`nrt_xxx`（リフレッシュ） |
| 旧形式 | `secret_xxx`（2024年9月25日以前） |
| トークン交換 | HTTP Basic Auth（`client_id:client_secret` を Base64） |
| 有効期限 | `expires_in` が返されない（明示的な期限なし） |
| リフレッシュ | `grant_type: refresh_token` で新しいトークンペア取得 |

### 実装ポイント

1. **alternativeAuth パターン**
   - OAuth と内部インテグレーショントークン（API Key）の両方に対応
   - UI で両方のオプションを表示（「または」で区切り）

2. **metadata の型変更**
   - `map[string]string` → `map[string]interface{}`
   - Notion の `owner` オブジェクト（ネスト）に対応

3. **リフレッシュロジック**
   - `expires_at` がある場合のみリフレッシュ（5分前にトリガー）
   - `expires_at` がない場合はリフレッシュしない（現状の Notion 仕様）

### 変更ファイル

| ファイル | 変更内容 |
|----------|----------|
| `apps/console/src/app/api/oauth/notion/authorize/route.ts` | 新規作成 |
| `apps/console/src/app/api/oauth/notion/callback/route.ts` | 新規作成、expires_at 保存 |
| `apps/console/src/lib/oauth-apps.ts` | Notion プロバイダー追加 |
| `apps/console/src/app/(console)/services/page.tsx` | alternativeAuth UI 追加 |
| `apps/server/internal/modules/notion/client.go` | リフレッシュロジック追加 |
| `apps/server/internal/store/token.go` | Metadata 型を interface{} に変更 |

---

## Trello OAuth 1.0a 対応 ✅

### 完了タスク

| ID | タスク | 状態 | 備考 |
|----|--------|------|------|
| D22-020 | Trello OAuth 1.0a authorize ルート作成 | ✅ | HMAC-SHA1 署名生成、Request Token 取得 |
| D22-021 | Trello OAuth 1.0a callback ルート作成 | ✅ | Access Token 取得、Vault 保存 |
| D22-022 | oauth-apps.ts に Trello 追加 | ✅ | OAUTH_PROVIDERS, OAUTH_CONFIGS |
| D22-023 | services/page.tsx を OAuth 方式に変更 | ✅ | API Key 入力から OAuth ボタンへ |
| D22-024 | 全17ツールの動作確認 | ✅ | ボード、リスト、カード、チェックリスト操作 |

### OAuth 1.0a の特徴

| 項目 | 内容 |
|------|------|
| プロトコル | OAuth 1.0a（3-legged） |
| 署名方式 | HMAC-SHA1 |
| トークン有効期限 | 無期限（`expiration: never`） |
| 必要な設定 | API Key + Secret + Allowed Origins |

### OAuth 1.0a フロー

```
1. authorize: Request Token 取得
   POST /1/OAuthGetRequestToken
   ↓
2. ユーザー認可
   GET /1/OAuthAuthorizeToken?oauth_token=xxx
   ↓
3. callback: Access Token 取得
   POST /1/OAuthGetAccessToken
   ↓
4. Vault に保存
   - access_token: OAuth token
   - username: API Key (consumer key)
   - metadata.token_secret: OAuth token secret
```

### 実装ポイント

1. **署名生成**
   - パラメータをアルファベット順にソート
   - Signature Base String = HTTP method + URL + params
   - Signing Key = consumer_secret + & + token_secret
   - HMAC-SHA1 でハッシュ、Base64 エンコード

2. **状態管理**
   - oauth_token_secret を Cookie に Base64url エンコードで保存
   - callback で取得して署名に使用
   - 10分で有効期限切れ

3. **トークン保存形式**
   - Go モジュールとの互換性のため API Key を `username` フィールドに保存
   - Trello API は `key` + `token` クエリパラメータで認証

### テスト結果

| ツール | 結果 |
|--------|------|
| `list_boards` | ✅ |
| `get_board` | ✅ |
| `get_lists` | ✅ |
| `get_cards` | ✅ |
| `get_card` | ✅ |
| `create_card` | ✅ |
| `update_card` | ✅ |
| `move_card` | ✅ |
| `create_checklist` | ✅ |
| `get_checklists` | ✅ |
| `add_checklist_item` | ✅ |
| `get_checklist_items` | ✅ |
| `update_checklist_item` | ✅ |
| `delete_checklist_item` | ✅ |
| `delete_checklist` | ✅ |
| `delete_card` | ✅ |

### 変更ファイル

| ファイル | 変更内容 |
|----------|----------|
| `apps/console/src/app/api/oauth/trello/authorize/route.ts` | 新規作成（OAuth 1.0a Request Token） |
| `apps/console/src/app/api/oauth/trello/callback/route.ts` | 新規作成（Access Token 取得） |
| `apps/console/src/lib/oauth-apps.ts` | Trello プロバイダー追加 |
| `apps/console/src/app/(console)/services/page.tsx` | authConfig を OAuth 方式に変更 |

### コミット履歴

| コミット | 内容 |
|----------|------|
| af566a2 | feat(trello): add OAuth 1.0a authentication support |

---

## GitHub OAuth 対応 ✅

### 完了タスク

| ID | タスク | 状態 | 備考 |
|----|--------|------|------|
| D22-011 | GitHub OAuth authorize/callback ルート作成 | ✅ | OAuth 2.0、CSRF 対策あり |
| D22-012 | oauth-apps.ts に GitHub 追加 | ✅ | OAUTH_PROVIDERS, OAUTH_CONFIGS |
| D22-013 | services/page.tsx に alternativeAuth パターン追加 | ✅ | OAuth + Fine-grained PAT |
| D22-014 | 動作確認 | ✅ | get_user, list_repos 動作確認 |

### GitHub OAuth の特徴

| 項目 | 内容 |
|------|------|
| プロトコル | OAuth 2.0 |
| トークン有効期限 | **無期限**（revoke されるまで有効） |
| リフレッシュトークン | **なし** |
| スコープ | `repo read:user` |
| ツール数 | 20ツール |

### 実装ポイント

1. **alternativeAuth パターン**
   - OAuth をプライマリ、Fine-grained PAT をフォールバックとして提供
   - Notion と同様の UI パターン（「または」で区切り）

2. **トークン保存**
   - `refresh_token: null`、`expires_at: null` で保存
   - リフレッシュ不要のため expires_at を設定しない

3. **CSRF 対策**
   - state パラメータに returnTo を Base64url エンコードで埋め込み
   - callback で検証

### テスト結果

| ツール | 結果 | 備考 |
|--------|------|------|
| `get_user` | ✅ | ユーザー "shibaleo" を取得 |
| `list_repos` | ✅ | リポジトリ一覧取得（mcpist, go-mcp-dev 等） |

### 変更ファイル

| ファイル | 変更内容 |
|----------|----------|
| `apps/console/src/app/api/oauth/github/authorize/route.ts` | 新規作成（OAuth 2.0 認可リクエスト） |
| `apps/console/src/app/api/oauth/github/callback/route.ts` | 新規作成（トークン交換、Vault 保存） |
| `apps/console/src/lib/oauth-apps.ts` | GitHub プロバイダー追加 |
| `apps/console/src/app/(console)/services/page.tsx` | alternativeAuth UI 追加（OAuth + PAT） |

---

## Asana OAuth 対応 ✅

### 完了タスク

| ID | タスク | 状態 | 備考 |
|----|--------|------|------|
| D22-030 | Asana モジュール実装 | ✅ | 12 ツール（読み取り専用） |
| D22-031 | OAuth authorize/callback ルート作成 | ✅ | OAuth 2.0 + リフレッシュトークン |
| D22-032 | oauth-apps.ts に Asana 追加 | ✅ | OAUTH_PROVIDERS, OAUTH_CONFIGS |
| D22-033 | OAuth スコープエラー修正 | ✅ | Identity/OpenID Connect スコープ無効化 |
| D22-034 | expires_at 型互換性修正 | ✅ | FlexibleTime 型導入 |
| D22-035 | 全ツール動作確認 | ✅ | get_me, list_projects, list_tasks |

### Asana OAuth の特徴

| 項目 | 内容 |
|------|------|
| プロトコル | OAuth 2.0 |
| トークン有効期限 | 約1時間（`expires_in` 返却） |
| リフレッシュトークン | あり |
| スコープ | パラメータ不要（アプリ設定で制御） |
| ツール数 | 12ツール（読み取り専用） |

### 実装したツール

| ツール | 説明 | readOnlyHint |
|--------|------|--------------|
| get_me | 認証ユーザー情報取得 | true |
| list_workspaces | ワークスペース一覧 | true |
| get_workspace | ワークスペース詳細 | true |
| list_projects | プロジェクト一覧 | true |
| get_project | プロジェクト詳細 | true |
| list_sections | セクション一覧 | true |
| list_tasks | タスク一覧 | true |
| get_task | タスク詳細 | true |
| list_subtasks | サブタスク一覧 | true |
| list_stories | ストーリー（コメント）一覧 | true |
| list_tags | タグ一覧 | true |
| search_tasks | タスク検索 | true |

### トラブルシューティング

#### 1. OAuth "forbidden_scopes" エラー

**エラー:**
```
forbidden_scopes: Your app is not allowed to request user authorization for `default identity` scopes
```

**原因:** Asana Developer Console で「Identity / OpenID Connect」スコープ（OpenID, Profile）が有効になっていた

**解決:** Developer Console で Identity/OpenID Connect スコープを無効化

#### 2. 401 Unauthorized エラー

**原因:** `expires_at` の型不一致
- Console: ISO 文字列で保存 (`"2026-02-02T13:01:57.797Z"`)
- Go: int64 Unix タイムスタンプを期待

**解決:** `FlexibleTime` カスタム型を導入

```go
// FlexibleTime handles both Unix timestamp (int64) and ISO string formats
type FlexibleTime int64

func (ft *FlexibleTime) UnmarshalJSON(data []byte) error {
    // Try as number first
    var num int64
    if err := json.Unmarshal(data, &num); err == nil {
        *ft = FlexibleTime(num)
        return nil
    }

    // Try as string (ISO format)
    var str string
    if err := json.Unmarshal(data, &str); err == nil {
        t, err := time.Parse(time.RFC3339, str)
        // ...
        *ft = FlexibleTime(t.Unix())
        return nil
    }
    return fmt.Errorf("expires_at must be number or string")
}
```

**影響範囲:** 以下のモジュールを FlexibleTime に更新
- asana/module.go
- google_calendar/module.go
- google_tasks/module.go
- microsoft_todo/module.go
- notion/client.go

### テスト結果

| ツール | 結果 | 備考 |
|--------|------|------|
| `get_me` | ✅ | shibaleo (shiba.dog.leo.private@gmail.com) |
| `list_workspaces` | ✅ | "My workspace" 取得 |
| `list_projects` | ✅ | "shibaleo's first project" 取得 |
| `list_tasks` | ✅ | Task 1, Task 2, Task 3 取得 |

### 変更ファイル

| ファイル | 変更内容 |
|----------|----------|
| `apps/server/internal/modules/asana/module.go` | 新規作成（12ツール） |
| `apps/server/internal/store/token.go` | FlexibleTime 型追加 |
| `apps/server/internal/modules/google_calendar/module.go` | FlexibleTime 対応 |
| `apps/server/internal/modules/google_tasks/module.go` | FlexibleTime 対応 |
| `apps/server/internal/modules/microsoft_todo/module.go` | FlexibleTime 対応 |
| `apps/server/internal/modules/notion/client.go` | FlexibleTime 対応 |
| `apps/server/cmd/server/main.go` | RegisterModule(asana.New()) 追加 |
| `apps/server/cmd/tools-export/main.go` | RegisterModule(asana.New()) 追加 |
| `apps/console/src/app/api/oauth/asana/authorize/route.ts` | 新規作成 |
| `apps/console/src/app/api/oauth/asana/callback/route.ts` | 新規作成 |
| `apps/console/src/lib/oauth-apps.ts` | Asana プロバイダー追加 |

---

## Google OAuth 統合 ✅

### 完了タスク

| ID | タスク | 状態 | 備考 |
|----|--------|------|------|
| D22-040 | OAuth スコープ統合設計 | ✅ | Calendar, Tasks, Drive, Docs, Sheets を 1 回の認証で |
| D22-041 | oauth-apps.ts 修正 | ✅ | 統合スコープ、serviceId = "google" |
| D22-042 | authorize/callback ルート修正 | ✅ | 全スコープ要求、"google" として保存 |
| D22-043 | Go モジュール修正 | ✅ | google_calendar, google_tasks が "google" を参照 |
| D22-044 | Console UI 修正 | ✅ | "google" 接続で Calendar/Tasks 両方接続済みに |
| D22-045 | ビルド確認 | ✅ | Go / TypeScript 両方成功 |

### 設計方針

**問題**: Google Calendar と Google Tasks で別々の OAuth 認証が必要だった
- ユーザーが 2 回認証を行う必要があった
- 将来 Drive, Docs, Sheets 追加でさらに増加

**解決**: 1 つの OAuth 認証で全 Google サービスにアクセス

```
OAuth 認証（1回）
  スコープ: calendar + tasks + drive + docs + sheets
     ↓
user_credentials テーブル
  module = "google" として 1 レコードだけ保存
     ↓
Go モジュール (google_calendar, google_tasks, etc.)
  すべて "google" クレデンシャルを参照
```

### 統合スコープ

```typescript
const GOOGLE_SCOPES = [
  // Calendar
  "https://www.googleapis.com/auth/calendar",
  "https://www.googleapis.com/auth/calendar.events",
  // Tasks
  "https://www.googleapis.com/auth/tasks",
  // Drive
  "https://www.googleapis.com/auth/drive",
  // Docs
  "https://www.googleapis.com/auth/documents",
  // Sheets
  "https://www.googleapis.com/auth/spreadsheets",
]
```

### Console UI の変更

- `services/page.tsx` で "google" 接続があれば `google_calendar`, `google_tasks` も接続済み表示
- `getOAuthProviderForService()` が Google 系モジュールすべてに "google" を返す

### 変更ファイル

| ファイル | 変更内容 |
|----------|----------|
| `apps/console/src/lib/oauth-apps.ts` | 統合スコープ、google-tasks 削除、getOAuthProviderForService 修正 |
| `apps/console/src/app/api/oauth/google/authorize/route.ts` | GOOGLE_SCOPES 統合、module パラメータ廃止 |
| `apps/console/src/app/api/oauth/google/callback/route.ts` | "google" として保存、複数モジュールのツール設定を保存 |
| `apps/console/src/app/(console)/services/page.tsx` | connectedModuleIds に google → calendar/tasks マッピング追加 |
| `apps/server/internal/modules/google_calendar/module.go` | GetModuleToken, UpdateModuleToken を "google" に変更 |
| `apps/server/internal/modules/google_tasks/module.go` | GetModuleToken, UpdateModuleToken を "google" に変更 |

---

## Google Sheets モジュール テスト ✅

### 完了タスク

| ID | タスク | 状態 | 備考 |
|----|--------|------|------|
| D22-050 | google_sheets モジュール全ツールテスト | ✅ | 28ツール全て動作確認 |

### テスト結果

| カテゴリ | ツール | 結果 |
|---------|--------|------|
| **スプレッドシート** | `create_spreadsheet` | ✅ |
| | `get_spreadsheet` | ✅ |
| | `search_spreadsheets` | ✅ |
| **シート操作** | `list_sheets` | ✅ |
| | `create_sheet` | ✅ |
| | `rename_sheet` | ✅ |
| | `duplicate_sheet` | ✅ |
| | `copy_sheet_to` | ✅ |
| | `delete_sheet` | ✅ |
| **値の読み取り** | `get_values` | ✅ |
| | `batch_get_values` | ✅ |
| | `get_formulas` | ✅ |
| **値の書き込み** | `update_values` | ✅ |
| | `batch_update_values` | ✅ |
| | `append_values` | ✅ |
| | `clear_values` | ✅ |
| **行・列操作** | `insert_rows` | ✅ |
| | `delete_rows` | ✅ |
| | `insert_columns` | ✅ |
| | `delete_columns` | ✅ |
| **書式設定** | `format_cells` | ✅ |
| | `merge_cells` | ✅ |
| | `unmerge_cells` | ✅ |
| | `set_borders` | ✅ |
| | `auto_resize` | ✅ |
| **その他** | `find_replace` | ✅ |
| | `protect_range` | ✅ |

### 備考

- スプレッドシートの削除は `google_drive:delete_file` を使用
- テスト用スプレッドシート 2 件を作成し、テスト後に google_drive で削除

---

## Google Apps Script モジュール実装 ✅

### 完了タスク

| ID | タスク | 状態 | 備考 |
|----|--------|------|------|
| D22-060 | コミュニティ実装調査 | ✅ | mohalmah/google-appscript-mcp-server, whichguy/gas_mcp |
| D22-061 | ツール設計 | ✅ | 17ツール（トリガー管理除外） |
| D22-062 | `modules/google_apps_script/module.go` 作成 | ✅ | 943行、全17ツール |
| D22-063 | OAuth スコープ設定追加 | ✅ | oauth-apps.ts に google-apps-script 追加 |
| D22-064 | `main.go` に RegisterModule 追加 | ✅ | server, tools-export 両方 |
| D22-065 | Console UI に authConfig 追加 | ✅ | services/page.tsx、module-data.ts |
| D22-066 | tools.json 再生成 | ✅ | Google Apps Script 17ツール追加 |

### コミュニティ実装調査

| 実装 | ツール数 | 特徴 |
|------|----------|------|
| [mohalmah/google-appscript-mcp-server](https://github.com/mohalmah/google-appscript-mcp-server) | 16 | 基本的な CRUD 操作 |
| [whichguy/gas_mcp](https://github.com/whichguy/gas_mcp) | ~50 | ローカルファイルシステム操作（ls, cat, write, rsync, git）を含む |

**whichguy/gas_mcp が 50 ツールある理由:**
- ローカルファイルシステム操作（ls, cat, write, mkdir, rsync）
- Git 操作（init, status, add, commit, push, pull）
- clasp 連携（login, clone, pull, push, deploy）

**mcpist で実装できない理由:**
- mcpist はステートレスアーキテクチャ（Cloudflare Worker + Go サーバー）
- ローカルファイルシステムへのアクセス不可
- ユーザーのマシン上での clasp/git 操作不可

### 設計判断

1. **トリガー管理除外**: 公式 API エンドポイントが存在しない。`ScriptApp.newTrigger()` の実行で実装可能だが、ハッキー。
2. **copy_project / delete_project 除外**: `google_drive:copy_file` / `google_drive:delete_file` で代替可能（GAS プロジェクトは Drive ファイル）。

### 実装したツール（17ツール）

| カテゴリ | ツール | 説明 | readOnlyHint |
|----------|--------|------|--------------|
| **プロジェクト** | `list_projects` | プロジェクト一覧 | true |
| | `get_project` | プロジェクト詳細 | true |
| | `create_project` | 新規プロジェクト作成 | false |
| | `get_content` | スクリプトファイル内容取得 | true |
| | `update_content` | スクリプトファイル更新 | false |
| **バージョン** | `list_versions` | バージョン一覧 | true |
| | `get_version` | バージョン詳細 | true |
| | `create_version` | バージョン作成（スナップショット） | false |
| **デプロイメント** | `list_deployments` | デプロイメント一覧 | true |
| | `get_deployment` | デプロイメント詳細 | true |
| | `create_deployment` | 新規デプロイメント | false |
| | `update_deployment` | デプロイメント更新 | false |
| | `delete_deployment` | デプロイメント削除 | false (destructive) |
| **実行** | `run_function` | 関数実行 | false |
| | `list_executions` | 実行履歴 | true |
| **モニタリング** | `list_processes` | プロセス一覧 | true |
| | `get_metrics` | メトリクス取得 | true |

### OAuth スコープ

```typescript
"google-apps-script": {
  authUrl: "https://accounts.google.com/o/oauth2/v2/auth",
  scopes: [
    "https://www.googleapis.com/auth/script.projects",
    "https://www.googleapis.com/auth/script.deployments",
    "https://www.googleapis.com/auth/script.metrics",
    "https://www.googleapis.com/auth/script.processes",
    "https://www.googleapis.com/auth/drive.readonly",
  ],
  serviceId: "google_apps_script",
}
```

### 変更ファイル

| ファイル | 変更内容 |
|----------|----------|
| `apps/server/internal/modules/google_apps_script/module.go` | 新規作成（17ツール、943行） |
| `apps/server/cmd/server/main.go` | RegisterModule(google_apps_script.New()) 追加 |
| `apps/server/cmd/tools-export/main.go` | RegisterModule(google_apps_script.New()) 追加 |
| `apps/console/src/lib/oauth-apps.ts` | google-apps-script OAuth 設定追加 |
| `apps/console/src/app/(console)/services/page.tsx` | google_apps_script authConfig 追加 |
| `apps/console/src/lib/module-data.ts` | google_apps_script アイコン追加 |
| `apps/console/src/lib/tools.json` | Google Apps Script 17ツール追加 |

### コミット履歴

| コミット | 内容 |
|----------|------|
| (未コミット) | feat(google_apps_script): add Google Apps Script MCP module with 17 tools |

### テスト結果

| ツール | 結果 | 備考 |
|--------|------|------|
| `list_projects` | ✅ | 4件取得 |
| `get_project` | ✅ | メタデータ取得 |
| `create_project` | ✅ | "MCP Test Project" 作成 |
| `get_content` | ✅ | ソースコード取得 |
| `update_content` | ✅ | Code.gs 更新 |
| `list_versions` | ✅ | 8件取得 |
| `get_version` | ✅ | v8 詳細取得 |
| `create_version` | ✅ | v1 作成 |
| `list_deployments` | ✅ | 8件取得 |
| `get_deployment` | ✅ | デプロイメント詳細取得 |
| `create_deployment` | ✅ | 新規デプロイメント作成 |
| `update_deployment` | ⚠️ | API制限（read-only deployment は変更不可） |
| `delete_deployment` | ✅ | デプロイメント削除 |
| `run_function` | ❌ | 追加スコープ必要（後述） |
| `list_executions` | ✅ | 空の結果（実行履歴なし） |
| `list_processes` | ✅ | 空の結果 |
| `get_metrics` | ❌ | API引数エラー（要調査） |

### 未解決: `run_function` のスコープ問題

**エラー:**
```
Request had insufficient authentication scopes.
ACCESS_TOKEN_SCOPE_INSUFFICIENT
```

**原因:** `scripts.run` API は、実行するスクリプトが使用するリソースに応じたスコープが必要。

**参考:** [Method: scripts.run | Apps Script API](https://developers.google.com/apps-script/api/reference/rest/v1/scripts/run)

**必要なスコープ例:**
- `https://www.googleapis.com/auth/script.scriptapp` - トリガー、権限管理
- `https://www.googleapis.com/auth/spreadsheets` - スプレッドシート操作
- `https://www.googleapis.com/auth/documents` - ドキュメント操作
- `https://www.googleapis.com/auth/drive` - ドライブ操作
- `https://www.googleapis.com/auth/script.external_request` - 外部API呼び出し

**対応方針:**
1. **方針A: 汎用スコープ追加** - `script.scriptapp` と主要なスコープを OAuth 認証時に追加
   - メリット: 多くのスクリプトが実行可能
   - デメリット: 過剰な権限要求、ユーザーが警戒する可能性
2. **方針B: 動的スコープ** - スクリプトの `appsscript.json` を読み取り、必要なスコープを特定
   - メリット: 最小限の権限
   - デメリット: 実装が複雑、再認証が必要になる場合あり
3. **方針C: run_function を除外** - プロジェクト管理に特化し、実行機能は除外
   - メリット: シンプル、セキュリティリスク低
   - デメリット: 機能制限

**推奨:** 方針A で `script.scriptapp` を追加し、簡単なスクリプト実行をサポート。
複雑なスクリプトはユーザーに再認証を促す形で対応。

**対応完了:** ✅
1. `google/authorize/route.ts` の `google_apps_script` スコープに `script.scriptapp` を追加
2. `oauth-apps.ts` の `google-apps-script` スコープも同様に更新
3. 再認証して `run_function` をテスト → 成功

### 解決済み: `get_metrics` のエラー ✅

**エラー:**
```
Request contains an invalid argument.
INVALID_ARGUMENT
```

**原因:** `metricsGranularity` パラメータが必須だったが、指定していなかった

**修正内容:**
- `metricsGranularity` を必須パラメータとして追加（WEEKLY/DAILY、デフォルト: WEEKLY）
- `filter` パラメータを `metrics_granularity` と `deployment_id` に明確化

---

## DAY023 追加作業 (2026-02-03)

### Google Apps Script run_function / get_metrics 修正 ✅

| ID | タスク | 状態 | 備考 |
|----|--------|------|------|
| D23-001 | `script.scriptapp` スコープ追加 | ✅ | authorize/route.ts, oauth-apps.ts |
| D23-002 | `get_metrics` 必須パラメータ修正 | ✅ | `metricsGranularity` (WEEKLY/DAILY) |
| D23-003 | `run_function` テスト | ✅ | API 実行可能デプロイ後に動作確認 |

### 変更ファイル

| ファイル | 変更内容 |
|----------|----------|
| `apps/console/src/app/api/oauth/google/authorize/route.ts` | `script.scriptapp` スコープ追加 |
| `apps/console/src/lib/oauth-apps.ts` | `script.scriptapp` スコープ追加 |
| `apps/server/internal/modules/google_apps_script/module.go` | `get_metrics` に `metricsGranularity` 必須パラメータ追加 |

### テスト結果

| ツール | 結果 | 備考 |
|--------|------|------|
| `get_metrics` | ✅ 成功 | `metricsGranularity=WEEKLY` で動作 |
| `run_function` (引数なし) | ✅ 成功 | `hello()` → "Hello from MCP!" |
| `run_function` (引数あり) | ✅ 成功 | `add(3, 5)` → 8 |

### run_function の前提条件

`run_function` を使用するには、Apps Script プロジェクト側で以下の設定が必要:

1. **GCP プロジェクト紐付け**: Apps Script エディタ → プロジェクトの設定 → GCP プロジェクト番号を設定
2. **API 実行可能デプロイ**: デプロイ → 新しいデプロイ → 「実行可能 API」を選択

### コミット

| コミット | 内容 |
|----------|------|
| (ステージ済み) | fix(google_apps_script): add script.scriptapp scope and fix get_metrics API |

### PostgreSQL モジュール実装 ✅

| ID | タスク | 状態 | 備考 |
|----|--------|------|------|
| D23-004 | PostgreSQL モジュール実装 | ✅ | 7ツール全て動作確認済み |
| D23-005 | pgx v5 依存追加 | ✅ | go.mod に追加 |
| D23-006 | Console UI 統合 | ✅ | authConfig、icon追加 |
| D23-007 | UUID変換修正 | ✅ | `[16]byte` → 文字列形式 |

### 実装したツール

| ツール | 種別 | 説明 |
|--------|------|------|
| `test_connection` | read-only | 接続テスト |
| `list_schemas` | read-only | スキーマ一覧取得 |
| `list_tables` | read-only | テーブル一覧取得 |
| `describe_table` | read-only | テーブル定義取得 |
| `query` | read-only | SELECT実行（max_rows制限付き） |
| `execute` | destructive | INSERT/UPDATE/DELETE実行 |
| `execute_ddl` | destructive | CREATE/ALTER/DROP実行 |

### セキュリティ対策

- localhost/127.0.0.1/::1 接続禁止（SSRF対策）
- `sslmode=require` デフォルト
- SQL種別バリデーション（query→SELECTのみ、execute→DMLのみ、execute_ddl→DDLのみ）
- 行数制限（デフォルト1000、最大10000）
- タイムアウト設定（接続10秒、クエリ30秒）

### 変更ファイル

| ファイル | 変更内容 |
|----------|----------|
| `apps/server/go.mod` | pgx v5.7.2 依存追加 |
| `apps/server/internal/modules/postgresql/module.go` | 新規作成（7ツール実装） |
| `apps/server/cmd/server/main.go` | postgresql モジュール登録 |
| `apps/server/cmd/tools-export/main.go` | postgresql モジュール登録 |
| `apps/console/src/app/(console)/services/page.tsx` | authConfig追加 |
| `apps/console/src/lib/module-data.ts` | icon追加 |
| `apps/console/src/lib/tools.json` | tools-exportで更新 |

### テスト結果

| ツール | 結果 | 備考 |
|--------|------|------|
| `test_connection` | ✅ 成功 | Supabase Session pooler (port 6543) |
| `list_schemas` | ✅ 成功 | pg_temp_*, pg_toast含む |
| `list_tables` | ✅ 成功 | public, auth スキーマ確認 |
| `describe_table` | ✅ 成功 | カラム・インデックス情報取得 |
| `query` | ✅ 成功 | UUID文字列変換動作 |
| `execute` | ✅ 成功 | INSERT 3行 (dummys.test_users) |
| `execute_ddl` | ✅ 成功 | CREATE SCHEMA/TABLE, ALTER TABLE, DROP SCHEMA CASCADE |

### 解決した問題

1. **IPv6接続エラー**: Render は IPv6 未対応。Supabase Session pooler (port 6543) に切り替えて解決
2. **UUID表示問題**: pgx は UUID を `[16]byte` で返す。`convertValue` 関数で文字列形式に変換

### コミット

| コミット | 内容 |
|----------|------|
| `cd6c42a` | feat(postgresql): add PostgreSQL direct connection module with 7 tools |
| `43800d0` | fix(console): add tools.json postgresql service |
| (ステージ済み) | fix(postgresql): convert UUID bytes to string format in query results |

---

## 次回の作業

1. Google OAuth 統合の動作確認（実際に認証フローをテスト）
2. PostgreSQL execute/execute_ddl のエラーハンドリング強化（必要に応じて）
