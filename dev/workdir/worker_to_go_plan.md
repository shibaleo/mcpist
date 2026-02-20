# Worker廃止でGo Serverに機能集約する実装計画

## ゴール
Cloudflare Workerを廃止し、認証・MCPプロキシ・REST API・OAuth関連の機能をすべてGo Serverに集約する。ConsoleはGo Serverへ直接接続する。

## 前提
- 現在のWorkerは以下を担当: 認証(JWT/API Key), MCPプロキシ, /v1/me, /v1/admin, /v1/oauth の代理, OAuthメタデータ, JWKS配布, CORS, セキュリティヘッダ
- Go ServerはMCP/REST本体とDBアクセスを担当

## 実装計画

### 1. ルーティング統合
- Go ServerにWorker相当のHTTPエンドポイントを追加
  - `/v1/mcp/*`
  - `/v1/me/*`
  - `/v1/admin/*`
  - `/v1/oauth/apps/{provider}/credentials`
  - `/.well-known/*` (oauth-authorization-server, oauth-protected-resource, jwks)
- ConsoleのAPIクライアントは `WORKER_URL` ではなく Go Server を直参照

### 2. 認証/認可の内製化
- Go Serverで `Authorization: Bearer` を直接検証
  - Clerk JWT検証 (JWKS取得)
  - API Key JWT検証
- APIキー検証は「署名検証 + DB照合」を必須化
  - `jti` / `key_id` をJWTに含める
  - 失効/期限切れをDBで判定
- 既存 `X-Gateway-Token` を廃止

### 3. MCPプロキシ削除/統合
- Workerの `/v1/mcp/*` プロキシをGo Serverに直接組み込み
- transport middleware (SSE/inline) をそのままGoで受ける
- CORS/セキュリティヘッダをGo側で設定

### 4. OAuthメタデータの再実装
- `/.well-known/oauth-authorization-server`
- `/.well-known/oauth-protected-resource`
- 既存のClerkプロキシ仕様に合わせる

### 5. CORS/セキュリティヘッダ
- Go ServerにCORSミドルウェア導入
- セキュリティヘッダ (CSP/Frame-ancestors) 付与

### 6. レート制限
- 現行のMCP向け rate limiter を全APIに適用可能に拡張
- per-user or per-ip の制限

### 7. Console/Env変更
- `NEXT_PUBLIC_MCP_SERVER_URL` を Go Server URL に切り替え
- Worker関連の環境変数削除
  - `GATEWAY_SIGNING_KEY`, `WORKER_JWKS_URL`, `SERVER_JWKS_URL` 等

### 8. 監査ログ/観測
- Workerで行っていた `request/security` ログ送信をGo側へ移植
- ロギング項目に `user_id`, `auth_type`, `request_id` を含める

### 9. デプロイ構成
- Worker削除
- Go ServerのTLS/公開URLを固定
- CORS許可ドメインを環境変数化

## 影響範囲
- Console APIクライアント
- Go Serverの認証/RESTルーティング
- OAuthフロー
- 環境変数/デプロイ

## リスク
- JWT検証やCORS不備による認証バイパス
- SSEの負荷集中
- エッジのDDoS耐性低下

## 検証項目
- Clerk JWT / API Key での認証成功
- API Key revoke後にアクセス不可
- OAuth callback/refresh動作
- MCP SSEとinline両方で正常応答
- CORS preflight

