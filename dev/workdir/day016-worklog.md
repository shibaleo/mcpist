# DAY016 作業ログ

## 日付

2026-01-27

---

## 作業内容

### D16-001: サービス接続時のデフォルトツール設定自動保存

BL-001の対応。サービス接続成功時にtools.jsonのannotationsを元にデフォルトツール設定をDBに自動保存する機能を実装。

#### MCP Tool Annotations移行

tools.jsonのカスタムフィールド(`defaultEnabled`/`dangerous`)をMCP仕様（2025-11-25）の`annotations`に移行。

| フィールド | 説明 | デフォルト |
|-----------|------|-----------|
| `readOnlyHint` | 環境を変更しない | false |
| `destructiveHint` | 破壊的な更新を行う可能性がある | true |
| `idempotentHint` | 繰り返し呼び出しても追加的影響がない | false |
| `openWorldHint` | 外部エンティティとやり取りする | true |

#### ヘルパー関数

| 関数 | ロジック |
|------|---------|
| `isDefaultEnabled(tool)` | `readOnlyHint === true` のツールをデフォルト有効 |
| `isDangerous(tool)` | `readOnlyHint !== true && destructiveHint !== false` のツールを危険表示 |

#### 自動保存の実装

`saveDefaultToolSettings()` をトークン保存後に呼び出す:
- API Key接続: `token-vault.ts` の `upsertTokenWithVerification()` 内
- OAuth接続: `google/callback` と `microsoft/callback` のRoute Handler内

#### 再接続時のユーザー設定保持

`saveDefaultToolSettings()` に既存設定チェックを追加。設定が存在する場合はスキップし、ユーザーのカスタム設定を保持する。

#### 変更ファイル

| ファイル | 変更内容 |
|----------|----------|
| `apps/console/src/lib/tools.json` | 全115ツールを`annotations`形式に変換 |
| `apps/console/src/lib/module-data.ts` | `ToolAnnotations`型、`isDefaultEnabled()`、`isDangerous()` 追加 |
| `apps/console/src/lib/tool-settings.ts` | `saveDefaultToolSettings()` 追加、既存設定スキップロジック |
| `apps/console/src/lib/token-vault.ts` | 接続成功時に`saveDefaultToolSettings()`呼び出し |
| `apps/console/src/app/api/oauth/google/callback/route.ts` | 接続成功時にデフォルト設定保存 |
| `apps/console/src/app/api/oauth/microsoft/callback/route.ts` | 接続成功時にデフォルト設定保存 |
| `apps/console/src/lib/supabase/database.types.ts` | `get_my_tool_settings`、`upsert_my_tool_settings` 型定義追加 |

#### ビルドエラー修正

| エラー | 原因 | 修正 |
|--------|------|------|
| `"get_my_tool_settings"` not assignable | `database.types.ts` に型定義がなかった | 型定義追加 |
| `Property 'error' does not exist on type 'Json'` | `upsert_my_tool_settings` の戻り値が`Json`型 | `Record<string, unknown>` にキャスト |

---

### Console UI改善

#### ツール設定UIの改善

| 改善内容 | 詳細 |
|----------|------|
| MCP Annotationsバッジ | ReadOnly（青）、Destructive（黄）、Idempotent（灰）のバッジ表示 |
| Switch (トグル) | Checkboxを`@radix-ui/react-switch`ベースのSwitchに置換（左側配置） |
| 全解除ボタン | 「デフォルト」「全選択」に加え「全解除」ボタンを追加 |
| 認証ヘルプリンク | シークレット入力ダイアログにサービスごとのトークン取得ページへのリンクを追加 |

#### 認証ヘルプリンク

| サービス | URL |
|----------|-----|
| Notion | https://www.notion.so/profile/integrations |
| GitHub | https://github.com/settings/tokens?type=beta |
| Jira/Confluence | https://id.atlassian.com/manage-profile/security/api-tokens |
| Supabase | https://supabase.com/dashboard/account/tokens |

#### ダッシュボード

「有効なツール」カードにDBから取得した有効ツール数/全ツール数を表示（モック値`-`を置換）。

#### 変更ファイル

| ファイル | 変更内容 |
|----------|----------|
| `apps/console/src/app/(console)/tools/page.tsx` | Switch、全解除、認証リンク、Annotationsバッジ |
| `apps/console/src/components/ui/switch.tsx` | Switchコンポーネント（新規作成） |
| `apps/console/package.json` | `@radix-ui/react-switch` 追加 |
| `apps/console/src/app/(console)/dashboard/page.tsx` | 有効ツール数表示 |

---

### DB検証

Supabase MCP経由で本番DBのツール設定を検証。

| 検証内容 | 結果 |
|----------|------|
| 全8モジュールのツール設定登録 | OK（supabase, airtable, notion, github, jira, confluence, google_calendar, microsoft_todo） |
| Annotations基準のデフォルト値 | OK（readOnlyHint=trueのツールが有効） |
| ユーザーカスタム設定の保存 | OK |
| 切断後のツール設定残存 | 確認（孤立データとして残存） |
| 再接続時の設定保持（スキップロジック） | OK（既存設定が保持された） |

---

## コミット履歴

| コミット | メッセージ |
|----------|-----------|
| `645b79d` | feat(console): adopt MCP tool annotations and auto-save default tool settings |
| `a7fc850` | fix(console): skip default tool settings if user settings already exist |
| `ddbdeb5` | feat(console): add switch toggle, deselect-all button, and auth help links |
| `79807cd` | feat(console): show enabled tool count on dashboard and fix switch position |

---

## タスク進捗

| ID | タスク | 状態 |
|----|--------|------|
| D16-001 | サービス接続時のデフォルトツール設定自動保存 | **完了** |
| D16-002 | 仕様書の実装追従更新 | 未着手 |
| D16-003 | next.config.ts デバッグログ削除 | 未着手 |
| D16-004 | VitePress docsビルド修正 | 未着手 |
| D16-005 | Phase 4: UI要件定義 | 未着手 |
| D16-006 | E2Eテスト設計 | 未着手 |

---

## バックログ更新

| ID | タスク | 状態 | 備考 |
|----|--------|------|------|
| BL-001 | サービス接続時にデフォルトツール設定を自動保存 | **完了** | D16-001で実装。MCP Annotations移行も含む |
| BL-002 | 切断時のツール設定クリーンアップ | 保留 | DB migration必要。ユーザー設定復元のためのデータ構造変更が前提 |
