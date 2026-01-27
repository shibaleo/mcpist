# DAY016 計画

## 日付

2026-01-27

---

## 概要

DAY015で完了したSprint-005の残タスクを整理し、仕様書と実装の差分是正および新機能実装を行う。

---

## DAY015の成果（振り返り）

| 完了タスク | 備考 |
|------------|------|
| Phase 2: RPC呼び出しリファクタ | Console/Worker/Go Server 全て完了 (9/9) |
| Phase 7: OAuth トークンリフレッシュ | Google Calendar + Microsoft To Do 対応 |
| Microsoft To Do OAuth実装 | authorize/callback + Goモジュール (11ツール) |
| ツール設定API | tool_settings RPC + Console UI連携 |

---

## 本日のタスク

### 優先度: 高

| ID | タスク | 見積り | 備考 |
|----|--------|--------|------|
| D16-001 | サービス接続時のデフォルトツール設定自動保存 | - | BL-001。OAuth callback後にtools.jsonのdefaultEnabledを元にDB保存 |
| D16-002 | 仕様書の実装追従更新 | - | spec-impl-compare.mdの差分を仕様書に反映 |

### 優先度: 中

| ID | タスク | 見積り | 備考 |
|----|--------|--------|------|
| D16-003 | next.config.ts デバッグログ削除 | - | B-006。console.log削除 |
| D16-004 | VitePress docsビルド修正 | - | デッドリンク削除、ビルド成功させる |

### 優先度: 低

| ID | タスク | 見積り | 備考 |
|----|--------|--------|------|
| D16-005 | Phase 4: UI要件定義 | - | spc-ui.md作成 |
| D16-006 | E2Eテスト設計 | - | OAuth認可フロー等 |

---

## 仕様書更新計画（D16-002詳細）

spec-impl-compare.mdで特定された主要な差分を仕様書に反映する:

### 更新対象

| 仕様書 | 更新内容 |
|--------|----------|
| spc-itf.md | メタツール名: `call` → `run` に修正 |
| spc-itf.md | Long-lived Tokenプレフィックス: `mcpist_` → `mpt_` に修正 |
| spc-itf.md | Token Refresh: 各モジュールが担当する設計に変更 |
| dtl-spc/itf-tvl.md | Token Vault: Edge Functions → Supabase RPC方式に変更 |
| idx-ept.md | Auth Server: Supabase Auth + Worker構成に変更 |
| dsn-tbl.md | api_keys: key_prefix, last_used_at, revoked_at列追加 |
| dsn-tbl.md | service_tokens, oauth_appsテーブル追加 |

### 方針

- 実装が正であり、仕様書を実装に合わせる
- 未実装機能（PSP Webhook等）は仕様書から削除または「将来実装予定」に変更

---

## BL-001 実装計画（D16-001詳細）

### 現状の問題

- サービス接続成功時、ツール設定がDBに保存されない
- ユーザーが「設定を保存」を明示的に押す必要がある

### 解決策

1. OAuth callback成功後、リダイレクト時にクエリパラメータでmoduleを渡す
2. Connectionsページでmoduleパラメータを検知したら自動保存処理を実行
3. tools.jsonのdefaultEnabledを元にenabled/disabled配列を作成
4. `upsert_my_tool_settings` RPCを呼び出してDB保存

### 変更ファイル

| ファイル | 変更内容 |
|----------|----------|
| `apps/console/src/app/api/oauth/google/callback/route.ts` | リダイレクトURLに`?connected=google_calendar`追加 |
| `apps/console/src/app/api/oauth/microsoft/callback/route.ts` | リダイレクトURLに`?connected=microsoft_todo`追加 |
| `apps/console/src/app/(console)/connections/page.tsx` | クエリパラメータ検知 → 自動保存処理追加 |

---

## 参考

- [day015-worklog.md](./day015-worklog.md) - 前日作業ログ
- [day015-review.md](./day015-review.md) - 前日振り返り
- [day015-backlog.md](./day015-backlog.md) - バックログ
- [spec-impl-compare.md](./spec-impl-compare.md) - 仕様・実装差分
