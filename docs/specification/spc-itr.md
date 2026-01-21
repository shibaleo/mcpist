# MCPist インタラクション仕様書（spc-itr）

## 概要

本ドキュメントは、spc-sys.mdで定義されたコンポーネント間のインタラクションを規定する。

MCP Serverは内部コンポーネント（Auth Middleware, MCP Handler, Module Registry, Modules）の集合である。外部コンポーネント（CLT, CON等）に対してはMCP Serverとして抽象化し、内部コンポーネント間は詳細に定義する。

---

## コンポーネント略称一覧

| #   | コンポーネント               | 略称 | 備考           |
| --- | ---------------------------- | ---- | -------------- |
| 1   | MCP Client (OAuth2.0)        | CLO  | 実装範囲外     |
| 2   | MCP Client (API KEY)         | CLK  | 実装範囲外     |
| 3   | API Gateway                  | GWY  |                |
| 4   | Auth Server                  | AUS  |                |
| 5   | Session Manager              | SSM  |                |
| 6   | Data Store                   | DST  |                |
| 7   | Token Vault                  | TVL  |                |
| 8   | MCP Server                   | SRV  | 外部向け抽象化 |
| 9   | Auth Middleware              | AMW  | MCP Server内部 |
| 10  | MCP Handler                  | HDL  | MCP Server内部 |
| 11  | Module Registry              | REG  | MCP Server内部 |
| 12  | Modules                      | MOD  | MCP Server内部 |
| 13  | User Console                 | CON  |                |
| 14  | Identity Provider            | IDP  | 実装範囲外     |
| 15  | External Auth Server         | EAS  | 実装範囲外     |
| 16  | External Service API         | EXT  | 実装範囲外     |
| 17  | Payment Service Provider     | PSP  | 実装範囲外     |

---

## 1. MCP Client (OAuth2.0)（CLO）

| 相手                     | 方向        | やり取り                                 |
| ------------------------ | ----------- | ---------------------------------------- |
| API Gateway              | CLO → GWY   | MCP通信（JSON-RPC over SSE）             |
| Auth Server              | CLO → AUS   | OAuth 2.1認証フロー（認可コード、トークン交換） |
| その他                   | -           | 直接やり取りなし                         |

---

## 2. MCP Client (API KEY)（CLK）

| 相手                     | 方向        | やり取り                                 |
| ------------------------ | ----------- | ---------------------------------------- |
| Token Vault              | CLK → TVL   | API KEY認証                              |
| その他                   | -           | 直接やり取りなし                         |

---

## 3. API Gateway（GWY）

| 相手                     | 方向        | やり取り                                 |
| ------------------------ | ----------- | ---------------------------------------- |
| MCP Client (OAuth2.0)    | GWY ← CLO   | MCP通信リクエスト受付                    |
| Auth Server              | GWY ← AUS   | JWT受信・検証                            |
| Token Vault              | GWY ← TVL   | API KEY受信・検証                        |
| Auth Middleware          | GWY → AMW   | リクエスト転送                           |
| その他                   | -           | 直接やり取りなし                         |

---

## 4. Auth Server（AUS）

| 相手                     | 方向        | やり取り                                 |
| ------------------------ | ----------- | ---------------------------------------- |
| MCP Client (OAuth2.0)    | AUS ← CLO   | OAuth 2.1認証リクエスト受付              |
| API Gateway              | AUS → GWY   | JWT提供                                  |
| Data Store               | AUS ↔ DST   | ユーザーID共有                           |
| その他                   | -           | 直接やり取りなし                         |

---

## 5. Session Manager（SSM）

| 相手                     | 方向        | やり取り                                 |
| ------------------------ | ----------- | ---------------------------------------- |
| Identity Provider        | SSM ← IDP   | ID連携（ソーシャルログイン）             |
| Data Store               | SSM → DST   | ユーザーID共有                           |
| その他                   | -           | 直接やり取りなし                         |

---

## 6. Data Store（DST）

| 相手                     | 方向        | やり取り                                 |
| ------------------------ | ----------- | ---------------------------------------- |
| Session Manager          | DST ← SSM   | ユーザーID共有                           |
| Auth Server              | DST ↔ AUS   | ユーザーID共有                           |
| Token Vault              | DST → TVL   | ユーザーID共有                           |
| Module Registry          | DST → REG   | ツール設定提供                           |
| MCP Handler              | DST → HDL   | カスタムプロンプト提供                   |
| Payment Service Provider | DST ← PSP   | プラン情報受信                           |
| User Console             | DST ← CON   | ツール設定登録                           |
| その他                   | -           | 直接やり取りなし                         |

---

## 7. Token Vault（TVL）

| 相手                     | 方向        | やり取り                                 |
| ------------------------ | ----------- | ---------------------------------------- |
| MCP Client (API KEY)     | TVL ← CLK   | API KEY認証受付                          |
| API Gateway              | TVL → GWY   | API KEY提供                              |
| Data Store               | TVL ← DST   | ユーザーID共有                           |
| User Console             | TVL ← CON   | トークン登録                             |
| External Auth Server     | TVL ← EAS   | 認証（トークン受信）                     |
| Modules                  | TVL → MOD   | トークン提供                             |
| その他                   | -           | 直接やり取りなし                         |

---

## 8. MCP Server（SRV）

MCP Serverは以下の内部コンポーネントで構成される。外部からはSRVとして抽象化される。

| 相手                     | 方向        | やり取り                                 |
| ------------------------ | ----------- | ---------------------------------------- |
| API Gateway              | SRV ← GWY   | リクエスト受付（Auth Middleware経由）    |
| その他                   | -           | 内部コンポーネント経由                   |

---

## 9. Auth Middleware（AMW）

| 相手                     | 方向        | やり取り                                 |
| ------------------------ | ----------- | ---------------------------------------- |
| API Gateway              | AMW ← GWY   | リクエスト受信                           |
| MCP Handler              | AMW → HDL   | 認証済みリクエスト転送                   |
| その他                   | -           | 直接やり取りなし                         |

---

## 10. MCP Handler（HDL）

| 相手                     | 方向        | やり取り                                 |
| ------------------------ | ----------- | ---------------------------------------- |
| Auth Middleware          | HDL ← AMW   | 認証済みリクエスト受信                   |
| Data Store               | HDL ← DST   | カスタムプロンプト取得                   |
| Module Registry          | HDL → REG   | メタツール呼び出し                       |
| その他                   | -           | 直接やり取りなし                         |

---

## 11. Module Registry（REG）

| 相手                     | 方向        | やり取り                                 |
| ------------------------ | ----------- | ---------------------------------------- |
| MCP Handler              | REG ← HDL   | メタツールリクエスト受信                 |
| Data Store               | REG ← DST   | ツール設定取得                           |
| Modules                  | REG → MOD   | ツール実行委譲                           |
| その他                   | -           | 直接やり取りなし                         |

---

## 12. Modules（MOD）

| 相手                     | 方向        | やり取り                                 |
| ------------------------ | ----------- | ---------------------------------------- |
| Module Registry          | MOD ← REG   | ツール実行リクエスト受信                 |
| Token Vault              | MOD ← TVL   | トークン取得                             |
| External Service API     | MOD → EXT   | リソースアクセス（HTTPS）                |
| その他                   | -           | 直接やり取りなし                         |

---

## 13. User Console（CON）

| 相手                     | 方向        | やり取り                                 |
| ------------------------ | ----------- | ---------------------------------------- |
| Payment Service Provider | CON → PSP   | 決済リクエスト                           |
| Token Vault              | CON → TVL   | トークン登録                             |
| Data Store               | CON → DST   | ツール設定登録                           |
| External Auth Server     | CON → EAS   | 認可フロー                               |
| Identity Provider        | CON → IDP   | ソーシャルログイン                       |
| その他                   | -           | 直接やり取りなし                         |

---

## 14. Identity Provider（IDP）

| 相手                     | 方向        | やり取り                                 |
| ------------------------ | ----------- | ---------------------------------------- |
| Session Manager          | IDP → SSM   | ID連携                                   |
| User Console             | IDP ← CON   | ソーシャルログイン                       |
| その他                   | -           | 直接やり取りなし                         |

---

## 15. External Auth Server（EAS）

| 相手                     | 方向        | やり取り                                 |
| ------------------------ | ----------- | ---------------------------------------- |
| User Console             | EAS ← CON   | 認可フロー受付                           |
| Token Vault              | EAS → TVL   | 認証トークン提供                         |
| External Service API     | EAS ↔ EXT   | 同一サービス内連携                       |
| その他                   | -           | 直接やり取りなし                         |

---

## 16. External Service API（EXT）

| 相手                     | 方向        | やり取り                                 |
| ------------------------ | ----------- | ---------------------------------------- |
| Modules                  | EXT ← MOD   | API呼び出し受付（HTTPS）                 |
| External Auth Server     | EXT ↔ EAS   | 同一サービス内連携                       |
| その他                   | -           | 直接やり取りなし                         |

---

## 17. Payment Service Provider（PSP）

| 相手                     | 方向        | やり取り                                 |
| ------------------------ | ----------- | ---------------------------------------- |
| User Console             | PSP ← CON   | 決済リクエスト受付                       |
| Data Store               | PSP → DST   | プラン情報提供（Webhook含む）            |
| その他                   | -           | 直接やり取りなし                         |
