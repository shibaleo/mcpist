# DAY014 バックログ

## Sprint目標

**Phase 1（Supabase移行）完了後の次ステップ：CI/CDパイプライン構築とOAuth認可フローの安定化**

---

## 完了済みスプリント

### Sprint-004 (DAY013) ✅ 完了

| タスク | 状態 | 備考 |
|--------|------|------|
| Supabase Vault移行 | ✅ 完了 | migration-plan.md準拠 |
| Notion API 401エラー修正 | ✅ 完了 | AuthContext使用に修正 |
| Vercelビルドエラー修正 | ✅ 完了 | RPC型定義手動追加 |
| Go Server 0.1.3デプロイ | ✅ 完了 | Render更新済み |
| OAuth callbackパス統一 | ✅ 完了 | `/oauth/callback` に統一 |

---

## 残課題（DAY012から継続）

### 引き継ぎ課題

| ID | タスク | 優先度 | 状態 | 備考 |
|----|--------|--------|------|------|
| B-001 | Supabase OAuth Server 有効化 | 高 | ⬜ 未着手 | ベータ機能、本番で使用 |
| B-002 | HTTPS 設定（本番） | 高 | ⬜ 未着手 | TLS 証明書設定 |
| B-003 | next.config.ts のデバッグログ削除 | 低 | ⬜ 未着手 | console.log削除 |
| B-005 | CI/CD パイプライン構築 | 中 | ⬜ 未着手 | GitHub Actions |
| B-006 | 本番デプロイ時の鍵管理 | 高 | ⬜ 未着手 | 環境変数で管理 |

### 完了済み

| ID | タスク | 完了日 |
|----|--------|--------|
| B-004 | refresh_token grant テスト | 2026-01-22 |
| B-007 | Console → Worker `/internal/*` 認証追加 | 2026-01-22 |

---

## タスク一覧

### Phase 1: APIキー認証機能 ✅ 完了

すべて完了（T-001〜T-007）

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
| T-017 | refresh_token grant テスト | ✅ 完了 | - |
| T-018 | next.config.ts デバッグログ削除 | ⬜ 未着手 | 0.25h |
| T-019 | ドキュメント整備 | 🔄 継続 | 1h |

### Phase 5: OAuth認可フロー ✅ 完了

| ID | タスク | 状態 | 備考 |
|----|--------|------|------|
| T-020 | OAuth callbackパス統一 | ✅ 完了 | `/oauth/callback` に統一 (ea400c3) |
| T-021 | 管理UIからの認可フローテスト | 🔄 要テスト | consent画面 → callback |
| T-022 | Claude Web からの認可フローテスト | 🔄 要テスト | Claude.ai MCP連携 |

### Phase 6: UI設計・ドキュメント ⬜ 未着手

| ID | タスク | 状態 | 備考 |
|----|--------|------|------|
| T-023 | UI要求仕様書作成 | ⬜ 未着手 | 画面一覧・機能要件 |
| T-024 | パスルーティング設計 | ⬜ 未着手 | Next.js App Routerルート設計 |
| T-025 | ユーザーフロー図作成 | ⬜ 未着手 | 主要フローの可視化 |

### Phase 7: テスト設計 ⬜ 未着手

| ID | タスク | 状態 | 備考 |
|----|--------|------|------|
| T-026 | テスト戦略策定 | ⬜ 未着手 | E2E/単体/統合テスト方針 |
| T-027 | E2Eテストシナリオ設計 | ⬜ 未着手 | 主要ユースケースのシナリオ |
| T-028 | 詳細テスト設計 | ⬜ 未着手 | テストケース・期待結果 |
| T-029 | OAuth認可フローテスト手順書 | ⬜ 未着手 | T-021/T-022の詳細手順 |

### Phase 8: クレジット管理機能 ⬜ 未着手

| ID | タスク | 状態 | 備考 |
|----|--------|------|------|
| T-030 | 管理者画面にクレジット増量UI追加 | ⬜ 未着手 | 自分のクレジットを入力して増量 |
| T-031 | クレジット増量RPC作成 | ⬜ 未着手 | トランザクション＋残高検証ロジック |
| T-032 | 不正クレジット増加防止 | ⬜ 未着手 | SQL直接操作によるクレジット増加を防止 |

---

## 完了条件

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

## 次にやるべきこと

### 優先度: 高
1. **CI/CDパイプライン構築 (T-008〜T-010)**
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

## 技術的負債・注意事項

### Supabase RPC 型定義の管理

**問題**: Supabase CLIの型生成がRPC関数の型を正しく生成しない場合がある。

**現状の対応**: `apps/console/src/lib/supabase/database.types.ts` に手動でRPC型定義を追加。

**新しいRPCを追加した場合**:
1. `database.types.ts` の `Functions` セクションに型定義を追加
2. `Args`: 引数の型
3. `Returns`: 戻り値の型

```typescript
new_rpc_function: {
  Args: {
    p_param1: string
    p_param2?: number | null  // optional
  }
  Returns: {
    id: string
    name: string
  }[]  // 配列の場合
}
```

### ローカルとVercelの型チェックの違い

**問題**: ローカルの `pnpm tsc --noEmit` では検出できないエラーがVercelビルドで発生する

**対策**: デプロイ前に以下を実行
```bash
cd apps/console && pnpm exec next build
```

---

## デプロイ手順メモ

### Go Server (Render/Koyeb)

```bash
cd apps/server
docker build --platform linux/amd64 -t shibaleo/mcpist-api:X.X.X -t shibaleo/mcpist-api:latest .
docker push shibaleo/mcpist-api:X.X.X
docker push shibaleo/mcpist-api:latest
```

Renderダッシュボードで Image URL を `shibaleo/mcpist-api:X.X.X` に更新して再デプロイ

**注意**: `latest`タグはキャッシュされやすいので、明示的なバージョンタグを使用

### Worker ルーティング

`wrangler.toml` の設定:
- Primary: Render (`mcpist-api-dev.onrender.com`)
- Secondary: Koyeb (フォールバック)

---

## 参考資料

- [DAY013 レビュー](../DAY013/review.md)
- [DAY012 バックログ](../DAY012/backlog.md)
- [システムアーキテクチャ](../DAY011/mcpist-system-architecture.md)
