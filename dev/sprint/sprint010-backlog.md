# Sprint 010 バックログ (実績ベース)

## 基本情報

| 項目 | 値 |
|------|-----|
| スプリント番号 | SPRINT-010 |
| 期間 | 2026-02-15 〜 2026-02-21 (7日間) |
| 作業日 | DAY030 〜 DAY034 (5 セッション) |
| コミット数 | 49 |

---

## Sprint 010 計画との対比

### Phase 1: ~~Cloudflare Workers OAuth 2.1 Server~~ (S10-001〜016) → 不要

**判断:** Clerk の認証基盤をそのまま使用することで、自前 OAuth 2.1 Server の構築は不要と判断。Clerk DCR (Dynamic Client Registration) の有効化のみで MCP クライアントの OAuth 接続に対応可能。

代わりに既存 OAuth 2.0 フローの改善 (Atlassian OAuth 2.0, Trello 修正, OAuth Apps UI) を実施。

### Phase 2: Neon PG 移行 (S10-020〜037)

| ID | タスク | 状態 |
|----|--------|------|
| S10-020 | Neon プロジェクト作成 | 未着手 (ローカル Docker PG で代替) |
| S10-021 | スキーマ移植 (DDL) | ✅ 完了 (7b65748: Supabase 除去 + database/migrations/ に移行) |
| S10-022 | RPC 関数移植 | ✅ 完了 (PostgREST RPC → GORM 直接クエリに全面移行) |
| S10-023 | pgsodium → pgcrypto 暗号化変更 | 未着手 (Supabase 除去で pgsodium 依存は消滅。平文のまま) |
| S10-024 | users テーブルの auth.users FK 除去 | ✅ 完了 (Supabase Auth 依存を完全除去) |
| S10-025〜027 | データ移行 (dump/re-encrypt/import) | 未着手 (本番 DB は既に Neon で稼働中) |
| S10-030〜034 | Go Server 接続先変更 | ✅ 完了 (PostgREST → GORM 直接 DB。broker/ 全面書き直し) |
| S10-035 | Console DB アクセスを Neon に切替 | ✅ 完了 (Supabase SDK → Worker REST API 経由) |
| S10-036 | admin.ts の service role 接続変更 | ✅ 完了 (Supabase admin → Clerk + Worker API) |
| S10-037 | Worker API Key 検証先変更 | ✅ 完了 (Ed25519 JWT ゲートウェイ認証に移行) |

**結果:** 計画とは異なるアプローチで同等以上の成果。PostgREST/Supabase を完全除去し、GORM 直接接続 + ogen 自動生成サーバーに移行。

### Phase 3: テスト基盤 (S10-040〜043)

| ID | タスク | 状態 |
|----|--------|------|
| S10-040 | authz middleware ユニットテスト | 未着手 |
| S10-041 | broker/user.go ユニットテスト | 未着手 |
| S10-042 | broker/retry.go ユニットテスト | 未着手 |
| S10-043 | CI トリガーを push/PR に変更 | 未着手 |

**結果:** 全未着手。

### Phase 4: 小タスク (S10-050)

| ID | タスク | 状態 |
|----|--------|------|
| S10-050 | 仕様書更新: credit → subscription model | 未着手 |

**結果:** 未着手。

---

## 計画外の実施内容

Sprint 010 では計画外の大規模アーキテクチャ改善が中心となった。

### DAY030 (2026-02-15): トランスポート分離 + RPC 設計統一

| コミット | 内容 |
|----------|------|
| 29f787a | refactor: extract SSE/Inline transport from handler into middleware |
| 20ed596 | chore: remove unused Dockerfiles and .dockerignore files |
| 1b0c50b, 6e87e26, 6f26d41 | refactor: decouple PostgREST RPC from Supabase-specific headers |
| 4dc3703 | fix: remove duplicate lines in user.go causing build failure |
| cd70a3c | refactor: consolidate RPC design and remove sync_modules |
| 6f89327 | fix: add apikey header to all PostgREST RPC calls |
| 8a77771 | refactor: rename UserStore to UserBroker |
| 9fd9d14 | feat: sync modules and tools to DB dynamically at startup |
| b4eed0b | feat: replace static tools.json with dynamic DB fetch |

### DAY031 (2026-02-16〜17): RESTful 移行 + Console 型安全化

| コミット | 内容 |
|----------|------|
| 708f7dd | docs: update architecture diagrams and work logs |
| 1adce8b | docs: add Console RPC migration plan (34 RPCs) |
| 7d20b49 | feat: add /rpc/* proxy route with Hono |
| cbf6c9e | refactor: route RPC calls through Worker, remove getUserId |
| 698c1e8 | refactor: reorganize lib/ by domain and remove Supabase from API routes |
| 6d19032 | fix: fix WORKER_URL fallback and admin RPC proxy bugs |
| b512d3c | feat: add /v1 route prefix, OpenAPI 3.1 spec |
| 8ce90cc | fix: update Console endpoints to /v1 |
| 765ce3c | feat: replace RPC proxy with RESTful routes |
| e23f66b | feat: introduce openapi-fetch typed client |
| 81d91d6 | feat: add user_id/email to get_user_context |
| 27ed869 | feat: migrate all RPCs to PostgREST with p_user_id |
| 5911965 | refactor: add cache in auth check |

### DAY032 (2026-02-19): PostgREST → GORM + Clerk 認証移行

| コミット | 内容 |
|----------|------|
| fd8c4ab | docs: add day031 backlog and day032 Stripe webhook migration plan |
| f0fa3fd | feat: migrate Stripe webhook from Console to Worker |
| 95ecf75 | feat: add gateway auth endpoints for Go Server PostgREST migration |
| c36695a | refactor: migrate from PostgREST/Supabase Auth to GORM + Clerk auth |
| 99fabda | fix: correct module identification (UUID→name), harden lifecycle |
| f1d1c7f | docs: worklog and backlog on day 32 |

### DAY033 (2026-02-20): Ed25519 認証 + Supabase 完全除去 + ogen 移行

| コミット | 内容 |
|----------|------|
| 9997295 | refactor: replace GATEWAY_SECRET with Ed25519 JWT gateway auth |
| a5e1bdd | fix: pass required query params to /v1/me/usage endpoint |
| 0aa3edc | fix: align Stripe customer linking with OpenAPI spec |
| 7b65748 | refactor: remove Supabase, migrate DB schema to database/ |
| a34d4e4 | chore: integrate local PostgreSQL via Docker |
| 66ec4ce | refactor: replace hand-written REST handlers with ogen-generated server |

### DAY034 (2026-02-21): spec 統一 + ツール設定修正 + OAuth 改善

| コミット | 内容 |
|----------|------|
| e84f67e | docs: add day033 worklog, day034 plan, security report |
| 8760c08 | fix: use find-then-update for UpsertOAuthApp |
| a5cd582 | fix: return per-tool rows from GetModuleConfig |
| 3931055 | fix: align UpsertToolSettingsBody field names with Worker spec |
| 8043210 | refactor: unify OpenAPI specs — Go Server as single source of truth |
| 5b0c65f | fix: add [build] command to wrangler.toml |
| bc5c485 | fix: restore tools field in /v1/modules response |
| a0f0377 | fix: unmarshal module tools via json.RawMessage |
| 2f5e4b9 | fix: force GORM to include enabled column in UPSERT |
| b6354ce | fix: use raw SQL for tool_settings UPSERT (GORM bool bug) |
| afc9b66 | refactor: remove Console-side default tool settings logic |
| 6e2b4c1 | refactor: unify MCP schema language to English (26 files) |
| 1217b73 | chore: simplify Worker config, remove env.dev section |
| f9f870e | fix: use OAuth 1.0a standard field names for Trello |
| c938136 | feat: redesign OAuth apps page as card grid, add Atlassian OAuth 2.0 |
| 495a643 | fix: switch Confluence OAuth scopes from Classic to Granular |
| fbaacfa | fix: separate Jira and Confluence OAuth flows |

---

## day032-backlog 消化状況

| ID | 項目 | 状態 |
|----|------|------|
| 1A | Clerk DCR 有効化 | 未着手 |
| 1B | OAuth App Credentials 再登録 | ✅ 完了 (OAuth Apps 管理ページ + admin API で再登録) |
| 1C | OAuth コールバック URL 本番設定 | ✅ 完了 (全プロバイダの callback URL を設定) |
| 2A | Render 環境変数 | ✅ 完了 |
| 2B | Workers シークレット | ✅ 完了 |
| 2C | Console 環境変数 | ✅ 完了 |
| 2D | Stripe Webhook 設定 | 一部完了 (Go Server 実装済み、Stripe Dashboard 未設定) |
| 3A | expires_at 形式統一 | 未修正 |
| 3B | Clerk セッション無限リダイレクト | ✅ 完了 (AuthProvider ref guard 修正) |
| 3C | Console 本番デプロイ設定 | ✅ 完了 (Vercel デプロイ稼働中) |
| 4A | INTERNAL_SECRET 廃止 | 未着手 |
| 4B | SECONDARY_API_URL 廃止 | 未着手 |
| 5A | ユーザークレデンシャル暗号化 | 未着手 |
| 5B | OAuth App Credentials 暗号化 | 未着手 |
| 5C | 暗号化キーのローテーション | 未着手 |
| 5D | 既存データの暗号化マイグレーション | 未着手 |
| 6A | Go Server Render デプロイ確認 | ✅ 完了 |
| 6B | Worker → Go Server 接続テスト | ✅ 完了 |
| 6C | .env.dev テンプレート更新 | ✅ 完了 (1217b73 で env.dev セクション削除) |
| 6D | Stripe Webhook Go Server 実装 | ✅ 完了 |
| 6E | seed.sql の OAuth Apps 初期データ | 未着手 |

## day034-issue_report (セキュリティ調査) 消化状況

| # | リスク | 優先度 | 状態 |
|---|--------|--------|------|
| 1 | API キー失効が実質効かない | Critical | 未修正 |
| 2 | OAuth state の真正性検証不足 | High | 未修正 |
| 3 | 資格情報・OAuth 秘密の平文保存 | High | 未修正 |
| 4 | トークン検証 API が未認証 + 任意先 fetch | Medium | 未修正 |
| 5 | `.env.dev` が Git 管理対象 | Medium | ✅ 完了 (1217b73 で env.dev セクション削除、ドメイン変更で秘密値をローテーション) |

## day031-backlog 消化状況

| # | タスク | 状態 |
|---|--------|------|
| 5 | Stripe Webhook の Worker 移行 | ✅ 完了 (f0fa3fd) |
| 6 | `get_user_context` の責務分離 | ✅ 解消 (PostgREST RPC 全廃で不要に。GORM で個別クエリに分離済み) |
| 7 | OAuth authorize ルートの Supabase 依存 | ✅ 完了 (Supabase 完全除去。Clerk auth に移行) |
| 8 | OAuth 2.1 Server | 不要 (Clerk 認証をそのまま使用) |
| 9 | PostgREST 廃止 + DB 移行 | ✅ PostgREST 廃止完了 (GORM 移行)。Drizzle 案は不採用、Go Server 直接 DB に変更 |
| 10 | テスト基盤 | 未着手 |
| 11 | 旧 `worker-client.ts` の完全除去 | ✅ 完了 (Supabase 除去時に削除) |
| 12 | OpenAPI spec 同期 | ✅ 解消 (ogen 自動生成で spec とコードが一致) |
| 13 | Go Server 型安全性 | ✅ 解消 (ogen 生成コードで型安全) |
| 14 | 仕様書の残課題 | 未着手 |

---

## Sprint 010 サマリ

| 項目 | 内容 |
|------|------|
| 計画達成率 (Phase別) | Phase 1: 不要 (Clerk 採用で解消), Phase 2: 70%, Phase 3: 0%, Phase 4: 0% |
| 計画外作業の割合 | 約 80% (アーキテクチャ改善が中心) |
| コミット数 | 49 |
| 主要成果 | Supabase 完全除去、PostgREST → GORM 移行、ogen 自動生成サーバー、Ed25519 JWT ゲートウェイ認証、Worker RESTful 化、Console openapi-fetch 型安全化、OpenAPI spec 統一、MCP スキーマ英語統一、OAuth Apps カード UI、Atlassian OAuth 2.0、Confluence Granular スコープ、全サービス動作確認 |

---

## Sprint 011 繰越し候補

### 優先度: Critical / High

| # | タスク | 由来 |
|---|--------|------|
| 1 | API キー失効の実効性確保 (JWT key_id + DB 照合) | day034-issue #1 |
| 2 | OAuth state の真正性検証 (HMAC 署名) | day034-issue #2 |
| 3 | 資格情報の暗号化保存 (AES-256-GCM) | day034-issue #3, day032-backlog 5A-5D |
| 4 | Stripe Webhook Stripe Dashboard 設定 | day032-backlog 2D |

### 優先度: Medium

| # | タスク | 由来 |
|---|--------|------|
| 5 | トークン検証 API の認証必須化 | day034-issue #4 |
| 6 | expires_at 形式統一 | day032-backlog 3A |
| 7 | INTERNAL_SECRET 廃止 | day032-backlog 4A |
| 8 | SECONDARY_API_URL 廃止 | day032-backlog 4B |
| 9 | テスト基盤 (authz, broker ユニットテスト + CI) | sprint010 Phase 3 |
| 10 | Clerk DCR 有効化 (MCP クライアント OAuth 接続) | day032-backlog 1A |

### 優先度: Low

| # | タスク | 由来 |
|---|--------|------|
| 11 | seed.sql の OAuth Apps 初期データ | day032-backlog 6E |
| 13 | 仕様書更新 (credit → subscription model 等) | sprint010 Phase 4, day031-backlog #14 |
