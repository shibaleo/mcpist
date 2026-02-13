# DAY028 バックログ

## 日付

2026-02-13

---

## 完了タスク

### 1. ~~OAuth2 リフレッシュ共通化 + store → broker リネーム~~ ✅ 完了

**実績:** DAY028 で実施。+679/-1,306 行。11 モジュールの重複コード削除。ローカルテストで asana, airtable のリフレッシュ動作確認済み。

### 2. ~~batch リファクタ: raw_output 廃止 + 二重変換バグ修正~~ ✅ 完了

**実績:** DAY028 で実施。`Run()` から compact 変換を分離し `ApplyCompact()` 公開関数に。`raw_output` 廃止（`output: true` + `params.format: "json"` が代替）。`resolveStringVariables` の JSON 配列対応修正。ローカルテストで run/batch の compact/json/変数参照すべて正常動作確認済み。

---

## 未完了タスク

### 3. Sprint 007 Phase 3: 仕様書の実装追従更新 (S7-020〜026)

| ID | タスク | 状態 | 備考 |
|----|--------|------|------|
| S7-020 | spc-dsn.md Rate Limit 記述更新 | ❌ 未着手 | 「将来実装予定」に変更 |
| S7-021 | spc-itf.md JWT `aud` チェック要件整理 | ❌ 未着手 | |
| S7-022 | spc-itf.md MCP 拡張エラーコード整理 | ❌ 未着手 | |
| S7-023 | spc-itf.md Console API 設計更新 | ❌ 未着手 | |
| S7-024 | spc-itf.md PSP Webhook 仕様整理 | ❌ 未着手 | |
| S7-025 | dtl-itr-MOD-TVL.md credentials JSON 構造整理 | ❌ 未着手 | ogen 移行 + broker 化を反映 |
| S7-026 | spec-impl-compare.md 更新 | ❌ 未着手 | |

### 4. クレジットモデル仕様書更新

| タスク | 状態 | 備考 |
|--------|------|------|
| dtl-spc-credit-model.md をランニングバランス方式に更新 | ❌ 未着手 | credits テーブル廃止・running balance 移行を反映 |

### 5. Sprint 007 Phase 4: 機能実装

| ID | タスク | 状態 | 備考 |
|----|--------|------|------|
| S7-030 | usage_stats 参照 API 実装 | ❌ 未着手 | Console で使用量表示 |
| S7-031 | enabled_modules 参照 API 完成 | ❌ 未着手 | 残作業完了 |

### 6. Grafana ダッシュボード改善

| タスク | 状態 | 備考 |
|--------|------|------|
| アラート設定 | ❌ 未着手 | エラーレート閾値のアラートルール作成 |
| パネル改善 | ❌ 未着手 | 必要に応じて追加パネル |

### 7. dsn-modules.md の更新

| タスク | 状態 | 備考 |
|--------|------|------|
| 3 層アーキテクチャ (dsn-layers.md) との整合 | ❌ 未着手 | 手書きモジュール節の削除、format 層の記述追加、ogen 採用基準の更新 |
| 既存モジュール別設計書の整理 | ❌ 未着手 | composite ツール + 認証の記述以外は不要 |

---

## 優先度

| 優先度 | タスク |
|--------|--------|
| 高 | dsn-modules.md の 3 層アーキテクチャ整合 |
| 中 | 仕様書更新 (S7-020〜026) |
| 中 | クレジットモデル仕様書更新 |
| 低 | Grafana ダッシュボード改善 |
| 低 | usage_stats / enabled_modules API |
