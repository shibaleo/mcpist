# AUS - GWY インタラクション詳細（dtl-itr-AUS-GWY）

## ドキュメント管理情報

| 項目      | 値                                            |
| ------- | -------------------------------------------- |
| Status  | `reviewed`                                   |
| Version | v2.0                                         |
| Note    | Auth Server - API Gateway Interaction Detail |

---

## 概要

| 項目 | 内容 |
|------|------|
| 連携元 | API Gateway (GWY) |
| 連携先 | Auth Server (AUS) |
| 内容 | JWT 検証 |
| プロトコル | HTTPS |

---

## 詳細

| 項目 | 内容 |
|------|------|
| 方向 | GWY → AUS（単方向） |
| 用途 | クライアントから受信した JWT の有効性検証 |

GWY はクライアントの Bearer トークン（JWT）を AUS に問い合わせて検証する。検証は以下の優先順で試行し、いずれかが成功した時点でユーザーIDを確定する。

### 検証フロー

| 優先度 | 方式 | エンドポイント | 説明 |
|--------|------|--------------|------|
| 1 | OAuth userinfo | /auth/v1/oauth/userinfo | Bearer トークンを送信し、`sub` クレームからユーザーIDを取得 |
| 2 | Auth user | /auth/v1/user | Bearer トークン + API キーを送信し、`id` フィールドからユーザーIDを取得 |
| 3 | JWKS 署名検証 | /auth/v1/.well-known/jwks.json | JWKS から公開鍵を取得し、JWT 署名を検証。`issuer` クレームを検証。`sub` からユーザーIDを取得 |

### 認証失敗時のレスポンス

すべての検証方式が失敗した場合、GWY は以下を返却する。

| 項目 | 値 |
|------|------|
| ステータス | 401 Unauthorized |
| ヘッダー | `WWW-Authenticate: Bearer resource_metadata="{base_url}/mcp/.well-known/oauth-protected-resource"` |

### 期待する振る舞い

- GWY は優先度1から順に検証を試行し、いずれかが成功した時点でユーザーIDを確定する
- 優先度1・2は AUS がトークンの有効性を判定するため、GWY 側でのクレーム検証は不要
- 優先度3（JWKS）では `issuer` クレームを検証する。JWKS はライブラリのキャッシュ機構に従う
- 検証成功時、GWY はユーザー情報をヘッダーに付与して AMW へ転送する（ヘッダー詳細は [dtl-itr-AMW-GWY.md](./dtl-itr-AMW-GWY.md) 参照）
- すべての検証が失敗した場合、401 + `WWW-Authenticate` ヘッダーで OAuth Discovery フローを案内する

---

## 関連ドキュメント

| ドキュメント                               | 内容                  |
| ------------------------------------ | ------------------- |
| [itr-GWY.md](./itr-GWY.md)           | API Gateway 仕様      |
| [itr-AUS.md](./itr-AUS.md)           | Auth Server 仕様      |
| [dtl-itr-AMW-GWY.md](./dtl-itr-AMW-GWY.md) | GWY→AMW 転送ヘッダー詳細 |

