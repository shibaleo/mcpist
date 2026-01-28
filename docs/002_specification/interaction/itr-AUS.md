# Auth Server インタラクション仕様書（itr-AUS）

## ドキュメント管理情報

| 項目 | 値 |
|------|-----|
| Status | `reviewed` |
| Version | v2.0 |
| Note | Auth Server Interaction Specification |

---

## 概要

Auth Server（AUS）は、OAuth 2.1準拠の認可サーバー。MCP Client (OAuth2.0) およびUser Consoleに対してOAuth 2.1認証フローを提供する。

**実装:** Supabase Auth

主な責務：
- OAuth 2.1 + PKCE認証フローの提供
- JWT（access_token）の発行
- JWKS公開鍵の提供
- トークンリフレッシュ

---

## 連携サマリー

| 相手 | 方向 | やり取り | 詳細 |
|------|------|----------|------|
| MCP Client (OAuth2.0) | AUS ← CLO | OAuth 2.1認証リクエスト受付 | [dtl-itr-AUS-CLO.md](./dtl-itr-AUS-CLO.md) |
| API Gateway | AUS → GWY | JWT提供 | [dtl-itr-AUS-GWY.md](./dtl-itr-AUS-GWY.md) |
| Session Manager | AUS ↔ SSM | ユーザー認証連携 | [dtl-itr-AUS-SSM.md](./dtl-itr-AUS-SSM.md) |

---

## エンドポイント一覧

| エンドポイント | メソッド | 用途 | 準拠仕様 |
|---------------|--------|------|----------|
| `/.well-known/openid-configuration` | GET | メタデータ | [OpenID Connect Discovery 1.0](https://openid.net/specs/openid-connect-discovery-1_0.html) |
| `/authorize` | GET | 認可リクエスト | [OAuth 2.1](https://datatracker.ietf.org/doc/html/draft-ietf-oauth-v2-1-12), [RFC 7636 (PKCE)](https://datatracker.ietf.org/doc/html/rfc7636) |
| `/token` | POST | トークン交換・リフレッシュ | OAuth 2.1 |
| `/.well-known/jwks.json` | GET | JWT検証用公開鍵 | [RFC 7517 (JWK)](https://datatracker.ietf.org/doc/html/rfc7517) |

---

## メタデータエンドポイント

### /.well-known/openid-configuration

[OpenID Connect Discovery 1.0](https://openid.net/specs/openid-connect-discovery-1_0.html) 準拠。

**メタデータ例:**

```json
{
  "issuer": "{Auth Server Domain}",
  "authorization_endpoint": "{Auth Server Domain}/authorize",
  "token_endpoint": "{Auth Server Domain}/token",
  "jwks_uri": "{Auth Server Domain}/.well-known/jwks.json",
  "response_types_supported": ["code"],
  "grant_types_supported": ["authorization_code", "refresh_token"],
  "code_challenge_methods_supported": ["S256"],
  "token_endpoint_auth_methods_supported": ["none"],
  "scopes_supported": ["openid", "profile", "email"]
}
```

---

## JWT（access_token）仕様

[RFC 7519 (JWT)](https://datatracker.ietf.org/doc/html/rfc7519)、[RFC 7515 (JWS)](https://datatracker.ietf.org/doc/html/rfc7515) 準拠。

| 項目 | 内容 |
|------|------|
| 形式 | JWT（JWS、RS256署名） |
| 有効期限 | 3600秒（1時間） |
| 検証 | JWKS公開鍵による署名検証 |

**JWTペイロード例:**

```json
{
  "iss": "{Auth Server Domain}",
  "sub": "user-uuid",
  "aud": "authenticated",
  "exp": 1234567890,
  "iat": 1234564290,
  "scope": "openid profile"
}
```

| クレーム | 説明 |
|----------|------|
| iss | 発行者（Auth Server） |
| sub | ユーザー識別子（user_id） |
| aud | 対象リソース（MCP Server） |
| exp | 有効期限（Unix timestamp） |
| iat | 発行日時（Unix timestamp） |
| scope | 許可されたスコープ |

---

## AUSが直接やり取りしないコンポーネント

| コンポーネント | 理由 |
|----------------|------|
| MCP Client (API KEY) (CLK) | API KEY認証はTVL担当 |
| Data Store (DST) | SSM経由（DBトリガーでユーザー作成） |
| Token Vault (TVL) | 外部サービストークン管理 |
| Auth Middleware (AMW) | GWY経由でJWKS取得 |
| MCP Handler (HDL) | MCP Server内部 |
| Modules (MOD) | MCP Server内部 |
| User Console (CON) | 認可フローのリダイレクト先（直接連携ではない） |
| Identity Provider (IDP) | SSM経由 |
| External Auth Server (EAS) | 外部サービス認証専用 |
| External Service API (EXT) | MOD経由 |
| Payment Service Provider (PSP) | 課金専用 |

---

## 関連ドキュメント

| ドキュメント | 内容 |
|-------------|------|
| [spc-sys.md](../spc-sys.md) | システム仕様書 |
| [spc-itr.md](../spc-itr.md) | インタラクション仕様書 |
| [itr-CLO.md](./itr-CLO.md) | MCP Client (OAuth2.0)詳細仕様 |
| [itr-GWY.md](./itr-GWY.md) | API Gateway詳細仕様 |
| [itr-SSM.md](./itr-SSM.md) | Session Manager詳細仕様 |
