# DAY021 作業ログ

## 日付

2026-02-01

---

## 完了タスク

### Phase 1: Google Tasks モジュール実装 ✅

| ID | タスク | 状態 | 備考 |
|----|--------|------|------|
| D21-001 | Google Tasks API 調査 | ✅ | OAuth scope、エンドポイント確認 |
| D21-002 | google_tasks モジュール作成 | ✅ | 9ツール実装 |
| D21-003 | ツール実装 | ✅ | 全9ツール動作確認 |
| D21-004 | Console OAuth 設定 | ✅ | 共有コールバック方式に変更 |
| D21-005 | E2E テスト | ✅ | mcpist-dev MCP経由で全ツール検証 |

---

## 作業詳細

### 1. Server (Go) モジュール実装

**新規作成:**
- `apps/server/internal/modules/google_tasks/module.go` (663行)

**実装ツール (9個):**

| ツール | 説明 | Annotation |
|--------|------|------------|
| list_task_lists | タスクリスト一覧 | ReadOnly |
| get_task_list | タスクリスト詳細 | ReadOnly |
| list_tasks | タスク一覧 | ReadOnly |
| get_task | タスク詳細 | ReadOnly |
| create_task | タスク作成 | Create |
| update_task | タスク更新 | Update |
| delete_task | タスク削除 | Delete |
| complete_task | タスク完了トグル | Update |
| clear_completed | 完了タスク一括削除 | Delete |

**登録:**
- `apps/server/cmd/server/main.go` - import追加、RegisterModule追加
- `apps/server/cmd/tools-export/main.go` - 同様

### 2. Console OAuth 実装

**当初の計画:** google-tasks 専用の authorize/callback を作成

**実際の実装:** 共有コールバック方式に変更

**理由:**
- Google Cloud Console に登録済みの redirect_uri は `/api/oauth/google/callback` のみ
- 新規エンドポイント登録より、state パラメータでモジュールを識別する方が効率的

**変更内容:**

| ファイル | 変更 |
|----------|------|
| `google/authorize/route.ts` | `module` パラメータ対応、モジュール別スコープ定義 |
| `google/callback/route.ts` | state から `module` を取り出し、適切なモジュールにトークン保存 |
| `oauth-apps.ts` | `google-tasks` を `google` authorize にルーティング |
| `tools/page.tsx` | `google_tasks` の authConfig 追加 |

### 3. Vault RPC 修正

**問題:** `vault.update_secret` は Supabase Vault に存在しない

**解決:** マイグレーション追加
- `00000000000013_fix_oauth_apps_vault_update.sql`
- delete + create パターンで upsert_oauth_app, delete_oauth_app を修正

### 4. clear_completed の挙動修正

**問題:** Google Tasks API の `/clear` エンドポイントは `hidden=true` を設定するだけで、UI からタスクが消えない

**解決:** 完了タスクを取得して各々を DELETE する実装に変更

---

## 発見・学び

### Google Tasks API の hidden フィールド

| 操作 | 結果 | UI表示 |
|------|------|--------|
| `complete_task` | `status: "completed"` | 「Completed」折り畳みに表示 |
| Google API `/clear` | `hidden: true` | **UIに影響なし** |
| `delete_task` | 完全削除 | 即座に消える |

- `hidden: true` は論理削除だが、Google Tasks Web UI では無視される
- 実用的には `delete_task` を使うべき

### OAuth App とトークンの分離

```
OAuth App: "google" (client_id/client_secret) - 共有
  ├── google_calendar トークン (scope: calendar)
  └── google_tasks トークン (scope: tasks)
```

- OAuth App 認証情報は `GetOAuthAppCredentials(ctx, "google")` で共有
- トークンは `GetModuleToken(ctx, userID, "google_tasks")` でモジュール固有
- Console のコールバックは state パラメータでモジュールを識別

---

## コミット履歴

| コミット | 内容 |
|----------|------|
| 3df724e | feat(google_tasks): add Google Tasks MCP module with OAuth support |
| 392c1a0 | fix(google_tasks): add missing authConfig for OAuth flow |
| 42dc427 | refactor(oauth): unify Google OAuth callback for calendar and tasks |
| (staged) | fix(google_tasks): change clear_completed to actually delete tasks + vault fix |

---

## 未完了タスク（Phase 2以降）

| ID | タスク | 状態 |
|----|--------|------|
| D21-006〜010 | prompts MCP 実装 | 未着手 |
| D21-011〜013 | Console プロンプト管理 UI | 未着手 |
| D21-014〜015 | 仕様書整備 | 未着手 |

---

## 次回の作業

1. prompts MCP 実装（Phase 2）
2. Console プロンプト管理 UI（Phase 3）
