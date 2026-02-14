# Sprint 009 計画書

## 基本情報

| 項目 | 値 |
|------|-----|
| スプリント番号 | SPRINT-009 |
| 期間 | 2026-02-15 〜 2026-02-21 (7日間) |
| マイルストーン | M8: サブスク移行・堅牢性・ツール動的配信 |
| 前提 | Sprint-008 完了（設計書 -15、CI 6 ジョブ、Grafana アラート 3 + 通知設定） |
| 状態 | 計画中 |

---

## Sprint 目標

**クレジットを捨てる。サブスクで制御する。壊れにくくする**

Sprint 008 で品質基盤（CI、アラート、監査ログ）が整った。次は課金モデルの転換と堅牢性強化：
1. クレジット従量課金 → サブスクリプション + 利用上限に移行
2. 外部依存障害時の脆弱性（リトライなし）
3. tools.json パイプラインのぎこちなさ

---

## 背景

### クレジットモデルの問題

現在のフロー:
```
CanAccessTool (残高チェック) → ツール実行 → ConsumeCredit → 失敗 → ログのみ
```

**問題点:**
1. ConsumeCredit 失敗時に結果を返すかブロックするか、どちらも不適切
   - ブロック: ユーザーは実行済みの結果を受け取れない (Supabase 障害のせい)
   - 放置: 無料で使い放題になるリスク
2. リソースサーバー (Google 429, GitHub 403) のエラー時、クレジットを消費すべきか曖昧
3. ConsumeCredit は Supabase RPC 呼出しで、全リクエストに 1 往復追加

**結論:** クレジット従量課金を放棄し、サブスクリプションモデルに移行する。

### サブスクリプションモデルの設計

Claude のようなプラン別利用上限モデル:

```
[プラン] Free / Plus
[上限]  日次上限 (プランごとに異なる)
[判定]  利用量カウント → 上限超過でブロック → 翌日リセット
```

**利用量の記録は維持する。** サブスクでも「誰が何をどれだけ使ったか」の記録は必要:
- ユーザーへの利用量表示
- 管理者のレポート
- 不正利用の検知

### クレジットとの比較

| 観点 | クレジット | サブスク + 利用上限 |
|------|----------|-------------------|
| Server 負荷 | 毎回 ConsumeCredit RPC | カウント記録のみ (非同期可) |
| 障害耐性 | ConsumeCredit 失敗 = 判断不能 | 上限チェックはキャッシュ可能 |
| ユーザー体験 | 残高気にしながら使う | 上限内は自由に使える |
| 課金 | Stripe 1回払い | Stripe Subscription |
| 実装複雑度 | 高 (残高管理, 消費, 返金) | 低 (プランとカウンター) |

---

## タスク一覧

### Phase 1: サブスクリプション移行（優先度：最高）

クレジット従量課金を廃止し、プラン別利用上限モデルに移行する。

#### 1a. DB スキーマ + RPC

| ID | タスク | 変更対象 | 備考 |
|----|--------|---------|------|
| S9-001 | plans テーブル作成 | DB migration | plan_id, name, daily_limit, price_monthly 等 |
| S9-002 | user_profiles にプラン関連カラム追加 | DB migration | plan_id (FK), current_period_start, current_period_end |
| S9-003 | 利用量カウント用 RPC (record_usage) | DB migration | credit_transactions を usage_log にリネーム/再利用、running balance 不要 |
| S9-004 | 利用上限チェック RPC (check_usage_limit) | DB migration | 日次カウントを返す。Server がキャッシュ可能な形式 |
| S9-005 | consume_user_credits RPC 廃止 | DB migration | 削除 |
| S9-006 | get_user_context RPC 更新 | DB migration | credits → plan_id, daily_used, daily_limit |

#### 1b. Server (Go)

| ID | タスク | 変更対象 | 備考 |
|----|--------|---------|------|
| S9-010 | UserContext: credit フィールド → plan + usage フィールド | broker/user.go | FreeCredits/PaidCredits → PlanID, DailyUsed, DailyLimit |
| S9-011 | AuthContext: credit → plan + usage | middleware/authz.go | TotalCredits() → WithinDailyLimit() |
| S9-012 | CanAccessTool: 残高チェック → 利用上限チェック | middleware/authz.go | INSUFFICIENT_CREDITS → USAGE_LIMIT_EXCEEDED |
| S9-013 | handler run: ConsumeCredit 削除 → RecordUsage (非同期) | mcp/handler.go | 失敗してもブロックしない。ログ記録のみ |
| S9-014 | handler batch: ConsumeCredit 削除 → RecordUsage (非同期) | mcp/handler.go | 同上 |
| S9-015 | checkBatchPermissions: credit チェック → usage limit チェック | mcp/handler.go | TotalCredits() < toolCount → !WithinDailyLimit(toolCount) |
| S9-016 | ErrInsufficientCredit → ErrUsageLimitExceeded | mcp/types.go | エラーコード変更 |
| S9-017 | ConsumeCredit 関数削除 | broker/user.go | ConsumeResult 型も削除 |
| S9-018 | RecordUsage 関数追加 | broker/user.go | 非同期。失敗時はログのみ (上限チェックは事前に済んでいる) |
| S9-019 | BatchResult 簡素化 | modules/modules.go | SuccessCount/SuccessfulTasks は残す (RecordUsage 用) |

#### 1c. Console (Next.js)

| ID | タスク | 変更対象 | 備考 |
|----|--------|---------|------|
| S9-020 | credits ページ → plan ページに書き換え | credits/page.tsx | プラン表示、利用量表示、Stripe サブスク管理リンク |
| S9-021 | dashboard: クレジットカード → プラン + 利用量カード | dashboard/page.tsx | 日次の利用量バー表示 |
| S9-022 | sidebar: "クレジット" → "プラン" | sidebar.tsx | ラベル変更 |
| S9-023 | credits.ts → plan.ts に変更 | lib/credits.ts | getUserCredits → getUserPlan 等 |
| S9-024 | Stripe checkout → Subscription 作成に変更 | stripe/checkout/route.ts | mode: "payment" → "subscription" |
| S9-025 | Stripe webhook: checkout → invoice.paid 対応 | stripe/webhook/route.ts | add_user_credits → activate_subscription |
| S9-026 | サインアップボーナス → Free プラン自動付与 | grant-signup-bonus/route.ts | オンボーディング簡素化 |
| S9-027 | stripe.ts: price ID 更新 | lib/stripe.ts | freeCreditPriceId → subscriptionPriceIds |
| S9-028 | サービスページにレート制限情報を表示 | services/page.tsx | サービスごとの推定バースト制限・レート制限を表示 |

### Phase 2: 堅牢性改善（優先度：高）

| ID | タスク | 変更対象 | 備考 |
|----|--------|---------|------|
| S9-030 | Supabase クライアントにリトライ (指数バックオフ + ジッター) | broker/ | DB 呼び出し全般 |
| S9-031 | ヘルスチェック障害時のキャッシュ延長 | middleware/ | 現在 30s キャッシュ → 障害検知時に延長 |
| S9-032 | セキュリティヘッダー追加 (Worker) | apps/worker/ | CSP, HSTS, X-Content-Type-Options 等 |
| S9-033 | OAuth2 トークンリフレッシュを golang.org/x/oauth2 に移行 | broker/token.go | 手書き refreshOAuthToken (~80行) を x/oauth2 TokenSource に置換。OAuthRefreshConfig テーブル駆動は維持 |

### Phase 3: ツール定義の動的配信（優先度：中）

tools.json パイプラインを廃止し、Server API で動的に配信する。

| ID | タスク | 変更対象 | 備考 |
|----|--------|---------|------|
| S9-040 | Server に GET /tools エンドポイント追加 | handler/ or 新ファイル | 認証なし。ツール定義一覧を JSON で返す |
| S9-041 | Worker に /tools プロキシルート追加 | apps/worker/ | 認証不要パスとして Server に中継 |
| S9-042 | Console を Worker /tools API 呼び出しに切り替え | apps/console/ | tools.json import → Worker 経由で fetch |
| S9-043 | tools.json と tools-export コマンド削除 | apps/console/, apps/server/cmd/ | SSoT を Server に一本化 |

### Phase 4: 繰越し・小タスク（優先度：低）

| ID | タスク | 変更対象 | 備考 |
|----|--------|---------|------|
| S9-050 | セッション ID を暗号学的ランダムに変更 | SSE 関連 | 現在ポインタアドレス |

---

## 設計詳細

### プランテーブル

```sql
CREATE TABLE plans (
  id              TEXT PRIMARY KEY,       -- 'free', 'plus'
  name            TEXT NOT NULL,          -- 表示名
  daily_limit     INTEGER NOT NULL,       -- 日次ツール実行上限
  price_monthly   INTEGER DEFAULT 0,     -- 月額 (円)
  stripe_price_id TEXT,                  -- Stripe Price ID (NULL = Free)
  features        JSONB DEFAULT '{}'     -- 追加機能フラグ
);

-- 初期データ
INSERT INTO plans VALUES
  ('free', 'Free', 100, 0,   NULL,        '{}'),
  ('plus', 'Plus', 500, 980, 'price_xxx', '{}');
```

### 利用量チェックの流れ

```
[リクエスト]
  → GetUserContext (キャッシュ30s) → plan_id, daily_used, daily_limit
  → CanAccessTool:
      1. ツールホワイトリスト ✓
      2. アカウントステータス ✓
      3. daily_used < daily_limit ✓
  → ツール実行
  → RecordUsage (非同期 goroutine, fire-and-forget)
      → Supabase RPC insert (失敗してもブロックしない、リトライなし)
```

### 設計判断

**日次リセット:** UTC 0:00 で統一。DB クエリは `WHERE created_at >= current_date AT TIME ZONE 'UTC'`。

**RecordUsage:** fire-and-forget。利用量記録はプロバイダ側の責務であり、記録失敗はユーザーに影響させない。冪等性保証・リトライも不要。

**プラン変更の即時反映:** Free → Plus アップグレード時、ユーザーは即座に上限拡大を体感できる必要がある。Stripe webhook でプラン更新後、Server の UserContext キャッシュを即時無効化する。キャッシュ TTL 30s の遅延は UX として許容できない (上限到達 → 課金 → まだ使えない)。

**障害耐性:** RecordUsage が失敗しても結果を返す。理由:
- 上限チェックは事前に完了している
- カウントの厳密性よりユーザー体験を優先
- 次の GetUserContext で DB から正確な値を取得する

### authErrorToRPC マッピング変更

```go
// Before
"INSUFFICIENT_CREDITS" → ErrInsufficientCredit (-32002)

// After
"USAGE_LIMIT_EXCEEDED" → ErrUsageLimitExceeded (-32002)  // コード番号は維持
```

---

## 実装方針

### 作業順序

```
Day 1-2: Phase 1a (DB) + Phase 1b (Server) — クレジット削除 + サブスク基盤
Day 3:   Phase 1c (Console) — UI 書き換え
Day 4:   Phase 2 (堅牢性) — リトライ・キャッシュ延長・x/oauth2
Day 5:   Phase 3 (ツール動的配信) — Server + Console 両方
Day 6-7: Phase 4 + バッファ + Stripe 連携テスト
```

### スコープ外（Sprint 010 以降）

以下はバックログに残す：

| 項目 | 理由 |
|------|------|
| 認可 OAuth Server 自作 | 規模大 (2-3 Sprint)。現状 Supabase Auth で動いている |
| DB 移行 (Supabase → Neon) | 規模大。認証基盤と連動するため同時に計画すべき |
| Console Prisma 導入 | DB 移行と同時が自然。今入れると二重マイグレーションリスク |
| 分散 Rate Limiter | マルチインスタンス運用開始時まで不要 |
| Stripe ogen 化 | 機能追加であり堅牢性改善の後 |

---

## 完了条件

- [ ] クレジット残高の概念が Server から消えている (ConsumeCredit, TotalCredits 削除)
- [ ] plans テーブルにプラン定義がある (free/plus)
- [ ] ツール実行時に日次利用上限でブロックされる
- [ ] 利用量が非同期で記録される
- [ ] Console にプランページがあり、利用量が表示される
- [ ] Supabase RPC 呼び出しに指数バックオフリトライが入っている
- [ ] Worker にセキュリティヘッダーが付与されている
- [ ] Console のサービスページにサービスごとのレート制限情報が表示されている
- [ ] Console がツール定義を Server API から動的取得している
- [ ] tools.json ファイルと tools-export コマンドが削除されている

---

## リスク

| リスク | 影響 | 対策 |
|--------|------|------|
| Stripe Subscription 設定の複雑さ | サブスク作成/解約フローが増える | 最初は Free + 手動アップグレードで最小実装 |
| 利用上限の数値設定が不適切 | ユーザー体験悪化 or 無制限に近い状態 | Grafana で利用量を監視し、数値を調整 |
| RecordUsage 非同期で利用量の取りこぼし | 上限を少し超える可能性 | キャッシュ TTL 30s の範囲で許容。厳密性より可用性 |
| /tools API が Server ダウン時に Console 設定画面が見えない | 設定変更不可 | Server が落ちていればツール実行もできないため許容 |
| リトライによるレイテンシ増加 | ツール応答が遅くなる | バックオフ上限を短く (最大 2s) |

---

## 参考

- [sprint008-review.md](./sprint008-review.md) - Sprint 008 レビュー
- [sprint008-backlog.md](./sprint008-backlog.md) - Sprint 008 バックログ
