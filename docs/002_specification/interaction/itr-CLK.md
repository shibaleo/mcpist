# MCP Client (API KEY) インタラクション仕様書（itr-CLK）

## ドキュメント管理情報

| 項目 | 値 |
|------|-----|
| Status | `draft` |
| Version | v2.0 |
| Note | MCP Client (API KEY) Interaction Specification - 実装範囲外 |

---

## 概要

MCP Client (API KEY)（CLK）は、LLM Host（Claude Code, Cursor等）からMCP Serverへ接続するクライアント。API KEY認証を使用する。

**実装範囲外**だが、他コンポーネントとのやり取りを明確にするため仕様を記載する。

OAuth 2.1よりシンプルな認証方式で、主にサーバーサイドアプリケーションや自動化ツールでの利用を想定。

---

## 連携サマリー（dtl-itrまとめ）

### GWY
- [dtl-itr-CLK-GWY.md](./dtl-itr-CLK-GWY.md)
  - MCP通信

---

## CLKが保持するデータ

| データ | 用途 |
|--------|------|
| api_key | MCP Serverへのリクエストに使用 |

OAuth 2.1と異なり、refresh_tokenやトークン有効期限の管理は不要。

---

## CLKが直接やり取りしないコンポーネント

| コンポーネント | 理由 |
|----------------|------|
| MCP Client (OAuth2.0) (CLO) | 別の認証方式 |
| Auth Server (AUS) | OAuth 2.1専用 |
| Session Manager (SSM) | OAuth 2.1専用 |
| Data Store (DST) | サーバー側 |
| Token Vault (TVL) | GWY経由（CLKから直接アクセスしない） |
| Auth Middleware (AMW) | GWY経由 |
| MCP Handler (HDL) | GWY経由 |
| Modules (MOD) | GWY経由 |
| User Console (CON) | API KEY発行時のみ（クライアント利用時は直接やり取りなし） |
| Identity Provider (IDP) | OAuth 2.1専用 |
| External Auth Server (EAS) | CON経由 |
| External Service API (EXT) | MOD経由 |
| Payment Service Provider (PSP) | CON経由 |

---

## 関連ドキュメント

| ドキュメント | 内容 |
|-------------|------|
| [spc-sys.md](../spc-sys.md) | システム仕様書 |
| [spc-itr.md](spc-itr.md) | インタラクション仕様書 |
| [itr-TVL.md](./itr-TVL.md) | Token Vault詳細仕様 |
| [itr-GWY.md](./itr-GWY.md) | API Gateway詳細仕様 |
| [itr-CLO.md](./itr-CLO.md) | MCP Client (OAuth2.0)詳細仕様 |




