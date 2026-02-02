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

## 次回の作業

1. ステージされた変更をコミット
2. Phase 2: Trello モジュール実装
3. Phase 3: Google Docs モジュール実装
