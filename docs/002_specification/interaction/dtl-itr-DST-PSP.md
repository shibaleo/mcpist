# DST - PSP インタラクション詳細（dtl-itr-DST-PSP）

## ドキュメント管理情報

| 項目      | 値                                                        |
| ------- | -------------------------------------------------------- |
| Status  | `draft`                                                  |
| Version | v1.0                                                     |
| Note    | Data Store - Payment Service Provider Interaction Detail |

---

## 概要

| 項目 | 内容 |
|------|------|
| 連携元 | Payment Service Provider (PSP) |
| 連携先 | Data Store (DST) |
| 内容 | 有料クレジット情報 |
| プロトコル | Webhook (HTTPS) |

---

## 詳細

| 項目 | 内容 |
|------|------|
| 方式 | Webhook |
| プロトコル | HTTPS |
| 認証 | 署名検証（Webhook Secret） |
| データ形式 | JSON |

**Webhookエンドポイント:** `https://api.mcpist.app/webhooks/psp`

### 通知されるイベント

| イベント | 説明 | DSTの処理 |
|----------|------|-----------|
| checkout.session.completed | 決済完了 | 有料クレジット残高加算 |
| checkout.session.expired | セッション期限切れ | （処理なし） |

**注:** PSPは有料クレジットのみを扱う。無料クレジットはシステム内部で管理される

### 署名検証

- リクエストヘッダー `Stripe-Signature` に署名を含む
- Webhook Secretを使用して署名を検証
- 検証失敗時は400エラーを返却

### 冪等性

- event.idを使用して重複処理を防止
- 処理済みイベントは無視

### 注意事項

- イベントの順序は保証されない
- 非同期キューでの処理を推奨

---

## 関連ドキュメント

| ドキュメント                             | 内容                            |
| ---------------------------------- | ----------------------------- |
| [itr-PSP.md](./itr-PSP.md)         | Payment Service Provider 詳細仕様 |
| [itr-DST.md](./itr-DST.md)         | Data Store 詳細仕様               |

