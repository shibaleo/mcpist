# DAY022 振り返り（学び）

## 1. Next.js Route Handler での Cookie 処理

### `cookies()` の罠

Next.js の `cookies()` で取得した cookieStore に `set()` しても、Route Handler から返す `NextResponse` には**自動的に反映されない**。

**間違った理解:**
```typescript
const cookieStore = await cookies()
cookieStore.set('key', 'value')  // これでレスポンスに含まれる...はず
return NextResponse.redirect(url)  // 実際は cookie が含まれない
```

**正しい方法:**
```typescript
const response = NextResponse.redirect(url)
response.cookies.set('key', 'value', options)  // 明示的にレスポンスに設定
return response
```

**教訓:** Route Handler では、Supabase の `createClient()` をそのまま使わず、cookie の設定を追跡して明示的にレスポンスに含める必要がある。

---

## 2. Supabase PKCE 認証フロー

### フローの理解

```
1. ログインページ (Client)
   └─ signInWithOAuth() → code_verifier を cookie に保存
                        → OAuth プロバイダーへリダイレクト

2. OAuth プロバイダー
   └─ ユーザー認証 → /auth/callback?code=xxx へリダイレクト

3. Auth Callback (Server)
   └─ exchangeCodeForSession(code)
      → cookie から code_verifier を読み取り
      → Supabase へトークン交換リクエスト
      → セッション cookie を設定
```

**重要:** Step 3 で cookie を正しくレスポンスに含めないと、セッションが確立されない。

---

## 3. Cookie の SameSite 属性

### OAuth リダイレクトと cookie

| SameSite | OAuth コールバック時の動作 |
|----------|---------------------------|
| `Strict` | cookie が送信されない（クロスサイトリダイレクトのため） |
| `Lax` | TOP-LEVEL ナビゲーションでは送信される |
| `None` | 常に送信（Secure 必須） |

OAuth フローでは `Lax` が適切。`Strict` だと code_verifier cookie がコールバック時に送信されない。

---

## 4. デバッグの進め方

### エラーメッセージを段階的に追う

| 順番 | エラー | 意味 | 対応 |
|------|--------|------|------|
| 1 | `PKCE code verifier not found` | cookie が届いていない | cookie 設定を確認 |
| 2 | `Flow state not found` | Supabase 側の問題 | メンテナンス確認 |
| 3 | (成功) | cookie 到達、セッション確立 | - |

**教訓:** エラーメッセージが変わったら、問題が前進している証拠。同じエラーが続くなら修正が効いていない。

---

## 5. 本番環境での認証デバッグ

### ローカルでは再現しない問題

OAuth 認証の問題はローカル環境では再現しにくい：
- リダイレクト URL が違う
- cookie のドメインが違う
- HTTPS vs HTTP

**対策:**
1. デバッグログを本番にデプロイ
2. ログから cookie の有無を確認
3. 一つずつ原因を潰す

---

## 6. @supabase/ssr の正しい使い方

### Server Components vs Route Handlers

| コンテキスト | cookie 処理 |
|-------------|------------|
| Server Component | `cookies()` で読み取りのみ（書き込みは無視される） |
| Route Handler | 明示的に `NextResponse.cookies.set()` が必要 |
| Middleware | `request.cookies` と `response.cookies` 両方操作 |

**教訓:** 同じ `createClient()` でも、呼び出すコンテキストで動作が異なる。特に cookie の書き込みは要注意。

---

---

## 7. Notion OAuth トークンの仕様変更

### トークン形式の変更（2024年9月25日〜）

| 時期 | アクセストークン | リフレッシュトークン |
|------|-----------------|---------------------|
| 以前 | `secret_xxx` | なし（長期トークン） |
| 現在 | `ntn_xxx` | `nrt_xxx` |

**背景:** セキュリティスキャナーとの互換性向上、トークン識別の明確化

### 有効期限が明示されない問題

Notion の OAuth レスポンスには `expires_in` フィールドが含まれていない：

```json
{
  "access_token": "ntn_xxx",
  "refresh_token": "nrt_xxx",
  "token_type": "bearer",
  "bot_id": "xxx",
  "workspace_id": "xxx",
  "workspace_name": "..."
  // expires_in がない！
}
```

**対応戦略:**
1. `expires_at` がなければリフレッシュしない
2. 将来 Notion が `expires_in` を返し始めたら自動対応
3. 401 エラー時のリトライは将来の拡張として保留

### リフレッシュの実装パターン

```go
func needsRefresh(creds *store.Credentials) bool {
    if creds.ExpiresAt == 0 {
        return false  // 期限不明ならリフレッシュしない
    }
    now := time.Now().Unix()
    return now >= (creds.ExpiresAt - tokenRefreshBuffer)
}
```

**教訓:**
- OAuth プロバイダーの仕様は変わる（トークン形式、有効期限の有無）
- 防御的なコーディング：期限がなければ無期限として扱う
- 公式ドキュメントに明記されていない仕様は実際のレスポンスで確認

---

## 8. ネストした JSON の型定義

### Go での対応

サービスによって metadata の構造が異なる：

| サービス | metadata の構造 |
|----------|----------------|
| Atlassian | `{"domain": "xxx.atlassian.net"}` |
| Notion | `{"owner": {"type": "user", "user": {...}}}` |

**問題:** `map[string]string` ではネストしたオブジェクトを保存できない

**解決:** `map[string]interface{}` に変更

```go
// Before
Metadata map[string]string `json:"metadata,omitempty"`

// After
Metadata map[string]interface{} `json:"metadata,omitempty"`
```

**教訓:** 汎用的なクレデンシャル構造体を設計する際は、将来の拡張性を考慮して柔軟な型を使う

---

## まとめ

1. Route Handler で cookie を設定するときは `NextResponse.cookies.set()` を使う
2. OAuth フローでは `SameSite: 'lax'` が必要
3. エラーメッセージの変化は問題の前進を示す
4. 本番でしか再現しない問題はデバッグログをデプロイして追う
5. Supabase SSR の動作はコンテキスト依存、ドキュメントを注意深く読む
6. OAuth プロバイダーの仕様は変わる（トークン形式、有効期限）—防御的に実装
7. 汎用構造体は柔軟な型（interface{}）で将来の拡張に備える
