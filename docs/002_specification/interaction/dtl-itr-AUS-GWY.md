# AUS - GWY インタラクション詳細（dtl-itr-AUS-GWY）

## ドキュメント管理情報

| 項目 | 値 |
|------|-----|
| Status | `draft` |
| Version | v1.0 |
| ID | ITR-REL-004 |
| Note | Auth Server - API Gateway Interaction Detail |

---

## 概要

| 項目 | 内容 |
|------|------|
| 連携元 | API Gateway (GWY) |
| 連携先 | Auth Server (AUS) |
| 内容 | トークン検証 |
| プロトコル | HTTPS (JWKS) |

---

## 詳細

| 項目      | 内容                                           |
| ------- | -------------------------------------------- |
| 方向      | GWY が AUS の JWKS を取得                         |
| エンドポイント | `{Auth Server Domain}/.well-known/jwks.json` |
| 用途      | JWT署名検証用公開鍵の提供                               |
| キャッシュ   | Cache-Controlヘッダーを返却                     |

GWYはAUSからJWKSを取得し、MCP ClientからのJWTを検証する。

### 検証フロー

1. GWYがJWTを受信
2. JWKSから公開鍵を取得（キャッシュ優先）
3. JWT署名を検証
4. クレーム（iss, aud, exp）を検証（クレーム定義は[itr-aus.md](./itr-aus.md#jwtaccess_token仕様)参照）
5. 検証成功時：ユーザー情報をヘッダーに付与してAMWへ転送

### 付与するヘッダー

```
X-User-Id: {jwt.sub}
X-Gateway-Secret: {shared_secret}
```

### JWKSレスポンス例（[RFC 7517](https://datatracker.ietf.org/doc/html/rfc7517) 準拠）

```json
{
  "keys": [
    {
      "kty": "RSA",
      "kid": "key-id-1",
      "use": "sig",
      "alg": "RS256",
      "n": "...",
      "e": "AQAB"
    }
  ]
}
```

---

## 関連ドキュメント

| ドキュメント | 内容 |
|-------------|------|
| [itr-gwy.md](./itr-gwy.md) | API Gateway 詳細仕様 |
| [itr-aus.md](./itr-aus.md) | Auth Server 詳細仕様 |
| [idx-itr-rel.md](./idx-itr-rel.md) | インタラクション関係ID一覧 |
