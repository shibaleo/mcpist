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
