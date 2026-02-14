# Sprint 007 レビュー

## 基本情報

| 項目 | 値 |
|------|-----|
| スプリント番号 | SPRINT-007 |
| 計画期間 | 2026-02-03 〜 2026-02-09 (7日間) |
| 実績期間 | 2026-02-03 〜 2026-02-13 (延長 11日間) |
| マイルストーン | M6: Observability・セキュリティ・仕様整備 |
| 状態 | **完了** |

---

## 計画 vs 実績サマリ

| 項目 | 計画 | 実績 | 達成度 |
|------|------|------|--------|
| Phase 1: Observability 実装 | 6タスク | 4タスク完了 | ⚠️ 67% |
| Phase 2: セキュリティ設計書 | 6タスク | 6タスク完了 | ✅ 100% |
| Phase 3: 仕様書更新 | 7タスク | 0タスク | ❌ 0% |
| ogen 全面移行 | 計画外 | 19/19 モジュール | ⭐ 計画外成果 |
| 3 層アーキテクチャ確立 | 計画外 | spec/tool/format 分離 | ⭐ 計画外成果 |
| OAuth2 リフレッシュ集約 | 計画外 | broker パッケージ | ⭐ 計画外成果 |
| Console リデザイン | 計画外 | UI 全面刷新 | ⭐ 計画外成果 |

---

## Sprint 006 → 007 差分

| 項目 | S006 終了時 | S007 終了時 | 差分 |
|------|-----------|-----------|------|
| モジュール数 | 18 | 20 | +2 |
| ツール数 | 248 | ~280 | +~32 |
| ogen 移行 | 0/19 | **19/19** | +19 |
| compact format | 0/19 | **19/19** | +19 |
| OpenAPI subset spec | 0 | **19** | +19 |
| OAuth2 リフレッシュ | 11 モジュールに分散 | broker 集約 | -1,306 行 |
| httpclient 依存 | 18 モジュール | **0** | -18 |
| テスト | なし | ツール定義テスト | +1 |

---

## Phase 別詳細

### Phase 1: Observability 実装 ⚠️ 部分完了

| ID | タスク | 計画 | 実績 | 状態 |
|----|--------|------|------|------|
| S7-001 | Observability 設計書 | dsn-observability.md | 完了 | ✅ |
| S7-002 | 構造化ログ統一 | Go Server | Loki 統合完了 | ✅ |
| S7-003 | ツール実行ログに user_id | Go Server | 完了 | ✅ |
| S7-004 | invalid_gateway_secret ログ | Go Server | 完了 | ✅ |
| S7-005 | エラー分類とログレベル | Go Server | 未着手 | 次Sprint |
| S7-006 | Grafana ダッシュボード | 設計書 | 設計のみ | 次Sprint |

### Phase 2: セキュリティ設計書 ✅ 完了

| ID | タスク | 計画 | 実績 | 状態 |
|----|--------|------|------|------|
| S7-010 | セキュリティ設計書 | dsn-security.md | 完了 | ✅ |
| S7-011 | 認証・認可フロー整理 | dsn-security.md | 完了 | ✅ |
| S7-012 | OAuth セキュリティ整理 | dsn-security.md | 完了 | ✅ |
| S7-013 | データ保護整理 | dsn-security.md | 完了 | ✅ |
| S7-014 | SSRF 対策整理 | dsn-security.md | 完了 | ✅ |
| S7-015 | セキュリティチェックリスト | dsn-security.md | 完了 | ✅ |

### Phase 3: 仕様書更新 ❌ 未着手

| ID | タスク | 状態 |
|----|--------|------|
| S7-020 | Rate Limit 記述更新 | 次Sprint |
| S7-021 | JWT `aud` チェック要件整理 | 次Sprint |
| S7-022 | MCP 拡張エラーコード整理 | 次Sprint |
| S7-023 | Console API 設計更新 | 次Sprint |
| S7-024 | PSP Webhook 仕様整理 | 次Sprint |
| S7-025 | credentials JSON 構造整理 | 次Sprint |
| S7-026 | spec-impl-compare.md 更新 | 次Sprint |

**未達成の理由:** ogen 全面移行・3 層アーキテクチャ確立が想定外の大規模作業となり、仕様書更新に着手できなかった。

---

## 計画外成果

### 1. ogen 全面移行 (19/19 モジュール)

全モジュールの HTTP クライアントを手書きから ogen 自動生成に移行。OpenAPI subset spec を Single Source of Truth とし、型安全な API クライアントを生成。

| 移行日 | モジュール |
|--------|-----------|
| DAY026 | GitHub, Supabase, Grafana, Asana |
| DAY027 | Jira, Confluence, Notion, TickTick, Todoist, Trello |
| DAY028 | Dropbox, Airtable, Google Calendar/Tasks/Docs/Drive/Sheets/Apps Script, Microsoft Todo |

### 2. 3 層アーキテクチャの確立

```
[spec 層]   openapi-subset.yaml → ogen → 型安全な関数
[tool 層]   ハンドラ (パラメータ変換 → 関数呼び出し → JSON 返却)
[format 層] JSON → compact 表現 (CSV/MD/key-value)
```

各層は隣の層の存在を知らない。設計書 dsn-layers.md を作成。

### 3. OAuth2 リフレッシュ集約

`internal/store` → `internal/broker` にリネーム。11 モジュールの個別 `refreshToken()` を `OAuthRefreshConfig` テーブル駆動に統合 (+679/-1,306 行)。

### 4. batch リファクタ

- `raw_output` (壊れていた) を廃止、`params.format: "json"` に統一
- Run() から compact 変換を分離し、二重変換バグを修正
- `resolveStringVariables` の JSON 配列対応で変数参照を修正

### 5. Console リデザイン

- sidebar、テーマ (ivory light)、レスポンシブ改善
- ランディングページ刷新 (アーキテクチャ図)
- ルート名変更 (connections→mcp-server, billing→credits, prompts→templates)

### 6. クレジットモデル改善

- running balance パターン移行 (credits テーブル廃止)
- details JSONB 統合、レガシーカラム削除
- per-user burst limit + batch size cap

---

## 数値サマリ

| 項目 | 値 |
|------|-----|
| コミット数 | 68 |
| 変更ファイル | 440 |
| 追加行数 | +179,935 |
| 削除行数 | -9,722 |
| 新規 OpenAPI subset spec | 19 |
| 新規 format.go | 19 |
| 新規 ogen gen/ パッケージ | 19 |

---

## 振り返り

### 良かった点

1. **ogen 全面移行**: 19 モジュールの HTTP クライアントを型安全な自動生成に置き換え、保守性が大幅向上
2. **3 層アーキテクチャ**: spec/tool/format の責務分離により、ハンドラが 100% 機械的に。将来の toolgen の設計根拠が確立
3. **broker 集約**: OAuth2 リフレッシュの重複コード 1,306 行を削除。新モジュール追加時のリフレッシュ実装が設定 1 行で完了
4. **batch 修正**: 壊れていた raw_output と変数参照を修正し、format パラメータを run/batch で統一

### 改善点

1. **仕様書更新が再び未着手**: S006 から引き継いだ S7-020〜026 が進まなかった
2. **計画 vs 実績の乖離**: 当初計画 (Observability・セキュリティ・仕様整備) から ogen 移行にピボット。結果は良かったが計画精度に課題
3. **スプリント期間超過**: 7 日 → 11 日に延長

### 次 Sprint への教訓

1. **仕様書更新を最初にやる**: 実装が先行すると常にドキュメントが後回しになる
2. **大規模リファクタの見積もり**: ogen 移行は 1 日で終わる想定だったが 3 日かかった。依存するリファクタ (format 分離、broker 集約) が連鎖的に発生した
3. **設計書が減ったのは良いこと**: 3 層分離により「spec = 実装 = 設計書」。モジュール別設計書の大半が不要になった

---

## 残課題 (次 Sprint へ引き継ぎ)

| 優先度 | タスク | 備考 |
|--------|--------|------|
| 高 | dsn-modules.md の 3 層整合 | 2 層 → 3 層に更新 |
| 中 | 仕様書更新 (S7-020〜026) | S006 から引き継ぎ |
| 中 | クレジットモデル仕様書更新 | running balance 反映 |
| 低 | Grafana ダッシュボード改善 | アラート設定、パネル追加 |
| 低 | usage_stats / enabled_modules API | Console 使用量表示 |
| 低 | CI/CD 整備、E2E テスト基盤 | S006 から引き継ぎ |

→ 詳細は [sprint007-backlog.md](./sprint007-backlog.md) を参照

---

## 参考

- [sprint007-plan.md](./sprint007-plan.md) - Sprint 007 計画
- [sprint006-review.md](./sprint006-review.md) - Sprint 006 レビュー
- [dsn-layers.md](../../docs/003_design/modules/dsn-layers.md) - 3 層アーキテクチャ設計書
- [day026-worklog.md](day026-worklog.md)
- [day027-worklog.md](day027-worklog.md)
- [day028-worklog.md](day028-worklog.md)
