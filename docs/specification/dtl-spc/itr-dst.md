# Data Store インタラクション仕様書（itr-dst）

## ドキュメント管理情報

| 項目 | 値 |
|------|-----|
| Status | `draft` |
| Version | v2.0 |
| Note | Data Store Interaction Specification |

---

## 概要

Data Store（DST）は、ユーザーの権限・設定・課金情報を管理するデータストア。

主な責務：
- ユーザー情報の永続化
- アカウント状態の管理
- クレジット残高の管理
- モジュール有効/無効設定の管理
- ツール単位の有効/無効設定の管理
- カスタムプロンプトの保存
- 課金情報の管理（PSPからのWebhook）

---

## 連携サマリー（spc-itrより）

| 相手 | 方向 | やり取り |
|------|------|----------|
| Session Manager | DST ← SSM | ユーザーID共有 |
| Auth Server | DST ↔ AUS | ユーザーID共有 |
| Token Vault | DST → TVL | ユーザーID共有 |
| MCP Handler | DST ← HDL | ユーザー設定提供 |
| Modules | DST ← MOD | クレジット消費 |
| Payment Service Provider | DST ← PSP | プラン情報受信 |
| User Console | DST ← CON | ツール設定登録 |

---

## 連携詳細

### SSM → DST（ユーザーID共有）

| 項目 | 内容 |
|------|------|
| トリガー | 新規ユーザー登録時 |
| 操作 | ユーザーレコード作成 |

**フロー:**
1. SSMがソーシャルログインでユーザー認証
2. SSMがDSTにユーザーID（UUID）を通知
3. DSTが新規ユーザーレコードを作成（デフォルト設定）

**作成されるデフォルト設定:**
- current_plan: `free`
- account_status: `active`
- enabled_modules: `[]`

---

### AUS ↔ DST（ユーザーID共有）

| 項目 | 内容 |
|------|------|
| トリガー | OAuth認証成功時 |
| 操作 | ユーザー存在確認、アカウント状態取得 |

**フロー:**
1. AUSがOAuth認証成功
2. DSTにuser_idで問い合わせ
3. 存在しなければ新規レコード作成
4. アカウント状態（active/suspended/disabled）を返却

---

### DST → TVL（ユーザーID共有）

| 項目 | 内容 |
|------|------|
| 用途 | トークン管理のユーザー紐付け |
| 方向 | DSTのuser_idをTVLが参照 |

TVLはDSTと同じuser_id体系を使用してトークンを管理する。

---

### HDL → DST（ユーザー設定取得）

| 項目 | 内容 |
|------|------|
| トリガー | MCPメソッド実行時 |
| 操作 | ユーザー設定・状態の取得 |

**提供する情報:**

| フィールド | 型 | 説明 |
|-----------|-----|------|
| account_status | string | アカウント状態（active/suspended/disabled） |
| credit_balance | number | クレジット残高 |
| enabled_modules | array | 有効なモジュール一覧 |
| tool_settings | object | ツール単位の有効/無効設定 |

**注:** プランによるモジュール制限は行わない。

**レスポンス例:**
```json
{
  "user_id": "user-123",
  "account_status": "active",
  "credit_balance": 1000,
  "enabled_modules": ["notion", "google_calendar"],
  "tool_settings": {
    "notion": {
      "search": true,
      "create_page": true,
      "delete_page": false
    },
    "google_calendar": {
      "list_events": true,
      "create_event": true,
      "delete_event": false
    }
  }
}
```

---

### DST → HDL（カスタムプロンプト提供）

| 項目 | 内容 |
|------|------|
| トリガー | prompts/list, prompts/get 実行時 |
| 操作 | ユーザー定義プロンプト取得 |

**提供する情報:**
- ユーザーが登録したカスタムプロンプト
- プロンプトテンプレート

---

### PSP → DST（プラン情報受信）

| 項目 | 内容 |
|------|------|
| トリガー | Webhook受信（決済完了、サブスクリプション更新等） |
| 操作 | 課金状態の同期 |

**同期するイベント:**
- `checkout.session.completed` → サブスクリプション作成
- `customer.subscription.updated` → プラン情報更新
- `customer.subscription.deleted` → プランをfreeに変更
- `invoice.paid` → billing_status更新
- `invoice.payment_failed` → billing_statusをpast_dueに

**保存する情報:**
- psp_customer_id
- psp_subscription_id
- current_plan（free/pro/enterprise）
- billing_status（active/past_due/canceled）
- current_period_start
- current_period_end

---

### CON → DST（ツール設定登録）

| 項目 | 内容 |
|------|------|
| トリガー | ユーザーが設定画面で操作 |
| 操作 | モジュール有効/無効、設定変更 |

**操作内容:**
- モジュールの有効/無効切り替え
- 個別ツールの有効/無効切り替え
- カスタムプロンプトの登録/更新/削除

---

### MOD → DST（クレジット消費）

| 項目 | 内容 |
|------|------|
| トリガー | ツール/プロンプト実行成功時 |
| 操作 | クレジット残高の減算、消費記録の保存 |

**消費リクエスト:**
```json
{
  "user_id": "user-123",
  "module": "notion",
  "primitive_type": "tool",
  "primitive_name": "search",
  "amount": 1,
  "request_id": "req-456"
}
```

**消費レスポンス:**
```json
{
  "success": true,
  "credit_balance": 999
}
```

**保存する消費記録:**

| フィールド | 型 | 説明 |
|-----------|-----|------|
| user_id | string | ユーザーID |
| module | string | モジュール名 |
| primitive_type | string | プリミティブ種別 |
| primitive_name | string | プリミティブ名 |
| amount | number | 消費量 |
| request_id | string | リクエストID |
| consumed_at | timestamp | 消費日時 |

**注:**
- 消費量（amount）は現時点では固定1
- 消費記録は監査ログとして保存

---

## DSTが直接やり取りしないコンポーネント

| コンポーネント | 理由 |
|----------------|------|
| MCP Client (OAuth2.0) (CLO) | AUS経由 |
| MCP Client (API KEY) (CLK) | TVL経由 |
| API Gateway (GWY) | AMW経由 |
| Auth Middleware (AMW) | HDL経由 |
| Identity Provider (IDP) | SSM経由 |
| External Auth Server (EAS) | CON経由 |
| External Service API (EXT) | MOD経由 |

---

## 関連ドキュメント

| ドキュメント | 内容 |
|-------------|------|
| [spc-sys.md](../spc-sys.md) | システム仕様書 |
| [spc-itr.md](../spc-itr.md) | インタラクション仕様書 |
| [spc-tbl.md](../spc-tbl.md) | テーブル仕様書 |
| [itr-ssm.md](./itr-ssm.md) | Session Manager詳細仕様 |
| [itr-aus.md](./itr-aus.md) | Auth Server詳細仕様 |
| [itr-mod.md](./itr-mod.md) | Modules詳細仕様 |
| [itr-psp.md](./itr-psp.md) | Payment Service Provider詳細仕様 |
| [itr-con.md](./itr-con.md) | User Console詳細仕様 |
