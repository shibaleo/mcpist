# DAY028 バックログ

## 日付

2026-02-13

---

## 未完了タスク

### 1. OAuth2 リフレッシュ共通化 + store → broker リネーム

| タスク | 状態 | 備考 |
|--------|------|------|
| 各モジュールの `refreshToken()` を共通関数に集約 | ❌ 未着手 | 11 モジュールにコピペされている |
| `store` パッケージを `broker` にリネーム | ❌ 未着手 | 「保存場所」ではなく「認証情報の仲介者」が本質 |
| `getCredentials()` をリフレッシュ込みの共通関数に | ❌ 未着手 | モジュールは有効なトークンを受け取るだけ |

**現状の問題:**
- `refreshToken()` が Asana, Google (6 モジュール), Dropbox, Notion, Microsoft Todo にコピペ
- プロバイダごとの差異は token endpoint URL と refresh_token の扱い（返す/返さない）だけ
- リフレッシュは store (broker) 層の責務であり、モジュールが知るべきではない

**あるべき姿:**
```go
// broker.GetCredentials — リフレッシュを透過的に処理
creds, err := broker.GetCredentials(ctx, "asana")
// モジュールは有効なトークンを受け取るだけ。リフレッシュの有無を知らない
```

**判断:** リフレッシュ共通化とリネームは影響範囲が広い。format 層分離 + ogen 移行完了後にまとめて実施する。

### 2. Sprint 007 Phase 3: 仕様書の実装追従更新 (S7-020〜026)

| ID | タスク | 状態 | 備考 |
|----|--------|------|------|
| S7-020 | spc-dsn.md Rate Limit 記述更新 | ❌ 未着手 | 「将来実装予定」に変更 |
| S7-021 | spc-itf.md JWT `aud` チェック要件整理 | ❌ 未着手 | |
| S7-022 | spc-itf.md MCP 拡張エラーコード整理 | ❌ 未着手 | |
| S7-023 | spc-itf.md Console API 設計更新 | ❌ 未着手 | |
| S7-024 | spc-itf.md PSP Webhook 仕様整理 | ❌ 未着手 | |
| S7-025 | dtl-itr-MOD-TVL.md credentials JSON 構造整理 | ❌ 未着手 | ogen 移行 + broker 化を反映 |
| S7-026 | spec-impl-compare.md 更新 | ❌ 未着手 | |

### 3. クレジットモデル仕様書更新

| タスク | 状態 | 備考 |
|--------|------|------|
| dtl-spc-credit-model.md をランニングバランス方式に更新 | ❌ 未着手 | credits テーブル廃止・running balance 移行を反映 |

### 4. Sprint 007 Phase 4: 機能実装

| ID | タスク | 状態 | 備考 |
|----|--------|------|------|
| S7-030 | usage_stats 参照 API 実装 | ❌ 未着手 | Console で使用量表示 |
| S7-031 | enabled_modules 参照 API 完成 | ❌ 未着手 | 残作業完了 |

### 5. Grafana ダッシュボード改善

| タスク | 状態 | 備考 |
|--------|------|------|
| アラート設定 | ❌ 未着手 | エラーレート閾値のアラートルール作成 |
| パネル改善 | ❌ 未着手 | 必要に応じて追加パネル |

### 6. dsn-modules.md の更新

| タスク | 状態 | 備考 |
|--------|------|------|
| 3 層アーキテクチャ (dsn-layers.md) との整合 | ❌ 未着手 | 手書きモジュール節の削除、format 層の記述追加、ogen 採用基準の更新 |
| 既存モジュール別設計書の整理 | ❌ 未着手 | composite ツール + 認証の記述以外は不要 |

---

## 優先度

| 優先度 | タスク |
|--------|--------|
| 高 | OAuth2 リフレッシュ共通化 + store → broker リネーム |
| 高 | dsn-modules.md の 3 層アーキテクチャ整合 |
| 中 | 仕様書更新 (S7-020〜026) |
| 中 | クレジットモデル仕様書更新 |
| 低 | Grafana ダッシュボード改善 |
| 低 | usage_stats / enabled_modules API |
