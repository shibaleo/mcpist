# OAuth Mock Server 統合テスト結果（tst-oauth-mock-server）

## テスト実施日

2026-01-21

---

## テスト環境

| 項目 | 値 |
|------|-----|
| OAuth Mock Server | http://oauth.localhost (port 4000) |
| Console | http://console.localhost (port 3000) |
| Supabase | http://127.0.0.1:54321 |
| Docker Profile | default |

---

## テスト結果サマリー

| テスト | エンドポイント | 結果 |
|--------|---------------|------|
| Health check | GET /health | OK |
| OAuth metadata | GET /.well-known/oauth-authorization-server | OK |
| JWKS | GET /jwks | OK |
| Authorization request | GET /authorize | OK (302 redirect) |
| Get authorization details | GET /authorization/:id | OK |
| Consent page | GET /oauth/consent | OK (200) |

---

## 詳細ログ

### 1. Health Check

**リクエスト:**
```bash
curl -s http://oauth.localhost/health
```

**レスポンス:**
```json
{"status":"ok","service":"oauth-mock-server"}
```

### 2. OAuth Authorization Server Metadata (RFC 8414)

**リクエスト:**
```bash
curl -s http://oauth.localhost/.well-known/oauth-authorization-server
```

**レスポンス:**
```json
{
  "issuer": "http://oauth.localhost",
  "authorization_endpoint": "http://oauth.localhost/authorize",
  "token_endpoint": "http://oauth.localhost/token",
  "jwks_uri": "http://oauth.localhost/jwks",
  "response_types_supported": ["code"],
  "grant_types_supported": ["authorization_code", "refresh_token"],
  "code_challenge_methods_supported": ["S256"],
  "token_endpoint_auth_methods_supported": ["none"],
  "scopes_supported": ["openid", "profile", "email"]
}
```

### 3. JWKS (JSON Web Key Set)

**リクエスト:**
```bash
curl -s http://oauth.localhost/jwks
```

**レスポンス:**
```json
{
  "keys": [
    {
      "kty": "RSA",
      "n": "ivTqWdSX_v8stdgivn5Svk8yish8DBRfj3MMV5cdyxcnFHVGGKHgxBDBuGMBIVLsG8qF_dKiLId_elmjPR-bKsl-7UkcpkhAOaDRPB_0mENpTqqZbIBZBT-YnRhcwVIJKspbbiB7S-cELu0oNYZyNb8Fii1MWv11gzPq6J2xajhB3t-Tp9oiAFU2T50qAB_MGZIbVERyGnmwuHoJZmNNWOrTuo0nyKtJ0q5i1FibDHOKNBD8Tg32wgspsXVBJXEiX5uYxiPQFQo7WMkljCS7_zMMArq3p_cmdlU6cWFavH21v82CJcjvetJSg632d-DZ5BZdr_CEUaVvOvahVG9_Tw",
      "e": "AQAB",
      "kid": "mcpist-auth-key-1",
      "use": "sig",
      "alg": "RS256"
    }
  ]
}
```

**備考:** 開発環境ではRSA鍵が自動生成され、`/app/.auth-private-key.pem`に保存される

### 4. Authorization Request (PKCE)

**リクエスト:**
```bash
curl -s -v 'http://oauth.localhost/authorize?response_type=code&client_id=test-client&redirect_uri=http://localhost:8080/callback&code_challenge=E9Melhoa2OwvFrEMTJguCHaoeK1t8URWbuGJSstw-cM&code_challenge_method=S256&state=test123&scope=openid+profile+email'
```

**レスポンス:**
```
HTTP/1.1 302 Found
Location: http://console.localhost/oauth/consent?authorization_id=18823742ae93013a41de226043febe97
```

**サーバーログ:**
```
[authorize] Redirecting to consent: http://console.localhost/oauth/consent?authorization_id=18823742ae93013a41de226043febe97
```

### 5. Get Authorization Details

**リクエスト:**
```bash
curl -s 'http://oauth.localhost/authorization/18823742ae93013a41de226043febe97'
```

**レスポンス:**
```json
{
  "id": "18823742ae93013a41de226043febe97",
  "client_id": "test-client",
  "redirect_uri": "http://localhost:8080/callback",
  "scope": "openid profile email",
  "state": "test123",
  "created_at": "2026-01-21T13:00:57.0669+00:00",
  "expires_at": "2026-01-21T13:10:56.932+00:00"
}
```

### 6. Consent Page

**リクエスト:**
```bash
curl -s 'http://console.localhost/oauth/consent?authorization_id=18823742ae93013a41de226043febe97'
```

**レスポンス:**
```
HTTP/1.1 200 OK
Content-Type: text/html
(Next.js React アプリケーションが正常にレンダリング)
```

---

## OAuth 認証フロー

```
1. MCP Client → OAuth Server: GET /authorize (PKCE parameters)
2. OAuth Server → Database: Store authorization request
3. OAuth Server → Console: 302 redirect to /oauth/consent?authorization_id=xxx
4. Console → OAuth Server: GET /authorization/:id (fetch details)
5. Console → User: Display consent screen
6. User → Console: Click "Allow"
7. Console → OAuth Server: POST /authorization/:id/approve (user_id)
8. OAuth Server → Database: Generate authorization code
9. OAuth Server → Console: Return code + redirect_uri
10. Console → MCP Client: 302 redirect with code + state
11. MCP Client → OAuth Server: POST /token (code + code_verifier)
12. OAuth Server: Verify PKCE, generate JWT
13. OAuth Server → MCP Client: access_token (JWT)
```

---

## エンドポイント一覧

| Method | Path | 説明 |
|--------|------|------|
| GET | /health | ヘルスチェック |
| GET | /.well-known/oauth-authorization-server | OAuth メタデータ (RFC 8414) |
| GET | /jwks | 公開鍵 (JWKS) |
| GET | /authorize | 認可リクエスト開始 |
| GET | /authorization/:id | 認可リクエスト詳細取得 |
| POST | /authorization/:id/approve | 認可承認 (コード発行) |
| POST | /authorization/:id/deny | 認可拒否 |
| POST | /token | トークン交換 |

---

## 確認事項

- OAuth Mock Server がポート4000で正常起動
- Traefik経由で oauth.localhost にルーティング成功
- RSA鍵の自動生成と永続化 (開発環境)
- Supabase RPC経由で認可リクエストがDBに保存
- 認可リクエストの詳細取得が正常動作
- Consent pageへのリダイレクトが正常動作

---

## 未テスト項目

- POST /authorization/:id/approve (認証済みユーザーセッションが必要)
- POST /authorization/:id/deny (認証済みユーザーセッションが必要)
- POST /token (完全なOAuthフロー完了が必要)
- E2Eテスト (ブラウザでの完全なフロー)

---

## Docker コンテナ状態

```
CONTAINER ID   IMAGE           STATUS          NAMES
0967e89dbfcf   mcpist-oauth    Up              mcpist-oauth
21f8fb0d8d49   mcpist-console  Up              mcpist-console
2cefd1b4f7d2   traefik:v3.3    Up              mcpist-traefik
48b448ddd873   mcpist-server   Up              mcpist-server
bc0456e4918b   mcpist-worker   Up              mcpist-worker
```

---

## 備考

- Supabase OAuth Serverと同じAPI契約を実装
- `authorization_id`パターンを採用 (Supabase互換)
- 本番環境ではSupabase OAuth Serverを使用し、開発環境でのみこのMock Serverを使用
