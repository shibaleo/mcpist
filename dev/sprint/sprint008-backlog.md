# Sprint 008 バックログ

Sprint 008 のスコープ外だが、次スプリント以降で対処すべき項目。

---

## 優先度：高

### 認証基盤の整理（Sprint 009 候補）

認証（ユーザーは誰か）と認可（LLM に何を許可するか）は別のフロー。それぞれ独立に改善する。

#### 認証（OIDC）— Console ログイン

| 項目 | 現状 | 方針 |
|------|------|------|
| Console ログイン | Supabase Auth | **当面そのまま**。問題は起きていない |
| 将来の移行先 | — | Neon Auth (Better Auth ベース、OIDC 準拠) |

Supabase Auth は認証用途では問題なし。移行は DB 移行と同時に行えばよい。

#### 認可（OAuth Server）— LLM → MCPist API

| 項目 | 現状 | 方針 |
|------|------|------|
| LLM 認可フロー | Supabase Auth の OAuth | **自作が必要**。デバッグ不能が致命的 |
| 問題 | Claude App の認可フローが失敗してもログが出ない | Supabase Auth がブラックボックス |
| 移行先候補 | Cloudflare Worker 上に OAuth 2.1 Server | ログ・デバッグ完全制御 |
| 代替案 | Better Auth の OAuth 2.1 Provider プラグイン | OIDC Provider にもなれる |

```
[認証] ユーザー → OIDC (Supabase Auth / 将来 Neon Auth) → Console ログイン
[認可] LLM     → OAuth 2.1 (自作 or Better Auth)         → MCPist API アクセス許可
```

### DB・インフラ移行（Sprint 009〜010 候補）

| 項目 | 現状 | 移行先 | 備考 |
|------|------|--------|------|
| PostgreSQL | Supabase PG | Neon PG | PG 関数はそのまま移植可能 |
| PostgREST | Supabase 経由 | Neon PostgREST | Go/Console からの呼び出しコードは変更不要 |
| Token Vault | Supabase Vault | PG 暗号化 (pgcrypto) or Worker KV | 要設計 |

**動機:**
- 3 サービス (DB + Auth + Vault) が 1 プロバイダーに集中 → 障害時に全停止
- Neon は PostgREST を提供 → DB + REST API レイヤーの移行コスト低
- 認証は Supabase に残しても、DB は先に移行可能

**Neon ブランチ活用:**
- 無料プランで 10 ブランチまで利用可能
- staging/preview 環境をブランチで作成 → インフラコスト 0
- PR ごとにデータベースブランチを切るワークフローが可能

### 堅牢性改善

| 項目 | 現状 | 対策 |
|------|------|------|
| Supabase/Neon 障害時 | ~~全ユーザーブロック~~ | ~~サーキットブレーカー +~~ フォールバックキャッシュ **(Sprint 009 完了)** |
| リトライ・バックオフ | ~~一切なし~~ | 指数バックオフ + ジッター **(Sprint 009 完了)** |
| Worker セキュリティヘッダー | ~~なし~~ | CSP, HSTS, X-Content-Type-Options 等 **(Sprint 009 完了)** |
| OAuth2 x/oauth2 移行 | 手書き refreshOAuthToken | **見送り**: Notion (JSON body) / Microsoft Todo (extra scope) など x/oauth2 の画一性に合わない。現在のテーブル駆動で十分 |
| Rate Limiter | インスタンス独立 | 分散レートリミット（KV or PG ベース） |

---

## 優先度：中

### Stripe ogen 化

Stripe は OpenAPI spec を公開。ogen で型安全クライアントを生成し、他の 19 モジュールと同じ 3 層パターンに乗せる。

### Go GC 理解

ogen クライアントのタイムアウト設定（S8-037）で goroutine 滞留は防げるが、GC の挙動を理解しておく必要がある。特に：
- goroutine が参照するオブジェクトの GC 遅延
- 大量 JSON パース時のメモリプレッシャー
- `GOGC` / `GOMEMLIMIT` チューニング

### Console の DB アクセス一元化

現在 Console は supabase-js 経由で直接 DB にアクセス。Go Server に REST API を立てて一元化すれば：
- 認可ロジックが Go に集約
- Console は API クライアントのみ
- ただし Supabase 脱出後に検討（PostgREST が残るなら不要）

### ツール定義配信の自動化

現在のパイプライン: Server の Go コード変更 → `tools-export` で `tools.json` 生成 → Console にコミット → Vercel ビルド。ツール定義の SSoT は Server の Go コードだが、Console への配信が手動。

**Sprint 009 での検討結果:**
- Vercel prebuild で `go run` → Vercel ビルド環境に Go がない (`go: command not found`)
- Server に `/tools` エンドポイント追加 → Server の責務外 (MCP プロトコルに集中すべき)
- Supabase に定義を寄せる → DB スキーマ変更 + Console 全ページ非同期化が必要で過剰
- TS で Go ソースをパースするスクリプト → Go ソースのパーサーをメンテする二重負担

**現状維持の判断:** tools.json は git に残し、ツール定義変更時にローカルで `go run ./cmd/tools-export` を実行してコミット。根本的な解決には CI/CD パイプライン (GitHub Actions 等) でアプリ間の依存関係を管理する仕組みが必要。

**将来方針:** CI/CD 導入時に、Server のツール定義変更を検知 → tools.json 自動生成 → Console デプロイ、のパイプラインを構築する。

---

## 優先度：低

### SSE 改善

- ハートビート/ping-pong 追加
- メッセージバッファ溢れ時の対策（現在サイレントドロップ）
- セッション ID を暗号学的ランダムに変更（現在ポインタアドレス）

### Loki goroutine プール

現在 Loki push は goroutine を無制限に生成。Loki が落ちると goroutine が溜まる。プール + バックプレッシャーで制御。

---

## 参考

- [sprint008-plan.md](./sprint008-plan.md) - Sprint 008 計画（監査結果含む）
- [sprint007-review.md](./sprint007-review.md) - Sprint 007 レビュー
