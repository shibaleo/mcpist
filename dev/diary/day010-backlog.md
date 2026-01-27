# Backlog（DAY010 更新版）

最終更新: 2026-01-20

---

## 優先度: 高

### JWT 認証テスト（OAuth トークン）

**状態**: 未着手
**前提**: API Key 認証は完了

**要件**:
1. OAuth 2.0 トークンでの MCP リクエスト検証
2. 無効なトークンでの拒否確認
3. トークン有効期限切れ時の動作確認

**関連ファイル**:
- `apps/worker/src/index.ts` - JWT検証ロジック
- `apps/console/src/app/api/auth/token/route.ts`

---

### 本番環境デプロイ

**状態**: 未着手

| タスク | 状態 | 備考 |
|--------|------|------|
| Render MCP Server デプロイ（Primary） | 未着手 | |
| Koyeb MCP Server デプロイ（Failover） | 未着手 | |
| Cloudflare Worker デプロイ | 未着手 | |
| Vercel Console デプロイ | 未着手 | |
| 本番 Supabase 設定 | 未着手 | OAuth Server (BETA) 設定含む |
| DNS 設定（api.mcpist.app 等） | 未着手 | |
| SSL/TLS 証明書 | 未着手 | Cloudflare で自動 |
| 環境変数設定 | 未着手 | `ENVIRONMENT=production` |

---

### Supabase OAuth Server 設定（Dashboard）

**状態**: 未着手

**設定項目**:
- Authorization URL: `/oauth/consent`
- Token URL: `/api/auth/token`
- クライアント登録

---

### Worker デバッグログ削除

**状態**: 未着手

**対象**:
- `apps/worker/src/index.ts` の `console.log` 削除
- 本番デプロイ前に実施

---

## 優先度: 中

### ネットワーク接続検証（DinD 内）

**状態**: 未着手

**要件**:
- Devcontainer 内での Docker-in-Docker 通信確認
- コンテナ間のネットワーク疎通

---

### LB/ヘルスチェック動作検証

**状態**: 未着手

**要件**:
1. 複数バックエンドへの分散確認
2. `docker stop api-render` → koyeb のみに流れることを確認
3. フェイルオーバー/復旧シナリオ

---

### OAuth 2.0 クライアント認証フローの完成

**状態**: 進行中（コールバック未実装）

**要件**:
1. コールバックページ実装（`/my/mcp-connection/callback`）
2. 認可コードをトークンに交換
3. トークンを Vault に保存
4. MCP クライアントへのトークン返却

**関連ファイル**:
- `apps/console/src/app/(console)/my/mcp-connection/callback/page.tsx`（新規）
- `apps/console/src/app/api/auth/token/route.ts`

---

### API Key 認証の接続テスト改善

**状態**: 未着手

**要件**:
1. MCP Server のヘルスチェックを先に行う
2. エラーメッセージを分かりやすく
3. 成功時にツール一覧を表示

---

### 開発用 OAuth Server の別コンテナ化

**状態**: 未着手
**優先度**: 中（本番 Supabase OAuth Server の挙動確認後）

**背景**:
- 本番: Supabase OAuth Server (BETA) を使用
- 開発: 現在は Next.js API Routes で実装（`/api/auth/*`）
- Supabase OAuth Server の挙動が判明したら、開発用も分離可能

**実装案**:
```
┌─────────────────────────┐    ┌─────────────────────┐
│ Supabase Local          │    │ Custom OAuth Server │
│  ├─ Auth (ログイン用)   │    │  ├─ /authorize      │
│  ├─ PostgreSQL          │◄───│  ├─ /token          │
│  └─ Vault               │    │  └─ /consent        │
└─────────────────────────┘    └─────────────────────┘
```

**ポイント**:
- OAuth Server 部分のみ分離（SDK 非依存）
- ログイン/セッション/RLS は Supabase のまま
- Go または Node.js で実装
- `compose/oauth-server.yml` として追加

**関連ファイル**:
- `apps/console/src/app/api/auth/authorize/route.ts`
- `apps/console/src/app/api/auth/token/route.ts`
- `apps/console/src/app/oauth/consent/page.tsx`

---

### モジュール追加

**優先度**: 中

| モジュール | 状態 | 説明 |
|-----------|------|------|
| Notion | ✅ 実装済み | ページ・データベース操作 |
| Google Calendar | 未実装 | 予定の取得・作成 |
| Microsoft Todo | 未実装 | タスク管理 |
| Jira | 未実装 | Issue/Project 操作 |
| Confluence | 未実装 | Wiki 操作 |
| GitHub | 未実装 | リポジトリ、Issue、PR 操作 |
| RAG | 未実装 | ドキュメント検索（セマンティック/キーワード） |
| Supabase | 未実装 | DB 操作、マイグレーション、ログ、ストレージ |

---

## 優先度: 低

### 外見設定の DB 永続化

**状態**: 未着手
**現状**: localStorage に保存

**要件**: Supabase DB に保存して複数デバイスで共有

**実装内容**:
1. `user_preferences` テーブル作成
2. `appearance-context.tsx` を更新
3. API Route または Server Action の実装

**関連ファイル**:
- `apps/console/src/lib/appearance-context.tsx`
- `apps/console/src/app/(console)/settings/page.tsx`

---

### サービス認証方法の DB 管理

**状態**: 未着手
**現状**: フロントエンド側でハードコード

**要件**: サービスごとに利用可能な認証方法を DB 側で管理

**関連ファイル**:
- `apps/console/src/app/(console)/my/connections/page.tsx`
- `apps/console/src/lib/data.ts`

---

### CI/CD パイプライン改善

**状態**: 未着手

**要件**:
1. プレビューデプロイ（PR 単位）
2. マイグレーション自動適用
3. E2E テスト（Playwright）

---

### API 仕様書

**状態**: 未着手

**要件**:
1. OpenAPI/Swagger 定義
2. MCP Protocol 仕様（JSON-RPC over HTTP）
3. 認証フロー図

---

## 技術的負債

### useSearchParams の Suspense 対応

**状態**: 未着手

**問題**: Next.js 15 でビルドエラー
```
useSearchParams() should be wrapped in a suspense boundary at page "/oauth/consent"
```

**解決策**:
```tsx
<Suspense fallback={<Loading />}>
  <ConsentPageContent />
</Suspense>
```

**関連ファイル**:
- `apps/console/src/app/oauth/consent/page.tsx`（パス変更済み）

---

### 未使用コード・変数の整理

**状態**: 未着手

**対象**:
- OAuth 関連の state 変数
- 未使用のインポート

---

### DockerHub へのプッシュ完了

**状態**: 中断（エラー発生）

**対象**:
- `shibaleo/mcpist-devcontainer:latest`

---

## 完了済み（DAY010）

- [x] Devcontainer + DinD 構築
- [x] Cloudflare Worker API Gateway 実装
- [x] E2E テスト環境構築（API Key 認証）
- [x] OpenTofu モジュール作成
- [x] 環境変数統合（.env.local）
- [x] Windows 互換スクリプト
- [x] OAuth Server 切り替え実装
- [x] consent ページのパス変更（/auth → /oauth）

---

## 次のスプリント候補

### Sprint-003: 本番デプロイ & JWT 認証

| タスク | 成果物 |
|--------|--------|
| 本番デプロイ | Render + Koyeb + Cloudflare + Vercel |
| JWT 認証テスト | OAuth トークンでの E2E |
| Supabase OAuth Server 設定 | Dashboard 設定 |

### Sprint-004: モジュール拡充

| タスク | 成果物 |
|--------|--------|
| Google Calendar モジュール | OAuth 連携、予定 CRUD |
| Microsoft Todo モジュール | タスク CRUD |

### Sprint-005: 課金機能

| タスク | 成果物 |
|--------|--------|
| プリペイドクレジット実装 | リクエスト数課金 |
| 決済連携 | Stripe 等 |
