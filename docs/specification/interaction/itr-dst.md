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
| Session Manager | DST ← SSM | ユーザーレコード作成（DBトリガー） |
| MCP Handler | DST ← HDL | ユーザー設定取得 |
| Modules | DST ← MOD | クレジット消費 |
| Payment Service Provider | DST ← PSP | クレジット情報受信 |
| User Console | DST ← CON | ツール設定登録 |

---

## 連携詳細

### SSM → DST（ユーザーレコード作成）

| 項目 | 内容 |
|------|------|
| トリガー | 新規ユーザー登録時（DBトリガー） |
| 操作 | ユーザーレコード作成 |

SSMでユーザーが作成されると、DBトリガーによりDSTにユーザーレコードが作成される。

**作成されるデフォルト設定:**
- current_plan: `free`
- account_status: `active`
- enabled_modules: `[]`

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
| free_credit_balance | number | 無料クレジット残高 |
| paid_credit_balance | number | 有料クレジット残高 |
| enabled_modules | array | 有効なモジュール一覧 |
| tool_settings | object | ツール単位の有効/無効設定 |
| user_prompts | array | ユーザー定義プロンプト |

**クレジット詳細:** [dtl-spc-credit-model.md](../dtl-spc/dtl-spc-credit-model.md)を参照

**注:** プランによるモジュール制限は行わない。

**レスポンス例:**
```json
{
  "user_id": "user-123",
  "account_status": "active",
  "free_credit_balance": 50,
  "paid_credit_balance": 1000,
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
  },
  "user_prompts": [
    {
      "name": "weekly_report",
      "description": "週次レポート作成",
      "template": "..."
    }
  ]
}
```

---

### PSP → DST（有料クレジット情報受信）

| 項目 | 内容 |
|------|------|
| トリガー | Webhook受信（決済完了） |
| 操作 | 有料クレジット残高の加算 |

**同期するイベント:**
- `checkout.session.completed` → 有料クレジット残高加算

**保存する情報:**
- psp_customer_id
- paid_credit_balance（加算後の残高）

---

### CON → DST（ツール設定登録）

| 項目 | 内容 |
|------|------|
| トリガー | ユーザーが設定画面で操作 |
| 操作 | モジュール有効/無効、設定変更、Auto-recharge設定 |

**操作内容:**
- モジュールの有効/無効切り替え
- 個別ツールの有効/無効切り替え
- カスタムプロンプトの登録/更新/削除
- Auto-recharge設定の登録/更新

**Auto-recharge設定:** [dtl-spc-credit-model.md](../dtl-spc/dtl-spc-credit-model.md)を参照

---

### MOD → DST（クレジット消費）

| 項目 | 内容 |
|------|------|
| トリガー | ツール実行成功時 |
| 操作 | クレジット残高の減算、消費記録の保存 |
| 対象 | 外部API呼び出しを伴うツール（メタツールを除く） |
| 除外 | get_module_schema, run, batch（メタツール）、リソース取得 |

**消費リクエスト:**
```json
{
  "user_id": "user-123",
  "module": "notion",
  "tool": "search",
  "amount": 1,
  "request_id": "req-456",
  "task_id": null
}
```

| フィールド | 必須 | 説明 |
|-----------|------|------|
| user_id | ✅ | ユーザーID |
| module | ✅ | モジュール名 |
| tool | ✅ | ツール名 |
| amount | ✅ | 消費量（現時点では固定1） |
| request_id | ✅ | リクエスト追跡用ID |
| task_id | ✅ | batch内タスクID（runの場合はnull） |

**run/batch の識別:**

| 呼び出し方式 | task_id | 説明 |
|-------------|---------|------|
| run | `null` | 単発ツール実行 |
| batch | `"task_name"` | batch内の各ツール実行 |

**消費レスポンス:**
```json
{
  "success": true
}
```

**注:** 監査・請求・分析の詳細設計は[dsn-adt.md](../../design/dsn-adt.md)を参照。

---

## DSTが直接やり取りしないコンポーネント

| コンポーネント | 理由 |
|----------------|------|
| MCP Client (OAuth2.0) (CLO) | GWY経由 |
| MCP Client (API KEY) (CLK) | GWY経由 |
| API Gateway (GWY) | AMW経由 |
| Auth Server (AUS) | SSM経由（同一DB） |
| Auth Middleware (AMW) | HDL経由 |
| Token Vault (TVL) | user_idリレーション参照のみ |
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
| [itr-hdl.md](./itr-hdl.md) | MCP Handler詳細仕様 |
| [itr-mod.md](./itr-mod.md) | Modules詳細仕様 |
| [itr-psp.md](./itr-psp.md) | Payment Service Provider詳細仕様 |
| [itr-con.md](./itr-con.md) | User Console詳細仕様 |
| [dsn-adt.md](../../design/dsn-adt.md) | 監査・請求・分析設計書 |
| [dtl-spc-credit-model.md](../dtl-spc/dtl-spc-credit-model.md) | クレジットモデル詳細仕様 |
