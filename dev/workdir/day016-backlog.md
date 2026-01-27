# DAY016 バックログ

## 概要

D16-001（デフォルトツール設定自動保存）とConsole UI改善を完了。仕様書更新（D16-002）は主要5項目を修正済み、残り5項目が未修正。

---

## 完了タスク

| ID | タスク | 備考 |
|----|--------|------|
| BL-001 | サービス接続時にデフォルトツール設定を自動保存 | D16-001。MCP Annotations移行含む |
| BL-003 | Token Vault仕様書をSupabase RPC方式に更新 | itf-tvl.md v2.0 (DAY16) |
| BL-004 | メタツール名 `call` → `run` 修正 | spc-itf.md |
| BL-005 | Tokenプレフィックス `mcpist_` → `mpt_` 修正 | spc-itf.md |
| BL-006 | Auth Server仕様をSupabase Auth + Worker構成に更新 | idx-ept.md |
| BL-007 | Token Refresh責務を仕様書に反映 | itf-tvl.md（モジュール側で実装） |

---

## 残タスク

### 仕様書・実装差分（spec-impl-compare.md 未修正分）

| ID | タスク | 状態 | 対象ファイル | 備考 |
|----|--------|------|-------------|------|
| BL-010 | Rate Limit記述の更新 | ⬜ | spc-dsn.md | 実装では削除済み。仕様から削除または「将来実装予定」に変更 |
| BL-011 | JWT `aud`チェック要件の整理 | ⬜ | spc-itf.md | 実装では明示チェックなし。仕様の要件を実装に合わせるか、実装側で追加するか判断が必要 |
| BL-012 | MCP拡張エラーコード(2001-2005)の整理 | ⬜ | spc-itf.md | 実装ではJSON-RPC標準コードのみ。仕様を実装に合わせるか、実装側で追加するか判断が必要 |
| BL-013 | Console API設計の更新 | ⬜ | spc-itf.md | `/api/dashboard`等のREST API定義→Supabase RPC直接呼び出し方式に更新 |
| BL-014 | PSP Webhook仕様の整理 | ⬜ | spc-itf.md | 未実装。仕様から削除または「将来実装予定」に変更 |

### その他

| ID | タスク | 状態 | 備考 |
|----|--------|------|------|
| BL-002 | 切断時のツール設定クリーンアップ | 保留 | DB migration必要。データ構造変更が前提 |
| BL-020 | next.config.ts デバッグログ削除 | ⬜ | B-006からの引き継ぎ |
| BL-021 | VitePress docsビルド修正 | ⬜ | デッドリンク削除、ビルド成功させる |
| BL-022 | Phase 4: UI要件定義 | ⬜ | spc-ui.md作成 |
| BL-023 | E2Eテスト設計 | ⬜ | OAuth認可フロー等 |

---

## 参考

- [day016-worklog.md](./day016-worklog.md) - 作業ログ
- [day016-plan.md](./day016-plan.md) - 計画
- [day015-backlog.md](./day015-backlog.md) - 前日バックログ
- [spec-impl-compare.md](./spec-impl-compare.md) - 仕様・実装差分
