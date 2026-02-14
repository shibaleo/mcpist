# Sprint 009 バックログ

Sprint 006〜009 の残課題を統合・棚卸し。完了・廃止済みは除外。

---

## 優先度：高

### 認証基盤の整理

認証（ユーザーは誰か）と認可（LLM に何を許可するか）は別のフロー。

#### 認証（OIDC）— Console ログイン

| 項目 | 現状 | 方針 |
|------|------|------|
| Console ログイン | Supabase Auth | **当面そのまま**。問題は起きていない |
| 将来の移行先 | — | Neon Auth (Better Auth ベース、OIDC 準拠) |

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

### DB・インフラ移行

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

### CI/CD 整備

| タスク | 備考 | 由来 |
|--------|------|------|
| CI トリガーを push/PR に変更 | 現在 workflow_dispatch (手動)。自動化すべき | S008 |
| tools.json 自動生成パイプライン | Server のツール定義変更を検知 → tools.json 自動生成 → Console デプロイ | S006〜S009 |
| Console lint + build CI の安定化 | ESLint + pnpm build は S008 で構築済み | S008 |

**補足 (Sprint 009 でのツール定義配信の検討結果):**
- Vercel prebuild で `go run` → Vercel ビルド環境に Go がない
- Server に `/tools` エンドポイント追加 → Server の責務外
- Supabase に定義を寄せる → DB スキーマ変更 + Console 全ページ非同期化が過剰
- TS で Go ソースをパース → Go パーサーの二重負担

**現状:** tools.json は git に残し、手動で `go run ./cmd/tools-export` を実行してコミット。根本的な解決には CI/CD パイプラインでアプリ間の依存関係を管理する仕組みが必要。

---

## 優先度：中

### 堅牢性改善（残り）

| 項目 | 現状 | 対策 |
|------|------|------|
| Rate Limiter | インスタンス独立 | 分散レートリミット（KV or PG ベース）。マルチインスタンス運用開始時に対応 |

> リトライ・バックオフ、フォールバックキャッシュ、セキュリティヘッダーは Sprint 009 で完了。
> OAuth2 x/oauth2 移行は見送り（Notion JSON body / Microsoft Todo extra scope 等、画一性に合わない。テーブル駆動で十分）。

### Stripe ogen 化

Stripe は OpenAPI spec を公開。ogen で型安全クライアントを生成し、他モジュールと同じ 3 層パターンに乗せる。

### Console の DB アクセス一元化

現在 Console は supabase-js 経由で直接 DB にアクセス。Go Server に REST API を立てて一元化すれば：
- 認可ロジックが Go に集約
- Console は API クライアントのみ
- ただし Supabase 脱出後に検討（PostgREST が残るなら不要）

### Go GC 理解

ogen クライアントのタイムアウト設定（S8-037 完了）で goroutine 滞留は防げるが、GC の挙動を理解しておく：
- goroutine が参照するオブジェクトの GC 遅延
- 大量 JSON パース時のメモリプレッシャー
- `GOGC` / `GOMEMLIMIT` チューニング

### 仕様書の残課題

| タスク | 備考 | 由来 |
|--------|------|------|
| credentials JSON 構造の整理 | ogen + broker 化を反映 | S006 (BL-090) |
| credit model → subscription model に更新 | credits テーブル廃止・plans テーブル移行を反映 | S007 |

> Sprint 007 の大規模リファクタで仕様と実装の整合はほぼ取れている (S008 Phase 1c で確認)。上記 2 点のみ残存。

---

## 優先度：低

### SSE 改善

- ハートビート/ping-pong 追加
- メッセージバッファ溢れ時の対策（現在サイレントドロップ）

> セッション ID の暗号学的ランダム化は Sprint 009 (S9-050) で完了。

### Loki goroutine プール

現在 Loki push は goroutine を無制限に生成。Loki が落ちると goroutine が溜まる。プール + バックプレッシャーで制御。

### テスト基盤

| タスク | 備考 | 由来 |
|--------|------|------|
| E2E テスト設計 | OAuth 認可フロー等 | S006 (S6-020) |
| Go Server ユニットテスト拡充 | CI は S008 で構築済み。テストカバレッジ向上 | S006 (S6-021) |

### UI/UX

| タスク | 備考 | 由来 |
|--------|------|------|
| ブランディング・ロゴ作成 | | S006 (BL-083) |
| ソーシャルログイン拡充 | GitHub, Apple など | S006 (BL-084) |

### 将来検討

| タスク | 備考 | 由来 |
|--------|------|------|
| Stg/Prd 環境構築 | Blue-Green 方式 | S006 (BL-030) |
| 追加モジュール（Slack/Linear） | 需要に応じて | S006 (BL-032) |
| RFC 8707 Resource Indicators 対応 | OAuth 2.0 拡張 | S006 (BL-060) |

---

## 廃止・完了タスク（Sprint 009 時点）

以下は Sprint 006〜009 で完了または廃止済み。バックログから除外。

### Sprint 009 で完了

- サブスクリプションモデル移行 (Phase 1: DB + Server + Console)
- リトライ・指数バックオフ + ジッター (S9-030)
- フォールバックキャッシュ (S9-031)
- セキュリティヘッダー追加 (S9-032)
- セッション ID 暗号学的ランダム化 (S9-050)

### Sprint 009 で見送り・方針決定

- OAuth2 x/oauth2 移行 → **見送り** (テーブル駆動で十分)
- tools.json 動的配信 (Phase 3) → **見送り** (CI/CD パイプラインで解決すべき)

### Sprint 008 以前で完了

- 設計書削減 (15 ファイル削除、dsn-observability.md -440 行)
- CI/CD 基盤構築 (Go lint + test, Console lint + build)
- Grafana アラートルール 3 本 + 通知設定
- panic recovery ミドルウェア、グレースフルシャットダウン、ogen タイムアウト
- usage_stats / enabled_modules 参照 (Console が Supabase RPC 直接呼出)
- resources MCP 実装 → **廃止** (CORE-005〜009)

---

## 参考

- [sprint009-plan.md](./sprint009-plan.md) - Sprint 009 計画
- [sprint008-backlog.md](./sprint008-backlog.md) - Sprint 008 バックログ
- [sprint007-backlog.md](./sprint007-backlog.md) - Sprint 007 バックログ
- [sprint006-backlog.md](./sprint006-backlog.md) - Sprint 006 バックログ
