# MCPist インタラクション仕様書（spc-itr）

## ドキュメント管理情報

| 項目      | 値                         |
| ------- | ------------------------- |
| Status  | `reviewed`                |
| Version | v2.0 (Sprint-006)         |
| Note    | Interaction Specification |

---

## 概要

本ドキュメントは、spc-sys.mdで定義されたコンポーネント間のインタラクションを規定する。

MCP Serverは内部コンポーネント（Auth Middleware, MCP Handler, Modules）の集合である。外部コンポーネント（CLO, CON等）に対してはMCP Serverとして抽象化し、内部コンポーネント間は詳細に定義する。

---

## コンポーネント略称一覧

| #   | コンポーネント               | 略称 | 備考           |
| --- | ---------------------------- | ---- | -------------- |
| 1   | MCP Client (OAuth2.0)        | CLO  | 実装範囲外     |
| 2   | MCP Client (API KEY)         | CLK  | 実装範囲外     |
| 3   | API Gateway                  | GWY  |                |
| 4   | Auth Server                  | AUS  |                |
| 5   | Auth Middleware              | AMW  | MCP Server内部 |
| 6   | MCP Handler                  | HDL  | MCP Server内部 |
| 7   | Modules                      | MOD  | MCP Server内部 |
| 8   | Data Store                   | DST  |                |
| 9   | Token Vault                  | TVL  |                |
| 10  | Observability                | OBS  |                |
| 11  | MCP Server                   | SRV  | 外部向け抽象化 |
| 12  | User Console                 | CON  |                |
| 13  | Session Manager              | SSM  |                |
| 14  | Identity Provider            | IDP  | 実装範囲外     |
| 15  | External Auth Server         | EAS  | 実装範囲外     |
| 16  | External Service API         | EXT  | 実装範囲外     |
| 17  | Payment Service Provider     | PSP  | 実装範囲外     |

---

## 1. MCP Client (OAuth2.0)（CLO）

| 相手          | 方向        | やり取り                           |
| ----------- | --------- | ------------------------------ |
| API Gateway | CLO → GWY | MCP通信（JSON-RPC over Streamable HTTP） |
| Auth Server | CLO → AUS | OAuth 2.1認証フロー（認可コード取得、トークン交換） |
| その他         | -         | 直接やり取りなし                       |

---

## 2. MCP Client (API KEY)（CLK）

| 相手                     | 方向        | やり取り                                 |
| ------------------------ | ----------- | ---------------------------------------- |
| API Gateway              | CLK → GWY   | MCP通信（JSON-RPC over Streamable HTTP、APIキーをヘッダで送信） |
| その他                   | -           | 直接やり取りなし（APIキーはユーザーがリクエストヘッダに含めて送信） |

---

## 3. API Gateway（GWY）

| 相手                    | 方向        | やり取り                                                    |
| --------------------- | --------- | ------------------------------------------------------- |
| MCP Client (OAuth2.0) | GWY ← CLO | MCP通信リクエスト受付                                            |
| MCP Client (API KEY)  | GWY ← CLK | MCP通信リクエスト受付（APIキー認証）                                   |
| Auth Server           | GWY → AUS | JWT検証（userinfo / Auth API / JWKS の3段構え）                 |
| Data Store            | GWY → DST | APIキー検証（SHA-256 → KVキャッシュ → Supabase RPC fallback）      |
| Auth Middleware       | GWY → AMW | リクエスト転送（X-User-ID, X-Auth-Type, X-Gateway-Secret ヘッダ付与） |
| Observability         | GWY → OBS | ログ送信                                                    |
| その他                   | -         | 直接やり取りなし                                                |

---

## 4. Auth Server（AUS）

| 相手                    | 方向        | やり取り               |
| --------------------- | --------- | ------------------ |
| MCP Client (OAuth2.0) | AUS ← CLO | OAuth 2.1認証リクエスト受付 |
| API Gateway           | AUS ← GWY | JWT検証リクエスト受付       |
| Data Store            | AUS ↔ DST | ユーザーID共有（トリガー実行）   |
| Session Manager       | AUS ← SSM | ユーザー登録時の関数トリガー     |
| その他                   | -         | 直接やり取りなし           |

---

## 5. Auth Middleware（AMW）

| 相手                     | 方向        | やり取り                                 |
| ------------------------ | ----------- | ---------------------------------------- |
| API Gateway              | AMW ← GWY   | リクエスト受信（X-Gateway-Secret検証）   |
| Data Store               | AMW → DST   | ユーザーコンテキスト取得（アカウント状態・クレジット残高・有効モジュール・無効ツール） |
| MCP Handler              | AMW → HDL   | 認証済みリクエスト転送（コンテキスト付き） |
| その他                   | -           | 直接やり取りなし                         |

---

## 6. MCP Handler（HDL）

| 相手                     | 方向        | やり取り                                 |
| ------------------------ | ----------- | ---------------------------------------- |
| Auth Middleware          | HDL ← AMW   | 認証済みリクエスト受信                   |
| Modules                  | HDL → MOD   | メタツール経由でのモジュールツール実行、スキーマ取得 |
| Data Store               | HDL → DST   | ツール実行成功時のクレジット消費（冪等） |
| Observability            | HDL → OBS   | ツール実行ログ、セキュリティイベント     |
| その他                   | -           | 直接やり取りなし                         |

---

## 7. Modules（MOD）

| 相手                   | 方向        | やり取り                           |
| -------------------- | --------- | ------------------------------ |
| MCP Handler          | MOD ← HDL | ツール実行リクエスト受信                   |
| Token Vault          | MOD → TVL | 外部サービスアクセス用トークン取得、トークンリフレッシュ   |
| External Service API | MOD → EXT | リソースアクセス（HTTPS + Bearer Token） |
| その他                  | -         | 直接やり取りなし                       |

---

## 8. Data Store（DST）

| 相手                     | 方向        | やり取り                                 |
| ------------------------ | ----------- | ---------------------------------------- |
| Auth Server              | DST ↔ AUS   | ユーザーID共有                           |
| Token Vault              | DST → TVL   | トークン管理のためのユーザー紐付け       |
| Session Manager          | DST ← SSM   | ユーザー情報の登録・参照                 |
| API Gateway              | DST ← GWY   | APIキー検証                              |
| Auth Middleware          | DST ← AMW   | ユーザーコンテキスト取得                 |
| MCP Handler              | DST ← HDL   | クレジット消費                           |
| Payment Service Provider | DST ← PSP   | 課金情報の同期（Webhook）                |
| User Console             | DST ← CON   | ツール有効/無効設定の書き込み            |
| その他                   | -           | 直接やり取りなし                         |

---

## 9. Token Vault（TVL）

| 相手                     | 方向        | やり取り                                 |
| ------------------------ | ----------- | ---------------------------------------- |
| Data Store               | TVL ← DST   | ユーザーID共有                           |
| User Console             | TVL ← CON   | OAuth連携完了時のトークン保存            |
| Modules                  | TVL ← MOD   | 外部サービスアクセス用トークンの復号化・提供 |
| その他                   | -           | 直接やり取りなし                         |

---

## 10. Observability（OBS）

| 相手                     | 方向        | やり取り                                 |
| ------------------------ | ----------- | ---------------------------------------- |
| API Gateway              | OBS ← GWY   | HTTPリクエストログ受信                   |
| MCP Handler              | OBS ← HDL   | ツール実行ログ、セキュリティイベント受信 |
| その他                   | -           | 直接やり取りなし                         |

---

## 11. MCP Server（SRV）

MCP Serverは以下の内部コンポーネントで構成される。外部からはSRVとして抽象化される。

内部コンポーネント: Auth Middleware (AMW), MCP Handler (HDL), Modules (MOD)

| 相手                     | 方向        | やり取り                                 |
| ------------------------ | ----------- | ---------------------------------------- |
| API Gateway              | SRV ← GWY   | リクエスト受付（Auth Middleware経由）    |
| その他                   | -           | 内部コンポーネント経由                   |

---

## 12. User Console（CON）

| 相手                     | 方向        | やり取り                                 |
| ------------------------ | ----------- | ---------------------------------------- |
| Session Manager          | CON → SSM   | ソーシャルログイン（Session Manager経由でIdP連携） |
| Payment Service Provider | CON → PSP   | Checkout処理                             |
| Token Vault              | CON → TVL   | OAuth連携完了時のトークン保存            |
| Data Store               | CON → DST   | ツール有効/無効設定の書き込み            |
| External Auth Server     | CON → EAS   | 外部サービスOAuth認可フロー              |
| その他                   | -           | 直接やり取りなし                         |

---

## 13. Session Manager（SSM）

| 相手                     | 方向        | やり取り                                 |
| ------------------------ | ----------- | ---------------------------------------- |
| Identity Provider        | SSM → IDP   | ソーシャルログイン認証リクエスト         |
| Auth Server              | SSM → AUS   | ユーザー登録時の関数トリガー             |
| Data Store               | SSM → DST   | ユーザー情報の登録・参照                 |
| その他                   | -           | 直接やり取りなし                         |

---

## 14. Identity Provider（IDP）

| 相手                     | 方向        | やり取り                                 |
| ------------------------ | ----------- | ---------------------------------------- |
| Session Manager          | IDP ← SSM   | ソーシャルログイン認証リクエスト受付     |
| その他                   | -           | 直接やり取りなし                         |

---

## 15. External Auth Server（EAS）

| 相手                     | 方向        | やり取り                                 |
| ------------------------ | ----------- | ---------------------------------------- |
| User Console             | EAS ← CON   | 認可フロー受付                           |
| その他                   | -           | 直接やり取りなし                         |

---

## 16. External Service API（EXT）

| 相手                     | 方向        | やり取り                                 |
| ------------------------ | ----------- | ---------------------------------------- |
| Modules                  | EXT ← MOD   | API呼び出し受付（HTTPS + Bearer Token）  |
| その他                   | -           | 直接やり取りなし                         |

---

## 17. Payment Service Provider（PSP）

| 相手                     | 方向        | やり取り                                 |
| ------------------------ | ----------- | ---------------------------------------- |
| User Console             | PSP ← CON   | 決済リクエスト受付                       |
| Data Store               | PSP → DST   | 課金情報の同期（Webhook）                |
| その他                   | -           | 直接やり取りなし                         |

---

## 関連ドキュメント

| ドキュメント | 内容 |
|-------------|------|
| [spc-sys.md](spc-sys.md) | システム仕様書（コンポーネント定義） |
| [spc-itf.md](spc-itf.md) | インタフェース仕様書 |
