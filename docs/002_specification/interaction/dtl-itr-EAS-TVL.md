# EAS - TVL インタラクション詳細（dtl-itr-EAS-TVL）

## ドキュメント管理情報

| 項目 | 値 |
|------|-----|
| Status | `draft` |
| Version | v1.0 |
| ID | ITR-REL-013 |
| Note | External Auth Server - Token Vault Interaction Detail |

---

## 概要

| 項目 | 内容 |
|------|------|
| 連携元 | Token Vault (TVL) |
| 連携先 | External Auth Server (EAS) |
| 内容 | トークンリフレッシュ |
| プロトコル | OAuth 2.0 / HTTPS |

---

## 詳細

| 項目 | 内容 |
|------|------|
| トリガー | トークン取得時に有効期限切れを検出 |
| 操作 | refresh_tokenを使用した新トークン取得 |

### フロー

1. MODからトークン取得リクエスト
2. access_tokenの有効期限切れを検出
3. EASのトークンエンドポイントにリフレッシュリクエスト
4. 新しいaccess_token（+ refresh_token）を取得
5. TVL内のトークン情報を更新
6. 新しいaccess_tokenをMODに返却

### リフレッシュリクエスト（共通形式）

```
POST /oauth/token
Content-Type: application/x-www-form-urlencoded

grant_type=refresh_token
&refresh_token={refresh_token}
&client_id={client_id}
&client_secret={client_secret}
```

### リフレッシュ失敗時

- refresh_tokenも無効の場合はエラー返却
- ユーザーはCONで再連携が必要

---

## 関連ドキュメント

| ドキュメント | 内容 |
|-------------|------|
| [itr-tvl.md](./itr-tvl.md) | Token Vault 詳細仕様 |
| [itr-eas.md](./itr-eas.md) | External Auth Server 詳細仕様 |
| [idx-itr-rel.md](./idx-itr-rel.md) | インタラクション関係ID一覧 |
