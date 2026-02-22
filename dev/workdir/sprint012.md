# Sprint 012 計画書

## 基本情報

| 項目 | 値 |
|------|-----|
| スプリント番号 | SPRINT-012 |
| 期間 | 2026-02-23 〜 2026-03-01 (7日間) |
| マイルストーン | M11: 仕様整備 + テスト基盤 + 法務ページ |
| 前提 | Sprint 011 完了 (セキュリティ強化: API キー失効、OAuth state HMAC、資格情報暗号化、SSRF 対策) |

---

## Sprint 目標

**仕様書を現行実装に追従させ、テスト基盤を整備する**

Sprint 010 で大規模アーキテクチャ刷新 (Supabase → Neon/GORM, Clerk 移行, Render 移行) を完了し、Sprint 011 でセキュリティ強化を完了した。しかし仕様書は Sprint 006 時点の設計のまま大きく乖離している。仕様を先に整備し、正確な仕様に基づいてテストを書く。

---

## タスク一覧

### Phase 1: 仕様書リファクタリング (優先度: 高)

**背景:** 仕様書が Supabase/PostgREST/Koyeb 時代の記述のままで、現行の Neon/GORM/Clerk/Render 構成と大幅に乖離している。テストを書く前に SSoT を確立する。

| ID | タスク | 対象ファイル | 主な変更点 |
|----|--------|-------------|-----------|
| S12-001 | テーブル仕様書の更新 | `docs/002_specification/spc-tbl.md` | 現行 DB 定義 (GORM models.go) に合わせる。mcpist スキーマ、processed_webhook_events 等の追加テーブル反映 |
| S12-002 | 課金設計書の更新 | `docs/003_design/details/dsn-billing.md` | Supabase Edge Function → Worker proxy + Go Server 構成に変更。Stripe Webhook フロー更新 |
| S12-003 | サブスクリプション設計書の更新 | `docs/003_design/details/dsn-subscription.md` | 5 プラン → Free/Plus 2 プラン。Koyeb/Fly.io → Render。使用量制御の現行実装反映 |
| S12-004 | クレジットモデル仕様書の更新 | `docs/002_specification/details/dtl-spc-credit-model.md` | Supabase RPC → GORM 実装。consume/add の現行フロー反映 |
| S12-005 | インターフェース仕様書の更新 | `docs/002_specification/spc-itf.md` | ogen 自動生成ベースの API エンドポイント一覧に更新 |

---

### Phase 2: テスト基盤 (優先度: 高)

**背景:** Sprint 011 Phase 3 からの繰越し。Phase 1 で整備した仕様に基づいてテストを記述する。

| ID | タスク | 変更対象 | 備考 |
|----|--------|---------|------|
| S12-010 | authz middleware ユニットテスト | `middleware/authz_test.go` (新規) | CanAccessTool, WithinDailyLimit |
| S12-011 | broker/user.go ユニットテスト | `broker/user_test.go` (新規) | GetUserContext, RecordUsage |
| S12-012 | crypto パッケージ ユニットテスト | `internal/crypto/crypto_test.go` (新規) | Encrypt/Decrypt, キーローテーション |
| S12-013 | CI トリガーを push/PR に変更 | `.github/workflows/ci.yml` | workflow_dispatch → push + pull_request |
| S12-014 | `go test ./...` の全パス確認 | `apps/server/` | ビルド通過 + 既存テストのパス |

---

### Phase 3: 法務ページ (優先度: 中)

**背景:** サービス公開に向けて必須の法務ページ。Jira backlog MCPIST-15〜17。

| ID | タスク | 変更対象 | 備考 |
|----|--------|---------|------|
| S12-020 | Privacy Policy ページ | `apps/console/` | MCPIST-15 |
| S12-021 | Terms of Service ページ | `apps/console/` | MCPIST-16 |
| S12-022 | Security Policy ページ | `apps/console/` | MCPIST-17 |

---

### Phase 4: バッファ (優先度: 低、時間があれば)

| ID | タスク | 備考 |
|----|--------|------|
| S12-030 | キーバージョニング対応 (MCPIST-13) | 暗号化キーのローテーション。S11-025 繰越し |
| S12-031 | Worker キャッシュ KV 移行検討 | RV-009/010 の課題。設計のみ |

---

## 作業順序

```
Day 1:   Phase 1 (S12-001〜002) — テーブル仕様 + 課金設計
Day 2:   Phase 1 (S12-003〜005) — サブスク + クレジット + IF 仕様
Day 3:   Phase 2 (S12-010〜012) — ユニットテスト作成
Day 4:   Phase 2 (S12-013〜014) — CI 設定 + 全テストパス
Day 5:   Phase 3 (S12-020〜022) — 法務ページ 3 件
Day 6-7: バッファ + Phase 4
```

---

## リスク

| リスク | 影響 | 対策 |
|--------|------|------|
| 仕様書の乖離が想定以上に大きい | Phase 1 がオーバーフローし Phase 2 に食い込む | 仕様書は「現行実装の正確な記述」に徹し、将来設計の議論は含めない |
| テスト対象コードの依存関係が複雑 | モック作成に時間がかかる | インターフェース境界が明確な crypto パッケージから着手 |
| 法務ページの内容確定 | 法律的な正確性 | テンプレートベースで最小限の内容で作成、後から法務レビュー |

---

## 完了条件

- [ ] 仕様書 5 件が現行実装と整合している
- [ ] authz middleware のユニットテストが pass
- [ ] crypto パッケージのユニットテストが pass
- [ ] CI が push/PR トリガーで自動実行される
- [ ] `go test ./...` が全パスする
- [ ] Privacy Policy / Terms of Service / Security Policy ページが Console に存在する

---

## 参考

- [sprint011.md](sprint011.md) - Sprint 011 計画書
- [day035.md](day035.md) - Sprint 011 作業ログ (クロージング含む)
- [review.jsonl](review.jsonl) - レビュー・知見ログ
- [backlog.csv](backlog.csv) - Jira 同期バックログ
