# Stripe クレジットシステム実装計画

## 概要

クレジット配布の全パターンに対応する統一RPCと、各トリガーの実装。

---

## クレジット配布パターン

| # | パターン | トリガー | クレジット種別 | クレジット数 |
|---|----------|----------|---------------|-------------|
| 1 | 初回サインアップ | ユーザー登録完了 | free | 100 |
| 2 | $0 Checkout + カード登録 | checkout.session.completed | paid | 100 |
| 3 | 有料サブスク | invoice.paid | paid | 1000/月 |
| 4 | スポット購入 | checkout.session.completed | paid | 購入量 |

---

## 現状分析

### 既存RPC

| 関数名 | 用途 | 問題点 |
|--------|------|--------|
| `add_paid_credits` | paid_credits 追加 | free_credits 非対応 |
| `link_stripe_customer` | Stripe Customer ID 紐付け | 問題なし |
| `get_user_by_stripe_customer` | Customer ID からユーザー検索 | 問題なし |
| `get_stripe_customer_id` | ユーザーの Customer ID 取得 | 問題なし |

### 既存Webhook

| エンドポイント | イベント | 処理 |
|---------------|----------|------|
| `/api/stripe/webhook` | checkout.session.completed | `add_paid_credits` 呼び出し |

---

## 実装計画

### Phase 1: 統一RPC作成

#### 1.1 `add_credits` RPC

**既存の `add_paid_credits` を拡張**して、free/paid 両対応に。

```sql
CREATE OR REPLACE FUNCTION mcpist.add_credits(
    p_user_id UUID,
    p_amount INTEGER,
    p_credit_type TEXT,  -- 'free' or 'paid'
    p_event_id TEXT      -- 冪等性キー
)
RETURNS JSONB
```

**パラメータ:**

| 名前 | 型 | 必須 | 説明 |
|------|-----|------|------|
| p_user_id | UUID | ○ | 対象ユーザーID |
| p_amount | INTEGER | ○ | 付与クレジット数 |
| p_credit_type | TEXT | ○ | 'free' または 'paid' |
| p_event_id | TEXT | ○ | 冪等性キー（重複処理防止） |

**処理フロー:**

1. event_id で重複チェック（`processed_webhook_events`）
2. amount > 0 バリデーション
3. credit_type に応じて `free_credits` or `paid_credits` を加算
4. `credit_transactions` に記録
5. `processed_webhook_events` に記録

**戻り値:**

```json
{
  "success": true,
  "credit_type": "free",
  "free_credits": 100,
  "paid_credits": 0,
  "added": 100
}
```

#### 1.2 マイグレーションファイル

```
supabase/migrations/00000000000032_unified_add_credits.sql
```

**内容:**
- `mcpist.add_credits()` 関数作成
- `public.add_credits()` ラッパー作成
- 既存 `add_paid_credits` は後方互換のため残す（内部で `add_credits` を呼ぶラッパーに変更可）

---

### Phase 2: 初回サインアップ時クレジット付与

#### 2.1 Console 側実装

**ファイル:** `apps/console/src/app/auth/callback/route.ts`

**処理フロー:**

1. OAuth callback でユーザー認証完了
2. ユーザーが新規かどうか判定（`created_at` と現在時刻の差 or 専用フラグ）
3. 新規の場合: `add_credits(user_id, 100, 'free', 'signup:{user_id}')` を呼び出し

**冪等性:**
- event_id: `signup:{user_id}` 形式
- 同一ユーザーへの重複付与を防止

#### 2.2 新規ユーザー判定

**方法A: created_at チェック**
```typescript
// 登録から10秒以内なら新規とみなす
const isNewUser = (new Date().getTime() - new Date(user.created_at).getTime()) < 10000
```

**方法B: credits テーブルチェック**
```typescript
// free_credits + paid_credits が 0 なら新規
const { data: credits } = await supabase.rpc('get_my_credits')
const isNewUser = credits.free_credits === 0 && credits.paid_credits === 0
```

→ **方法B を採用**（より確実）

---

### Phase 3: $0 Checkout カード登録必須化

#### 3.1 Checkout Session 設定変更

**ファイル:** `apps/console/src/app/api/stripe/checkout/route.ts`

**変更点:**

```typescript
const session = await stripe.checkout.sessions.create({
  // ... 既存設定
  payment_method_collection: "always",  // 追加: カード入力必須
  // ...
})
```

#### 3.2 UI文言変更

**ファイル:** `apps/console/src/app/(console)/billing/page.tsx`

**変更:**
- 「無料クレジットを取得」→「カード登録で100クレジット獲得」
- 説明文を追加: 「クレジットカードを登録すると、ボーナスとして100クレジットが付与されます」

---

### Phase 4: 有料サブスク対応

#### 4.1 Stripe Product/Price 作成

**Dashboard で作成:**

| 項目 | 値 |
|------|-----|
| Product Name | MCPist Pro |
| Price | $9.99/month (recurring) |
| metadata.credits | 1000 |

#### 4.2 Webhook イベント追加

**ファイル:** `apps/console/src/app/api/stripe/webhook/route.ts`

**追加イベント:** `invoice.paid`

```typescript
case "invoice.paid":
  await handleInvoicePaid(event.data.object as Stripe.Invoice, event.id)
  break
```

**handleInvoicePaid 処理:**

1. Invoice から subscription を取得
2. Subscription の metadata からクレジット数を取得
3. Customer ID からユーザーを特定
4. `add_credits(user_id, credits, 'paid', event_id)` を呼び出し

#### 4.3 サブスク開始UI

**新規ページ or billing ページ拡張:**
- Pro プラン説明
- 「月額 $9.99 で 1,000 クレジット/月」
- サブスク開始ボタン → Checkout Session (mode: "subscription")

---

### Phase 5: スポット購入拡張

#### 5.1 追加 Product/Price

**Dashboard で作成:**

| Product | Price | Credits |
|---------|-------|---------|
| Credit Pack Small | $5 | 500 |
| Credit Pack Medium | $10 | 1,200 |

#### 5.2 UI拡張

**billing ページに追加:**
- クレジットパック選択
- 購入ボタン
- 既存の Checkout 処理を流用（Price ID を動的に変更）

---

## ファイル変更一覧

### DB Migration

| ファイル | 内容 |
|----------|------|
| `00000000000032_unified_add_credits.sql` | `add_credits` RPC |

### Console

| ファイル | 変更内容 |
|----------|----------|
| `src/app/auth/callback/route.ts` | 新規ユーザーへの初回クレジット付与 |
| `src/app/api/stripe/checkout/route.ts` | `payment_method_collection: "always"` |
| `src/app/api/stripe/webhook/route.ts` | `invoice.paid` ハンドラ追加 |
| `src/app/(console)/billing/page.tsx` | UI文言変更、サブスク・パック購入UI |
| `src/lib/supabase/database.types.ts` | `add_credits` RPC 型定義 |

---

## 実装順序

| Phase | タスク | 依存 |
|-------|--------|------|
| 1 | 統一RPC `add_credits` 作成 | なし |
| 2 | 初回サインアップ時クレジット付与 | Phase 1 |
| 3 | $0 Checkout カード必須化 | なし |
| 4 | 有料サブスク対応 | Phase 1 |
| 5 | スポット購入拡張 | Phase 1 |

**推奨順序:** Phase 1 → Phase 3 → Phase 2 → Phase 4 → Phase 5

（Phase 3 は既存コード修正のみで簡単なため先に実施）

---

## テスト項目

### Phase 1: add_credits RPC

- [ ] free_credits 付与が正常に動作
- [ ] paid_credits 付与が正常に動作
- [ ] 同一 event_id での重複付与が拒否される
- [ ] 不正な credit_type でエラー
- [ ] amount <= 0 でエラー

### Phase 2: 初回サインアップ

- [ ] 新規ユーザー登録時に 100 free_credits 付与
- [ ] 既存ユーザー再ログイン時は付与されない
- [ ] event_id による重複防止が機能

### Phase 3: カード登録必須

- [ ] Checkout 画面でカード入力が必須
- [ ] カード登録完了で 100 paid_credits 付与

### Phase 4: サブスク

- [ ] サブスク開始で初回 1,000 paid_credits 付与
- [ ] 翌月更新で 1,000 paid_credits 付与
- [ ] キャンセル後は付与されない

### Phase 5: スポット購入

- [ ] $5 で 500 credits 付与
- [ ] $10 で 1,200 credits 付与

---

## 環境変数（追加予定）

### サブスク用

```
STRIPE_PRO_PRICE_ID=price_xxx
```

### スポット購入用

```
STRIPE_CREDIT_PACK_SMALL_PRICE_ID=price_xxx
STRIPE_CREDIT_PACK_MEDIUM_PRICE_ID=price_xxx
```

---

## 注意事項

1. **冪等性**: すべてのクレジット付与は event_id で重複チェック
2. **トランザクション**: RPC 内でクレジット更新とログ記録を同一トランザクションで実行
3. **エラーハンドリング**: Webhook 失敗時は Stripe が自動リトライ（冪等性で対応）
4. **テストモード**: 本番移行までは Stripe テストモードを使用

---

## 本日の実装範囲

**Phase 1 + Phase 3 を実施**

| # | タスク | 見積もり |
|---|--------|----------|
| 1 | `add_credits` RPC 作成・テスト | 1h |
| 2 | Checkout カード必須化 | 0.5h |
| 3 | UI文言変更 | 0.5h |
| 4 | E2Eテスト | 0.5h |

**合計: 2.5h**

Phase 2（初回サインアップ）、Phase 4（サブスク）、Phase 5（スポット購入）は次回以降に実施。
