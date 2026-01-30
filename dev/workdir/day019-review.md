# DAY019 レビュー・設計メモ

## クレジット実装方針

### 決定事項

- **都度購入モデル**で進める（サブスクリプションは後回し）
- Stripe サンドボックスで **0円商品** を作成して購入フローを実装

### 理由

1. **安定供給できるか不明**
   - MCPサーバーの運用コストが読めない
   - API呼び出し頻度に応じたコスト変動
   - インフラ負荷の予測が困難

2. **UXの一貫性は維持したい**
   - ユーザーにはStripe Checkout経由で購入させる
   - テスト期間中は0円だが、本番では有料化予定
   - 購入→クレジット付与のフローを先に確立しておく

### 実装詳細

- Stripe Product: "MCPist Credits (100)"
- Stripe Price: $0.00
- Webhook: `checkout.session.completed` → `add_credits` RPC
- 冪等性: `processed_webhook_events` テーブルで event_id 重複チェック

### 将来の拡張

- 有料化時は Price ID を本番用に差し替え
- サブスクリプション対応は需要を見て判断
- 使用量ベースの課金も検討可能（Stripe Metered Billing）

---

## RPC設計の反省

### 問題点

#### 1. 命名規則・抽象度の統一が難しい

テーブル設計とRPC設計の間で、命名規則や抽象度を揃えるのが困難だった。

**例：テーブル名とRPC名の不一致**
- テーブル: `prompts` → RPC: `list_my_prompts`, `upsert_my_prompt`
- テーブル: `api_keys` → RPC: `list_my_api_keys`, `create_api_key`
- テーブル: `user_credentials` → RPC: `get_user_credential`, `upsert_user_credential`

**例：カラム名の揺れ**
- `settings` vs `preferences`（最終的に `settings` に統一）
- `name` vs `display_name`（テーブルによって異なる）

**例：引数名の揺れ**
- `p_module_name` vs `p_module` vs `p_module_id`
- `p_prompt_id` vs `p_id`

#### 2. 呼び出し元を最初に考えないとカオス化する

RPC設計時に「誰が呼ぶのか」を明確にしないまま進めると、以下の問題が発生した：

| 問題 | 具体例 |
|------|--------|
| 認証方式の混在 | `auth.uid()` と `p_user_id` 引数の使い分けが不明確に |
| 権限管理の漏れ | service_role 専用RPCに authenticated が呼べてしまう設計 |
| 命名の不統一 | `_my_` と `_user_` プレフィックスの使い分けが後付けに |

### 呼び出し元の分類（今回整理した結果）

| 呼び出し元 | 環境 | 認証 | RPC命名 | 例 |
|------------|------|------|---------|-----|
| Console (User) | ブラウザ | authenticated + `auth.uid()` | `_my_` | `get_my_settings` |
| Console (Router) | Next.js Server | service_role + `p_user_id` | `_user_` | `add_user_credits` |
| Gateway | CF Worker | service_role | なし | `lookup_user_by_key_hash` |
| API Server | MCP Server | service_role + `p_user_id` | `_user_` | `consume_user_credits` |
| Trigger | DB | 内部 | `handle_` | `handle_new_user` |

### 教訓

1. **設計フェーズで呼び出し元を先に決める**
   - 各RPCが「誰から呼ばれるか」をCanvasに必ず記載
   - 呼び出し元に応じて認証方式と命名規則が自動的に決まる

2. **命名規則を最初に文書化する**
   - テーブル名・カラム名・RPC名・引数名のルールを先に決める
   - 例: `p_` prefix for parameters, `v_` prefix for variables

3. **CRUD操作の抽象度を統一する**
   - `list` / `get` / `create` / `update` / `delete` を基本形に
   - `upsert` は create + update の場合のみ使用

4. **レビュー時のチェックリスト**
   - [ ] 呼び出し元が明記されているか
   - [ ] 命名規則に従っているか
   - [ ] 認証方式が呼び出し元と一致しているか
   - [ ] テーブル名とRPC名の関係が分かりやすいか
