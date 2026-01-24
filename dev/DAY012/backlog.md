# DAY012 バックログ

## Sprint目標

**DAY011で完成した開発環境を基盤に、本番デプロイ準備とAPIキー認証機能を実装する。**

## 前回からの引き継ぎ課題

DAY011のバックログから引き継いだ残課題：

| ID | タスク | 優先度 | 備考 |
|----|--------|--------|------|
| B-001 | Supabase OAuth Server 有効化 | 高 | ベータ機能、本番で使用 |
| B-002 | HTTPS 設定（本番） | 高 | TLS 証明書設定 |
| B-003 | next.config.ts のデバッグログ削除 | 低 | console.log for NEXT_PUBLIC_SUPABASE_URL |
| B-004 | refresh_token grant テスト | 中 | ✅ 完了 (2026-01-22) |
| B-005 | CI/CD パイプライン構築 | 中 | GitHub Actions |
| B-006 | 本番デプロイ時の鍵管理 | 高 | 環境変数で管理 |
| B-007 | Console → Worker `/internal/*` 認証追加 | 中 | ✅ 完了: `INTERNAL_SECRET` で検証 |

---

## 2026-01-22 完了タスク

### APIキー管理機能の統一

| タスク | 状態 | 備考 |
|--------|------|------|
| `/my/mcp-connection` からAPIキー生成機能を削除 | ✅ 完了 | `/my/api-keys`に統一 |
| `/api/apikey` route 削除 | ✅ 完了 | 古い`oauth_tokens`テーブル使用の実装 |
| Worker Dockerfile.dev に `INTERNAL_SECRET` 追加 | ✅ 完了 | キャッシュinvalidateが動作 |
| README.md 更新 | ✅ 完了 | Hostヘッダー、スクリプト名修正 |

### 発見した問題と解決

1. **APIキーが削除後もアクセスできた問題**
   - 原因: Worker の `/internal/invalidate-api-key` が 401 Unauthorized
   - 原因: Dockerfile.dev で `INTERNAL_SECRET` が `.dev.vars` に含まれていなかった
   - 解決: grep パターンに `INTERNAL_SECRET` を追加

2. **Docker外からMCPサーバーにアクセスできない問題**
   - 原因: `*.localhost` はDocker内部でのみ解決される
   - 解決: `Host` ヘッダーを使ってnginxにルーティングさせる

---

## タスク一覧

### Phase 1: APIキー認証機能実装 ✅ 完了

| ID | タスク | 状態 | 見積 |
|----|--------|------|------|
| T-001 | APIキーテーブル設計 (DBマイグレーション) | ✅ 完了 | 1h |
| T-002 | APIキー生成・管理RPC関数作成 | ✅ 完了 | 1h |
| T-003 | Console: APIキー管理画面UI作成 | ✅ 完了 | 2h |
| T-004 | Console: APIキー発行・削除API作成 | ✅ 完了 | 1h |
| T-005 | Worker: APIキー検証ミドルウェア実装 | ✅ 完了 | 1h |
| T-006 | Worker: KVキャッシュ統合 | ✅ 完了 | 1h |
| T-007 | APIキー認証E2Eテスト | ✅ 完了 | 1h |

### Phase 2: CI/CD パイプライン構築 ⬜ 次のタスク

| ID | タスク | 状態 | 見積 |
|----|--------|------|------|
| T-008 | GitHub Actions ワークフロー作成 (dev環境) | ⬜ 未着手 | 2h |
| T-009 | 環境シークレット設定 | ⬜ 未着手 | 0.5h |
| T-010 | dev環境デプロイテスト | ⬜ 未着手 | 1h |
| T-011 | stage/production ワークフロー作成 | ⬜ 未着手 | 1h |

### Phase 3: 本番環境準備

| ID | タスク | 状態 | 見積 |
|----|--------|------|------|
| T-012 | Cloudflare Workers デプロイ設定 | ⬜ 未着手 | 1h |
| T-013 | Render/Koyeb デプロイ設定 | ⬜ 未着手 | 1h |
| T-014 | HTTPS/TLS 設定 | ⬜ 未着手 | 1h |
| T-015 | 本番用環境変数・シークレット設定 | ⬜ 未着手 | 1h |
| T-016 | 本番JWT鍵管理（環境変数設定） | ⬜ 未着手 | 0.5h |

### Phase 4: テスト・クリーンアップ

| ID | タスク | 状態 | 見積 |
|----|--------|------|------|
| T-017 | refresh_token grant テスト | ✅ 完了 | 0.5h |
| T-018 | next.config.ts デバッグログ削除 | ⬜ 未着手 | 0.25h |
| T-019 | ドキュメント整備 | 🔄 進行中 | 1h |

---

## 完了条件

### Phase 1: APIキー認証 ✅ 完了
- [x] Console画面でAPIキーを発行できる
- [x] APIキーでMCP Serverに接続できる
- [x] APIキーを削除すると接続が拒否される
- [x] KVキャッシュが機能している

### Phase 2: CI/CD
- [ ] mainブランチへのマージでdev環境に自動デプロイ
- [ ] タグ作成でstage/production環境にデプロイ

### Phase 3: 本番環境
- [ ] `https://mcp.mcpist.app` でMCP Serverにアクセス可能
- [ ] `https://console.mcpist.app` でConsole UIにアクセス可能
- [ ] 本番JWTが正しく署名・検証される

### Phase 4: テスト・クリーンアップ
- [x] refresh_token grantが動作する
- [ ] デバッグログが削除されている

---

## 技術メモ

### APIキー検証フロー（実装済み）

```
初回: Worker --KVミス--> Supabase RPC (30-50ms) --> KVにキャッシュ
2回目以降: Worker --KVヒット--> (1-5ms)
```

| 項目 | 値 |
|------|-----|
| キャッシュ | Cloudflare KV |
| TTL | ソフト: 1時間, ハード: 1日 |
| キャッシュ内容 | APIキーハッシュ → ユーザーID |
| 無効化 | Console削除時に即時無効化（Server Action → Worker `/internal/invalidate-api-key`）|

### Cloudflare KV 無料枠

| 項目 | 無料枠 | 有料 |
|------|--------|------|
| Reads | 100,000/day | $0.50/million |
| Writes | 1,000/day | $5.00/million |

**注意**: キャッシュヒットも1 readとしてカウント。スケール時は有料プラン移行が必要。

### 内部サービス間認証（実装済み）

| 通信経路 | 認証 | 状態 |
|---------|------|------|
| Console → Worker (`/internal/*`) | `INTERNAL_SECRET` | ✅ 実装済み |
| Console → DB (Supabase) | `SUPABASE_ANON_KEY` + Session | ✅ |
| Worker → DB (Supabase RPC) | `SUPABASE_ANON_KEY` | ✅ |
| Worker → API (Go Server) | `GATEWAY_SECRET` | ✅ |
| API → DB (Supabase) | `SUPABASE_ANON_KEY` + ユーザーコンテキスト | ✅ |

### CI/CD 環境構成（DAY011で設計済み）

| 環境 | Supabase/Render/Koyeb | Cloudflare | ドメイン |
|------|----------------------|------------|---------|
| dev | shiba.dog.leo.private | shiba.dog.leo.private | dev.mcpist.app |
| stage | fukudamakoto.private | shiba.dog.leo.private | stage.mcpist.app |
| production | fukudamakoto.work | shiba.dog.leo.private | cloud.mcpist.app |

---

## 2026-01-23 追加タスク

### OAuth認可フローテスト

| ID | タスク | 状態 | 備考 |
|----|--------|------|------|
| T-020 | 管理UIからの認可フローテスト | ⬜ 未着手 | Console `/my/mcp-connection` から OAuth フロー開始 → consent画面 → callback |
| T-021 | Claude Web からの認可フローテスト | ⬜ 未着手 | Claude.ai MCP連携から OAuth フロー開始 → consent画面 → callback |

**現在の問題**: Supabase OAuth Server で `authorization request cannot be processed` エラーが発生
- consent画面は正常に表示される
- 「許可する」クリック時に `approveAuthorization()` がエラーを返す
- Supabase OAuth Server (BETA) の制限または設定の問題の可能性

### 参考実装（dwhbi）との比較

`C:\Users\shiba\HOBBY\dwhbi\packages\console` の実装を確認。mcpistとほぼ同じアプローチだが、以下の点が異なる：

**dwhbiの実装（シンプル）**:
```typescript
// consent/page.tsx
const { data, error: authError } = await supabase.auth.oauth.getAuthorizationDetails(authorizationId)
// ↓ dataをそのまま使用
setAuthDetails(data as AuthorizationDetails)

// approveも同様にシンプル
const { data, error } = await supabase.auth.oauth.approveAuthorization(authorizationId)
if (data?.redirect_url) {
  window.location.href = data.redirect_url
}
```

**mcpistの実装（やや冗長）**:
- `AuthorizationDetails` interfaceを独自に定義してマッピングしている
- `data.scope` を `split(' ')` でパースしている
- `data.client?.name` や `data.redirect_url` を手動で取り出している

**改善計画**:
1. mcpistのconsent/page.tsxをdwhbi版に近いシンプルな実装に置き換え
2. 余計なデータ変換を排除してSupabase SDKのレスポンスをそのまま使用
3. エラーハンドリングの改善（"cannot be processed" → ユーザーフレンドリーなメッセージ）

**.well-known/oauth-authorization-server の違い**:
- dwhbi: Supabase の `/auth/v1/.well-known/openid-configuration` をプロキシ
- mcpist: 手動でmetadataを構築

**改善計画**:
```typescript
// Supabaseのメタデータをそのままプロキシする方式に変更
const response = await fetch(`${SUPABASE_URL}/auth/v1/.well-known/openid-configuration`)
if (response.ok) {
  return NextResponse.json(await response.json())
}
```

**テスト手順（管理UIから）**:
1. `http://localhost:3000/my/mcp-connection` にアクセス
2. OAuth認可フローを開始
3. Supabase OAuth consent画面（`/oauth/consent`）に遷移
4. 「許可する」をクリック
5. callback (`/my/mcp-connection/callback`) にリダイレクト
6. アクセストークン取得を確認

**テスト手順（Claude Webから）**:
1. Claude.ai の MCP 設定から mcpist サーバーを追加
2. OAuth フロー開始（Supabase OAuth Server へリダイレクト）
3. Console の consent画面（`/oauth/consent`）に遷移
4. 「許可する」をクリック
5. Claude.ai にリダイレクト、MCP接続確立を確認

---

## 次にやるべきこと

### 優先度: 最高
1. **OAuth認可フローの修正 (T-020〜T-021)**
   - **方針: dwhbiの動作実績ある実装をそのままコピー**
   - dwhbiは実際に動作しているので、真似すれば必ず成功する

   **作業手順:**
   1. `dwhbi/packages/console/src/app/auth/consent/page.tsx` → `mcpist/apps/console/src/app/oauth/consent/page.tsx` にコピー
   2. `dwhbi/packages/console/src/app/.well-known/oauth-authorization-server/route.ts` → `mcpist/apps/console/src/app/.well-known/oauth-authorization-server/route.ts` にコピー
   3. import パスなど必要最小限の調整のみ行う
   4. テスト実行

### 優先度: 高
2. **CI/CDパイプライン構築 (T-008〜T-010)**
   - GitHub Actions ワークフロー作成
   - dev環境シークレット設定
   - dev環境デプロイテスト

### 優先度: 中
3. **next.config.ts デバッグログ削除 (T-018)**
4. **ドキュメント整備 (T-019)**

### 優先度: 低
5. **技術的負債の解消**
   - useSearchParams の Suspense 対応

---

## 参考資料

- [DAY011 レビュー](../DAY011/review.md)
- [DAY011 バックログ](../DAY011/backlog.md)
- [システムアーキテクチャ](../DAY011/mcpist-system-architecture.md)
- [Cloudflare Workers KV Pricing](https://developers.cloudflare.com/kv/platform/pricing/)
