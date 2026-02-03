# OIDC login: 現状 vs 正しい実装 比較と実装計画

作成日: 2026-02-03
対象: apps/console ログインフロー（UIのGitHub/Microsoftボタンは無視）

## 1) 現状実装 vs 正しいOpenID Connect（比較表）

| 観点 | 現状の実装（把握できた範囲） | 正しいOIDC実装（推奨） | ギャップ/リスク |
|---|---|---|---|
| プロトコル | Supabase auth の OAuth flow を使用。`apps/console/src/app/login/page.tsx` で `supabase.auth.signInWithOAuth()` | OIDC Authorization Code + PKCE | OIDCの必須要素（ID Token検証/nonce）不足の可能性 |
| Discovery | なし（明示的エンドポイント指定）。`apps/console/next.config.ts` で `NEXT_PUBLIC_OAUTH_SERVER_URL` | `/.well-known/openid-configuration` からメタデータ取得 | Issuer/endpointの固定化で環境差異に弱い |
| state | Supabase内部に依存（コード側では明示管理なし）。`apps/console/src/app/login/page.tsx` | `state` を自前で生成し、HttpOnly cookieで照合 | CSRF対策の明示性不足 |
| nonce | なし | `nonce` を発行しID Tokenの `nonce` を検証 | リプレイ対策不足 |
| PKCE | Supabaseが code_verifier cookie を発行。`apps/console/src/app/auth/callback/route.ts` で cookie ログ | code_verifier を自前生成しcookie保存 | code_verifier cookie 不達時の失敗をコントロールできない |
| Cookie属性 | `apps/console/src/lib/supabase/client.ts` の `cookieOptions`（sameSite=lax, secure=!isLocalhost） | SameSite=Lax, Secure, Path=/, 可能なら `__Host-` prefix | ホスト/HTTPS差で初回失敗が起き得る |
| コールバック | `/auth/callback` で `exchangeCodeForSession`。`apps/console/src/app/auth/callback/route.ts` | `/oidc/callback` で token交換→ID Token検証→セッション発行 | ID Token検証が欠落、エラーの原因切り分けが難しい |
| セッション | Supabaseセッションcookieに依存。`apps/console/src/lib/supabase/middleware.ts` で `getUser()` | アプリ独自のセッションcookie（短期/長期） | Supabase依存と実装の不整合が起きやすい |
| Middleware | `apps/console/middleware.ts` → `updateSession()` が `getUser()` で判定 | アプリセッション検証のみ（署名/期限/失効） | 外部依存と障害切り分けが難しい |
| ログ | `/auth/callback` にログあり。`apps/console/src/app/auth/callback/route.ts` | 主要ステップの構造化ログ（state/nonce/iss/aud/exp） | 再現性のない障害で原因追跡が困難 |
| リダイレクト | `/auth/callback?returnTo=` に依存。`apps/console/src/app/login/page.tsx` | 許可リスト化（returnToの検証） | Open Redirect のリスク |

## 2) 正しいOIDCの正規フロー（推奨）

1. `/oidc/login` で以下を生成
   - `state`, `nonce`, `code_verifier`
   - `code_challenge = S256(code_verifier)`
2. HttpOnly cookie で `state/nonce/code_verifier` を保存
3. Authorization Endpoint にリダイレクト
   - response_type=code
   - scope=openid profile email
   - client_id, redirect_uri
   - state, nonce
   - code_challenge, code_challenge_method=S256
4. `/oidc/callback` で `state` を照合
5. Token Endpoint に `authorization_code` + `code_verifier` で交換
6. 受け取った `id_token` を検証
   - `iss`, `aud`, `exp`, `nonce` を検証
   - JWKS で署名検証
7. アプリ独自のセッションcookie発行
8. returnTo を検証してリダイレクト

## 3) 実装計画（最小構成 → 安定化）

### Phase 0: 前提整理（1日）
- OIDC提供者を確定（例: Supabase Auth / 外部IdP）
- 本番/開発のベースURLを固定（HTTPS必須）
- returnTo 許可リストの定義（相対パスのみ）
- 既存Supabase依存コードの影響範囲を棚卸し

### Phase 1: 最小OIDC実装（2-3日）
- `apps/console/src/app/oidc/login/route.ts` を新設
  - discovery 取得
  - state/nonce/code_verifier 発行
  - cookie保存（HttpOnly/SameSite=Lax/Secure/Path=/）
- `apps/console/src/app/oidc/callback/route.ts` を新設
  - state照合
  - token交換
  - id_token 検証（jose等）
  - セッションcookie発行
- `/login` のボタンは `/oidc/login` に向ける

### Phase 2: セッション/ミドルウェア整理（1-2日）
- `apps/console/middleware.ts` をアプリセッション検証に置換
- Supabase依存の `getUser()` 判定を削除
- ログを構造化（request_id, iss, aud, error_code 等）

### Phase 3: 既存フローの段階的廃止（1日）
- `/auth/callback` を段階的に無効化
- Supabaseセッションcookieへの依存を除去

### Phase 4: テストと安定化（継続）
- 初回ログイン/シークレット/クロスドメイン/HTTPS の再現テスト
- Cookie属性の検証（Secure/SameSite/Domain/Path）
- 失敗時のフォールバック（エラー画面/再試行）

## 4) OIDC最小実装スケルトン（詳細）

### 4.1 ルート構成
- `apps/console/src/app/oidc/login/route.ts`
- `apps/console/src/app/oidc/callback/route.ts`
- `apps/console/src/lib/oidc/config.ts`（設定とディスカバリ）
- `apps/console/src/lib/oidc/crypto.ts`（PKCE生成）
- `apps/console/src/lib/oidc/session.ts`（セッションcookie）

### 4.2 必要環境変数
- `OIDC_ISSUER`
- `OIDC_CLIENT_ID`
- `OIDC_CLIENT_SECRET`（公開クライアントなら不要）
- `OIDC_REDIRECT_URI`（例: https://dev.mcpist.app/oidc/callback）
- `APP_SESSION_SECRET`

### 4.3 /oidc/login 処理（要点）
- discovery: `/.well-known/openid-configuration`
- `state`, `nonce`, `code_verifier` を生成
- `code_challenge = base64url(SHA256(code_verifier))`
- `state/nonce/verifier` を HttpOnly cookie 保存
- Authorization Endpoint にリダイレクト

### 4.4 /oidc/callback 処理（要点）
- `state` をcookieと比較（CSRF対策）
- `code_verifier` を使って token 交換
- `id_token` を検証
  - `iss`, `aud`, `exp`, `nonce`
  - 署名検証（JWKS）
- ユーザー情報を抽出（sub, email, name）
- アプリセッションcookieを発行
- returnTo を検証してリダイレクト

### 4.5 ログと監視（最小）
- request_id を付与
- `state_mismatch`, `nonce_mismatch`, `token_exchange_failed` 等を区別
- 認証エラー時は reason を返す

## 5) 追加メモ（現状の再現性なし事象に関する仮説）
- 初回のみ `code_verifier` cookie が付与されないケース
- HTTP/HTTPS/サブドメイン差で `Secure` cookie が欠落
- ブラウザのストレージ初期状態での race/redirect タイミング差

以上


## 6) ????
- 2026-02-03: PKCE code_verifier ???????????????????????????`/auth/login` ??????????????????????
- 2026-02-03: `/auth/callback` ????? `PKCE code_verifier cookie present: true` ????????????????
