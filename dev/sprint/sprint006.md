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

### Phase 3: テスト基盤構築

| ID | タスク | 成果物 | 備考 |
|----|--------|--------|------|
| S6-020 | E2E テスト設計書作成 | tst-e2e.md | テスト対象フロー、テスト環境、実行方法 |
| S6-021 | Go Server ユニットテスト | *_test.go | handler, middleware, modules の主要パス |
| S6-022 | Batch 権限チェックのテスト | handler_test.go | All-or-Nothing、クレジット不足、正常系 |
| S6-023 | Console ビルドテスト | CI workflow | `pnpm next build` が CI で通ることを保証 |

### Phase 4: CI/CD 整備

| ID | タスク | 成果物 | 備考 |
|----|--------|--------|------|
| S6-030 | tools.json 検証 CI | GitHub Actions | Go tools-export 出力と Console tools.json の差分チェック |
| S6-031 | Go lint + test CI | GitHub Actions | golangci-lint + go test |
| S6-032 | Console lint + build CI | GitHub Actions | eslint + next build |

### Phase 5: コード品質改善

| ID | タスク | 備考 |
|----|--------|------|
| S6-040 | 未使用 Go モジュール削除 | S5-077 からの引き継ぎ |
| S6-041 | spec-impl-compare.md の更新 | Phase 2 完了後に差分再チェック |

### Phase 6: Sprint-005 残タスク

| ID | タスク | 備考 |
|----|--------|------|
| S6-050 | UI 要求仕様書作成 (spc-ui.md) | S5-060。画面一覧・機能要件 |
| S6-051 | ユーザーフロー図作成 | S5-061。主要フローの可視化 |
| S6-052 | 画面遷移図作成 | S5-062。認証後のナビゲーション |

---

## 優先度

| 優先度 | Phase | 理由 |
|--------|-------|------|
| **高** | Phase 1: Stripe 連携 | コア機能。課金なしではサービスとして成立しない |
| **高** | Phase 2: 仕様書更新 | 仕様の信頼性回復。乖離が放置されると全ドキュメントの価値が低下 |
| **高** | Phase 3: テスト基盤 | リグレッション検出の仕組みがないまま機能追加を続けるのは危険 |
| **中** | Phase 4: CI/CD | テストを自動実行する基盤。Phase 3 と並行して進められる |
| **低** | Phase 5: コード品質 | クリーンアップ。他タスクの合間に実施 |
| **低** | Phase 6: UI 仕様 | 実装が安定してから仕様化する方が効率的 |

---

## 完了条件

### Phase 1: Stripe 連携
- [ ] サインアップ時に free_credits=1000 が自動付与される
- [ ] Stripe Checkout でクレジット購入ができる
- [ ] Webhook で決済完了 → paid_credits に反映される（冪等）
- [ ] Billing ページで残高と取引履歴が確認できる
- [ ] ツール実行 → クレジット消費 → 不足時拒否の一連フローが動作する

### Phase 2: 仕様書更新
- [ ] BL-010〜014 の5項目が全て resolved（修正 or「将来実装予定」明記）
- ~~Tool Sieve 設計書が存在する~~ （削除: セキュリティ仕様のスコープ）
- [ ] Observability 設計書が存在する
- [ ] spec-impl-compare.md の未解決項目がゼロ

### Phase 3: テスト基盤
- [ ] E2E テスト設計書が存在する
- [ ] Go Server の handler / middleware に最低限のユニットテストが存在する
- [ ] Console の `next build` が CI で自動実行される

### Phase 4: CI/CD
- [ ] PR 作成時に Go lint + test が自動実行される
- [ ] PR 作成時に Console build が自動実行される
- [ ] tools.json の整合性が CI で検証される

---

## 継続バックログ

Sprint-006 のスコープ外だが、記録しておくタスク。

| ID | タスク | 状態 | 備考 |
|----|--------|------|------|
| BL-002 | 切断時のツール設定クリーンアップ | 保留 | DB migration 前提 |
| BL-030 | Stg/Prd 環境構築 | 未着手 | Blue-Green 方式 |
| BL-032 | 追加モジュール（Slack/Linear） | 未着手 | 需要に応じて |

---

## 参考

- [sprint005-review.md](./sprint005-review.md) - Sprint-005 レビュー
- [sprint005.md](./sprint005.md) - Sprint-005 計画書
- [day016-backlog.md](../workdir/day016-backlog.md) - DAY016 バックログ（引き継ぎ候補）
- [sprint004.md](./sprint004.md) - Sprint-004 計画書（仕様書作成フェーズ）
