# Payment Service Provider インタラクション仕様書（itr-PSP）

## ドキュメント管理情報

| 項目 | 値 |
|------|-----|
| Status | `reviewed` |
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

## 連携サマリー

| 相手 | 方向 | やり取り | 詳細 |
|------|------|----------|------|
| User Console | PSP ← CON | 決済 | [dtl-itr-CON-PSP.md](./dtl-itr-CON-PSP.md) |
| Data Store | PSP → DST | 有料クレジット情報（Webhook） | [dtl-itr-DST-PSP.md](./dtl-itr-DST-PSP.md) |

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
| [itr-CON.md](./itr-CON.md) | User Console詳細仕様 |
| [itr-DST.md](./itr-DST.md) | Data Store詳細仕様 |
| [dtl-spc-credit-model.md](../dtl-spc/dtl-spc-credit-model.md) | クレジットモデル詳細仕様 |
