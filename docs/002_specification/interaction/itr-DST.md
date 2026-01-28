# Data Store インタラクション仕様書（itr-DST）

## ドキュメント管理情報

| 項目 | 値 |
|------|-----|
| Status | `draft` |
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

| 相手                       | 方向        | やり取り             | 詳細                                         |
| ------------------------ | --------- | ---------------- | ------------------------------------------ |
| MCP Handler              | DST ← HDL | ユーザー設定取得・クレジット消費 | [dtl-itr-DST-HDL.md](./dtl-itr-DST-HDL.md) |
| Auth Middleware          | DST ← AMW | ユーザーコンテキスト取得     | [dtl-itr-AMW-DST.md](./dtl-itr-AMW-DST.md) |
| Auth Server              | DST ↔ AUS | ユーザーID共有（トリガー）   | [dtl-itr-AUS-DST.md](./dtl-itr-AUS-DST.md) |
| API Gateway              | DST ← GWY | APIキー検証          | [dtl-itr-DST-GWY.md](./dtl-itr-DST-GWY.md) |
| Session Manager          | DST ← SSM | ユーザー情報登録・参照      | [dtl-itr-DST-SSM.md](./dtl-itr-DST-SSM.md) |
| Token Vault              | DST → TVL | ユーザー紐付け          | [dtl-itr-DST-TVL.md](./dtl-itr-DST-TVL.md) |
| Payment Service Provider | DST ← PSP | クレジット情報受信        | [dtl-itr-DST-PSP.md](./dtl-itr-DST-PSP.md) |
| User Console             | DST ← CON | ツール設定登録          | [dtl-itr-CON-DST.md](./dtl-itr-CON-DST.md) |

---

## DSTが直接やり取りしないコンポーネント

| コンポーネント | 理由 |
|----------------|------|
| MCP Client (OAuth2.0) (CLO) | GWY経由 |
| MCP Client (API KEY) (CLK) | GWY経由 |
| Modules (MOD) | HDL経由 |
| Observability (OBS) | 直接連携なし |
| Identity Provider (IDP) | SSM経由 |
| External Auth Server (EAS) | CON経由 |
| External Service API (EXT) | MOD経由 |

---

## 関連ドキュメント

| ドキュメント | 内容 |
|-------------|------|
| [spc-sys.md](../spc-sys.md) | システム仕様書 |
| [spc-itr.md](spc-itr.md) | インタラクション仕様書 |
| [spc-tbl.md](../spc-tbl.md) | テーブル仕様書 |
| [itr-SSM.md](./itr-SSM.md) | Session Manager詳細仕様 |
| [itr-HDL.md](./itr-HDL.md) | MCP Handler詳細仕様 |
| [itr-MOD.md](./itr-MOD.md) | Modules詳細仕様 |
| [itr-PSP.md](./itr-PSP.md) | Payment Service Provider詳細仕様 |
| [itr-CON.md](./itr-CON.md) | User Console詳細仕様 |
| [dsn-adt.md](../../003_design/dsn-adt.md) | 監査・請求・分析設計書 |
| [dtl-spc-credit-model.md](../dtl-spc/dtl-spc-credit-model.md) | クレジットモデル詳細仕様 |
