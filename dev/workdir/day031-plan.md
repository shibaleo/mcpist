# DAY031 作業計画

## 日付

2026-02-16

## 対応スプリント

Sprint 010 — 残タスク整理 + テスト基盤 or Phase 1 (OAuth)

---

## 前日の状況

DAY030 で以下を実施済み:

- トランスポート層リファクタリング (handler.go → transport.go 分離)
- PostgREST RPC の Supabase 依存解消 → apikey ヘッダー復元 (Supabase Kong 要件)
- RPC 設計見直し (sync_modules 廃止 → 復活、プロンプト統合)
- **tools.json 廃止 + DB フェッチ移行** (`list_modules_with_tools` RPC、`descriptions` カラム追加) — **未コミット**
- 設計図更新 (grh-rpc-design, grh-table-design, grh-componet-interactions)

### バックログ更新が必要な項目

| 項目 | 変更 |
|------|------|
| `tools.json 自動生成パイプライン` | **完了/不要** — tools.json 自体を廃止したため |
| `sync_modules` RPC | **復活** — DAY030 で廃止したが、tools + descriptions の DB 同期のために復活 |
| `broker/module.go` | **不要** — sync_modules は main.go の起動時処理に統合済み |
| RPC 数 | 7 → 9 (sync_modules 復活、list_modules_with_tools 追加) |

---

## 目標

DAY030 の未コミット変更を確定し、次の開発フェーズに着手する。

---

## タスク

### 0. DAY030 残作業

| # | タスク | 成果物 |
|---|--------|--------|
| 0-1 | DAY030 の未コミット変更をコミット | git commit (12 ファイル + canvas 3 ファイル) |
| 0-2 | sprint010-backlog.md の更新 | tools.json パイプライン → 完了、sync_modules 復活を反映 |

### 候補 A: Phase 3 — テスト基盤 (S10-040〜043)

現状テストがゼロ。リファクタリングが進んだ今が書きやすいタイミング。

| # | ID | タスク | 成果物 |
|---|-----|--------|--------|
| A-1 | S10-040 | authz middleware ユニットテスト | middleware/authz_test.go |
| A-2 | S10-041 | broker/user.go ユニットテスト | broker/user_test.go |
| A-3 | S10-042 | broker/retry.go ユニットテスト | broker/retry_test.go |
| A-4 | S10-043 | CI トリガーを push/PR に変更 | .github/workflows/ci.yml |

### 候補 B: Phase 1 — OAuth 2.1 Server (S10-001〜005)

Claude 認可フローが動いている間は緊急度低だが、Supabase Auth がブラックボックスである構造的問題は残る。

| # | ID | タスク | 成果物 |
|---|-----|--------|--------|
| B-1 | S10-001 | `@cloudflare/workers-oauth-provider` 導入 | package.json, wrangler.toml |
| B-2 | S10-002 | OAuth 2.1 Server エンドポイント実装 | oauth-server.ts |
| B-3 | S10-003 | Token storage (Workers KV) | oauth-server.ts |

### 候補 C: Console RPC 設計図の完成

RPC 設計図に Server RPC (9) のみ記載。Console RPC (15+) が未記載。

| # | タスク | 成果物 |
|---|--------|--------|
| C-1 | Console RPC の洗い出し・整理 | grh-rpc-design.canvas (Console セクション追加) |

---

## 作業順序

```
1. DAY030 未コミット変更のコミット + バックログ更新
2. 候補 A / B / C のいずれかに着手 (ユーザー判断)
```

---

## 参考

- [sprint010-plan.md](sprint010-plan.md) - Sprint 010 計画
- [sprint010-backlog.md](../sprint/sprint010-backlog.md) - Sprint 010 バックログ
- [day030-worklog.md](../diary/day030-worklog.md) - DAY030 作業ログ
