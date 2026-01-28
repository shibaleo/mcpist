# CON - PSP インタラクション詳細（dtl-itr-CON-PSP）

## ドキュメント管理情報

| 項目 | 値 |
|------|-----|
| Status | `draft` |
| Version | v1.0 |
| ID | ITR-REL-017 |
| Note | User Console - Payment Service Provider Interaction Detail |

---

## 概要

| 項目 | 内容 |
|------|------|
| 連携元 | User Console (CON) |
| 連携先 | Payment Service Provider (PSP) |
| 内容 | 決済 |
| プロトコル | HTTPS |

---

## 詳細

| 項目 | 内容 |
|------|------|
| プロトコル | HTTPS |
| 用途 | クレジット購入、課金管理 |

### フロー

```mermaid
sequenceDiagram
    participant User as ユーザー
    participant CON as User Console
    participant PSP as Payment Service Provider
    participant DST as Data Store

    User->>CON: クレジット購入ボタンクリック
    CON->>PSP: Checkout Session作成
    PSP-->>CON: checkout_url
    CON->>User: PSP決済ページへリダイレクト
    User->>PSP: 決済情報入力・完了
    PSP-->>DST: Webhook（checkout.session.completed）
    DST->>DST: クレジット情報更新
    PSP-->>CON: success_urlへリダイレクト
    CON->>DST: クレジット残高取得
    DST-->>CON: クレジット残高
    CON->>User: 購入完了表示
```

---

## 関連ドキュメント

| ドキュメント | 内容 |
|-------------|------|
| [itr-CON.md](./itr-CON.md) | User Console 詳細仕様 |
| [itr-PSP.md](./itr-PSP.md) | Payment Service Provider 詳細仕様 |
| [idx-itr-rel.md](./idx-itr-rel.md) | インタラクション関係ID一覧 |
