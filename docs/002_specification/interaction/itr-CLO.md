# MCP Client (OAuth2.0) インタラクション仕様書（itr-CLO）

## ドキュメント管理情報

| 項目 | 値 |
|------|-----|
| Status | `draft` |
| Version | v2.0 |
| Note | MCP Client (OAuth2.0) Interaction Specification - 実装範囲外 |

---

## 概要

MCP Client (OAuth2.0)（略号: CLO）は、LLM Host（Claude Code, Cursor等）の一部として実装され，MCP Serverへ接続するクライアント。OAuth 2.1認証を使用する。

**実装範囲外**だが、他コンポーネントとのやり取りを明確にするため仕様を記載する。

---

## 連携サマリー（dtl-itrまとめ）

### AUS
- [dtl-itr-AUS-CLO.md](./dtl-itr-AUS-CLO.md)
  - OAuth認可

### GWY
- [dtl-itr-CLO-GWY.md](./dtl-itr-CLO-GWY.md)
  - MCP通信

---

## CLOが保持するデータ

| データ | 用途 |
|--------|------|
| access_token | MCP Serverへのリクエストに使用 |
| refresh_token | トークン更新に使用 |
| expires_in | トークン有効期限 |
| code_verifier | PKCE検証用（トークン交換時に使用） |

---

## CLOが直接やり取りしないコンポーネント

| コンポーネント | 理由 |
|----------------|------|
| MCP Client (API KEY) (CLK) | 別の認証方式 |
| Session Manager (SSM) | AUS経由 |
| Data Store (DST) | サーバー側 |
| Token Vault (TVL) | サーバー側 |
| Auth Middleware (AMW) | GWY経由 |
| MCP Handler (HDL) | GWY経由 |
| Modules (MOD) | GWY経由 |
| User Console (CON) | 別アプリケーション |
| Identity Provider (IDP) | SSM経由 |
| External Auth Server (EAS) | CON経由 |
| External Service API (EXT) | MOD経由 |
| Payment Service Provider (PSP) | CON経由 |

---

## 関連ドキュメント

| ドキュメント | 内容 |
|-------------|------|
| [spc-sys.md](../spc-sys.md) | システム仕様書 |
| [spc-itr.md](spc-itr.md) | インタラクション仕様書 |
| [itr-GWY.md](./itr-GWY.md) | API Gateway詳細仕様 |
| [itr-AUS.md](./itr-AUS.md) | Auth Server詳細仕様 |
| [itr-CLK.md](./itr-CLK.md) | MCP Client (API KEY)詳細仕様 |




