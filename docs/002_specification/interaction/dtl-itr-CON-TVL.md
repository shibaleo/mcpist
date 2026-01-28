# CON - TVL インタラクション詳細（dtl-itr-CON-TVL）

## ドキュメント管理情報

| 項目 | 値 |
|------|-----|
| Status | `draft` |
| Version | v1.0 |
| ID | ITR-REL-015 |
| Note | User Console - Token Vault Interaction Detail |

---

## 概要

| 項目 | 内容 |
|------|------|
| 連携元 | User Console (CON) |
| 連携先 | Token Vault (TVL) |
| 内容 | トークン登録 |
| プロトコル | 内部API |

---

## 詳細

| 項目 | 内容 |
|------|------|
| 用途 | 外部サービスのトークン保存 |
| タイミング | OAuth連携完了時、または手動トークン登録時 |

### 登録リクエスト

```json
{
  "user_id": "user-123",
  "service_id": "notion",
  "auth_type": "bearer",
  "credentials": { ... }
}
```

| フィールド | 必須 | 説明 |
|-----------|------|------|
| user_id | ✅ | ユーザーID |
| service_id | ✅ | サービス識別子 |
| auth_type | ✅ | 認証方式（bearer, oauth1, basic, custom_header） |
| credentials | ✅ | auth_typeに応じた認証情報 |

credentialsの詳細は[dtl-itr-MOD-TVL.md](./dtl-itr-MOD-TVL.md)を参照。

### フロー

1. ユーザーがCONで外部サービス連携（またはAPIトークン手動入力）
2. OAuth連携の場合、EASからトークン取得
3. TVLにトークン情報を保存（credentials部分は暗号化）
4. 既存トークンがあれば上書き

---

## 関連ドキュメント

| ドキュメント | 内容 |
|-------------|------|
| [itr-CON.md](./itr-CON.md) | User Console 詳細仕様 |
| [itr-TVL.md](./itr-TVL.md) | Token Vault 詳細仕様 |
| [idx-itr-rel.md](./idx-itr-rel.md) | インタラクション関係ID一覧 |
