# Token Vault インタラクション仕様書（itr-TVL）

## ドキュメント管理情報

| 項目 | 値 |
|------|-----|
| Status | `draft` |
| Version | v2.1 |
| Note | Token Vault Interaction Specification |

---

## 概要

Token Vault（TVL）は、外部サービスのOAuthトークン・API KEYを安全に管理するデータストア。

主な責務：
- 外部サービスのOAuthトークン保存・取得
- APIシークレットの暗号化保存
- トークンの暗号化保存

---

## 連携サマリー（dtl-itrまとめ）

### CON
- [dtl-itr-CON-TVL.md](./dtl-itr-CON-TVL.md)
  - 外部サービストークン登録・管理

### DST
- [dtl-itr-DST-TVL.md](./dtl-itr-DST-TVL.md)
  - トークン管理のためのユーザー紐付け

### MOD
- [dtl-itr-MOD-TVL.md](./dtl-itr-MOD-TVL.md)
  - トークン取得・保存

---

## サービス別トークン形式

| Service | long_term_token | oauth_token |
|---------|-----------------|-------------|
| notion | Internal Integration Token (`ntn_xxx`) | OAuth Access Token |
| google_calendar | - | OAuth Access Token |
| microsoft_todo | - | OAuth Access Token |

---

## TVLが直接やり取りしないコンポーネント

| コンポーネント | 理由 |
|----------------|------|
| MCP Client (CLO/CLK) | GWY経由 |
| API Gateway (GWY) | 直接連携なし |
| Auth Server (AUS) | OAuth2.0はAUS担当 |
| Session Manager (SSM) | DST経由 |
| Auth Middleware (AMW) | MCP Server内部 |
| MCP Handler (HDL) | MOD経由 |
| Observability (OBS) | 直接連携なし |
| Identity Provider (IDP) | SSM経由 |
| External Auth Server (EAS) | EXT経由 |
| External Service API (EXT) | MOD経由 |
| Payment Service Provider (PSP) | DST経由 |

---

## 関連ドキュメント

| ドキュメント | 内容 |
|-------------|------|
| [spc-sys.md](../spc-sys.md) | システム仕様書 |
| [spc-itr.md](spc-itr.md) | インタラクション仕様書 |
|[itf-tvl.md](itf-tvl.md)) | Token Vault API仕様 |
| [itr-CON.md](./itr-CON.md) | User Console詳細仕様 |
| [itr-MOD.md](./itr-MOD.md) | Modules詳細仕様 |
| [itr-DST.md](./itr-DST.md) | Data Store詳細仕様 |




