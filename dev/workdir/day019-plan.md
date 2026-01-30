# DAY019 計画

## 日付

2026-01-30

---

## 概要

Sprint-006 3日目。DAY018で tool_id + 多言語対応の実装・デプロイが完了。本日は仕様書整備（BL解消）を優先し、Phase 2 を完了させる。

---

## DAY018 の成果（振り返り）

| 完了タスク | 備考 |
|------------|------|
| dtl-itr draft 5件レビュー | D18-001〜005 完了 |
| tool_id + 多言語対応 | 全Phase完了、本番デプロイ済み |
| database.types.ts 更新 | RPC型定義を手動修正 |
| E2Eテスト | 言語切替、ツール無効化、batch実行確認 |

### DAY018 からの引き継ぎ

| タスク | 状態 | 備考 |
|--------|------|------|
| BL-011〜014 解消 | 未着手 | spc-itf.md 更新 |
| Observability 設計書作成 | 未着手 | dsn-observability.md |
| Stripe Phase 1 | ✅ **完了** | 本日実装・E2Eテスト完了 |

---

## 本日のタスク

### 優先度: 高（Phase 2 完了）

| ID | タスク | Sprint ID | 備考 |
|----|--------|-----------|------|
| D19-001 | BL-011 JWT `aud` チェック要件整理 | S6-011 | 実装では明示チェックなし。方針決定・記載 |
| D19-002 | BL-012 MCP 拡張エラーコード整理 | S6-012 | JSON-RPC 標準コードのみに更新 |
| D19-003 | BL-013 Console API 設計更新 | S6-013 | REST API → Supabase RPC 方式 |
| D19-004 | BL-014 PSP Webhook 仕様整理 | S6-014 | Phase 1 実装に合わせて更新 |

### 優先度: 中（設計書作成）

| ID | タスク | Sprint ID | 備考 |
|----|--------|-----------|------|
| D19-005 | Observability 設計書作成 | S6-016 | dsn-observability.md |

### 優先度: 低（stretch）

| ID | タスク | Sprint ID | 備考 |
|----|--------|-----------|------|
| D19-006 | database.types.ts 自動生成フロー整備 | BL-070 | CI/CDに組み込み検討 |

---

## Stripe 実装計画（Phase 1: 無料クレジット）

### 概要

$0 の Stripe Checkout で 100 クレジットを付与するフロー。

```
ユーザー → [無料クレジット取得] → Stripe Checkout ($0) → Webhook → 100クレジット付与
```

### 事前準備

- [x] Stripe アカウント作成
- [x] テストモード API キー取得（`STRIPE_SECRET_KEY`, `STRIPE_WEBHOOK_SECRET`）
- [x] Product/Price 作成（$0, metadata: credits=100）

### 実装タスク

| # | タスク | 見積もり | 状態 | 備考 |
|---|--------|----------|------|------|
| 1 | Stripe Product/Price 作成 | 0.5h | ✅ | Dashboard で作成 |
| 2 | DB マイグレーション | 0.5h | ✅ | 029-031: `stripe_customer_id`, RPC関数 |
| 3 | Checkout Session 作成 API | 1h | ✅ | `/api/stripe/checkout` |
| 4 | Webhook 処理 | 1.5h | ✅ | `/api/stripe/webhook` → クレジット付与 |
| 5 | Console UI | 1h | ✅ | billing ページ更新 |
| 6 | テスト | 0.5h | ✅ | E2E: 100クレジット付与確認 |

**合計: 5h → 完了**

### 環境変数（追加予定）

```
STRIPE_SECRET_KEY=sk_test_...
STRIPE_WEBHOOK_SECRET=whsec_...
NEXT_PUBLIC_STRIPE_PUBLISHABLE_KEY=pk_test_...
```

---

## D19-001〜004 詳細: spc-itf.md 更新

| BL ID | 項目 | 現状 | 対応方針 |
|-------|------|------|----------|
| BL-011 | JWT `aud` チェック | 仕様: 必須。実装: チェックなし | 実装に合わせて「推奨」に変更 or 将来実装として明記 |
| BL-012 | MCP 拡張エラーコード | 仕様: 2001-2005。実装: JSON-RPC 標準のみ | 仕様から拡張コード削除 |
| BL-013 | Console API 設計 | 仕様: REST API。実装: Supabase RPC 直接 | Supabase RPC 方式に更新 |
| BL-014 | PSP Webhook 仕様 | 仕様: 詳細設計あり。実装: 未実装 | Phase 1 実装予定として更新 |

---

## 完了条件

- [ ] BL-011〜014 が resolved
- [ ] dsn-observability.md 作成完了
- [ ] spec-impl-compare.md の Phase 2 関連項目がゼロ
- [x] Stripe Phase 1 実装完了

---

## バックログ（継続）

### 仕様書・実装差分（優先度: 低）

| ID | タスク | 備考 |
|----|--------|------|
| BL-015 | enabled_modules 参照API実装 | Console ツール設定で一部実装済み |
| BL-016 | user_prompts 管理UI実装 | |
| BL-017 | usage_stats 参照API実装 | |
| BL-018 | クレジット付与機能（CON→DST） | |
| BL-019 | ツール実行ログにuser_id追加 | |
| BL-020 | invalid_gateway_secretログ実装 | |
| BL-060 | RFC 8707 Resource Indicators 対応 | |
| BL-061 | クレジット初期化をDBトリガーからアプリ層へ移行 | |
| BL-070 | database.types.ts 自動生成フロー整備 | |

### Sprint-005 残タスク

| ID | タスク | 備考 |
|----|--------|------|
| S5-060 | UI要求仕様書作成 | 画面一覧・機能要件 |
| S5-061 | ユーザーフロー図作成 | 主要フローの可視化 |
| S5-062 | 画面遷移図作成 | 認証後のナビゲーション |
| S5-076 | CIにtools.json検証追加 | |
| S5-077 | 未使用Goモジュール削除 | |

---

## 参考

- [day018-review.md](./day018-review.md) - 前日振り返り
- [day018-backlog.md](./day018-backlog.md) - 前日バックログ
- [day018-worklog.md](./day018-worklog.md) - 前日作業ログ
- [sprint006.md](../sprint/sprint006.md) - Sprint-006 計画書
