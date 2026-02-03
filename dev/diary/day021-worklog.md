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

### Phase 2: prompts MCP 実装 ✅

| ID | タスク | 状態 | 備考 |
|----|--------|------|------|
| D21-006 | `list_user_prompts` RPC 作成 | ✅ | サーバー用（p_user_id 引数） |
| D21-007 | `get_user_prompt_by_name` RPC 作成 | ✅ | サーバー用（単体取得） |
| D21-008 | handler.go に prompts/list 追加 | ✅ | description のみ返却 |
| D21-009 | handler.go に prompts/get 追加 | ✅ | content を messages で返却 |
| D21-010 | Capability 宣言更新 | ✅ | prompts サポート宣言済み |

**追加タスク:**
| タスク | 状態 | 備考 |
|--------|------|------|
| prompts テーブルに description カラム追加 | ✅ | MCP 仕様対応（list で description、get で content） |
| prompts/get で無効プロンプト拒否 | ✅ | enabled=false のプロンプトはエラー |

### Phase 3: Console プロンプト管理 UI ✅

| ID | タスク | 状態 | 備考 |
|----|--------|------|------|
| D21-011 | /prompts ページ作成 | ✅ | 既存実装を活用 |
| D21-012 | description フィールド追加 | ✅ | MCP クライアントに表示される短い説明文 |
| D21-013 | 有効/無効トグル楽観的更新 | ✅ | 即座に保存、失敗時は元に戻す |

### Phase 4: Console ツール設定改善

| タスク | 状態 | 備考 |
|--------|------|------|
| ツール設定に楽観的更新パターン適用 | ✅ | トグル/全選択/全解除/デフォルト復元 |
| 保存ボタン削除 | ✅ | 即座に保存されるため不要 |

---

## 追加の作業詳細

### 5. prompts MCP 実装

**MCP仕様準拠:**
- `prompts/list`: name + description（短い説明文）のみ
- `prompts/get`: content を messages 配列で返却

**マイグレーション:**
- `00000000000015_rpc_user_prompts.sql` - サーバー用 RPC 追加
- `00000000000016_prompts_description.sql` - description カラム追加、RPC 更新

**Go サーバー変更:**
- `store/user.go` - UserPrompt 構造体に Description 追加
- `mcp/handler.go` - prompts/list, prompts/get ハンドラ更新、無効プロンプト拒否

### 6. Console UI 改善

**prompts ページ:**
- description フィールド追加（編集ダイアログ）
- リスト表示で description を優先表示
- 有効/無効トグル楽観的更新

**tools ページ:**
- handleToggleTool, handleSelectAll, handleDeselectAll, handleSelectDefault を楽観的更新に変更
- 保存ボタンと関連状態 (savedModules, savingModules) 削除

### 7. 認証エラーデバッグ

**問題:** 本番環境でログインできない（auth_callback_error）

**対応:**
- `auth/callback/route.ts` にエラーログ追加
- OAuth プロバイダーからのエラーパラメータをログ出力
- 設定確認後、正常にログイン可能に

---

## コミット履歴

| コミット | 内容 |
|----------|------|
| 3df724e | feat(google_tasks): add Google Tasks MCP module with OAuth support |
| 392c1a0 | fix(google_tasks): add missing authConfig for OAuth flow |
| 42dc427 | refactor(oauth): unify Google OAuth callback for calendar and tasks |
| b243898 | fix(google_tasks): change clear_completed to actually delete tasks |
| ba5b52d | feat(prompts): separate description and content per MCP spec |
| 9bfe31e | fix(console): debug login issue |
| (staged) | refactor: optimistic update for tool settings, reject disabled prompts |

---

### Phase 5: Console テーマ改善 ✅

| タスク | 状態 | 備考 |
|--------|------|------|
| Liam ERD風ダークテーマ適用 | ✅ | ドットグリッド、温かいアイボリー文字色 |
| アクセントカラー調整 | ✅ | 黒と混ぜた落ち着いたトーン（6色） |
| デフォルトアクセントをオレンジに変更 | ✅ | |
| ロゴをアクセント非依存に | ✅ | foreground色を使用 |
| カード透明度調整 | ✅ | 50% → 70% |
| /services ページ分離 | ✅ | ツール設定から接続管理を分離 |
| カスタムアクセントカラー設定削除 | ✅ | |
| 背景色プリセット機能削除 | ✅ | slate/zinc/custom を削除、常に black |
| PKCE認証エラー修正 | ✅ | skipBrowserRedirect で確実にクッキー設定 |

---

## コミット履歴（追加分）

| コミット | 内容 |
|----------|------|
| a349efa | feat(console): apply Liam ERD-inspired calm dark theme |
| 3845743 | refactor(console): separate services page and refine color palette |
| 97cfe22 | fix(auth): resolve PKCE code verifier not found error on first login |
| d32fad9 | refactor(console): remove background color preset feature |
| (staged) | fix(console): remove custom filter from accent color list |

---

## 未完了タスク

| ID | タスク | 状態 |
|----|--------|------|
| D21-014〜015 | 仕様書整備 | 未着手 |

---

## 次回の作業

1. 仕様書整備（JWT aud チェック、MCP エラーコード）
2. resources MCP 実装の検討
3. ライトモード用の色定義見直し（現在はダーク基調のまま）
