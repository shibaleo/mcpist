# External Auth Server インタラクション仕様書（itr-EAS）

## ドキュメント管理情報

| 項目 | 値 |
|------|-----|
| Status | `draft` |
| Version | v2.0 |
| Note | External Auth Server Interaction Specification - 実装範囲外 |

---

## 概要

External Auth Server（EAS）は、外部サービス（Notion, Google Calendar等）のOAuth認証を提供する認可サーバー。

**実装範囲外**だが、他コンポーネントとのやり取りを明確にするため仕様を記載する。

External Service API（EXT）と同一サービス内で連携する。

---

## 連携サマリー

| 相手 | 方向 | やり取り | 詳細 |
|------|------|----------|------|
| User Console | EAS ← CON | 認可フロー受付 | [dtl-itr-CON-EAS.md](./dtl-itr-CON-EAS.md) |

---

## EASが直接やり取りしないコンポーネント

| コンポーネント | 理由 |
|----------------|------|
| MCP Client (CLO/CLK) | MCP通信専用 |
| API Gateway (GWY) | MCP通信専用 |
| Auth Server (AUS) | MCPist内部認証専用 |
| Session Manager (SSM) | ソーシャルログイン専用 |
| Data Store (DST) | CON経由 |
| Auth Middleware (AMW) | MCP Server内部 |
| MCP Handler (HDL) | MCP Server内部 |
| Modules (MOD) | EXT経由（トークンリフレッシュはMODがEXT側で処理） |
| Identity Provider (IDP) | ソーシャルログイン専用 |
| Payment Service Provider (PSP) | 課金専用 |

---

## 関連ドキュメント

| ドキュメント | 内容 |
|-------------|------|
| [spc-sys.md](../spc-sys.md) | システム仕様書 |
| [spc-itr.md](spc-itr.md) | インタラクション仕様書 |
| [itr-CON.md](./itr-CON.md) | User Console詳細仕様 |
| [itr-EXT.md](./itr-EXT.md) | External Service API詳細仕様 |
