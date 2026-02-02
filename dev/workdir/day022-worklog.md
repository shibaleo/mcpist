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

## 次回の作業

1. Phase 4: Google Docs モジュール実装
2. Phase 5: Asana モジュール実装
3. Phase 6: PostgreSQL モジュール実装
