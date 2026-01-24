# DAY013 Review

## 作業内容

### Phase 3: データモデル設計 ✅

テーブル設計ドキュメントを新規作成し、Phase 3を完了。

#### 作成したドキュメント

| ファイル | 内容 |
|---------|------|
| dsn-tbl.md | テーブル設計書（ER図・リレーション・mermaid図） |
| dtl-dsn-tbl.md | テーブル詳細設計書（列定義・制約・RLS・RPC関数） |
| grh-table-design.canvas | ER図（Obsidian Canvas形式） |

#### 設計したテーブル（11テーブル）

- auth.users（Supabase管理）
- vault.secrets（Supabase管理）
- mcpist.users
- mcpist.api_keys
- mcpist.credits
- mcpist.credit_transactions
- mcpist.modules
- mcpist.module_settings
- mcpist.tool_settings
- mcpist.prompts
- mcpist.processed_webhook_events

### アーキテクチャ整理

#### API Key認証の仕組み
- SHA-256ハッシュで保存（原本は発行時のみ表示）
- ソルト不要（高エントロピー: mpt_<32 hex chars>）
- vault.secretsは不要（認証用、外部サービス接続用ではない）

#### OAuth審査の調査
- Google/Microsoft: CASA審査必要（個人開発者には厳しい）
- 未検証アプリ: 100ユーザー制限、警告表示（MVPはこれで可）
- Notion/GitHub: 審査が軽い（MVP優先対象）
- Supabase Auth: ログイン用（審査不要）とMCPistのAPI利用（要審査）は別

#### コンポーネント別Supabaseキー
- User Console (Frontend): anon key（RLS適用）
- User Console (API Routes): service_role key（Webhook処理）
- MCP Server (Go): service_role key（ツール実行、クレジット消費）
- Cloudflare Worker: キー不要（ルーティングのみ）

### 議論・決定事項

1. **Webhook処理はVercel API Routes**
   - MCP ServerではなくVercel Serverless (Next.js API Routes)で処理
   - processed_webhook_eventsへのINSERTはservice_role keyで実行

2. **MVPモジュール戦略**
   - PAT/API Token系: Notion, GitHub（審査が軽い）
   - OAuth系: Google Calendar, Microsoft ToDo（法人化後に対応）

3. **著作権・特許**
   - ソースコードは自動的に著作権保護
   - 特許は出願必要（費用高、MCPistには向かない）

4. **個人運用でのマネタイズは不可能**
   - Google/Microsoft OAuth審査: CASA審査が必要で個人開発者には厳しい
   - 未検証アプリ: 100ユーザー制限があり、商用展開は現実的ではない
   - 現実的な運用: 友人に使ってもらってフィードバックをもらう程度
   - 法人化すればOAuth審査のハードルは下がるが、コストが見合わない

## 感想

ドキュメント整備・整合性担保の作業が結構きつい。
テーブル設計は一通り終わったので、次はRPC関数の実装に移れる。

---

## 2026-01-24 追加作業

### Notion API 401エラーの修正

**問題**: MCP Server経由でNotion APIを呼び出すと401 unauthorizedエラーが発生

**原因調査**:
1. 古い`vault/client.go`が存在しないWorkerエンドポイント`/token-vault`を呼び出していた
2. `notion/client.go`がcontextから`"user_id"`キーで直接取得しようとしていたが、実際には`entitlement.AuthContext`に格納されていた

**修正内容**:
- `internal/token/store.go` を新規作成（設計書 migration-plan.md に準拠）
- `notion/client.go` で `entitlement.GetAuthContext(ctx)` を使用してUserIDを取得
- 古い `vault/client.go` を削除

### Vercel ビルドエラーの修正

**問題**: Vercelでビルド時に型エラーが発生

**原因**: Supabase RPC関数の型定義が `database.types.ts` に含まれていなかった

**修正内容**: `apps/console/src/lib/supabase/database.types.ts` に全RPC関数の型定義を手動追加

---

## 学んだこと・注意事項

### Supabase RPC 型定義の管理

**問題**: Supabase CLIの型生成 (`supabase gen types typescript`) がRPC関数の型を正しく生成しない場合がある。

**現状の対応**: `apps/console/src/lib/supabase/database.types.ts` に手動でRPC型定義を追加している。

**新しいRPCを追加した場合**:
1. `database.types.ts` の `Functions` セクションに型定義を追加する
2. `Args`: 引数の型
3. `Returns`: 戻り値の型

**例**:
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

**今後の改善案**:
- Supabase CLIが改善されたら自動生成に切り替え
- または、独自のスクリプトでDBスキーマから型を生成

### ローカルとVercelの型チェックの違い

**問題**: ローカルの `pnpm tsc --noEmit` では検出できないエラーがVercelビルドで発生する

**原因**: Next.jsのビルド時の型チェックはより厳格

**対策**: デプロイ前に `pnpm turbo run build --filter=@mcpist/console --force` を実行してVercelと同等のチェックを行う

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
- Primary: Render (`REDACTED_HOST`)
- Secondary: Koyeb (フォールバック)

両方のバックエンドを更新する必要がある場合は注意

---

## 変更ファイル一覧

- `apps/server/internal/token/store.go` - 新規作成
- `apps/server/internal/modules/notion/client.go` - AuthContext使用に修正
- `apps/server/internal/vault/client.go` - 削除
- `apps/console/src/lib/supabase/database.types.ts` - RPC型定義追加
- `apps/console/src/lib/credits.ts` - ServiceConnection型修正
- `apps/console/src/lib/token-vault.ts` - 型アサーション追加
