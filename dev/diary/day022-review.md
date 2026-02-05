# DAY022 振り返り（学び）

**期間:** 2026-02-02 〜 2026-02-03

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

## 9. Trello は OAuth 1.0a

### OAuth 2.0 との違い

Trello は **OAuth 1.0a** を使用している。OAuth 2.0 とは認証フローが大きく異なる。

| 項目 | OAuth 2.0 | OAuth 1.0a |
|------|-----------|------------|
| トークン取得 | 1回のリクエスト | 3ステップ（Request Token → Authorize → Access Token） |
| 署名 | 不要（HTTPS に依存） | HMAC-SHA1 署名が必須 |
| リフレッシュトークン | あり | なし（トークンは無期限） |
| 複雑さ | シンプル | 複雑（署名生成が必要） |

### OAuth 1.0a の 3-legged フロー

```
1. Request Token 取得
   POST /1/OAuthGetRequestToken
   Authorization: OAuth oauth_consumer_key=xxx, oauth_signature=xxx, ...
   → oauth_token, oauth_token_secret を取得

2. ユーザー認可
   GET /1/OAuthAuthorizeToken?oauth_token=xxx
   → ユーザーが許可 → callback に oauth_token, oauth_verifier が返る

3. Access Token 取得
   POST /1/OAuthGetAccessToken
   Authorization: OAuth oauth_token=xxx, oauth_verifier=xxx, oauth_signature=xxx, ...
   → 最終的な oauth_token, oauth_token_secret を取得
```

### 署名生成の実装

OAuth 1.0a で最も複雑な部分は**署名生成**：

```typescript
function generateOAuthSignature(
  method: string,
  url: string,
  params: Record<string, string>,
  consumerSecret: string,
  tokenSecret: string = ""
): string {
  // 1. パラメータをアルファベット順にソート
  const sortedParams = Object.keys(params)
    .sort()
    .map((key) => `${encodeURIComponent(key)}=${encodeURIComponent(params[key])}`)
    .join("&")

  // 2. Signature Base String を作成
  const signatureBaseString = [
    method.toUpperCase(),
    encodeURIComponent(url),
    encodeURIComponent(sortedParams),
  ].join("&")

  // 3. Signing Key を作成
  const signingKey = `${encodeURIComponent(consumerSecret)}&${encodeURIComponent(tokenSecret)}`

  // 4. HMAC-SHA1 でハッシュ
  const signature = crypto
    .createHmac("sha1", signingKey)
    .update(signatureBaseString)
    .digest("base64")

  return signature
}
```

### 状態管理の課題

OAuth 1.0a では `oauth_token_secret` を Step 1 と Step 3 の間で保持する必要がある：

| 方法 | メリット | デメリット |
|------|---------|-----------|
| Session/DB | 安全 | 複雑、追加のストレージ必要 |
| Cookie | シンプル | クライアントに露出（要暗号化） |
| URL パラメータ | × | セキュリティリスク |

**今回の実装:** Cookie に Base64url エンコードで保存（10分で有効期限切れ）

### API 呼び出し時の認証

Trello API は OAuth 1.0a で取得したトークンでも、シンプルに `key` + `token` クエリパラメータで認証できる：

```
GET https://api.trello.com/1/boards/xxx?key=API_KEY&token=ACCESS_TOKEN
```

署名を毎回生成する必要がないため、API 呼び出し側の実装は OAuth 2.0 と同様にシンプル。

### 教訓

1. **OAuth 1.0a は現役** - Trello のような大手サービスでもまだ使われている
2. **署名生成が最大のハードル** - パラメータの順序、エンコード、ハッシュ方式を正確に実装
3. **状態管理が必要** - Request Token と Access Token の間で秘密情報を保持
4. **API 側はシンプルな場合も** - 認証フローは複雑でも、API 呼び出しはクエリパラメータで済むことがある

---

## 10. pgx の UUID 型は [16]byte で返る

### 問題

PostgreSQL の `uuid` 型カラムを pgx で取得すると、Go 側では `[16]byte` として返される。これをそのまま JSON にシリアライズすると、バイト配列として出力される：

```json
{
  "id": [19, 145, 213, 68, 245, 252, 65, 131, ...]
}
```

期待される形式：
```json
{
  "id": "1391d544-f5fc-4183-a277-ed0457816108"
}
```

### 原因

pgx は PostgreSQL のネイティブ型を Go の型に直接マッピングする：

| PostgreSQL 型 | pgx Go 型 |
|---------------|----------|
| `uuid` | `[16]byte` |
| `text` | `string` |
| `integer` | `int32` |
| `timestamp` | `time.Time` |

`rows.Values()` で取得した値をそのまま JSON にすると、`[16]byte` は配列として出力される。

### 解決策

Go 側で UUID バイト配列を文字列形式に変換するヘルパー関数を実装：

```go
func convertValue(v interface{}) interface{} {
    if v == nil {
        return nil
    }

    // Check for [16]byte (UUID)
    if b, ok := v.([16]byte); ok {
        return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
            b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
    }

    // Check for []byte that might be a UUID (16 bytes)
    if b, ok := v.([]byte); ok && len(b) == 16 {
        return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
            b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
    }

    return v
}
```

### 代替案との比較

| 方法 | メリット | デメリット |
|------|---------|-----------|
| Go 側変換（採用） | JOIN などでも正しく動作 | 変換ロジック必要 |
| SQL で `::text` キャスト | シンプル | JOIN 時に毎回キャスト必要 |
| pgx カスタム型登録 | 一箇所で対応 | 設定が複雑 |

### 教訓

1. **ライブラリの型マッピングを把握する** - pgx は PostgreSQL 型を忠実にマッピング
2. **JSON 出力を必ず確認** - Go の型と JSON の型は1対1ではない
3. **変換は API レイヤーで行う** - SQL を複雑にするより、Go 側で変換するほうが保守しやすい

---

## まとめ

1. Route Handler で cookie を設定するときは `NextResponse.cookies.set()` を使う
2. OAuth フローでは `SameSite: 'lax'` が必要
3. エラーメッセージの変化は問題の前進を示す
4. 本番でしか再現しない問題はデバッグログをデプロイして追う
5. Supabase SSR の動作はコンテキスト依存、ドキュメントを注意深く読む
6. OAuth プロバイダーの仕様は変わる（トークン形式、有効期限）—防御的に実装
7. 汎用構造体は柔軟な型（interface{}）で将来の拡張に備える
8. **OAuth 1.0a は現役**—署名生成と状態管理が OAuth 2.0 との主な違い
9. **pgx の UUID は `[16]byte`**—JSON 出力前に文字列形式へ変換が必要
