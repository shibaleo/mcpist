# RPC関数詳細設計書（dtl-dsn-rpc）

## ドキュメント管理情報

| 項目 | 値 |
|------|-----|
| Status | `draft` |
| Version | v1.0 |
| Note | RPC Function Detail Design |

---

## lookup_user_by_key_hash

APIキーのハッシュからuser_idを取得する。

### インタラクション

| 項目 | 内容 |
|------|------|
| ID | ITR-REL-005 |
| 呼び出し元 | MCP Server (Auth Middleware) |
| トリガー | Authorization: Bearer mpt_xxx 受信時 |

### 設計意図

Worker側でハッシュ計算を行い、キャッシュと組み合わせることでVault呼び出しを削減する。

```
User ─(mpt_xxx)─▶ Worker ─(sha256 hash)─▶ Cache hit? ─▶ user_id
                     │                         │
                     │                         ▼ miss
                     │                    Vault RPC
                     │                         │
                     ▼                         ▼
              sha256(mpt_xxx) ◀───────── user_id (cache)
```

### 入力

| パラメータ | 型 | 必須 | 説明 |
|-----------|-----|------|------|
| p_key_hash | TEXT | YES | APIキーのSHA-256ハッシュ（hex形式） |

### 出力（JSONB）

| フィールド | 型 | 説明 |
|-----------|-----|------|
| valid | BOOLEAN | 検証成功したか |
| user_id | UUID | ユーザーID（成功時のみ） |
| error | TEXT | エラー理由（失敗時のみ: invalid_key / revoked / expired） |

### 処理

1. api_keysテーブルからkey_hashで検索
2. revoked_atがNULLでないか確認（削除済みチェック）
3. expires_atが設定されている場合、有効期限をチェック
4. last_used_atを現在時刻に更新
5. 検証結果を返却

### Worker側の責務

- APIキー（mpt_xxx）をSHA-256でハッシュ化
- ハッシュをキーとしてキャッシュを参照
- キャッシュミス時のみRPC呼び出し
- 検証成功時はuser_idをキャッシュに保存

### 参照テーブル

- mcpist.api_keys

---

## get_user_context

ツール実行に必要なユーザー情報を一括取得する。

### インタラクション

| 項目 | 内容 |
|------|------|
| ID | ITR-REL-008 |
| 呼び出し元 | MCP Server (MCP Handler) |
| トリガー | MCPメソッドリクエスト時 |

### 入力

| パラメータ | 型 | 必須 | 説明 |
|-----------|-----|------|------|
| p_user_id | UUID | YES | ユーザーID |

### 出力

| フィールド | 型 | 説明 |
|-----------|-----|------|
| account_status | TEXT | アカウント状態（active/suspended/disabled） |
| free_credits | INTEGER | 無料クレジット残高 |
| paid_credits | INTEGER | 有料クレジット残高 |
| enabled_modules | TEXT[] | 有効なモジュール名の配列 |
| disabled_tools | JSONB | モジュール別の無効ツール `{"notion": ["delete_page"], ...}` |

### 処理

1. usersテーブルからaccount_statusを取得
2. creditsテーブルからfree_credits, paid_creditsを取得
3. module_settingsテーブルから有効なモジュールを取得
4. tool_settingsテーブルから無効なツールを取得
5. 結果を1レコードで返却

### エラー条件

- ユーザーが存在しない → 空の結果（0行）

### 参照テーブル

- mcpist.users
- mcpist.credits
- mcpist.module_settings
- mcpist.tool_settings
- mcpist.modules

---

## consume_credit

クレジットを消費し、履歴を記録する。

### インタラクション

| 項目 | 内容 |
|------|------|
| ID | ITR-REL-011 |
| 呼び出し元 | MCP Server (Modules) |
| トリガー | ツール実行成功時 |

### 入力

| パラメータ | 型 | 必須 | 説明 |
|-----------|-----|------|------|
| p_user_id | UUID | YES | ユーザーID |
| p_module | TEXT | YES | モジュール名 |
| p_tool | TEXT | YES | ツール名 |
| p_amount | INTEGER | YES | 消費量（通常1） |
| p_request_id | TEXT | YES | リクエスト追跡用ID |
| p_task_id | TEXT | NO | batch内タスクID（runの場合はNULL） |

### 出力

| フィールド | 型 | 説明 |
|-----------|-----|------|
| success | BOOLEAN | 消費成功したか |
| free_credits | INTEGER | 消費後の無料クレジット残高 |
| paid_credits | INTEGER | 消費後の有料クレジット残高 |

### 処理

1. creditsテーブルから現在の残高を取得（FOR UPDATE）
2. 合計残高（free + paid）がamount未満の場合 → success=false
3. 無料クレジットから優先的に消費
   - free_credits >= amount → free_credits -= amount
   - free_credits < amount → free_credits = 0, paid_credits -= (amount - free_credits)
4. creditsテーブルを更新
5. credit_transactionsにtype='consume'で履歴を記録
6. 更新後の残高とsuccess=trueを返却

### エラー条件

- 残高不足 → success=false, 残高は変更しない

### 冪等性

- request_id + task_idの組み合わせでUNIQUE制約
- 重複リクエストは既存レコードを返却（再消費しない）

### 参照テーブル

- mcpist.credits（更新）
- mcpist.credit_transactions（挿入）

---

## get_module_token

モジュールが使用する外部サービスのトークンを取得する。

### インタラクション

| 項目 | 内容 |
|------|------|
| ID | ITR-REL-010 |
| 呼び出し元 | MCP Server (Modules) |
| トリガー | 外部API呼び出し前 |

### 入力

| パラメータ | 型 | 必須 | 説明 |
|-----------|-----|------|------|
| p_user_id | UUID | YES | ユーザーID |
| p_module | TEXT | YES | モジュール名（notion, github等） |

### 出力（JSONB）

Vaultに保存されたJSON全体をそのまま返却。`_auth_type`でトークン種別を判別する。

```json
{
  "access_token": "...",
  "refresh_token": "...",
  "client_id": "...",
  "client_secret": "...",
  "_auth_type": "oauth2",
  "_expires_at": "2024-01-01T00:00:00+00:00"
}
```

| _auth_type | 説明 |
|------------|------|
| oauth2 | OAuth 2.0トークン（access_token, refresh_token等） |
| api_key | 長期トークン/APIキー（token） |

### 設計意図

1サービス = 1 JSON secretとしてVaultに保存し、全情報をそのまま返却。モジュール側でトークン選択・リフレッシュ判定を行う。

### 処理

1. service_tokensテーブルからuser_id + module（=service）で検索
2. credentials_secret_idからvault.decrypted_secretsを参照
3. 復号されたJSON全体を返却

### モジュール側の責務

- `_auth_type`を見てトークン種別を判定
- `_expires_at`を見て期限切れ/期限間近を判定
- OAuthの場合、必要に応じてサービス固有エンドポイントでリフレッシュ
- リフレッシュ成功 → `update_module_token`で新トークン保存

### 参照テーブル

- mcpist.service_tokens（user_id + service → credentials_secret_id）
- vault.decrypted_secrets

---

## update_module_token

モジュールがリフレッシュしたトークンをVaultに保存する。

### インタラクション

| 項目 | 内容 |
|------|------|
| ID | ITR-REL-010 |
| 呼び出し元 | MCP Server (Modules) |
| トリガー | トークンリフレッシュ成功時 |

### 入力

| パラメータ | 型 | 必須 | 説明 |
|-----------|-----|------|------|
| p_user_id | UUID | YES | ユーザーID |
| p_module | TEXT | YES | モジュール名（notion, github等） |
| p_credentials | JSONB | YES | 保存する認証情報（全体を上書き） |

### 出力

| フィールド | 型 | 説明 |
|-----------|-----|------|
| success | BOOLEAN | 更新成功したか |

### 処理

1. service_tokensテーブルからuser_id + moduleで検索
2. credentials_secret_idからvault.secretsを特定
3. p_credentialsをそのまruま保存（vault.update_secret）

### モジュール側の責務

リフレッシュ時は、既存の認証情報に新しいトークンを反映した完全なJSONBを構築して渡す。

### 参照テーブル

- mcpist.service_tokens（参照）
- vault.secrets（更新）

---

## add_paid_credits

有料クレジットを加算する（Webhook処理用）。

### インタラクション

| 項目 | 内容 |
|------|------|
| ID | ITR-REL-021 |
| 呼び出し元 | User Console API Routes |
| トリガー | PSP Webhook (checkout.session.completed) |

### 入力

| パラメータ | 型 | 必須 | 説明 |
|-----------|-----|------|------|
| p_user_id | UUID | YES | ユーザーID |
| p_amount | INTEGER | YES | 加算するクレジット量 |
| p_event_id | TEXT | YES | PSPのevent.id（冪等性用） |

### 出力

| フィールド | 型 | 説明 |
|-----------|-----|------|
| success | BOOLEAN | 処理成功したか |
| paid_credits | INTEGER | 加算後の有料クレジット残高 |
| already_processed | BOOLEAN | 既に処理済みだったか |

### 処理

1. processed_webhook_eventsにevent_idが存在するかチェック
2. 存在する場合 → already_processed=true, 現在の残高を返却
3. 存在しない場合:
   - processed_webhook_eventsにINSERT
   - creditsテーブルのpaid_creditsにamountを加算
   - credit_transactionsにtype='purchase'で履歴を記録
4. 更新後の残高を返却

### 冪等性

- event_idで重複処理を防止

### 参照テーブル

- mcpist.processed_webhook_events（挿入）
- mcpist.credits（更新）
- mcpist.credit_transactions（挿入）

---

## reset_free_credits

月初に無料クレジットを補充する（Cron用）。

### インタラクション

| 項目 | 内容 |
|------|------|
| ID | - |
| 呼び出し元 | Cron（Supabase pg_cron） |
| トリガー | 毎月1日 00:00 UTC |

### 入力

なし

### 出力

| フィールド | 型 | 説明 |
|-----------|-----|------|
| updated_count | INTEGER | 更新されたユーザー数 |

### 処理

1. usersテーブルでaccount_status='active'のユーザーを対象
2. creditsテーブルのfree_creditsを1000に更新
3. credit_transactionsにtype='monthly_reset'で履歴を記録（free_credits < 1000だったユーザーのみ）
4. 更新件数を返却

### 冪等性

- free_credits = 1000に設定するため、何度実行しても同じ結果
- 既にfree_credits = 1000のユーザーは実質変化なし

### 参照テーブル

- mcpist.users（参照）
- mcpist.credits（更新）
- mcpist.credit_transactions（挿入）

---

## credit_transactions記録形式

### type別の記録内容

| type | amount | module | tool | request_id | task_id |
|------|--------|--------|------|------------|---------|
| consume | -1 | 'notion' | 'search' | 'req-xxx' | NULL or 'task-xxx' |
| purchase | +500 | NULL | NULL | NULL | NULL |
| monthly_reset | +N | NULL | NULL | NULL | NULL |

**注:**
- amountは消費時は負、加算時は正で記録
- monthly_resetのamountは補充量（1000 - 旧free_credits）

---

## 関連ドキュメント

| ドキュメント | 内容 |
|-------------|------|
| [dsn-rpc.md](./dsn-rpc.md) | RPC関数設計書（概要） |
| [dtl-dsn-tbl.md](./dtl-dsn-tbl.md) | テーブル詳細設計書 |
| [spc-tbl.md](../specification/spc-tbl.md) | テーブル仕様書 |
| [itr-dst.md](../specification/interaction/itr-dst.md) | Data Store インタラクション仕様 |
| [dtl-spc-credit-model.md](../specification/dtl-spc/dtl-spc-credit-model.md) | クレジットモデル詳細仕様 |
