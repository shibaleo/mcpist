# EXT - MOD インタラクション詳細（dtl-itr-EXT-MOD）

## ドキュメント管理情報

| 項目 | 値 |
|------|-----|
| Status | `draft` |
| Version | v1.0 |
| ID | ITR-REL-012 |
| Note | External Service API - Modules Interaction Detail |

---

## 概要

| 項目 | 内容 |
|------|------|
| 連携元 | Modules (MOD) |
| 連携先 | External Service API (EXT) |
| 内容 | API呼び出し |
| プロトコル | HTTPS |

---

## 詳細

| 項目 | 内容 |
|------|------|
| プロトコル | HTTPS |
| 認証 | TVLから取得したauth_typeに応じた方式 |
| データ形式 | JSON（サービスにより異なる） |

### 認証方式

TVLから取得した`auth_type`に基づきAuthorizationヘッダーを構築。詳細は[dtl-itr-MOD-TVL.md](./dtl-itr-MOD-TVL.md)のauth_type一覧を参照。

### API呼び出し例（Bearer Token - Notion）

```http
POST https://api.notion.com/v1/search
Authorization: Bearer ntn_xxx
Notion-Version: 2022-06-28
Content-Type: application/json

{
  "query": "設計ドキュメント"
}
```

### API呼び出し例（OAuth 1.0a - Zaim）

```http
GET https://api.zaim.net/v2/home/money
Authorization: OAuth oauth_consumer_key="xxx", oauth_token="xxx", oauth_signature_method="HMAC-SHA1", oauth_signature="xxx", oauth_timestamp="xxx", oauth_nonce="xxx", oauth_version="1.0"
```

### レスポンス処理

1. EXTからレスポンス受信
2. サービス固有のレスポンス形式を共通形式に変換
3. エラーの場合は適切なエラーコードにマッピング
4. HDLに結果を返却

---

## 関連ドキュメント

| ドキュメント | 内容 |
|-------------|------|
| [itr-mod.md](./itr-mod.md) | Modules 詳細仕様 |
| [itr-ext.md](./itr-ext.md) | External Service API 詳細仕様 |
| [idx-itr-rel.md](./idx-itr-rel.md) | インタラクション関係ID一覧 |
