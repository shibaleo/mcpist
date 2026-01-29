# API Gateway インタラクション仕様書（itr-GWY）

## ドキュメント管理情報

| 項目 | 値 |
|------|-----|
| Status | `draft` |
| Version | v2.0 |
| Note | API Gateway Interaction Specification |

---

## 概要

API Gateway（GWY）は、外部からのリクエストを受け付けるエントリーポイント。

主な責務：
- MCP Clientからのリクエスト受付
- JWT/API KEY検証の委譲
- MCP Serverへのリクエスト転送

---

## 連携サマリー（dtl-itrまとめ）

### AMW
- [dtl-itr-AMW-GWY.md](./dtl-itr-AMW-GWY.md)
  - リクエスト転送

### AUS
- [dtl-itr-AUS-GWY.md](./dtl-itr-AUS-GWY.md)
  - JWT 検証

### CLK
- [dtl-itr-CLK-GWY.md](./dtl-itr-CLK-GWY.md)
  - MCP通信

### CLO
- [dtl-itr-CLO-GWY.md](./dtl-itr-CLO-GWY.md)
  - MCP通信

### DST
- [dtl-itr-DST-GWY.md](./dtl-itr-DST-GWY.md)
  - APIキー検証

### OBS
- [dtl-itr-GWY-OBS.md](./dtl-itr-GWY-OBS.md)
  - リクエスト/認証/ルーティングのログ送信

---

## GWYが直接やり取りしないコンポーネント

| コンポーネント | 理由 |
|----------------|------|
| Session Manager (SSM) | 認証はAUS/DSTが担当 |
| Token Vault (TVL) | 直接連携なし |
| MCP Handler (HDL) | AMW経由 |
| Modules (MOD) | MCP Server内部 |
| User Console (CON) | 別アプリケーション |
| Identity Provider (IDP) | SSM経由 |
| External Auth Server (EAS) | CON経由 |
| External Service API (EXT) | MOD経由 |
| Payment Service Provider (PSP) | CON/DST経由 |

---

## 関連ドキュメント

| ドキュメント | 内容 |
|-------------|------|
| [spc-sys.md](../spc-sys.md) | システム仕様書 |
| [spc-itr.md](spc-itr.md) | インタラクション仕様書 |
| [itr-AUS.md](./itr-AUS.md) | Auth Server詳細仕様 |
| [itr-DST.md](./itr-DST.md) | Data Store詳細仕様 |
| [itr-AMW.md](./itr-AMW.md) | Auth Middleware詳細仕様 |
| [itr-OBS.md](./itr-OBS.md) | Observability詳細仕様 |



