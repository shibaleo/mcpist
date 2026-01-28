# DAY017 計画

## 日付

2026-01-28

---

## 概要

Sprint-006 初日。Stripe 決済連携の実装着手と、3日間先送りしてきた仕様書の実装追従更新を並行して進める。

---

## DAY016の成果（振り返り）

| 完了タスク | 備考 |
|------------|------|
| サービス接続時のデフォルトツール設定自動保存 | BL-001。MCP Annotations 移行含む |
| next.config.ts デバッグログ削除 | B-006 |
| VitePress docs ビルド修正 | rewrites でクリーンURL対応 |
| get_module_schema 複数モジュール対応 + ツールフィルタリング | 配列入力、DisabledTools 除外 |
| Observability — Loki 統合 + X-Request-ID トレーシング | 構造化ログ、エンドツーエンドトレース |
| Batch 権限チェック + クレジット残高事前検証 | All-or-Nothing、セキュリティログ |
| Go Server MCP Tool Annotations 実装 | 115ツール annotations |
| Console /mcp → /connections リネーム | ルート名の混同回避 |

### DAY016 からの引き継ぎ

- 仕様書の実装追従更新（D16-002 + BL-010〜014）: 3日連続先送り。Sprint-006 で最優先
- E2E テスト設計（D16-006）: 未着手
- CI に tools.json 検証追加（S5-076）: 未着手

---

## 本日のタスク

### 優先度: 高

| ID      | タスク               | Sprint ID  | 備考                                                                 |
| ------- | ----------------- | ---------- | ------------------------------------------------------------------ |
| D17-001 | 仕様書の実装追従更新        | S6-010〜017 | BL-010〜014 の5項目修正 + Tool Sieve / Observability / Annotations 設計書作成 |
| D17-002 | Stripe 商品・価格設定    | S6-001     | Stripe Dashboard で Product / Price 作成                              |
| D17-003 | サインアップ時の無料クレジット付与 | S6-002     | ユーザー作成時に free_credits=1000 を DB に反映                                |

### 優先度: 中

| ID | タスク | Sprint ID | 備考 |
|----|--------|-----------|------|
| D17-004 | Checkout Session API 実装 | S6-003 | Stripe Checkout Session 作成 Edge Function |
| D17-005 | Webhook ハンドラ実装 | S6-004 | checkout.session.completed → add_paid_credits。冪等性保証 |

### 優先度: 低

| ID | タスク | Sprint ID | 備考 |
|----|--------|-----------|------|
| D17-006 | E2E テスト設計書作成 | S6-020 | テスト対象フロー、テスト環境、実行方法 |

---

## D17-001 詳細: 仕様書の実装追従更新

### 仕様・実装差分の修正（BL-010〜014）

| BL ID | 対象ファイル | 更新内容 | 方針 |
|-------|-------------|----------|------|
| BL-010 | spc-dsn.md | Rate Limit 記述 | 実装では削除済み。「将来実装予定」に変更 |
| BL-011 | spc-itf.md | JWT `aud` チェック要件 | 実装では明示チェックなし。現状に合わせて修正 |
| BL-012 | spc-itf.md | MCP 拡張エラーコード (2001-2005) | JSON-RPC 標準コードのみに更新 |
| BL-013 | spc-itf.md | Console API 設計 | REST API → Supabase RPC 直接呼び出し方式に更新 |
| BL-014 | spc-itf.md | PSP Webhook 仕様 | Phase 1 実装（Stripe）に合わせて更新 |

### 新規設計書作成（S6-015〜017）

| Sprint ID | 成果物 | 内容 |
|-----------|--------|------|
| S6-015 | dsn-permission-system.md | Tool Sieve 設計（Layer 1: Filter / Layer 2: Guard / Layer 3: Gate） |
| S6-016 | dsn-observability.md | Loki, X-Request-ID, ログ種別、セキュリティイベントの設計 |
| S6-017 | spc-itf.md 追記 | MCP Tool Annotations の `annotations` フィールド仕様 |

---

## D17-002〜005 詳細: Stripe 決済連携

### 実装フロー

```
[D17-002] Stripe Dashboard: Product + Price 設定
    ↓
[D17-003] サインアップ → reset_free_credits RPC → free_credits=1000
    ↓
[D17-004] Billing ページ「購入」→ Edge Function → Stripe Checkout Session
    ↓
[D17-005] Stripe Webhook → checkout.session.completed → add_paid_credits RPC
```

### 既存資産

| 資産 | 状態 | 備考 |
|------|------|------|
| dsn-billing.md | 設計済み | Checkout フロー設計 |
| dtl-spc-credit-model.md | 設計済み | クレジットモデル仕様 |
| credits テーブル | 作成済み | free_credits, paid_credits |
| credit_transactions テーブル | 作成済み | 取引履歴 |
| add_paid_credits RPC | 作成済み | 有料クレジット追加 |
| reset_free_credits RPC | 作成済み | 無料クレジットリセット |
| consume_credit RPC | 作成済み | クレジット消費 |

### 変更ファイル（見込み）

| ファイル | 変更内容 |
|----------|----------|
| Supabase Edge Function (新規) | create-checkout-session |
| Supabase Edge Function (新規) | stripe-webhook |
| Supabase Auth Hook or Trigger | サインアップ時 free_credits 付与 |
| apps/console Billing ページ | 購入ボタン → Checkout Session API 呼び出し |

---

## 完了条件

- [ ] BL-010〜014 の5項目が全て resolved
- [ ] Tool Sieve 設計書 (dsn-permission-system.md) が作成されている
- [ ] Observability 設計書 (dsn-observability.md) が作成されている
- [ ] Stripe Dashboard に Product / Price が設定されている
- [ ] サインアップ時に free_credits=1000 が自動付与される
- [ ] (stretch) Checkout Session API が実装されている

---

## 参考

- [day016-review.md](day016-review.md) - 前日振り返り
- [day016-backlog.md](day016-backlog.md) - 前日バックログ
- [day016-worklog.md](day016-worklog.md) - 前日作業ログ
- [sprint006.md](../sprint/sprint006.md) - Sprint-006 計画書
- [day016-spec-impl-compare.md](./day016-spec-impl-compare.md) - 仕様・実装差分
