# Backlog

## Console UI

### 外見設定のDB永続化

**現状**: localStorage に保存
**要件**: Supabase DB に保存して複数デバイスで共有

**実装内容**:
1. `user_preferences` テーブル作成
   ```sql
   CREATE TABLE user_preferences (
     user_id UUID PRIMARY KEY REFERENCES auth.users(id) ON DELETE CASCADE,
     theme TEXT DEFAULT 'dark', -- 'light' | 'dark' | 'system'
     background_color TEXT DEFAULT 'black', -- 'black' | 'slate' | 'zinc' | 'stone'
     accent_color TEXT DEFAULT 'green', -- 'green' | 'blue' | 'purple' | 'pink' | 'orange' | 'yellow'
     created_at TIMESTAMPTZ DEFAULT NOW(),
     updated_at TIMESTAMPTZ DEFAULT NOW()
   );
   ```

2. `appearance-context.tsx` を更新
   - localStorage の代わりに Supabase から取得・保存
   - 初回アクセス時にデフォルト値でレコード作成

3. API Route または Server Action の実装
   - GET: ユーザー設定取得
   - PUT: ユーザー設定更新

**関連ファイル**:
- `apps/console/src/lib/appearance-context.tsx`
- `apps/console/src/app/(console)/settings/page.tsx`

---

### サービス認証方法のDB管理

**現状**: フロントエンド側でハードコード（`connections/page.tsx` の `extendedServices` 配列）

**要件**: サービスごとに利用可能な認証方法をDB側で管理

**実装内容**:
1. `service_auth_methods` テーブル作成
   ```sql
   CREATE TABLE service_auth_methods (
     id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
     service_id TEXT NOT NULL,
     auth_type TEXT NOT NULL, -- 'oauth2' | 'personal_token' | 'apikey' | 'integration_token'
     label TEXT NOT NULL, -- 表示名（例: "Personal Access Token", "内部インテグレーション"）
     help_text TEXT, -- 取得方法の説明
     oauth_config JSONB, -- OAuth用の設定（client_id, scopes等）
     priority INT DEFAULT 0, -- 表示順（複数認証方法がある場合）
     created_at TIMESTAMPTZ DEFAULT NOW(),
     UNIQUE(service_id, auth_type)
   );
   ```

2. 既存の `services` テーブルとの関連
   - `service_id` で紐付け
   - 1つのサービスに複数の認証方法を持てる設計

3. API実装
   - GET `/api/services/:id/auth-methods`: サービスの認証方法一覧取得

**関連ファイル**:
- `apps/console/src/app/(console)/my/connections/page.tsx`
- `apps/console/src/lib/data.ts`（既存の `ServiceAuthConfig` 型）

---

## 認証・セキュリティ

### OAuth 2.0 クライアント認証フローの完成

**現状**: 認可フロー開始まで実装済み（コールバック未実装）

**要件**:
1. コールバックページ実装（`/my/mcp-connection/callback`）
2. 認可コードをトークンに交換
3. トークンをVaultに保存
4. MCPクライアントへのトークン返却

**関連ファイル**:
- `apps/console/src/app/(console)/my/mcp-connection/callback/page.tsx`（新規）
- `apps/console/src/app/api/auth/token/route.ts`

---

### API Key認証の接続テスト改善

**現状**: 接続テストはできるが、MCP Serverが起動していないとエラー

**要件**:
1. MCP Serverのヘルスチェックを先に行う
2. エラーメッセージを分かりやすく
3. 成功時にツール一覧を表示

---

## MCP Server

### モジュール追加

**優先度: 高**

| モジュール | 状態 | 説明 |
|-----------|------|------|
| Notion | 実装済み | ページ・データベース操作 |
| Google Calendar | 未実装 | 予定の取得・作成 |
| Microsoft Todo | 未実装 | タスク管理 |
| Jira | 未実装 | Issue/Project操作 |
| Confluence | 未実装 | Wiki操作 |
| GitHub | 未実装 | リポジトリ、Issue、PR操作 |
| RAG | 未実装 | ドキュメント検索（セマンティック/キーワード） |
| Supabase | 未実装 | DB操作、マイグレーション、ログ、ストレージ |

---

### 認証ミドルウェアの統合テスト

**現状**: API Key検証のRPCは実装済み、Go側のミドルウェアも更新済み

**要件**:
1. E2Eテスト作成（API Key認証でMCPリクエスト）
2. OAuth 2.0トークン検証の実装
3. レート制限の実装

---

## インフラ

### 本番環境デプロイ準備

**優先度: 中**

| タスク | 状態 |
|--------|------|
| Vercelプロジェクト設定 | 未着手 |
| 本番Supabase設定 | 未着手 |
| 環境変数設定 | 未着手 |
| カスタムドメイン設定 | 未着手 |

---

### CI/CDパイプライン改善

**要件**:
1. プレビューデプロイ（PR単位）
2. マイグレーション自動適用
3. E2Eテスト（Playwright）

---

## ドキュメント

### API仕様書

**要件**:
1. OpenAPI/Swagger定義
2. MCP Protocol仕様（JSON-RPC over HTTP）
3. 認証フロー図

---

## 技術的負債

### useSearchParams の Suspense 対応

**問題**: Next.js 15でビルドエラー
```
useSearchParams() should be wrapped in a suspense boundary at page "/auth/consent"
```

**解決策**:
```tsx
<Suspense fallback={<Loading />}>
  <ConsentPageContent />
</Suspense>
```

**関連ファイル**:
- `apps/console/src/app/auth/consent/page.tsx`

---

### 未使用コード・変数の整理

**対象**:
- OAuth関連のstate変数（一部削除済み）
- 未使用のインポート

---

## 次のスプリント候補

### Sprint-002: MCP Server モジュール拡充

| タスク | 成果物 |
|--------|--------|
| Google Calendar モジュール | OAuth連携、予定CRUD |
| Microsoft Todo モジュール | タスクCRUD |
| 認証フローE2Eテスト | API Key + OAuth両方 |

### Sprint-003: 本番デプロイ

| タスク | 成果物 |
|--------|--------|
| Vercelデプロイ | Console本番公開 |
| MCP Serverデプロイ | Cloud Run or Fly.io |
| ドメイン設定 | mcpist.app |
