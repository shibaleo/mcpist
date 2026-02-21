# セキュリティ調査結果（apps/console, apps/server, apps/worker, database）

## 概要
対象: `apps/console`, `apps/server`, `apps/worker`, `database`
方式: リポジトリ内の実装・設定ファイルの静的レビュー

## 主要リスク（優先度順）

### 1. Critical: APIキー失効が実質効かない

**根拠**
- `apps/worker` はAPIキーJWTの署名検証のみで、DB上の失効状態を参照していない。
- `apps/server` も `claims.UserID` を信頼してユーザー解決するだけで、キーIDや失効状態の照合がない。
- APIキーは有効期限なしで発行可能。

**該当ファイル**
- `apps/worker/src/auth.ts`
- `apps/server/internal/rest/middleware.go`
- `apps/server/internal/rest/me.go`
- `apps/server/internal/auth/keys.go`

**影響**
- 漏えいしたキーが revoke 後も利用可能になるリスク。

**対応案**
- JWTに `jti` もしくは `key_id` を入れてDB照合を必須化。
- 失効済み・期限切れを拒否。
- 無期限発行を禁止。

---

### 2. High: OAuth `state` の真正性検証不足

**根拠**
- `state` 生成時に `nonce` を作成するが、保存・照合がなく、callback 側で検証していない。

**該当ファイル**
- `apps/console/src/app/api/oauth/google/authorize/route.ts`
- `apps/console/src/app/api/oauth/google/callback/route.ts`

**影響**
- OAuth CSRF / セッション固定のリスク。

**対応案**
- `state` を HttpOnly/SameSite cookie 等に保存し、callback で厳密照合。

---

### 3. High: 資格情報・OAuth秘密の平文保存

**根拠**
- `user_credentials.credentials` が平文保存。
- `oauth_apps.client_secret` が平文保存。
- `encrypted_credentials` カラムがあるが実装で未使用。

**該当ファイル**
- `database/migrations/00000000000001_baseline.sql`
- `apps/server/internal/db/models.go`
- `apps/server/internal/db/repo_credentials.go`

**影響**
- DB漏えい時に直接的な機密漏えいにつながる。

**対応案**
- KMS/Envelope暗号化などで保存時暗号化。
- `encrypted_credentials` を主経路に移行。

---

### 4. Medium: トークン検証APIが未認証 + 任意先fetch

**根拠**
- `/api/credentials/validate` が未認証で利用可能。
- `domain` / `base_url` を使用してサーバー側から外部 fetch を実施。

**該当ファイル**
- `apps/console/src/app/api/credentials/validate/route.ts`
- `apps/console/src/app/api/credentials/validate/validators/*.ts`

**影響**
- 外部からの濫用・SSRF面のリスク。

**対応案**
- 認証必須化。
- レート制限、許可ドメイン制限。

---

### 5. Medium: `.env.dev` がGit管理対象

**根拠**
- `.gitignore` で `.env.*` を除外しつつ `.env.dev` を例外で許可。
- `.env.dev` が実際に追跡されている。

**該当ファイル**
- `.gitignore`
- `.env.dev`

**影響**
- 開発用であっても秘密情報が履歴に残るリスク。

**対応案**
- `.env.dev` を追跡対象から外す。
- 漏えい済みとみなして値をローテーション。

---

## 良い点
- Worker→Server の境界が短命JWT(30秒)で保護されている。
- Admin API は Server 側で `withAdmin` を実装。

## 推奨優先度
1. APIキー失効の実効性確保
2. OAuth `state` の厳密検証
3. 資格情報の暗号化保存
4. 検証APIの認証/制限
5. `.env.dev` 運用改善
