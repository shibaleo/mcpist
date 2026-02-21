# Sprint 010 レビュー

## 基本情報

| 項目 | 値 |
|------|-----|
| スプリント番号 | SPRINT-010 |
| 期間 | 2026-02-15 〜 2026-02-21 (7日間) |
| 作業日 | DAY030 〜 DAY034 (5 セッション) |
| コミット数 | 49 |
| 前スプリント | Sprint 009 (サブスク移行、堅牢性改善) |

---

## 計画 vs 実績

### 計画の目標

> Supabase 依存を解消する。認可を自前で制御し、DB を Neon に移す

### 実際の成果

> Supabase を完全除去。PostgREST → GORM 直接 DB、ogen 自動生成サーバー、Ed25519 JWT 認証に移行。全サービスの OAuth 接続を刷新し動作確認完了。

| Phase | 計画 | 実績 |
|-------|------|------|
| Phase 1: OAuth 2.1 Server | 自前 OAuth 2.1 Server 構築 | **不要と判断** — Clerk 認証をそのまま使用 |
| Phase 2: Neon PG 移行 | Supabase → Neon | **70% 完了** — Supabase 完全除去、GORM 移行。本番は既に Neon で稼働 |
| Phase 3: テスト基盤 | ユニットテスト + CI | 未着手 |
| Phase 4: 小タスク | 仕様書更新 | 未着手 |

---

## 主要成果

### 1. アーキテクチャ刷新

| 変更 | Before | After |
|------|--------|-------|
| DB アクセス | PostgREST (HTTP RPC) | GORM 直接 DB 接続 |
| API サーバー | 手書き REST ハンドラー | ogen 自動生成 (28 EP) |
| 認証基盤 | Supabase Auth | Clerk |
| ゲートウェイ認証 | 共有シークレット (`GATEWAY_SECRET`) | Ed25519 JWT (30秒 TTL) |
| Console → Worker | 手動型定義 + `as unknown as` | openapi-fetch 型安全クライアント |
| OpenAPI spec | Worker spec / Go Server spec 二重管理 | Go Server spec を single source of truth |
| MCP スキーマ言語 | ユーザー言語設定に応じた日英切替 | 英語統一 (LLM 向け) |

### 2. Supabase 完全除去

- Supabase SDK、Auth、PostgREST、36 migration ファイルを全削除
- `-10,700 行`の負債を除去
- ローカル開発は Docker PostgreSQL で完結 (`pnpm db:up`)

### 3. OAuth 2.0 改善

- **Atlassian OAuth 2.0**: Jira + Confluence の OAuth ログイン対応
- **Confluence Granular スコープ**: Classic スコープ廃止に対応
- **OAuth フロー分離**: Jira 接続時に Confluence が自動接続される問題を修正
- **Trello OAuth 1.0a**: フィールド名不整合を修正
- **OAuth Apps 管理 UI**: カードグリッドレイアウトに再設計
- **全 8 サービスの動作確認完了**: Jira, Confluence, Google Calendar/Drive/Docs/Sheets/Tasks/Apps Script

### 4. バグ修正

- ツール設定トグルが保存されない問題 (GORM bool ゼロ値バグ → raw SQL)
- `/v1/modules` レスポンスに tools が含まれない問題 (jx.Raw → json.RawMessage)
- Admin サイドバーのハイライト二重表示
- UpsertOAuthApp の duplicate key violation
- Stripe customer linking の OpenAPI spec 不整合

---

## 設計判断の変更

### OAuth 2.1 Server → Clerk 認証をそのまま使用

**計画時の前提:** Supabase Auth の OAuth フローがブラックボックスで、LLM 認可フロー失敗時にログが出ない → 自前 OAuth 2.1 Server を構築する。

**変更理由:** Clerk に移行したことで、認証基盤の可観測性が確保された。Clerk は Dynamic Client Registration (DCR) をサポートしており、MCP クライアントの OAuth 接続にも対応可能。自前 OAuth 2.1 Server の構築は不要。

### PostgREST + Drizzle → GORM 直接 DB

**計画時の前提:** Worker に Drizzle ORM を導入し、PostgREST を廃止する。Go Server は Worker REST API 経由で DB アクセス。

**変更理由:** Go Server に GORM を導入し直接 DB 接続する方がシンプル。Worker を経由するレイテンシも排除できる。Worker は API ゲートウェイ + MCP トランスポート変換に専念。

### Worker 廃止計画 → Worker 継続

**計画時の検討:** Worker の機能を Go Server に集約し、Worker を廃止する案を策定 (worker_to_go_plan.md)。

**変更理由:** Worker は Cloudflare のエッジで動作し、MCP SSE/Streamable HTTP トランスポート変換とゲートウェイ認証に適している。Go Server に集約するメリットがコストに見合わない。Worker は継続使用。

---

## 数値サマリ

| 指標 | 値 |
|------|-----|
| コミット数 | 49 |
| 削除行数 | ~12,000 (Supabase + 旧コード) |
| 新規行数 | ~6,000 (ogen 生成 + spec) |
| 変更ファイル数 | 90+ |
| 新規パッケージ | `internal/ogenserver`, `internal/db`, `auth/gateway.go` |
| 削除パッケージ | `internal/rest`, `cmd/tools-export`, Supabase 関連 |

---

## 未完了・繰越し

### Critical / High

| # | タスク | 備考 |
|---|--------|------|
| 1 | API キー失効の実効性確保 | JWT key_id + DB 照合。Critical セキュリティ |
| 2 | OAuth state の真正性検証 | HMAC 署名。High セキュリティ |
| 3 | 資格情報の暗号化保存 | AES-256-GCM。High セキュリティ |
| 4 | Stripe Webhook Dashboard 設定 | Go Server 実装済み、Stripe 側未設定 |

### Medium

| # | タスク | 備考 |
|---|--------|------|
| 5 | トークン検証 API の認証必須化 | SSRF リスク |
| 6 | expires_at 形式統一 | OAuth callback 間の不整合 |
| 7 | INTERNAL_SECRET 廃止 | Clerk JWT に移行済みで不要の可能性 |
| 8 | SECONDARY_API_URL 廃止 | 未使用 |
| 9 | テスト基盤 | authz, broker ユニットテスト + CI |
| 10 | Clerk DCR 有効化 | MCP クライアント OAuth 接続 |

### Low

| # | タスク | 備考 |
|---|--------|------|
| 11 | seed.sql の OAuth Apps 初期データ | dev 環境の初期化効率化 |
| 12 | 仕様書更新 | credit → subscription model 等 |

---

## 振り返り

### Good

- Supabase 依存を完全に断ち切れた (計画の本質的な目標は達成)
- ogen 自動生成でハンドラーの型安全性が大幅に向上
- Ed25519 JWT ゲートウェイ認証で Worker → Go Server 間のセキュリティが改善
- 全サービス (8 モジュール) の OAuth 動作確認が完了し、プロダクトとして利用可能な状態

### Improve

- セキュリティ調査で検出した Critical/High 問題が 3 件未修正のまま残っている
- テスト基盤 (Phase 3) に全く着手できなかった
- 計画外の作業が 80% を占め、スプリント計画の精度が低かった

### Next

- Sprint 011 はセキュリティ修正 (API キー失効、OAuth state、暗号化) を最優先とする
- テスト基盤の構築も並行して進める
- スプリント計画時に、アーキテクチャ変更の波及効果をより精密に見積もる
