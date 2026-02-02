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

## 次回の作業

1. 変更をコミット
2. 仕様書整備（JWT aud チェック、MCP エラーコード）
3. resources MCP 実装の検討
