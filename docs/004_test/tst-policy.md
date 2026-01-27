# MCPist テスト方針書（tst-policy）

## ドキュメント管理情報

| 項目 | 値 |
|------|-----|
| Status | `draft` |
| Version | v1.0 |
| Note | MCPist Unit Test Policy |

---

## 概要

本ドキュメントは、MCPistシステムのユニットテスト方針を定義する。

コンポーネント単体の責務を検証することを目的とし、各コンポーネントを独立してテスト可能な単位として扱う。

---

## テスト対象コンポーネント（9個）

MCP Serverは包含概念であり、内部コンポーネント（AMW, HDL, MOD）を個別にテストする。

| # | コンポーネント | 略称 | テスト対象の責務 |
|---|----------------|------|------------------|
| 1 | API Gateway | GWY | ロードバランシング、JWT/API KEY検証、リクエスト転送 |
| 2 | Auth Server | AUS | OAuth 2.1フロー、JWT発行、JWKS公開 |
| 3 | Session Manager | SSM | ユーザーID発行、ソーシャルログイン連携、セッション管理 |
| 4 | Data Store | DST | ユーザー情報CRUD、課金情報、クレジット残高、ツール設定 |
| 5 | Token Vault | TVL | トークン暗号化/復号、トークン取得API、トークンリフレッシュ |
| 6 | Auth Middleware | AMW | X-Gateway-Secret検証、リクエスト転送 |
| 7 | MCP Handler | HDL | MCPメソッドルーティング、モジュール管理、メタツール |
| 8 | Modules | MOD | 外部サービスAPI呼び出し、トークン取得、ビジネスロジック |
| 9 | User Console | CON | 外部OAuth連携、トークン登録、ツール設定、課金 |

---

## テスト種別

### ユニットテスト

各コンポーネントの内部ロジックを独立して検証する。

| 観点 | 説明 |
|------|------|
| 目的 | コンポーネント単体の責務が正しく実装されていることを検証 |
| 依存関係 | モック/スタブで置換 |
| 実行環境 | ローカル開発環境 |
| 実行頻度 | コミット時（CI） |

### 統合テスト

コンポーネント間のインタラクションを検証する。

| 観点 | 説明 |
|------|------|
| 目的 | dtl-itr-XXX-YYY で定義されたインタラクションが正しく動作することを検証 |
| 依存関係 | 実コンポーネントまたはPrism mock |
| 実行環境 | ローカル開発環境 / ステージング環境 |
| 実行頻度 | PR時（CI） |

---

## コンポーネント別テスト方針

### 1. API Gateway (GWY)

| テスト項目 | 説明 | モック対象 |
|-----------|------|-----------|
| JWT検証 | 有効/無効/期限切れJWTの検証 | AUS (JWKS) |
| API KEY検証 | 有効/無効API KEYの検証 | TVL |
| リクエスト転送 | X-Gateway-Secretの付与、ヘッダー転送 | AMW |
| エラーハンドリング | 401/403レスポンスの返却 | - |

### 2. Auth Server (AUS)

| テスト項目 | 説明 | モック対象 |
|-----------|------|-----------|
| 認可エンドポイント | /authorize リダイレクト | - |
| トークンエンドポイント | /token JWT発行 | DST (ユーザー情報) |
| JWKSエンドポイント | /.well-known/jwks.json 公開鍵返却 | - |
| PKCE検証 | code_verifier/code_challenge検証 | - |

### 3. Session Manager (SSM)

| テスト項目 | 説明 | モック対象 |
|-----------|------|-----------|
| ソーシャルログイン | IDPからのコールバック処理 | IDP |
| ユーザーID発行 | 新規ユーザー作成 | DST |
| セッション管理 | セッション作成/検証/破棄 | - |

### 4. Data Store (DST)

| テスト項目 | 説明 | モック対象 |
|-----------|------|-----------|
| ユーザーCRUD | ユーザー情報の作成/読取/更新/削除 | - |
| クレジット残高 | 残高取得/消費/加算 | - |
| ツール設定 | 有効/無効設定の保存/取得 | - |
| Webhook処理 | PSPからのWebhook受信/検証 | PSP |

### 5. Token Vault (TVL)

| テスト項目 | 説明 | モック対象 |
|-----------|------|-----------|
| トークン取得 | POST /token-vault API | - |
| トークン暗号化 | 保存時の暗号化 | - |
| トークン復号 | 取得時の復号 | - |
| API KEY検証 | ハッシュ照合 | DST (API KEY情報) |

### 6. Auth Middleware (AMW)

| テスト項目 | 説明 | モック対象 |
|-----------|------|-----------|
| X-Gateway-Secret検証 | 有効/無効シークレットの検証 | - |
| リクエスト転送 | HDLへの転送 | HDL |
| エラーレスポンス | 401/403 JSON-RPCエラー返却 | - |

### 7. MCP Handler (HDL)

| テスト項目 | 説明 | モック対象 |
|-----------|------|-----------|
| tools/list | ツール一覧返却 | MOD |
| tools/call | ツール実行委譲 | MOD |
| resources/list | リソース一覧返却 | MOD |
| prompts/list | プロンプト一覧返却 | DST |
| メタツール | get_module_schema, run, batch | MOD |
| ユーザー設定フィルタリング | 有効ツールのみ返却 | DST |

### 8. Modules (MOD)

| テスト項目 | 説明 | モック対象 |
|-----------|------|-----------|
| トークン取得 | TVLからのトークン取得 | TVL |
| 外部API呼び出し | サービス固有のAPI呼び出し | EXT |
| エラーハンドリング | 外部APIエラーの変換 | - |
| クレジット消費 | 実行時のクレジット消費 | DST |

### 9. User Console (CON)

| テスト項目 | 説明 | モック対象 |
|-----------|------|-----------|
| ログイン | SSM経由ソーシャルログイン | SSM |
| 外部OAuth連携 | EASへの認可フロー開始 | EAS |
| トークン登録 | TVLへのトークン保存 | TVL |
| ツール設定 | DSTへの設定保存 | DST |
| 決済 | PSPへのCheckout処理 | PSP |

---

## モック戦略

### 外部依存（実装範囲外）のモック

| コンポーネント | モック方法 |
|----------------|-----------|
| MCP Client (CLO/CLK) | テストクライアント実装 |
| Identity Provider (IDP) | OAuth mockサーバー |
| External Auth Server (EAS) | OAuth mockサーバー |
| External Service API (EXT) | Prism mock / WireMock |
| Payment Service Provider (PSP) | Stripe Test Mode / Webhook mock |

### 内部コンポーネントのモック

| 方法 | 用途 |
|------|------|
| Interface mock | Goのinterface + モック実装 |
| Prism mock | OpenAPI仕様に基づくAPIモック |
| Test doubles | スタブ/スパイ/フェイク |

---

## テストファイル命名規則

| 種別 | ファイル名パターン | 例 |
|------|-------------------|-----|
| ユニットテスト | `*_test.go` | `handler_test.go` |
| 統合テスト | `*_integration_test.go` | `vault_integration_test.go` |
| テスト手順書 | `tst-{略称}.md` | `tst-tvl.md` |
| テスト結果 | `tst-{略称}-{対象}.md` | `tst-mod-notion.md` |

---

## テストカバレッジ目標

| 種別 | 目標 |
|------|------|
| ユニットテスト | 80%以上 |
| 統合テスト | 主要フロー100% |

---

## 関連ドキュメント

| ドキュメント | 内容 |
|-------------|------|
| [spc-sys.md](../002_specification/spc-sys.md) | システム仕様書 |
| [spc-itr.md](../002_specification/spc-itr.md) | インタラクション仕様書 |
| [idx-itr-rel.md](../002_specification/interaction/idx-itr-rel.md) | インタラクション関係ID一覧 |
| [tst-tvl.md](./tst-tvl.md) | Token Vault テスト手順書 |
| [tst-oauth-mock-server.md](./tst-oauth-mock-server.md) | OAuth mockサーバー手順書 |
