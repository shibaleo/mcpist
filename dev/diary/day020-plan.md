# DAY020 計画

## 日付

2026-01-31

---

## 概要

Sprint-006 4日目。DAY019でRPC設計・マイグレーション統合が完了。本日はRPC名変更に伴うコード更新と仕様書整備を行う。

---

## DAY019 の成果（振り返り）

| 完了タスク | 備考 |
|------------|------|
| Stripe Phase 1 完了 | Checkout + Webhook + billing UI |
| 初回クレジット付与（Signup Bonus） | pre_active → active 遷移 |
| オンボーディングフロー改善 | tools step 削除、残高アラート追加 |
| RPC設計・マイグレーション統合 | Canvas更新、RPCリネーム、prompts RPC作成 |

### 教訓（day019-review.md より）

- テーブル設計とRPC設計で命名規則・抽象度を揃えるのが困難
- RPC設計時に「誰が呼ぶのか」を最初に決めないとカオス化する
- 呼び出し元に応じた命名規則: `_my_` (Console User) / `_user_` (Router/API Server)

---

## 本日のタスク

### Phase 1: RPC変更に伴うコード更新（必須）

| ID | タスク | 見積 | 備考 |
|----|--------|------|------|
| D20-001 | database.types.ts 再生成 | 0.5h | `supabase gen types` 実行 |
| D20-002 | Console: RPC名変更対応 | 1h | `add_user_credits`, `complete_user_onboarding` |
| D20-003 | MCP Server: RPC名変更対応 | 0.5h | `consume_user_credits` |
| D20-004 | E2Eテスト実行 | 0.5h | 変更後の動作確認 |

**Phase 1 合計: 2.5h**

### Phase 2: 仕様書整備（spc-itf.md）

| ID | タスク | BL ID | 見積 | 備考 |
|----|--------|-------|------|------|
| D20-005 | JWT `aud` チェック要件整理 | BL-011 | 0.5h | 実装では明示チェックなし |
| D20-006 | MCP 拡張エラーコード整理 | BL-012 | 0.5h | JSON-RPC 標準コードのみに更新 |
| D20-007 | Console API 設計更新 | BL-013 | 1h | REST API → Supabase RPC 方式 |
| D20-008 | PSP Webhook 仕様整理 | BL-014 | 0.5h | Phase 1 実装に合わせて更新 |

**Phase 2 合計: 2.5h**

### Phase 3: 設計書作成（stretch）

| ID | タスク | 見積 | 備考 |
|----|--------|------|------|
| D20-009 | Observability 設計書作成 | 2h | dsn-observability.md |

**Phase 3 合計: 2h**

---

## RPC変更対応の詳細

### 変更前 → 変更後

| 旧RPC名 | 新RPC名 | 呼び出し元 |
|---------|---------|------------|
| `consume_credits` | `consume_user_credits` | API Server (MCP Server) |
| `add_credits` | `add_user_credits` | Console (Router) |
| `complete_onboarding` | `complete_user_onboarding` | Console (Router) |
| `get_my_preferences` | `get_my_settings` | Console (User) |
| `update_my_preferences` | `update_my_settings` | Console (User) |

### 影響ファイル（予想）

| コンポーネント | ファイル | 変更内容 |
|----------------|----------|----------|
| Console | `app/api/stripe/webhook/route.ts` | `add_user_credits` |
| Console | `app/api/onboarding/complete/route.ts` | `complete_user_onboarding` |
| Console | `lib/supabase/client.ts` or similar | RPC呼び出し箇所 |
| Console | `database.types.ts` | 型定義再生成 |
| MCP Server | `internal/supabase/credits.go` | `consume_user_credits` |

---

## 仕様書整備の詳細

### BL-011: JWT `aud` チェック

| 項目 | 現状 | 対応 |
|------|------|------|
| 仕様 | 必須チェック | 「推奨」に変更 |
| 実装 | チェックなし | 将来実装として明記 |

### BL-012: MCP 拡張エラーコード

| 項目 | 現状 | 対応 |
|------|------|------|
| 仕様 | 2001-2005 定義 | 削除（JSON-RPC標準のみ使用） |
| 実装 | JSON-RPC標準のみ | 変更なし |

### BL-013: Console API 設計

| 項目 | 現状 | 対応 |
|------|------|------|
| 仕様 | REST API `/api/v1/*` | Supabase RPC 直接呼び出しに更新 |
| 実装 | Supabase RPC 使用 | 変更なし |

### BL-014: PSP Webhook 仕様

| 項目 | 現状 | 対応 |
|------|------|------|
| 仕様 | 詳細設計あり | Phase 1 実装に合わせて更新 |
| 実装 | Phase 1 完了 | 変更なし |

---

## 完了条件

- [ ] database.types.ts が最新RPC定義を反映
- [ ] Console/MCP Serverのコードが新RPC名を使用
- [ ] E2Eテストが通過
- [ ] BL-011〜014 が resolved（spc-itf.md 更新完了）
- [ ] （stretch）dsn-observability.md 作成完了

---

## タイムライン

| 時間帯 | タスク |
|--------|--------|
| 午前 | Phase 1: RPC変更対応 (D20-001〜004) |
| 午後前半 | Phase 2: 仕様書整備 (D20-005〜008) |
| 午後後半 | Phase 3: 設計書作成 (D20-009) or バックログ消化 |

---

## 参考

- [day020-backlog.md](day020-backlog.md) - バックログ
- [day019-review.md](./day019-review.md) - 前日レビュー
- [day019-worklog.md](./day019-worklog.md) - 前日作業ログ
- [grh-rpc-design.canvas](../docs/graph/grh-rpc-design.canvas) - RPC設計図
