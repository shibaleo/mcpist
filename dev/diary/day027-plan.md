# DAY027 計画

## 日付

2026-02-12

---

## 概要

Sprint-007 延長 (6日目)。DAY026 で ogen 全面移行 (GitHub, Supabase, Grafana, Asana) + ヘルパー共通化が完了。本日は未コミット変更のコミットと Jira module ogen 移行を中心に進める。

---

## DAY026 からの引き継ぎ

| 項目 | 状態 |
|------|------|
| ogen 移行 (GitHub, Supabase, Grafana, Asana) | ✅ 完了 (83 tools, 76 ogen ops) |
| InputSchema バリデーション共通化 | ✅ 完了 |
| ヘルパー共通化 (ToJSON, ToStringSlice) | ✅ 実装済み・未コミット |
| 設計書 (dsn-modules.md, github.md, supabase.md, asana.md) | ✅ 作成済み・一部未コミット |
| 仕様書更新 (S7-020〜026) | ❌ 未着手 |
| クレジットモデル仕様書更新 | ❌ 未着手 |

---

## 本日のタスク

### 1. 未コミット変更のコミット（優先度：高）

DAY026 のヘルパー共通化 + ドキュメント変更をコミット。

| ID | タスク | 備考 | 状態 |
|----|--------|------|------|
| D27-001 | helpers.go + 4 module.go 共通化コミット | ビルド・テスト済み | |
| D27-002 | asana.md + dsn-modules.md ドキュメントコミット | 設計書追加 | |
| D27-003 | day026-worklog.md コミット | 作業ログ | |

### 2. Jira module ogen 移行（優先度：高）

公式 OpenAPI 3.0 spec あり。既存 11 tools を ogen 化。

| ID | タスク | 備考 | 状態 |
|----|--------|------|------|
| D27-004 | Jira OpenAPI subset spec 作成 | 公式 spec から必要エンドポイントを抽出 | |
| D27-005 | ogen 生成 + client.go 作成 | pkg/jiraapi/ | |
| D27-006 | module.go ハンドラ書き換え | httpclient → ogen | |
| D27-007 | ビルド・テスト・tools.json diff 確認 | 差分なし確認 | |
| D27-008 | 本番 MCP ツール動作確認 | 全 11 tools テスト | |

### 3. 仕様書更新（優先度：中・余裕があれば）

Sprint-007 Phase 3 の残タスク。

| ID | タスク | 備考 | 状態 |
|----|--------|------|------|
| D27-009 | spc-dsn.md Rate Limit 記述更新 (S7-020) | 「将来実装予定」に変更 | |
| D27-010 | spc-itf.md JWT/エラーコード/Console API 更新 (S7-021〜023) | まとめて対応 | |
| D27-011 | dtl-itr-MOD-TVL.md credentials 構造整理 (S7-025) | ogen 移行を反映 | |

---

## 実装方針

### 作業順序

1. **コミット (D27-001〜003)** → 未コミット変更を整理してコミット
2. **Jira ogen 移行 (D27-004〜008)** → GitHub/Asana と同じパターンで移行
3. **仕様書更新 (D27-009〜011)** → 時間があれば

### Jira module ogen 移行の方針

GitHub/Supabase/Grafana/Asana と同じ 2 層アーキテクチャ:

```
pkg/jiraapi/
  openapi-subset.yaml   # Jira REST API v3 から subset 抽出
  ogen.yaml             # generator config
  client.go             # SecuritySource (Bearer token)
  gen/                  # ogen 自動生成
internal/modules/jira/
  module.go             # ハンドラ書き換え
```

Jira 固有の考慮点:
- **認証**: Basic Auth (email:api_token) — Grafana と同じ dual auth パターンが参考になる
- **ベース URL**: `https://{domain}.atlassian.net/rest/api/3` — ユーザーごとに異なる
- **ページネーション**: `startAt` + `maxResults` パターン

---

## 完了条件

- [ ] DAY026 の未コミット変更が全てコミット済み
- [ ] Jira module が ogen 化され、ビルド・テスト pass
- [ ] tools.json に diff なし
- [ ] 本番 MCP ツールで Jira 11 tools が動作確認済み

---

## 参考

- [day026-worklog.md](day026-worklog.md) - DAY026 作業ログ
- [day026-backlog.md](day026-backlog.md) - DAY026 バックログ
- [sprint007-plan.md](../sprint/sprint007-plan.md) - Sprint 007 計画
- [dsn-modules.md](../../docs/003_design/modules/dsn-modules.md) - モジュールアーキテクチャ設計書
