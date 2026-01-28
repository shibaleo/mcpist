# AMW - GWY インタラクション詳細（dtl-itr-AMW-GWY）

## ドキュメント管理情報

| 項目      | 値                                                |
| ------- | ------------------------------------------------ |
| Status  | `reviewed`                                       |
| Version | v2.0                                             |
| Note    | Auth Middleware - API Gateway Interaction Detail |

---

## 概要

| 項目    | 内容                    |
| ----- | --------------------- |
| 連携元   | API Gateway (GWY)     |
| 連携先   | Auth Middleware (AMW) |
| 内容    | リクエスト転送               |
| プロトコル | HTTPS（パブリックネットワーク）    |

---

## 詳細

| 項目    | 内容                    |
| ----- | --------------------- |
| プロトコル | HTTPS（パブリックネットワーク）    |
| 認証    | X-Gateway-Secret ヘッダー |

### GWY が付与するヘッダー

| ヘッダー | 型 | 説明 |
|---------|------|------|
| X-User-ID | string (UUID) | 認証済みユーザーID |
| X-Auth-Type | string | 認証方式（`jwt` / `api_key`） |
| X-Request-ID | string (UUID) | リクエストトレース用の一意ID |
| X-Gateway-Secret | string | GWY-AMW 間の共有シークレット |

### 期待する振る舞い

- GWY はクライアントの `Authorization` ヘッダーを削除してから転送する
- AMW は `X-Gateway-Secret` を検証し、一致しないリクエストを拒否する（401 INVALID_GATEWAY_SECRET）
- `X-User-ID` は GWY の認証処理で確定した値であり、AMW はこれを信頼する

---

## 関連ドキュメント

| ドキュメント                     | 内容                 |
| -------------------------- | ------------------ |
| [itr-GWY.md](./itr-GWY.md) | API Gateway 仕様     |
| [itr-AMW.md](./itr-AMW.md) | Auth Middleware 仕様 |

