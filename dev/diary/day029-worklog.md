# DAY029 作業ログ

## 日付

2026-02-14

---

## コミット一覧 (22件)

| # | ハッシュ | 時刻 | メッセージ |
|---|---------|------|-----------|
| 1 | 47f122e | 00:43 | refactor(server): separate compact formatting from Run() and fix batch variable resolution |
| 2 | b7cda42 | 00:45 | refactor(server): separate compact formatting from Run() and remove dead code |
| 3 | dea88d1 | 00:59 | docs: close Sprint 007 and plan Sprint 008 with doc reduction focus |
| 4 | 315aa10 | 09:35 | docs: remove 65 redundant/empty files and simplify remaining specs (-6800 lines) |
| 5 | 3158426 | 11:34 | fix(server): use dedicated JSON-RPC error codes for authz and credit errors |
| 6 | 64edac8 | 12:00 | harden(server,worker): add panic recovery, graceful shutdown, tool timeout, and security headers |
| 7 | 40a9c3f | 12:09 | fix(ci): fix all failing CI jobs and remove unused deploy workflow |
| 8 | e5bf68f | 13:52 | fix(ci): upgrade Next.js to v16 with ESLint CLI migration and add golangci-lint config |
| 9 | 4bf7458 | 14:01 | fix(ci): fix golangci-lint errcheck exclusions with pointer receiver syntax |
| 10 | 8a03e87 | 14:22 | feat(server): add observability improvements and disable CI auto-triggers |
| 11 | a4b39b6 | 14:23 | docs: update Sprint 008 plan with audit results and sync specs to implementation |
| 12 | 393fa5f | 14:54 | feat(server): add Grafana contact point and notification policy tools |
| 13 | 464c7da | 15:01 | chore(console): regenerate tools.json with Grafana contact point and notification policy tools |
| 14 | bd99e29 | 19:16 | feat(server,db): migrate from credit billing to subscription-based daily usage limits |
| 15 | e2dc8e6 | 19:57 | feat(console): migrate from credit billing to subscription-based plans |
| 16 | 85df830 | 20:07 | fix(console): improve Stripe subscription webhook reliability |
| 17 | 9791738 | 20:16 | fix(console): remove deprecated STRIPE_FREE_CREDIT_PRICE_ID fallback |
| 18 | 87577f0 | 20:20 | feat(console): add Stripe Customer Portal for subscription management |
| 19 | dc66be5 | 20:26 | fix(console): fix portal RPC return type causing 404 |
| 20 | 212260a | 20:37 | feat(console): show external API rate limits on services page |
| 21 | de57cdf | 20:50 | feat: add retry with backoff, stale cache fallback, and security headers |
| 22 | 15d603e | 21:22 | fix(server): use cryptographic random for SSE session IDs |

> `88044c0` (prebuild 追加) → `3104773` (revert) は一連の修正で最終的に revert。コミット一覧からは省略しない。

---

## Sprint 008 完了

### Phase 1: 設計書の棚卸しと削減 ✅

| ID | タスク | 状態 | 備考 |
|----|--------|------|------|
| S8-001〜015 | 空テンプレート 15 ファイル削除 | ✅ | dsn-*.md, adr-*.md |
| S8-016〜018 | 設計書統合・簡略化 | ✅ | dsn-module-registry.md 削除、dsn-modules.md 287→66行 |
| S8-019 | canvas 棚卸し | ✅ | 9 中 2 ファイル削除 |
| S8-020〜025 | 仕様書の最小限更新 | ✅ | 全て「対応不要」または「既に反映済み」 |
| 計画外 | interaction/ 42 ファイル + details/ 4 ファイル削除 | ✅ | spc-itf.md が SSoT |

**docs/: 108 → 52 ファイル (-52%)、-6,800 行**

### Phase 2: セキュリティ・堅牢化 ✅

| ID | タスク | 状態 | 備考 |
|----|--------|------|------|
| S8-030 | panic recovery ミドルウェア | ✅ | Sprint 007 末で実装済みと判明 |
| S8-032 | CORS 制限検討 | ✅ | `*` 維持を意図的設計と確認 |
| S8-036 | グレースフルシャットダウン | ✅ | Sprint 007 末で実装済みと判明 |
| S8-037 | ogen HTTP タイムアウト | ✅ | Sprint 007 末で実装済みと判明 |

> S8-031 (セキュリティヘッダー) は Sprint 009 で完了。

### Phase 3: CI/CD 基盤 ✅

| ID | タスク | 状態 | 備考 |
|----|--------|------|------|
| S8-033 | Go lint + build + test CI | ✅ | golangci-lint + go build + go test -race |
| S8-035 | Console lint + build CI | ✅ | Next.js v16 + ESLint CLI 移行 |

CI は `workflow_dispatch` トリガー。6 ジョブ全 pass。

### Phase 4: Observability 仕上げ ✅

| ID | タスク | 状態 | 備考 |
|----|--------|------|------|
| S8-040 | ツールログに level フィールド追加 | ✅ | info/error を level ラベルで付与 |
| S8-041 | アクセス拒否・クレジット消費の監査ログ | ✅ | LogSecurityEvent で Loki 送信 |
| S8-042 | /health に DB 接続チェック追加 | ✅ | Supabase HEAD → 503 返却 |
| S8-043 | Grafana アラートルール設定 | ✅ | 3 ルール (Error Rate, Security Events, Log Silence) |

### Phase 5: 機能実装 ✅

| ID | タスク | 状態 | 備考 |
|----|--------|------|------|
| S8-050 | usage_stats 参照 | ✅ | Console が Supabase RPC 直接呼出で対応済み |
| S8-051 | enabled_modules 参照 | ✅ | 同上 |

### 計画外成果

- Grafana Contact Point / Notification Policy ツール 6 個追加 (ogen 再生成)
- 実際に mcpist-email Contact Point + Notification Policy を MCP ツール経由で構築
- dsn-observability.md: 584 → 144 行 (-75%)
- JSON-RPC カスタムエラーコード (-32001 PermissionDenied, -32002 InsufficientCredit)

---

## Sprint 009 完了

### Phase 1: サブスクリプション移行 ✅

#### 1a. DB スキーマ + RPC

| ID | タスク | 状態 |
|----|--------|------|
| S9-001 | plans テーブル作成 | ✅ |
| S9-002 | user_profiles にプラン関連カラム追加 | ✅ |
| S9-003 | record_usage RPC | ✅ |
| S9-004 | check_usage_limit RPC | ✅ |
| S9-005 | consume_user_credits RPC 廃止 | ✅ |
| S9-006 | get_user_context RPC 更新 | ✅ |

#### 1b. Server (Go)

| ID | タスク | 状態 |
|----|--------|------|
| S9-010 | UserContext: credit → plan + usage | ✅ |
| S9-011 | AuthContext: credit → plan + usage | ✅ |
| S9-012 | CanAccessTool: 残高 → 利用上限チェック | ✅ |
| S9-013 | handler run: ConsumeCredit → RecordUsage (非同期) | ✅ |
| S9-014 | handler batch: ConsumeCredit → RecordUsage (非同期) | ✅ |
| S9-015 | checkBatchPermissions: credit → usage limit | ✅ |
| S9-016 | ErrInsufficientCredit → ErrUsageLimitExceeded | ✅ |
| S9-017 | ConsumeCredit 関数削除 | ✅ |
| S9-018 | RecordUsage 関数追加 | ✅ |
| S9-019 | BatchResult 簡素化 | ✅ |

#### 1c. Console (Next.js)

| ID | タスク | 状態 |
|----|--------|------|
| S9-020 | credits → plan ページ書き換え | ✅ |
| S9-021 | dashboard: クレジット → プラン + 利用量カード | ✅ |
| S9-022 | sidebar: "クレジット" → "プラン" | ✅ |
| S9-023 | credits.ts → plan.ts | ✅ |
| S9-024 | Stripe checkout → Subscription | ✅ |
| S9-025 | Stripe webhook: invoice.paid 対応 | ✅ |
| S9-026 | サインアップ → Free プラン自動付与 | ✅ |
| S9-027 | stripe.ts: price ID 更新 | ✅ |
| S9-028 | サービスページにレート制限情報表示 | ✅ |

追加: Stripe Customer Portal 導入、Webhook 信頼性改善、portal RPC 型修正

### Phase 2: 堅牢性改善 ✅

| ID | タスク | 状態 | 備考 |
|----|--------|------|------|
| S9-030 | リトライ (指数バックオフ + ジッター) | ✅ | broker/ の DB 呼び出し全般 |
| S9-031 | 障害時キャッシュ延長 | ✅ | フォールバックキャッシュ |
| S9-032 | セキュリティヘッダー (Worker) | ✅ | CSP, HSTS, X-Content-Type-Options |
| S9-033 | OAuth2 x/oauth2 移行 | **見送り** | テーブル駆動で十分。画一性に合わない |

### Phase 3: ツール定義の動的配信 — 見送り

4 つのアプローチを検討し、いずれも不適切と判断:
- Vercel prebuild `go run` → Go が Vercel にない
- Server `/tools` エンドポイント → Server の責務外
- Supabase に定義 → DB スキーマ変更 + 非同期化が過剰
- TS で Go ソースをパース → 二重負担

**結論:** 手動 `go run ./cmd/tools-export` を維持。CI/CD パイプラインで根本解決。

### Phase 4: 繰越し・小タスク ✅

| ID | タスク | 状態 |
|----|--------|------|
| S9-050 | セッション ID を暗号学的ランダムに変更 | ✅ |

`fmt.Sprintf("%p", r)` → `crypto/rand` 16 bytes → hex (32 文字)

---

## tools-export リファクタ

| 変更 | 内容 |
|------|------|
| `main.go` | `generateToolsJSON()` を `exportTools()` から抽出 |
| `package.json` | prebuild 追加 → revert (Go が Vercel にない) |

---

## バックログ棚卸し

Sprint 006〜009 の残課題を統合し `sprint009-backlog.md` を作成。

| 優先度 | 件数 | 主な項目 |
|--------|------|----------|
| 高 | 3 | 認証基盤整理、DB 移行、CI/CD 整備 |
| 中 | 5 | 分散 Rate Limiter、Stripe ogen 化、Console DB 一元化、Go GC、仕様書残 2 点 |
| 低 | 5 | SSE 改善、Loki プール、テスト基盤、UI/UX、将来検討 |

---

## DAY029 サマリ

| 項目 | 内容 |
|------|------|
| テーマ | Sprint 008 完走 + Sprint 009 完走 (1 日で 2 スプリント) |
| コミット数 | 22 |
| Sprint 008 | 37/39 タスク完了 (95%) |
| Sprint 009 | Phase 1-2, 4 完了。Phase 3 見送り |
| 主な成果 | サブスク移行、堅牢性改善、設計書 -65 ファイル、Grafana +6 ツール、CI 6 ジョブ |
| バックログ | sprint009-backlog.md に全残課題を統合 |

---

## 未コミット

| ファイル | 内容 |
|----------|------|
| sprint008-plan.md | 監査結果の追記 |
| sprint008-backlog.md | Sprint 009 完了ステータス反映 |
| sprint009-backlog.md | 新規作成（バックログ統合） |
