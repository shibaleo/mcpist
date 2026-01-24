# API Gateway インタラクション仕様書（itr-gwy）

## ドキュメント管理情報

| 項目 | 値 |
|------|-----|
| Status | `reviewed` |
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

## 連携サマリー

| 相手 | 方向 | やり取り | 詳細 |
|------|------|----------|------|
| MCP Client (OAuth2.0) | GWY ← CLO | MCP通信リクエスト受付（JWT認証） | [dtl-itr-CLO-GWY.md](./dtl-itr-CLO-GWY.md) |
| MCP Client (API KEY) | GWY ← CLK | MCP通信リクエスト受付（API KEY認証） | [dtl-itr-CLK-GWY.md](./dtl-itr-CLK-GWY.md) |
| Auth Server | GWY → AUS | JWKS取得 | [dtl-itr-AUS-GWY.md](./dtl-itr-AUS-GWY.md) |
| Token Vault | GWY → TVL | API KEYハッシュ検証 | [dtl-itr-GWY-TVL.md](./dtl-itr-GWY-TVL.md) |
| Auth Middleware | GWY → AMW | リクエスト転送 | [dtl-itr-AMW-GWY.md](./dtl-itr-AMW-GWY.md) |

---

## GWYが直接やり取りしないコンポーネント

| コンポーネント | 理由 |
|----------------|------|
| Session Manager (SSM) | 認証はAUS/TVLが担当 |
| Data Store (DST) | AMW/HDL経由 |
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
| [spc-itr.md](../spc-itr.md) | インタラクション仕様書 |
| [itr-aus.md](./itr-aus.md) | Auth Server詳細仕様 |
| [itr-tvl.md](./itr-tvl.md) | Token Vault詳細仕様 |
| [itr-amw.md](./itr-amw.md) | Auth Middleware詳細仕様 |
