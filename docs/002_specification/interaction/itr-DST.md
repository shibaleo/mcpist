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

## 連携サマリー（dtl-itrまとめ）

### AMW
- [dtl-itr-AMW-DST.md](./dtl-itr-AMW-DST.md)
  - ユーザーコンテキスト取得

### AUS
- [dtl-itr-AUS-DST.md](./dtl-itr-AUS-DST.md)
  - ユーザーID共有

### CON
- [dtl-itr-CON-DST.md](./dtl-itr-CON-DST.md)
  - ユーザー設定管理

### GWY
- [dtl-itr-DST-GWY.md](./dtl-itr-DST-GWY.md)
  - APIキー検証

### HDL
- [dtl-itr-DST-HDL.md](./dtl-itr-DST-HDL.md)
  - ユーザーコンテキスト取得、クレジット消費

### PSP
- [dtl-itr-DST-PSP.md](./dtl-itr-DST-PSP.md)
  - 有料クレジット情報

### SSM
- [dtl-itr-DST-SSM.md](./dtl-itr-DST-SSM.md)
  - ユーザー情報の登録・参照

### TVL
- [dtl-itr-DST-TVL.md](./dtl-itr-DST-TVL.md)
  - トークン管理のためのユーザー紐付け

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




