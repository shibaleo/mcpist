# Sprint 006 計画書

## 基本情報

| 項目 | 値 |
|------|-----|
| スプリント番号 | SPRINT-006 |
| 期間 | 2026-01-28 〜 |
| マイルストーン | M5: Stripe連携・品質基盤・仕様整備 |
| 前提 | Sprint-005 完了（RPC基盤、OAuth基盤、8モジュール115ツール、Observability、セキュリティ多層防御） |

---

## Sprint 目標

**Stripe 決済連携の実装、仕様と実装の乖離解消、テスト基盤構築、CI/CD 整備**

Sprint-005 で機能基盤は揃ったが、課金（Stripe連携）はコア機能であり「作る」フェーズは終わっていない。Sprint-006 では Stripe 連携を実装しつつ、仕様書・テスト・CI の品質基盤を並行して構築する。

---

## 背景

### 未実装のコア機能

| 項目 | 状態 | 影響 |
|------|------|------|
| Stripe 決済連携 | 設計済み・未実装 | 課金なしではサービスとして成立しない |

既存の設計資産: dsn-billing.md（Checkout フロー設計）、dtl-spc-credit-model.md（クレジットモデル仕様）、credits/credit_transactions テーブル、add_paid_credits/reset_free_credits/consume_credit RPC

### Sprint-005 で蓄積した技術的負債

| 項目 | 状態 | リスク |
|------|------|--------|
| 仕様と実装の乖離 | BL-010〜014（5項目未修正） | 仕様書が信頼できない。新規メンバーが参照した際に混乱 |
| テスト基盤なし | E2E テスト未設計 | 回帰テストなしでリリースを継続している |
| CI/CD 不完全 | tools.json 検証なし | Go 側ツール定義変更時に Console 側との不整合を検出できない |
| UI 仕様なし | Phase 4 未着手 | 実装が先行し、画面の要件定義が存在しない |

---

## タスク一覧

### Phase 1: Stripe 決済連携

無料プランでのクレジット付与を起点に、Stripe Checkout → Webhook → DB反映の決済フローを実装する。

| ID | タスク | 成果物 | 備考 |
|----|--------|--------|------|
| S6-001 | Stripe 商品・価格設定 | Stripe Dashboard | Product（モジュール購読）、Price（¥500/月）の作成 |
| S6-002 | サインアップ時の無料クレジット付与 | RPC / Edge Function | ユーザー作成時に free_credits=1000 を DB に反映。既存 reset_free_credits RPC を活用 |
| S6-003 | Checkout Session API 実装 | Edge Function | Stripe Checkout Session を作成し、決済ページにリダイレクト |
| S6-004 | Webhook ハンドラ実装 | Edge Function | checkout.session.completed → add_paid_credits RPC。冪等性は processed_webhook_events で保証 |
| S6-005 | Billing ページ実装 | Console | モック → 実データ。残高表示、購入ボタン、取引履歴 |
| S6-006 | クレジット消費の動作検証 | テスト | ツール実行 → consume_credit → 残高減少 → 不足時拒否の一連フロー |

### Phase 2: 仕様書の実装追従更新

Sprint-004 で作成した仕様書群を現在の実装に合わせて更新する。

| ID | タスク | 対象ファイル | 備考 |
|----|--------|-------------|------|
| S6-010 | Rate Limit 記述の更新 | spc-dsn.md | **完了** 課金対象外・インフラ保護目的として維持 |
| S6-011 | JWT `aud` チェック要件の整理 | spc-itf.md | 実装では明示チェックなし。方針決定 |
| S6-012 | MCP 拡張エラーコード (2001-2005) の整理 | spc-itf.md | 実装では JSON-RPC 標準コードのみ |
| S6-013 | Console API 設計の更新 | spc-itf.md | REST API → Supabase RPC 直接呼び出し方式 |
| S6-014 | PSP Webhook 仕様の整理 | spc-itf.md | Phase 1 実装に合わせて更新 |
| S6-015 | ~~Tool Sieve 設計書の作成~~ | ~~dsn-permission-system.md~~ | **削除** セキュリティ仕様として扱うべき。設計仕様のスコープ外 |
| S6-016 | Observability 設計書の作成 | dsn-observability.md | Loki, X-Request-ID, ログ種別の設計を文書化 |
| S6-017 | MCP Tool Annotations 仕様反映 | spc-itf.md | annotations フィールドの仕様記述 |

**Phase 3〜6 は [sprint006-backlog.md](./sprint006-backlog.md) に移動**

---

## 完了条件

### Phase 1: Stripe 連携
- [x] サインアップ時に free_credits=1000 が自動付与される → **pre_active 状態で0、オンボーディング完了時に100クレジット付与に変更**
- [x] Stripe Checkout でクレジット購入ができる → **Phase 1: $0 Checkout で無料クレジット付与**
- [x] Webhook で決済完了 → paid_credits に反映される（冪等） → **free_credits として付与**
- [x] Billing ページで残高と取引履歴が確認できる
- [x] ツール実行 → クレジット消費 → 不足時拒否の一連フローが動作する

### Phase 2: 仕様書更新
- [ ] BL-010〜014 の5項目が全て resolved（修正 or「将来実装予定」明記）
- [ ] Observability 設計書が存在する
- [ ] spec-impl-compare.md の未解決項目がゼロ
- [x] dtl-itr-XXX-YYY.md 全25件が reviewed

---

## 進捗ログ

### 2026-01-28 (DAY017)

#### インタラクション仕様書の実装整合レビュー

dtl-itr-XXX-YYY.md ファイル25件を実装と比較し、整合性を確認。

| ステータス | 件数 | 内容 |
|-----------|------|------|
| `reviewed` | 20件 | 実装と整合確認済み |
| `draft` | 5件 | 未レビュー |

**reviewed (20件):**
dtl-itr-AMW-DST, AMW-GWY, AMW-HDL, AUS-CLO, AUS-DST, AUS-GWY, AUS-SSM, CLK-GWY, CLO-GWY, CON-DST, CON-EAS, CON-PSP, CON-SSM, CON-TVL, DST-HDL, EXT-MOD, HDL-MOD, HDL-OBS, IDP-SSM, MOD-TVL

**draft (5件):**
dtl-itr-DST-GWY, DST-PSP, DST-SSM, DST-TVL, GWY-OBS

#### 当日実施したレビュー

| ファイル | 結果 | バージョン |
|----------|------|-----------|
| dtl-itr-IDP-SSM.md | 差異なし | v1.0 → v2.0, reviewed |
| dtl-itr-MOD-TVL.md | 差異あり（許容） | v1.1 → v2.0, reviewed |

**dtl-itr-MOD-TVL.md の差異詳細:**
- Notionモジュールのリフレッシュ未実装（google_calendar/microsoft_todoは実装済み）
- oauth1/custom_header は定数定義のみ（使用モジュールなし）
- → 実装フェーズの問題であり仕様としては正しい

### 2026-01-29 (DAY018)

#### tool_id + 多言語対応

| タスク | 状態 |
|--------|------|
| MCP Server: tool_id 実装 | ✅ 完了 |
| MCP Server: 多言語対応（en-US/ja-JP） | ✅ 完了 |
| Console: 言語設定UI | ✅ 完了 |
| database.types.ts 更新 | ✅ 完了 |
| E2Eテスト | ✅ 完了 |

#### dtl-itr レビュー完了

残り5件（DST-GWY, DST-PSP, DST-SSM, DST-TVL, GWY-OBS）をレビュー完了。全25件 reviewed。

### 2026-01-30 (DAY019)

#### Stripe Phase 1 完了

| タスク | 状態 | 備考 |
|--------|------|------|
| S6-001 | ✅ 完了 | Stripe Product/Price ($0) 作成 |
| S6-002 | ✅ 完了 | サインアップ時 free_credits=0、オンボーディング完了時に100付与に変更 |
| S6-003 | ✅ 完了 | `/api/stripe/checkout` 実装 |
| S6-004 | ✅ 完了 | `/api/stripe/webhook` 実装（冪等性保証） |
| S6-005 | ✅ 完了 | Billing ページ更新（Signup Bonus カード追加） |
| S6-006 | ✅ 完了 | E2Eテスト完了 |

#### オンボーディングフロー改善

- tools step 削除（サービス選択のみに簡素化）
- 残高アラート追加（pre_active 時、残高50以下時）
- Signup Bonus 受け取りカード追加

#### RPC設計・マイグレーション統合

| 変更 | 内容 |
|------|------|
| マイグレーション統合 | 36ファイル → 9ファイル |
| RPC命名規則統一 | `_my_`=Console(User), `_user_`=Router/API Server |
| RPC名変更 | `consume_credits` → `consume_user_credits` 他 |
| 呼び出し元整理 | Gateway, Console Router, API Server の分類 |
| prompts RPC 作成 | `list_my_prompts`, `upsert_my_prompt`, `delete_my_prompt` |
| Canvas 更新 | grh-rpc-design.canvas に全変更反映 |

#### 教訓（day019-review.md）

- テーブル設計とRPC設計で命名規則・抽象度を揃えるのが困難
- RPC設計時に「誰が呼ぶのか」を最初に決めないとカオス化する
- 呼び出し元に応じた命名規則: `_my_` (Console User) / `_user_` (Router/API Server)

### 2026-01-31 (DAY020) - 予定

| タスク | 優先度 | 備考 |
|--------|--------|------|
| database.types.ts 再生成 | 高 | 新RPC反映 |
| Console/MCP Server RPC名変更対応 | 高 | コード修正 |
| BL-011〜014 解消 | 高 | spc-itf.md 更新 |
| Observability 設計書作成 | 中 | dsn-observability.md |

---

## 参考

- [sprint005-review.md](./sprint005-review.md) - Sprint-005 レビュー
- [sprint005.md](./sprint005.md) - Sprint-005 計画書
- [day016-backlog.md](day016-backlog.md) - DAY016 バックログ（引き継ぎ候補）
- [sprint004.md](./sprint004.md) - Sprint-004 計画書（仕様書作成フェーズ）
