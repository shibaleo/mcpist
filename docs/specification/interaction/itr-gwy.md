# API Gateway インタラクション仕様書（itr-gwy）

## ドキュメント管理情報

| 項目 | 値 |
|------|-----|
| Status | `draft` |
| Version | v2.0 |
| Note | API Gateway Interaction Specification |

---

## 概要

API Gateway（GWY）は、外部からのリクエストを受け付けるエントリーポイント。

主な責務：
- MCP Clientからのリクエスト受付
- JWT/API KEY検証の委譲
- MCP Serverへのリクエスト転送

---

## 連携サマリー（spc-itrより）

| 相手 | 方向 | やり取り |
|------|------|----------|
| MCP Client (OAuth2.0) | GWY ← CLO | MCP通信リクエスト受付（JWT認証） |
| MCP Client (API KEY) | GWY ← CLK | MCP通信リクエスト受付（API KEY認証） |
| Auth Server | GWY → AUS | JWKS取得 |
| Token Vault | GWY → TVL | API KEYハッシュ検証 |
| Auth Middleware | GWY → AMW | リクエスト転送 |

---

## 連携詳細

### CLO → GWY（MCP通信リクエスト受付）

| 項目 | 内容 |
|------|------|
| プロトコル | HTTPS |
| 認証 | Bearer Token（JWT） |

MCP ClientからのすべてのHTTPSリクエストはGWYを経由する。

**リクエストヘッダー:**
```
Authorization: Bearer {access_token}
Content-Type: application/json
Accept: application/json, text/event-stream
```

**GWYの処理:**
1. リクエスト受信
2. Authorizationヘッダーの存在確認
3. JWTの検証（AUSのJWKSで署名検証）
4. 検証成功時：AMWへ転送
5. 検証失敗時：401 Unauthorized返却

---

### CLK → GWY（MCP通信リクエスト受付）

| 項目 | 内容 |
|------|------|
| プロトコル | HTTPS |
| 認証 | API KEY（Bearer Token形式） |

**API KEY形式:**
```
mcpist_{random_string_32chars}
```

**リクエストヘッダー:**
```
Authorization: Bearer mpt_xxx
Content-Type: application/json
Accept: application/json, text/event-stream
```

**GWYの処理:**
1. リクエスト受信
2. Authorizationヘッダーの存在確認
3. API KEYの検証（TVLにハッシュで問い合わせ）
4. 検証成功時：AMWへ転送
5. 検証失敗時：401 Unauthorized返却

---

### GWY → AUS（JWT検証）

| 項目 | 内容 |
|------|------|
| 方向 | GWY が AUS の JWKS を取得 |
| キャッシュ | 必須（Cache-Controlヘッダーに従う） |

**検証フロー:**
1. GWYがJWTを受信
2. JWKSから公開鍵を取得（キャッシュ優先）
3. JWT署名を検証
4. クレーム（iss, aud, exp）を検証（クレーム定義は[itr-aus.md](./itr-aus.md#jwtaccess_token仕様)参照）
5. 検証成功時：ユーザー情報をヘッダーに付与してAMWへ転送

**付与するヘッダー:**
```
X-User-Id: {jwt.sub}
X-Gateway-Secret: {shared_secret}
```

---

### GWY → TVL（API KEY検証）

| 項目 | 内容 |
|------|------|
| 方向 | GWY が TVL に API KEY ハッシュを問い合わせ |
| 用途 | MCP Client (API KEY) からのリクエスト検証 |

**セキュリティ設計:**

API KEYの平文をGWYのメモリに保持しないため、ハッシュ比較方式を採用する。

```
保存時: TVLにSHA256(api_key)を保存
検証時: GWYがSHA256(受信したapi_key)を計算し、TVLに問い合わせ
```

**検証フロー:**
1. GWYがAPI KEYを受信（`Authorization: Bearer mpt_xxx`）
2. GWYがAPI KEYのSHA256ハッシュを計算（平文は即破棄）
3. キャッシュにハッシュ→user_idの対応があれば使用
4. キャッシュになければTVLにハッシュで検証リクエスト
5. TVLがハッシュに紐づくユーザーIDを返却
6. ハッシュ→user_idの対応をキャッシュ
7. 検証成功時：ユーザー情報をヘッダーに付与してAMWへ転送

**キャッシュ:**
- ハッシュ→user_idの対応をキャッシュ可能（ハッシュは平文ではないため）
- API KEY平文はキャッシュしない

---

### GWY → AMW（リクエスト転送）

| 項目 | 内容 |
|------|------|
| プロトコル | HTTP（内部通信） |
| 認証 | X-Gateway-Secret ヘッダー |

**転送時のヘッダー:**
```
X-User-Id: {user_id}
X-Gateway-Secret: {shared_secret}
X-Request-Id: {request_id}
X-Forwarded-For: {client_ip}
Content-Type: application/json
```

AMWはX-Gateway-Secretを検証し、信頼できるリクエストのみ処理する。

---

## GWYが直接やり取りしないコンポーネント

| コンポーネント | 理由 |
|----------------|------|
| Session Manager (SSM) | 認証はAUS/TVLが担当 |
| Data Store (DST) | AMW/HDL経由 |
| MCP Handler (HDL) | AMW経由 |
| Modules (MOD) | MCP Server内部 |
| User Console (CON) | 別アプリケーション |
| Identity Provider (IDP) | SSM経由 |
| External Auth Server (EAS) | CON経由 |
| External Service API (EXT) | MOD経由 |
| Payment Service Provider (PSP) | CON/DST経由 |

---

## 関連ドキュメント

| ドキュメント | 内容 |
|-------------|------|
| [spc-sys.md](../spc-sys.md) | システム仕様書 |
| [spc-itr.md](../spc-itr.md) | インタラクション仕様書 |
| [itr-aus.md](./itr-aus.md) | Auth Server詳細仕様 |
| [itr-tvl.md](./itr-tvl.md) | Token Vault詳細仕様 |
| [itr-amw.md](./itr-amw.md) | Auth Middleware詳細仕様 |
