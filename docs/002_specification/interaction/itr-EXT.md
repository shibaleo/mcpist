# External Service API インタラクション仕様書（itr-EXT）

## ドキュメント管理情報

| 項目 | 値 |
|------|-----|
| Status | `draft` |
| Version | v2.0 |
| Note | External Service API Interaction Specification - 実装範囲外 |

---

## 概要

External Service API（EXT）は、各モジュールがアクセスする外部サービスのAPIサーバーの総称。

**実装範囲外**だが、他コンポーネントとのやり取りを明確にするため仕様を記載する。

主なサービス：
- Notion API
- Google Calendar API
- Microsoft Graph API（To Do）

---

## 連携サマリー

| 相手 | 方向 | やり取り | 詳細 |
|------|------|----------|------|
| Modules | EXT ← MOD | API呼び出し受付（HTTPS） | [dtl-itr-EXT-MOD.md](./dtl-itr-EXT-MOD.md) |

---

## 認証方式

外部サービスの認証方式はサービスごとに異なる。認証方式の違いはToken Vault（TVL）が吸収し、MODは統一的なインターフェースでトークンを取得する。

| 認証方式      | 特徴                     | 例                           |
| --------- | ---------------------- | --------------------------- |
| OAuth 2.0 | refresh_tokenによるトークン更新 | Notion, Google Calendar     |
| 長期トークン    | APIキー形式、有効期限なし         | Notion Internal Integration |

**共通:**
- トークン/認証情報はTVLで暗号化保存
- MODはTVLからトークンを取得してEXTにアクセス
- 認証方式・トークン形式の差異はTVLが吸収

---

## EXTが直接やり取りしないコンポーネント

| コンポーネント | 理由 |
|----------------|------|
| MCP Client (CLO/CLK) | MCP通信専用 |
| API Gateway (GWY) | MCP通信専用 |
| Auth Server (AUS) | MCPist内部認証専用 |
| Session Manager (SSM) | ソーシャルログイン専用 |
| Data Store (DST) | MCPist内部 |
| Token Vault (TVL) | MOD経由（EXTからTVLへの直接通信はない） |
| Auth Middleware (AMW) | MCP Server内部 |
| MCP Handler (HDL) | MCP Server内部 |
| User Console (CON) | EAS経由 |
| Identity Provider (IDP) | ソーシャルログイン専用 |
| Payment Service Provider (PSP) | 課金専用 |

---

## 関連ドキュメント

| ドキュメント | 内容 |
|-------------|------|
| [spc-sys.md](../spc-sys.md) | システム仕様書 |
| [spc-itr.md](spc-itr.md) | インタラクション仕様書 |
| [itr-MOD.md](./itr-MOD.md) | Modules詳細仕様 |
| [itr-EAS.md](./itr-EAS.md) | External Auth Server詳細仕様 |
| [itr-TVL.md](./itr-TVL.md) | Token Vault詳細仕様 |
