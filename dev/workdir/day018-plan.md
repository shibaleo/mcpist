# DAY018 計画

## 日付

2026-01-29

---

## 概要

Sprint-006 2日目。Phase 2（仕様書整備）の完了を優先する。dtl-itr draft 5件のレビューと、BL-010〜014 の残項目を解消し、仕様書の整合性を確保する。

DAY017 の教訓: 仕様書整備とStripe実装の並行は非現実的。仕様書整備を完了させてから Phase 1 に着手する。

---

## DAY017 の成果（振り返り）

| 完了タスク | 備考 |
|------------|------|
| spc-dsn.md v2.0 更新 | Go 1.23+, Next.js 15, Render Primary 等 |
| spc-inf.md v3.0 更新 | 8レイヤー構成に刷新 |
| grh-infrastructure.canvas 作成 | インフラ構成図 |
| interaction/ 整合性検証・整備 | 25ファイル、7件新規、4件削除 |
| dtl-itr-*.md 9件更新 | 20/25 reviewed |
| canvas ノード/エッジID整理 | コンポーネント略称形式に統一 |

### DAY017 からの引き継ぎ

| タスク | 状態 | 備考 |
|--------|------|------|
| dtl-itr draft 5件レビュー | **本日実施** | DST-GWY, DST-PSP, DST-SSM, DST-TVL, GWY-OBS |
| BL-011〜014 解消 | **本日実施** | spc-itf.md 更新 |
| Observability 設計書作成 | **本日実施** | dsn-observability.md |
| Stripe Phase 1 | DAY019 以降 | 仕様書整備完了後に着手 |

---

## 本日のタスク

### 優先度: 高（Phase 2 完了）

| ID | タスク | Sprint ID | 備考 |
|----|--------|-----------|------|
| D18-001 | dtl-itr-DST-GWY.md レビュー | S6-017 | ✅ 完了（APIキー検証の期待動作を整理） |
| D18-002 | dtl-itr-DST-PSP.md レビュー | S6-017 | ✅ 完了（未実装前提で期待動作を明文化） |
| D18-003 | dtl-itr-DST-SSM.md レビュー | S6-017 | ✅ 完了（登録/参照の期待動作を明文化） |
| D18-004 | dtl-itr-DST-TVL.md レビュー | S6-017 | ✅ 完了（紐づけ中心の期待動作に整理） |
| D18-005 | dtl-itr-GWY-OBS.md レビュー | S6-017 | ✅ 完了（将来ログ仕様を詳細化） |

### 優先度: 中（spc-itf.md 更新）

| ID | タスク | Sprint ID | 備考 |
|----|--------|-----------|------|
| D18-006 | BL-011 JWT `aud` チェック要件整理 | S6-011 | 実装では明示チェックなし。方針決定・記載 |
| D18-007 | BL-012 MCP 拡張エラーコード整理 | S6-012 | JSON-RPC 標準コードのみに更新 |
| D18-008 | BL-013 Console API 設計更新 | S6-013 | REST API → Supabase RPC 方式 |
| D18-009 | BL-014 PSP Webhook 仕様整理 | S6-014 | Phase 1 実装に合わせて更新 |

### 優先度: 低（stretch）

| ID | タスク | Sprint ID | 備考 |
|----|--------|-----------|------|
| D18-010 | Observability 設計書作成 | S6-016 | dsn-observability.md |

---

## D18-001〜005 詳細: dtl-itr draft レビュー

| ID | ファイル | 連携内容 | レビュー観点 |
|----|----------|----------|-------------|
| D18-001 | dtl-itr-DST-GWY.md | APIキー検証 | store/api_keys.go、middleware/auth.go との整合 |
| D18-002 | dtl-itr-DST-PSP.md | Webhook受信 | processed_webhook_events、credit_transactions との整合 |
| D18-003 | dtl-itr-DST-SSM.md | ユーザー情報登録・参照 | Supabase Auth users テーブルとの関係 |
| D18-004 | dtl-itr-DST-TVL.md | ユーザー紐付け | Vault secrets の user_id 外部キー |
| D18-005 | dtl-itr-GWY-OBS.md | HTTPリクエストログ | observability/loki.go のログフィールド |

### レビュー手順

1. 仕様書を読み取り
2. 対応する実装コードを確認
3. 差異がある場合は判断（仕様修正 or 実装不足としてバックログ追加）
4. Status: `reviewed`、Version: v2.0 に更新

---

## D18-006〜009 詳細: spc-itf.md 更新

| BL ID | 項目 | 現状 | 対応方針 |
|-------|------|------|----------|
| BL-011 | JWT `aud` チェック | 仕様: 必須。実装: チェックなし | 実装に合わせて「推奨」に変更 or 将来実装として明記 |
| BL-012 | MCP 拡張エラーコード | 仕様: 2001-2005。実装: JSON-RPC 標準のみ | 仕様から拡張コード削除 |
| BL-013 | Console API 設計 | 仕様: REST API。実装: Supabase RPC 直接 | Supabase RPC 方式に更新 |
| BL-014 | PSP Webhook 仕様 | 仕様: 詳細設計あり。実装: 未実装 | Phase 1 実装予定として更新 |

---

## 完了条件

- [x] dtl-itr 全25件が reviewed（draft 0件）
- [ ] BL-011〜014 が resolved
- [ ] spec-impl-compare.md の Phase 2 関連項目がゼロ

---

## 参考

- [day017-review.md](./day017-review.md) - 前日振り返り
- [day017-backlog.md](./day017-backlog.md) - 前日バックログ
- [day017-worklog.md](./day017-worklog.md) - 前日作業ログ
- [sprint006.md](../sprint/sprint006.md) - Sprint-006 計画書
