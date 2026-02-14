# Sprint 009 レビュー

## 基本情報

| 項目 | 値 |
|------|-----|
| スプリント番号 | SPRINT-009 |
| 計画期間 | 2026-02-15 〜 2026-02-21 (7日間) |
| 実績期間 | 2026-02-14 (1日間、Sprint 008 と同日) |
| マイルストーン | M8: サブスク移行・堅牢性・ツール動的配信 |
| 状態 | **完了** |

---

## 計画 vs 実績サマリ

| 項目 | 計画 | 実績 | 達成度 |
|------|------|------|--------|
| Phase 1: サブスクリプション移行 | 28タスク | 28タスク完了 + 追加3件 | ✅ 100% |
| Phase 2: 堅牢性改善 | 4タスク | 3タスク完了, 1見送り | ⚠️ 75% (意図的見送り) |
| Phase 3: ツール定義の動的配信 | 4タスク | 0タスク (見送り) | ❌ 0% (意図的見送り) |
| Phase 4: 繰越し・小タスク | 1タスク | 1タスク完了 | ✅ 100% |

**全体達成度: 32/37 タスク完了 (86%)。未達 5 件は全て意図的な見送り・方針転換。**

---

## Sprint 008 → 009 差分

| 項目 | S008 終了時 | S009 終了時 | 差分 |
|------|-----------|-----------|------|
| 課金モデル | クレジット従量課金 | **サブスクリプション + 日次上限** | 全面移行 |
| ConsumeCredit RPC | 毎回呼出 | **削除** | -1 RPC/リクエスト |
| RecordUsage | なし | **非同期 fire-and-forget** | 障害耐性向上 |
| リトライ | なし | **指数バックオフ + ジッター** | 全 Supabase 呼出 |
| キャッシュ | 30s 固定 TTL | **障害時スティール延長** | 可用性向上 |
| セキュリティヘッダー | なし | **CSP, HSTS, X-Content-Type-Options** | Worker |
| SSE セッション ID | `fmt.Sprintf("%p", r)` | **crypto/rand 32文字** | 推測不能 |
| Stripe 課金 | 1回払い | **Subscription + Customer Portal** | 定期課金対応 |

---

## Phase 別詳細

### Phase 1: サブスクリプション移行 ✅ 完了

#### 1a. DB スキーマ + RPC (S9-001〜006)

- `plans` テーブル作成 (free/plus)
- `user_profiles` にプラン関連カラム追加
- `record_usage` / `check_usage_limit` RPC 追加
- `consume_user_credits` RPC 廃止
- `get_user_context` RPC をプラン形式に更新

#### 1b. Server (S9-010〜019)

- `UserContext`: credit フィールド → plan + usage フィールド
- `AuthContext`: `TotalCredits()` → `WithinDailyLimit()`
- `CanAccessTool`: 残高チェック → 利用上限チェック
- `handleRun` / `handleBatch`: `ConsumeCredit` 削除 → `RecordUsage` (非同期 goroutine)
- `ErrInsufficientCredit` → `ErrUsageLimitExceeded` (コード -32002 維持)
- `ConsumeCredit` / `ConsumeResult` 型を完全削除

#### 1c. Console (S9-020〜028 + 追加3件)

- credits ページ → plans ページに全面書き換え (利用量バー表示)
- dashboard: クレジットカード → プラン + 日次利用量カード
- sidebar: "クレジット" → "プラン"
- Stripe: `mode: "payment"` → `mode: "subscription"`
- Stripe webhook: `checkout.session.completed` → `invoice.paid` 対応
- サインアップボーナス → Free プラン自動付与
- サービスページにレート制限情報表示

**追加実装:**
- Stripe Customer Portal (サブスク管理画面) 導入
- Stripe webhook 信頼性改善 (べき等性)
- portal RPC 戻り型修正

### Phase 2: 堅牢性改善 ⚠️ 意図的見送り1件

| ID | タスク | 状態 | 備考 |
|----|--------|------|------|
| S9-030 | リトライ (指数バックオフ + ジッター) | ✅ | `broker/retry.go` 新設。全 Supabase 呼出に適用 |
| S9-031 | 障害時キャッシュ延長 | ✅ | スティールキャッシュ方式 |
| S9-032 | セキュリティヘッダー (Worker) | ✅ | `apps/worker/src/index.ts` |
| S9-033 | OAuth2 x/oauth2 移行 | **見送り** | 下記参照 |

**S9-033 見送り理由:**
- Notion: トークンリフレッシュが JSON body (x/oauth2 は form-urlencoded 前提)
- Microsoft Todo: `scope` パラメータに追加値が必要
- テーブル駆動 `OAuthRefreshConfig` (Sprint 007 で構築) が 6 プロバイダ 11 モジュールに対応済み
- 画一的な x/oauth2 に寄せるメリットがない

### Phase 3: ツール定義の動的配信 ❌ 見送り

4 つのアプローチを検討し、いずれも現時点では不適切と判断:

| アプローチ | 問題 |
|-----------|------|
| Vercel prebuild で `go run` | Vercel ビルド環境に Go がない (`go: command not found`) |
| Server に `/tools` エンドポイント | Server の責務外 (MCP プロトコルに集中すべき) |
| Supabase に定義を格納 | DB スキーマ変更 + Console 全ページ非同期化が過剰 |
| TS で Go ソースをパース | Go パーサーのメンテという二重負担 |

**結論:** tools.json は git に残し、手動 `go run ./cmd/tools-export` を維持。根本的な解決は CI/CD パイプライン整備時に行う。`main.go` の `generateToolsJSON()` 抽出リファクタのみ実施。

### Phase 4: 繰越し・小タスク ✅ 完了

| ID | タスク | 状態 | 備考 |
|----|--------|------|------|
| S9-050 | セッション ID を暗号学的ランダムに変更 | ✅ | `crypto/rand` 16 bytes → hex 32 文字 |

---

## 完了条件の検証

| 条件 | 状態 | 備考 |
|------|------|------|
| クレジット残高の概念が Server から消えている | ✅ | ConsumeCredit, TotalCredits 削除済み |
| plans テーブルにプラン定義がある | ✅ | free/plus |
| ツール実行時に日次利用上限でブロックされる | ✅ | WithinDailyLimit() |
| 利用量が非同期で記録される | ✅ | RecordUsage goroutine |
| Console にプランページがあり利用量が表示される | ✅ | plans/page.tsx |
| Supabase RPC に指数バックオフリトライ | ✅ | broker/retry.go |
| Worker にセキュリティヘッダー | ✅ | CSP, HSTS 等 |
| Console サービスページにレート制限情報 | ✅ | services/page.tsx |
| Console がツール定義を Server API から動的取得 | ❌ | 見送り (tools.json 維持) |
| tools.json と tools-export が削除されている | ❌ | 見送り (CI/CD で解決予定) |

**8/10 達成。未達 2 件は Phase 3 見送りに起因。**

---

## 数値サマリ

| 項目 | 値 |
|------|-----|
| Sprint 009 コミット数 | 12 (Sprint 008 と合わせて当日 24) |
| 変更ファイル数 | 156 (Sprint 008 含む) |
| 挿入行数 | +6,219 |
| 削除行数 | -8,446 |
| DB マイグレーション | 1 (508 行) |
| 新規 Console ページ | 1 (plans/page.tsx) |
| 削除 Console ページ | 1 (credits/page.tsx) |
| 新規 Server ファイル | 2 (broker/retry.go, middleware/recovery.go) |

---

## 振り返り

### 良かった点

1. **課金モデル全面移行を 1 日で完了**: DB → Server → Console → Stripe の 4 層を一貫して移行。クレジットの概念を完全に排除
2. **堅牢性の実質的向上**: リトライ・キャッシュ延長・セキュリティヘッダーで外部依存障害への耐性が大幅改善
3. **見送り判断が的確**: x/oauth2 移行と tools.json 動的配信は、調査の上で「現時点では不要」と判断。無駄な実装を回避
4. **Sprint 008 + 009 を同日完走**: 設計書削減 → CI 構築 → サブスク移行 → 堅牢性まで一気通貫

### 改善点

1. **Phase 3 の計画精度**: 4 タスク分の計画が全て見送りになった。事前調査を Sprint 内で行う形にすべきだった
2. **Vercel prebuild の試行錯誤**: Go が Vercel にない件は事前確認で防げた (commit → revert)

### 次 Sprint への教訓

1. **CI/CD パイプラインが次の最優先**: 手動デプロイ・手動 tools.json 更新の問題は Sprint 009 で顕在化した。CI トリガーの自動化と tools.json パイプラインをセットで解決すべき
2. **認証基盤の整理が迫っている**: サブスク移行で課金は整理された。次は認可 (OAuth Server) の課題に取り組む時期
3. **テスト基盤の拡充**: 今回のような大規模移行をテストなしで行っている。E2E テスト設計を先送りし続けるリスクが増大

---

## 繰越し (次 Sprint へ)

→ 詳細は [sprint009-backlog.md](./sprint009-backlog.md) を参照

| 優先度 | 項目 |
|--------|------|
| 高 | 認証基盤整理 (OAuth Server 自作) |
| 高 | DB・インフラ移行 (Supabase → Neon) |
| 高 | CI/CD 整備 (自動トリガー + tools.json パイプライン) |
| 中 | 分散 Rate Limiter、Stripe ogen 化、仕様書残課題 |
| 低 | SSE 改善、Loki プール、テスト基盤、UI/UX |

---

## 参考

- [sprint009-plan.md](./sprint009-plan.md) - Sprint 009 計画
- [sprint009-backlog.md](./sprint009-backlog.md) - Sprint 009 バックログ (統合版)
- [sprint008-review.md](./sprint008-review.md) - Sprint 008 レビュー
