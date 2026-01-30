# DAY019 初回クレジット付与実装計画

## 概要

billing ページで `pre_active` ユーザーに「初回クレジットを受け取る」カードを表示し、クリックで 100 free_credits を付与 + `active` に遷移させる。

---

## 背景

### 現在のフロー

```
新規ユーザー登録
    ↓
DBトリガー handle_new_user()
    ↓
account_status = 'pre_active', free_credits = 0, paid_credits = 0
    ↓
/onboarding（サービス選択）
    ↓
/dashboard
    ↓
??? ← 初回クレジット付与のタイミングが未定義
```

### 既存の実装（未使用）

- API: `POST /api/credits/grant-signup-bonus`
- RPC: `complete_onboarding(p_user_id, p_event_id)`
  - `add_credits(user_id, 100, 'free', event_id)` を呼ぶ
  - `account_status` を `active` に更新
  - 冪等性: `event_id = "onboarding:{user_id}"` で重複防止

---

## 実装方針

- オンボーディング画面 → 将来プロダクトツアー用（クレジット付与しない）
- ダッシュボード → オンボーディング体験のメイン（カードが光る）
- **billing ページ** → `pre_active` なら「初回クレジット取得」カードを表示

---

## 変更ファイル

| ファイル | 変更内容 |
|----------|----------|
| `apps/console/src/lib/credits.ts` | `UserContext` 型追加、`getUserContext()` 関数追加 |
| `apps/console/src/app/(console)/billing/page.tsx` | `pre_active` 判定、初回クレジットカード表示、API呼び出し |

---

## 詳細設計

### 1. credits.ts の変更

**追加する型:**

```typescript
export interface UserContext {
  account_status: string
  free_credits: number
  paid_credits: number
}
```

**追加する関数:**

```typescript
export async function getUserContext(): Promise<UserContext | null> {
  const supabase = createClient()
  const { data: { user } } = await supabase.auth.getUser()
  if (!user) return null

  const { data, error } = await supabase.rpc('get_user_context', {
    p_user_id: user.id
  })

  if (error || !data) return null

  const context = Array.isArray(data) ? data[0] : data
  return {
    account_status: context.account_status,
    free_credits: context.free_credits,
    paid_credits: context.paid_credits,
  }
}
```

### 2. billing/page.tsx の変更

**状態追加:**

```typescript
const [accountStatus, setAccountStatus] = useState<string | null>(null)
const [claiming, setClaiming] = useState(false)
```

**データ取得変更:**

```typescript
// getUserCredits() → getUserContext() に変更
const context = await getUserContext()
setAccountStatus(context?.account_status ?? null)
setCredits({
  free_credits: context?.free_credits ?? 0,
  paid_credits: context?.paid_credits ?? 0,
  updated_at: new Date().toISOString(),
})
```

**初回クレジット取得処理:**

```typescript
const handleClaimSignupBonus = async () => {
  setClaiming(true)
  try {
    const response = await fetch("/api/credits/grant-signup-bonus", {
      method: "POST",
    })
    const data = await response.json()

    if (data.success) {
      toast.success("クレジットを受け取りました！", {
        description: "100クレジットがアカウントに追加されました。",
      })
      // データ再取得
      const context = await getUserContext()
      setAccountStatus(context?.account_status ?? null)
      setCredits({
        free_credits: context?.free_credits ?? 0,
        paid_credits: context?.paid_credits ?? 0,
        updated_at: new Date().toISOString(),
      })
    } else if (data.error === "already_granted") {
      toast.info("既にクレジットを受け取っています")
      setAccountStatus("active")
    } else {
      throw new Error(data.message || "Failed to claim bonus")
    }
  } catch (error) {
    console.error("Claim error:", error)
    toast.error("エラーが発生しました", {
      description: "しばらくしてからもう一度お試しください。",
    })
  } finally {
    setClaiming(false)
  }
}
```

**UI表示ロジック:**

```tsx
{/* 初回クレジット取得（pre_active のみ） */}
{accountStatus === "pre_active" && (
  <Card className="animate-pulse-border border-primary shadow-lg shadow-primary/20">
    <CardHeader>
      <CardTitle className="text-lg flex items-center gap-2">
        <Sparkles className="h-5 w-5 text-primary" />
        ようこそ！初回クレジットを受け取る
      </CardTitle>
      <CardDescription>
        MCPistを始めるための100クレジットをプレゼント
      </CardDescription>
    </CardHeader>
    <CardContent>
      <div className="space-y-4">
        <div className="bg-primary/10 rounded-lg p-4">
          <div className="flex items-center justify-between">
            <div>
              <p className="font-medium">スタートボーナス</p>
              <p className="text-sm text-muted-foreground">
                今すぐ受け取れます
              </p>
            </div>
            <div className="text-right">
              <p className="text-2xl font-bold text-primary">100</p>
              <p className="text-sm text-muted-foreground">クレジット</p>
            </div>
          </div>
        </div>
        <Button
          className="w-full"
          size="lg"
          onClick={handleClaimSignupBonus}
          disabled={claiming}
        >
          {claiming ? (
            <>
              <Loader2 className="h-4 w-4 mr-2 animate-spin" />
              処理中...
            </>
          ) : (
            <>
              <Gift className="h-4 w-4 mr-2" />
              100クレジットを受け取る
            </>
          )}
        </Button>
      </div>
    </CardContent>
  </Card>
)}

{/* 既存の Stripe Checkout カード（active のみ） */}
{accountStatus !== "pre_active" && (
  // 現在の「無料クレジットを取得」カード
)}
```

---

## ユーザーフロー

```
新規ユーザー (pre_active, 0 credits)
    ↓
ログイン → /onboarding（サービス選択）
    ↓
/dashboard
    ↓
「クレジット残高」カードが光る（残高50以下）
    ↓
/billing ページへ移動
    ↓
「ようこそ！初回クレジットを受け取る」カードが光って表示
    ↓
「100クレジットを受け取る」ボタンクリック
    ↓
POST /api/credits/grant-signup-bonus
    ↓
complete_onboarding RPC 実行
    ↓
100 free_credits 付与 + account_status = "active"
    ↓
toast「クレジットを受け取りました！」
    ↓
初回カード消失 → 通常の billing ページ表示
```

---

## テスト項目

- [ ] pre_active ユーザーで billing ページにアクセス → 初回クレジットカードが表示される
- [ ] 初回クレジットカードが `animate-pulse-border` で光っている
- [ ] 「100クレジットを受け取る」クリック → 100 free_credits 付与
- [ ] 付与後、account_status が "active" に変更
- [ ] 付与後、初回カードが消えて通常の Stripe カードが表示される
- [ ] 既に active のユーザー → 初回カードは表示されない
- [ ] 2回クリック（冪等性）→ 「既にクレジットを受け取っています」

---

## 実装順序

1. `credits.ts` に `UserContext` 型と `getUserContext()` 関数追加
2. `billing/page.tsx` でデータ取得を `getUserContext()` に変更
3. `billing/page.tsx` に初回クレジットカードのUI追加
4. `handleClaimSignupBonus` 関数実装
5. 動作確認

---

## 注意事項

- 既存の `getUserCredits()` 関数は他で使われている可能性があるので削除しない
- `grant-signup-bonus` API は既存のものをそのまま使用
- `complete_onboarding` RPC も既存のものをそのまま使用
