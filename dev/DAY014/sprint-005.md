# Sprint 005 計画書

## 基本情報

| 項目 | 値 |
|------|-----|
| スプリント番号 | SPRINT-005 |
| 期間 | 2026-01-25 〜 |
| マイルストーン | M4: RPC実装・リファクタリング・モジュール拡張 |

---

## Sprint目標

**RPC関数の実装・統合、モジュール拡張（Airtable）、インフラ自動化**

---

## 進捗サマリー

| Phase | 状態 | 進捗 |
|-------|------|------|
| Phase 1: RPC関数実装 | ✅ 完了 | 16/17 (94%) |
| Phase 2: RPC呼び出しリファクタ | ⬜ 未着手 | 0/9 (0%) |
| Phase 3: パスルーティング設計 | ✅ 完了 | 3/3 (100%) |
| Phase 4: UI要件定義 | ⬜ 未着手 | 0/3 (0%) |
| Phase 5: ツール設定API | 🔄 進行中 | 2/8 (25%) |
| Phase 6: モジュール拡張 | ✅ 完了 | 1/1 (100%) |

---

## タスク一覧

### Phase 1: RPC関数実装 (Supabase) ✅

| ID | タスク | 状態 | 備考 |
|----|--------|------|------|
| S5-001 | lookup_user_by_key_hash | ✅ 完了 | APIキーハッシュ → user_id |
| S5-002 | get_user_context | ✅ 完了 | ツール実行用ユーザー情報 |
| S5-003 | consume_credit | ✅ 完了 | クレジット消費・履歴記録 |
| S5-004 | get_module_token | ✅ 完了 | モジュール用トークン取得 |
| S5-005 | update_module_token | ⬜ 未着手 | リフレッシュ後トークン保存 |
| S5-006 | generate_api_key | ✅ 完了 | APIキー生成 |
| S5-007 | list_api_keys | ✅ 完了 | APIキー一覧取得 |
| S5-008 | revoke_api_key | ✅ 完了 | APIキー論理削除 |
| S5-009 | list_service_connections | ✅ 完了 | サービス接続一覧 |
| S5-010 | upsert_service_token | ✅ 完了 | トークン登録/更新 |
| S5-011 | delete_service_token | ✅ 完了 | トークン削除 |
| S5-012 | add_paid_credits | ✅ 完了 | Webhook用クレジット加算 |
| S5-013 | reset_free_credits | ✅ 完了 | 月次リセット（pg_cron） |
| S5-014 | list_oauth_consents | ✅ 完了 | OAuth認可済みクライアント一覧 |
| S5-015 | revoke_oauth_consent | ✅ 完了 | OAuth認可取り消し |
| S5-016 | list_all_oauth_consents | ✅ 完了 | 全ユーザーOAuth認可一覧（admin） |
| S5-017 | sync_modules | ✅ 完了 | モジュール自動同期（サーバー起動時） |

---

### Phase 2: RPC呼び出しリファクタ ⬜

| ID | タスク | 状態 | 備考 |
|----|--------|------|------|
| S5-020 | api-keys ページをRPC使用に統一 | ⬜ 未着手 | generate/list/revoke |
| S5-021 | connections ページをRPC使用に統一 | ⬜ 未着手 | list/upsert/delete |
| S5-022 | dashboard クレジット表示をRPC化 | ⬜ 未着手 | 直接テーブル参照 → RPC |
| S5-023 | token-vault API Route リファクタ | ⬜ 未着手 | upsert_service_token RPC使用 |
| S5-024 | database.types.ts にRPC型定義追加 | ⬜ 未着手 | 新規RPC全てに対応 |
| S5-030 | Worker: lookup_user_by_key_hash 使用 | ⬜ 未着手 | 既存実装の整理 |
| S5-040 | Go: get_module_token RPC使用 | ⬜ 未着手 | 現在: 直接クエリ |
| S5-041 | Go: consume_credit RPC使用 | ⬜ 未着手 | 現在: 直接クエリ |
| S5-042 | Go: get_user_context RPC呼び出し | ⬜ 未着手 | アカウント状態・設定取得 |

---

### Phase 3: パスルーティング設計 ✅

| ID | タスク | 状態 | 備考 |
|----|--------|------|------|
| S5-050 | 現行ルート構造の整理 | ✅ 完了 | dsn-route.md作成 |
| S5-051 | ルーティング設計書作成 | ✅ 完了 | URL設計、認証要件 |
| S5-052 | 管理者Route Group分離 | ✅ 完了 | (admin)/layout.tsx |

---

### Phase 4: ユーザーコンソール要件定義 ⬜

| ID | タスク | 状態 | 備考 |
|----|--------|------|------|
| S5-060 | UI要求仕様書作成 (spc-ui.md) | ⬜ 未着手 | 画面一覧・機能要件 |
| S5-061 | ユーザーフロー図作成 | ⬜ 未着手 | 主要フローの可視化 |
| S5-062 | 画面遷移図作成 | ⬜ 未着手 | 認証後のナビゲーション |

---

### Phase 5: ツール設定API 🔄

| ID | タスク | 状態 | 備考 |
|----|--------|------|------|
| S5-070 | tools-exportコマンド作成 | ✅ 完了 | Go側ツール定義をJSON出力 |
| S5-071 | tools.json / services.json生成 | ✅ 完了 | console/src/lib配置 |
| S5-072 | tool_settingsテーブル作成 | ⬜ 未着手 | user_id, module, enabled_tools |
| S5-073 | get_tool_settings RPC作成 | ⬜ 未着手 | ユーザーのツール設定取得 |
| S5-074 | upsert_tool_settings RPC作成 | ⬜ 未着手 | ツール設定保存 |
| S5-075 | /tools ページをtools.json使用に変更 | ⬜ 未着手 | 現在のハードコード削除 |
| S5-076 | CIにtools.json検証追加 | ⬜ 未着手 | Go出力との差分チェック |
| S5-077 | 未使用Goモジュール削除 | ⬜ 未着手 | クリーンアップ |

---

### Phase 6: モジュール拡張 ✅

| ID | タスク | 状態 | 備考 |
|----|--------|------|------|
| S5-080 | Airtableモジュール実装 | ✅ 完了 | 11ツール（search, aggregate含む） |

---

## インフラ整備 ✅

| タスク | 状態 | 備考 |
|--------|------|------|
| Render GitHub連携 | ✅ 完了 | auto-deploy on commit |
| Koyeb GitHub連携 | ✅ 完了 | auto-deploy on commit |
| render.yaml追加 | ✅ 完了 | IaC設定 |
| 不要ファイル削除 | ✅ 完了 | .devcontainer/, compose/, infra/ |
| モジュール自動同期 | ✅ 完了 | sync_modules RPC |

---

## 作業ログ

### 2026-01-25

**OAuth Consent管理機能**
- `list_oauth_consents`, `revoke_oauth_consent`, `list_all_oauth_consents` RPC実装
- MCP接続ページ・管理者ページにOAuth認可管理UI追加

**パスルーティング設計**
- dsn-route.md作成
- 管理者Route Group分離 `(admin)/layout.tsx`

### 2026-01-26

**Airtableモジュール実装**
- 11ツール: list_bases, describe, query, get_record, create, update, delete, search_records, aggregate_records, create_table, update_table
- 新機能: テキスト検索、集計（group_by対応）

**モジュール自動同期**
- サーバー起動時にDBへ自動登録
- `sync_modules` RPC関数作成
- `apps/server/internal/store/module.go` 追加

**インフラ整備**
- Render/Koyeb: GitHub連携による自動デプロイ
- `render.yaml` 追加
- 不要ファイル削除: `.devcontainer/`, `compose/`, `infra/`

---

## 残タスク（優先度順）

1. **マイグレーションpush** - sync_modules RPC適用
2. **Phase 2: RPC呼び出しリファクタ** - Console/Worker/Goサーバーの統一
3. **Phase 5: ツール設定API** - tool_settingsテーブル・RPC作成
4. **Phase 4: UI要件定義** - spc-ui.md作成

---

## 参考資料

- [dsn-rpc.md](../../docs/design/dsn-rpc.md) - RPC関数設計書
- [dsn-route.md](../../docs/design/dsn-route.md) - ルート設計書
- [dtl-dsn-tbl.md](../../docs/design/dtl-dsn-tbl.md) - テーブル詳細設計書
