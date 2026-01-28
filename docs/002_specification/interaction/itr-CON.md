# User Console インタラクション仕様書（itr-CON）

## ドキュメント管理情報

| 項目 | 値 |
|------|-----|
| Status | `draft` |
| Version | v2.0 |
| Note | User Console Interaction Specification |

---

## 概要

User Console（CON）は、ユーザーが自分の設定を管理するWebアプリケーション。

主な機能：
- ユーザー認証（ログイン/ログアウト）
- OAuth同意画面の提供（MCP Client認可時）
- 外部サービス連携（OAuth認可フロー）
- 権限設定（モジュール有効/無効）
- 課金管理
- サーバーへの接続情報の提供

---

## 連携サマリー

| 相手 | 方向 | やり取り | 詳細 |
|------|------|----------|------|
| Payment Service Provider | CON → PSP | 決済リクエスト | [dtl-itr-CON-PSP.md](./dtl-itr-CON-PSP.md) |
| Token Vault | CON → TVL | トークン登録 | [dtl-itr-CON-TVL.md](./dtl-itr-CON-TVL.md) |
| Data Store | CON → DST | ツール設定登録 | [dtl-itr-CON-DST.md](./dtl-itr-CON-DST.md) |
| External Auth Server | CON → EAS | 認可フロー | [dtl-itr-CON-EAS.md](./dtl-itr-CON-EAS.md) |
| Session Manager | CON → SSM | ソーシャルログイン | [dtl-itr-CON-SSM.md](./dtl-itr-CON-SSM.md) |

---

## CONが直接やり取りしないコンポーネント

| コンポーネント | 理由 |
|----------------|------|
| MCP Client (OAuth2.0) (CLO) | 別アプリケーション |
| MCP Client (API KEY) (CLK) | 別アプリケーション |
| API Gateway (GWY) | MCP通信専用 |
| Auth Server (AUS) | CLO向け認証（CONはSSM経由） |
| Auth Middleware (AMW) | MCP Server内部 |
| MCP Handler (HDL) | MCP Server内部 |
| Modules (MOD) | MCP Server内部 |
| Identity Provider (IDP) | SSM経由 |
| External Service API (EXT) | EAS経由 |

---

## 関連ドキュメント

| ドキュメント | 内容 |
|-------------|------|
| [spc-sys.md](../spc-sys.md) | システム仕様書 |
| [spc-itr.md](spc-itr.md) | インタラクション仕様書 |
| [itr-PSP.md](./itr-PSP.md) | Payment Service Provider詳細仕様 |
| [itr-TVL.md](./itr-TVL.md) | Token Vault詳細仕様 |
| [itr-DST.md](./itr-DST.md) | Data Store詳細仕様 |
| [itr-EAS.md](./itr-EAS.md) | External Auth Server詳細仕様 |
| [itr-SSM.md](./itr-SSM.md) | Session Manager詳細仕様 |
