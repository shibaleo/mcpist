# DAY026 バックログ（やり残し）

## 日付

2026-02-11

---

## 未完了タスク

### 1. ヘルパー共通化コミット

| タスク | 状態 | 備考 |
|--------|------|------|
| helpers.go + 4 module.go + dsn-modules.md + asana.md のコミット | ❌ 未コミット | ビルド・テスト・tools.json diff なし確認済み |

**未コミットファイル:**
- `apps/server/internal/modules/helpers.go` (新規)
- `apps/server/internal/modules/github/module.go` (変更)
- `apps/server/internal/modules/supabase/module.go` (変更)
- `apps/server/internal/modules/grafana/module.go` (変更)
- `apps/server/internal/modules/asana/module.go` (変更)
- `docs/003_design/modules/dsn-modules.md` (変更)
- `docs/003_design/modules/asana.md` (新規)

### 2. Sprint 007 Phase 3: 仕様書の実装追従更新 (S7-020〜026)

| ID | タスク | 状態 | 備考 |
|----|--------|------|------|
| S7-020 | spc-dsn.md Rate Limit 記述更新 | ❌ 未着手 | BL-010: 「将来実装予定」に変更 |
| S7-021 | spc-itf.md JWT `aud` チェック要件整理 | ❌ 未着手 | BL-011 |
| S7-022 | spc-itf.md MCP 拡張エラーコード整理 | ❌ 未着手 | BL-012 |
| S7-023 | spc-itf.md Console API 設計更新 | ❌ 未着手 | BL-013 |
| S7-024 | spc-itf.md PSP Webhook 仕様整理 | ❌ 未着手 | BL-014 |
| S7-025 | dtl-itr-MOD-TVL.md credentials JSON 構造整理 | ❌ 未着手 | BL-090 |
| S7-026 | spec-impl-compare.md 更新 | ❌ 未着手 | D16-002 |

### 3. クレジットモデル仕様書更新

| タスク | 状態 | 備考 |
|--------|------|------|
| dtl-spc-credit-model.md をランニングバランス方式に更新 | ❌ 未着手 | credits テーブル廃止・running balance 移行を反映 |

### 4. Sprint 007 Phase 4: 機能実装

| ID | タスク | 状態 | 備考 |
|----|--------|------|------|
| S7-030 | usage_stats 参照 API 実装 | ❌ 未着手 | BL-017: Console で使用量表示 |
| S7-031 | enabled_modules 参照 API 完成 | ❌ 未着手 | BL-015: 残作業完了 |

### 5. Grafana ダッシュボード改善

| タスク | 状態 | 備考 |
|--------|------|------|
| アラート設定 | ❌ 未着手 | エラーレート閾値のアラートルール作成 |
| パネル改善 | ❌ 未着手 | 必要に応じて追加パネル |

### 6. 次の ogen 移行候補

| モジュール | ツール数 | OpenAPI spec | 優先度 |
|-----------|---------|-------------|--------|
| Jira | 11 | OpenAPI 3.0 (公式) | 高 |
| Confluence | 12 | Atlassian OpenAPI | 高 |
| Microsoft Todo | 8 | Graph API OpenAPI | 中 |

---

## 優先度

| 優先度 | タスク |
|--------|--------|
| 高 | ヘルパー共通化コミット（未コミット変更あり） |
| 高 | Jira module ogen 移行 |
| 中 | 仕様書更新 (S7-020〜026) |
| 中 | クレジットモデル仕様書更新 |
| 低 | Grafana ダッシュボード改善 |
| 低 | usage_stats / enabled_modules API |
