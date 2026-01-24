# Payment Service Provider インタラクション仕様書（itr-psp）

## ドキュメント管理情報

| 項目 | 値 |
|------|-----|
| Status | `draft` |
| Version | v2.0 |
| Note | Payment Service Provider Interaction Specification - 実装範囲外 |

---

## 概要

Payment Service Provider（PSP）は、課金処理を行う外部サービス。

**実装:** Stripe

**実装範囲外**だが、他コンポーネントとのやり取りを明確にするため仕様を記載する。

主な機能：
- 顧客管理（Customer）
- クレジット購入（Checkout Session）
- 顧客ポータル（Customer Portal）
- Webhook通知

---

## 連携サマリー（spc-itrより）

| 相手 | 方向 | やり取り |
|------|------|----------|
| User Console | PSP ← CON | 決済 |
| Data Store | PSP → DST | 有料クレジット情報（Webhook） |

---

## 連携詳細

### CON → PSP（決済）

| 項目 | 内容 |
|------|------|
| プロトコル | HTTPS |
| 認証 | API Key（Secret Key） |
| データ形式 | JSON |

**主な操作：**

1. **Checkout Session作成**
   - トリガー: ユーザーがクレジット購入ボタンをクリック
   - パラメータ: customer_id, price_id, success_url, cancel_url
   - レスポンス: checkout_url

2. **Customer Portal Session作成**
   - トリガー: ユーザーが課金管理ページにアクセス
   - パラメータ: customer_id, return_url
   - レスポンス: portal_url

---

### PSP → DST（有料クレジット情報）

| 項目 | 内容 |
|------|------|
| 方式 | Webhook |
| プロトコル | HTTPS |
| 認証 | 署名検証（Webhook Secret） |
| データ形式 | JSON |

**Webhookエンドポイント:** `https://api.mcpist.app/webhooks/psp`

**通知されるイベント:**

| イベント | 説明 | DSTの処理 |
|----------|------|-----------|
| checkout.session.completed | 決済完了 | 有料クレジット残高加算 |
| checkout.session.expired | セッション期限切れ | （処理なし） |

**注:** PSPは有料クレジットのみを扱う。無料クレジットはシステム内部で管理される

**署名検証:**
- リクエストヘッダー `Stripe-Signature` に署名を含む
- Webhook Secretを使用して署名を検証
- 検証失敗時は400エラーを返却

**冪等性:**
- event.idを使用して重複処理を防止
- 処理済みイベントは無視

**注意事項:**
- イベントの順序は保証されない
- 非同期キューでの処理を推奨

---

## PSPが提供する機能

### 顧客管理

| 機能 | 説明 |
|------|------|
| 顧客作成 | user_idに紐づく顧客レコード作成 |
| 顧客取得 | 顧客情報の参照 |
| 顧客更新 | メールアドレス等の更新 |

### クレジット購入

| 機能 | 説明 |
|------|------|
| Checkout Session作成 | クレジット購入ページへのリダイレクトURL生成 |
| Customer Portal Session作成 | 顧客向け管理ページへのリダイレクトURL生成 |

### 価格定義

PSP側で定義され、DSTが参照する情報。

| 項目 | 説明 |
|------|------|
| Product | クレジットパック定義 |
| Price | 価格定義（クレジット数量、金額） |
| Price ID | クレジットパックとマッピング |

---

## DSTが保持するPSP関連データ

| フィールド | 型 | 説明 |
|------------|-----|------|
| psp_customer_id | string | PSP顧客ID |
| paid_credit_balance | number | 有料クレジット残高 |

---

## PSPが直接やり取りしないコンポーネント

| コンポーネント | 理由 |
|----------------|------|
| MCP Client (CLO/CLK) | MCP通信専用 |
| API Gateway (GWY) | MCP通信専用 |
| Auth Server (AUS) | 認証専用 |
| Session Manager (SSM) | 認証専用 |
| Token Vault (TVL) | 外部サービストークン専用 |
| Auth Middleware (AMW) | MCP Server内部 |
| MCP Handler (HDL) | MCP Server内部 |
| Modules (MOD) | MCP Server内部 |
| Identity Provider (IDP) | 認証専用 |
| External Auth Server (EAS) | 外部サービス認証専用 |
| External Service API (EXT) | 外部サービスAPI専用 |

---

## 関連ドキュメント

| ドキュメント | 内容 |
|-------------|------|
| [spc-sys.md](../spc-sys.md) | システム仕様書 |
| [spc-itr.md](../spc-itr.md) | インタラクション仕様書 |
| [itr-con.md](./itr-con.md) | User Console詳細仕様 |
| [itr-dst.md](./itr-dst.md) | Data Store詳細仕様 |
| [dtl-spc-credit-model.md](../dtl-spc/dtl-spc-credit-model.md) | クレジットモデル詳細仕様 |
