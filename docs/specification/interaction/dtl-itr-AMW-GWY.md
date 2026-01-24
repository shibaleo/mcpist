# AMW - GWY インタラクション詳細（dtl-itr-AMW-GWY）

## ドキュメント管理情報

| 項目 | 値 |
|------|-----|
| Status | `draft` |
| Version | v1.0 |
| ID | ITR-REL-006 |
| Note | Auth Middleware - API Gateway Interaction Detail |

---

## 概要

| 項目 | 内容 |
|------|------|
| 連携元 | API Gateway (GWY) |
| 連携先 | Auth Middleware (AMW) |
| 内容 | リクエスト転送 |
| プロトコル | HTTP（内部通信） |

---

## 詳細

| 項目 | 内容 |
|------|------|
| プロトコル | HTTP（内部通信） |
| 認証 | X-Gateway-Secret ヘッダー |

### 転送時のヘッダー

```
X-User-Id: {user_id}
X-Gateway-Secret: {shared_secret}
X-Request-Id: {request_id}
X-Forwarded-For: {client_ip}
Content-Type: application/json
```

AMWはX-Gateway-Secretを検証し、信頼できるリクエストのみ処理する。

---

## 関連ドキュメント

| ドキュメント | 内容 |
|-------------|------|
| [itr-gwy.md](./itr-gwy.md) | API Gateway 詳細仕様 |
| [itr-amw.md](./itr-amw.md) | Auth Middleware 詳細仕様 |
| [idx-itr-rel.md](./idx-itr-rel.md) | インタラクション関係ID一覧 |
