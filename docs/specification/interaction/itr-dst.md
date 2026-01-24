# Data Store インタラクション仕様書（itr-dst）

## ドキュメント管理情報

| 項目 | 値 |
|------|-----|
| Status | `reviewed` |
| Version | v2.0 |
| Note | Data Store Interaction Specification |

---

## 概要

Data Store（DST）は、ユーザーの権限・設定・課金情報を管理するデータストア。

主な責務：
- ユーザー情報の永続化
- アカウント状態の管理
- クレジット残高の管理
- モジュール有効/無効設定の管理
- ツール単位の有効/無効設定の管理
- カスタムプロンプトの保存
- 課金情報の管理（PSPからのWebhook）

---

## 連携サマリー

| 相手 | 方向 | やり取り | 詳細 |
|------|------|----------|------|
| MCP Handler | DST ← HDL | ユーザー設定取得 | [dtl-itr-DST-HDL.md](./dtl-itr-DST-HDL.md) |
| Modules | DST ← MOD | クレジット消費 | [dtl-itr-DST-MOD.md](./dtl-itr-DST-MOD.md) |
| Payment Service Provider | DST ← PSP | クレジット情報受信 | [dtl-itr-DST-PSP.md](./dtl-itr-DST-PSP.md) |
| User Console | DST ← CON | ツール設定登録 | [dtl-itr-CON-DST.md](./dtl-itr-CON-DST.md) |

---

## DSTが直接やり取りしないコンポーネント

| コンポーネント | 理由 |
|----------------|------|
| MCP Client (OAuth2.0) (CLO) | GWY経由 |
| MCP Client (API KEY) (CLK) | GWY経由 |
| API Gateway (GWY) | AMW経由 |
| Auth Server (AUS) | SSM経由（同一DB） |
| Auth Middleware (AMW) | HDL経由 |
| Token Vault (TVL) | user_idリレーション参照のみ |
| Identity Provider (IDP) | SSM経由 |
| External Auth Server (EAS) | CON経由 |
| External Service API (EXT) | MOD経由 |

---

## 関連ドキュメント

| ドキュメント | 内容 |
|-------------|------|
| [spc-sys.md](../spc-sys.md) | システム仕様書 |
| [spc-itr.md](../spc-itr.md) | インタラクション仕様書 |
| [spc-tbl.md](../spc-tbl.md) | テーブル仕様書 |
| [itr-ssm.md](./itr-ssm.md) | Session Manager詳細仕様 |
| [itr-hdl.md](./itr-hdl.md) | MCP Handler詳細仕様 |
| [itr-mod.md](./itr-mod.md) | Modules詳細仕様 |
| [itr-psp.md](./itr-psp.md) | Payment Service Provider詳細仕様 |
| [itr-con.md](./itr-con.md) | User Console詳細仕様 |
| [dsn-adt.md](../../design/dsn-adt.md) | 監査・請求・分析設計書 |
| [dtl-spc-credit-model.md](../dtl-spc/dtl-spc-credit-model.md) | クレジットモデル詳細仕様 |
