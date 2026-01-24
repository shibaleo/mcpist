# MCPist システム構成図

## システムアーキテクチャ

```mermaid
flowchart LR
    subgraph ExternalLeft["外部システム"]
        direction TB
        subgraph Client["MCPクライアント"]
            CLIENT_DESC["・LLMからリクエスト受付<br>・ツール呼び出し"]
        end

        subgraph AuthProvider["認証プロバイダ"]
            AUTH_PROV_DESC["・ソーシャルログイン<br>・ID提供<br>・OAuth 2.0想定"]
        end

        subgraph PaymentService["決済代行サービス"]
            PAYMENT_DESC["・クレジットカード情報<br>・プラン情報<br>・webhook<br>・checkout"]
        end

        subgraph UserConsole["ユーザーコンソール"]
            CONSOLE_DESC["・外部OAuth連携<br>・外部シークレット保存<br>・ツール有効/無効設定<br>・クレジット課金"]
        end
    end

    subgraph APIGateway["APIゲートウェイ"]
        GW_DESC["・ロードバランシング<br>・ユーザー認証"]
    end

    subgraph SessionManager["Sessionマネージャー"]
        SM_DESC["・ユーザーID発行<br>・ソーシャルログイン連携<br>・セッション管理"]
    end

    subgraph AuthServer["Authサーバー"]
        AUTH_DESC["・OAuth 2.1準拠<br>・JWT発行<br>・JWKS公開<br>・MCPクライアントに認証<br>　情報を付与"]
    end

    subgraph DataStore["Data Store"]
        DS_DESC["・ユーザー情報<br>・課金情報<br>・クレジット残高情報<br>・ツール有効/無効 設定"]
    end

    subgraph TokenVault["Token Vault"]
        VAULT_DESC["・外部サービスシークレット<br>・OAuthリフレッシュトークン<br>・MCPサーバー認証シークレット<br>・MCPサーバー認証リフレッシュトークン"]
    end

    subgraph MCPServer["MCPサーバー"]
        direction TB
        subgraph AuthMiddleware["Authミドルウェア"]
            MW_DESC["・X-Gateway-Secret検証"]
        end

        subgraph MCPHandler["MCPハンドラ"]
            HANDLER_DESC["・tools/list, call<br>・resources<br>・prompts"]
        end

        subgraph ModuleRegistry["モジュールレジストリ"]
            REG_DESC["・get_module_schema<br>・run / batch"]
        end

        subgraph Modules["モジュール"]
            MOD_DESC["・MCPリソース<br>・MCPツール<br>・MCPプロンプト<br>・外部サービスアクセス"]
        end

        AuthMiddleware --> MCPHandler
        MCPHandler --> ModuleRegistry
        ModuleRegistry --> Modules
    end

    subgraph ExternalServices["外部WEBサービス"]
        subgraph ExtAuthServer["外部Authサーバー"]
            EXT_AUTH_DESC["・認可フロー<br>・OAuth 2.0想定"]
        end
        subgraph ExtServiceAPI["外部サービスAPI"]
            EXT_API_DESC["・API提供<br>・ユーザー所有リソース"]
        end
    end

    %% Main flows
    Client -->|"MCP通信"| APIGateway
    APIGateway --> MCPServer

    Client -->|"認証"| APIGateway
    AuthProvider -->|"ID連携"| SessionManager
    AuthServer -->|"JWT検証"| APIGateway

    PaymentService -->|"プラン情報"| DataStore

    UserConsole -->|"決済"| PaymentService
    UserConsole -->|"トークン登録"| TokenVault
    UserConsole -->|"ツール設定登録"| DataStore
    UserConsole -->|"認可フロー"| ExtAuthServer

    TokenVault -->|"トークン取得"| Modules

    DataStore -->|"ツール設定"| ModuleRegistry
    DataStore -->|"統計情報"| MCPHandler

    ExtAuthServer -->|"認証"| TokenVault

    Modules -->|"リソース<br>アクセス"| ExtServiceAPI

    %% Styling
    classDef clientBox fill:#fff,stroke:#333,stroke-dasharray:5 5
    classDef consoleBox fill:#ffe4b5,stroke:#333
    classDef gatewayBox fill:#87CEEB,stroke:#333
    classDef greenBox fill:#90EE90,stroke:#333
    classDef dataStoreBox fill:#f4a460,stroke:#333
    classDef mcpBox fill:#ffeb99,stroke:#333
    classDef externalBox fill:#fff,stroke:#333,stroke-dasharray:5 5

    class Client,AuthProvider,PaymentService clientBox
    class UserConsole consoleBox
    class APIGateway gatewayBox
    class SessionManager,AuthServer,TokenVault greenBox
    class DataStore dataStoreBox
    class MCPServer,AuthMiddleware,MCPHandler,ModuleRegistry,Modules mcpBox
    class ExternalServices,ExtAuthServer,ExtServiceAPI externalBox
```

## コンポーネント説明

### 外部システム（点線枠）

| コンポーネント | 説明 |
|---------------|------|
| MCPクライアント | LLMからリクエストを受け付け、ツール呼び出しを行う |
| 認証プロバイダ | ソーシャルログイン、ID提供（OAuth 2.0） |
| 決済代行サービス | クレジットカード情報、プラン情報、webhook、checkout |
| 外部WEBサービス | 外部Authサーバー（認可フロー）、外部サービスAPI（リソース提供） |

### 内部システム

| コンポーネント | 色 | 説明 |
|---------------|-----|------|
| ユーザーコンソール | オレンジ | OAuth連携、シークレット保存、ツール設定、課金管理 |
| APIゲートウェイ | 青 | ロードバランシング、ユーザー認証 |
| Sessionマネージャー | 緑 | ユーザーID発行、ソーシャルログイン連携、セッション管理 |
| Authサーバー | 緑 | OAuth 2.1準拠、JWT発行、JWKS公開 |
| Token Vault | 緑 | 外部サービスシークレット、OAuthトークン管理 |
| Data Store | オレンジ/赤 | ユーザー情報、課金情報、クレジット残高、ツール設定 |
| MCPサーバー | 黄色 | Authミドルウェア、MCPハンドラ、モジュールレジストリ、モジュール |

## データフロー

1. **MCP通信**: MCPクライアント → APIゲートウェイ → MCPサーバー
2. **認証フロー**: MCPクライアント → APIゲートウェイ（認証）、認証プロバイダ → Sessionマネージャー（ID連携）
3. **JWT検証**: Authサーバー → APIゲートウェイ
4. **課金フロー**: ユーザーコンソール → 決済代行サービス → Data Store（プラン情報）
5. **外部サービス連携**: ユーザーコンソール → 外部Authサーバー → Token Vault → モジュール → 外部サービスAPI
6. **ツール設定**: ユーザーコンソール → Data Store → モジュールレジストリ/MCPハンドラ
