# DAY016 バックログ

## 概要

Sprint-005 の DAY016 完了時点のバックログ。13タスク完了、3タスク未着手。
Sprint-005 は Phase 4, Phase 5 の一部を残して終了とする。

---

## DAY016 完了タスク

| ID | タスク | 備考 |
|----|--------|------|
| D16-001 | サービス接続時のデフォルトツール設定自動保存 | BL-001。MCP Annotations移行含む |
| D16-003 | next.config.ts デバッグログ削除 | B-006 |
| D16-004 | VitePress docsビルド修正 | rewrites でクリーンURL対応 |
| D16-007 | get_module_schema 複数モジュール対応 + ツールフィルタリング | 配列入力、DisabledTools除外、DynamicMetaTools |
| D16-008 | Observability — Loki統合 + X-Request-IDトレーシング | 構造化ログ、エンドツーエンドトレース |
| D16-009 | Batch権限チェック + クレジット残高事前検証 | All-or-Nothing、セキュリティログ |
| D16-010 | Go Server MCP Tool Annotations 実装 | Dangerous bool 完全削除、115ツール annotations |
| D16-011 | Console MCP接続設定から type:sse 削除 | MCP仕様準拠 |
| D16-012 | Console /mcp → /connections リネーム | ルート名の混同回避 |
| D16-013 | gRPC移行構想の実装可能性評価 | REJECTED判定 |

---

## 残タスク

### Sprint-005 残 (Phase 4: UI要件定義)

| ID | タスク | 状態 | 備考 |
|----|--------|------|------|
| S5-060 | UI要求仕様書作成 (spc-ui.md) | 未着手 | 画面一覧・機能要件 |
| S5-061 | ユーザーフロー図作成 | 未着手 | 主要フローの可視化 |
| S5-062 | 画面遷移図作成 | 未着手 | 認証後のナビゲーション |

### Sprint-005 残 (Phase 5: ツール設定API)

| ID | タスク | 状態 | 備考 |
|----|--------|------|------|
| S5-076 | CIにtools.json検証追加 | 未着手 | Go出力との差分チェック自動化 |
| S5-077 | 未使用Goモジュール削除 | 未着手 | クリーンアップ |

※ S5-070〜075 は DAY16 で完了済み（tools-export, tools.json生成, tool_settings RPC, /tools ページ対応）

### DAY016 計画の残

| ID | タスク | 状態 | 備考 |
|----|--------|------|------|
| D16-002 | 仕様書の実装追従更新 | 未着手 | spec-impl-compare.md の差分を仕様書に反映 |
| D16-006 | E2Eテスト設計 | 未着手 | OAuth認可フロー等 |

### 仕様書・実装差分（spec-impl-compare.md 未修正分）

| ID | タスク | 状態 | 対象ファイル | 備考 |
|----|--------|------|-------------|------|
| BL-010 | Rate Limit記述の更新 | 未着手 | spc-dsn.md | 実装では削除済み。仕様から削除または「将来実装予定」に変更 |
| BL-011 | JWT `aud`チェック要件の整理 | 未着手 | spc-itf.md | 実装では明示チェックなし |
| BL-012 | MCP拡張エラーコード(2001-2005)の整理 | 未着手 | spc-itf.md | 実装ではJSON-RPC標準コードのみ |
| BL-013 | Console API設計の更新 | 未着手 | spc-itf.md | REST API定義→Supabase RPC直接呼び出し方式に更新 |
| BL-014 | PSP Webhook仕様の整理 | 未着手 | spc-itf.md | 未実装。仕様から削除または「将来実装予定」に変更 |

### CON-DST 仕様・実装差分

| ID | タスク | 状態 | 対象ファイル | 備考 |
|----|--------|------|-------------|------|
| BL-015 | enabled_modules 参照API実装 | 未着手 | dtl-itr-CON-DST.md | 仕様: 登録/更新可。実装: module_settingsテーブル未使用 |
| BL-016 | user_prompts 管理UI実装 | 未着手 | dtl-itr-CON-DST.md | 仕様: 登録/更新/削除可。実装: テーブルのみ、UI未実装 |
| BL-017 | usage_stats 参照API実装 | 未着手 | dtl-itr-CON-DST.md | 仕様: 参照のみ。実装: 未実装 |
| BL-018 | クレジット付与機能（CON→DST） | 未着手 | dtl-itr-CON-DST.md | 仕様: CONから任意整数分のクレジット付与。実装: トリガーで初期化のみ |

---

## Sprint-005 Phase別 最終進捗

| Phase | 状態 | 進捗 |
|-------|------|------|
| Phase 1: RPC関数実装 | **完了** | 17/17 (100%) |
| Phase 2: RPC呼び出しリファクタ | **完了** | 9/9 (100%) |
| Phase 3: パスルーティング設計 | **完了** | 3/3 (100%) |
| Phase 4: UI要件定義 | 未着手 | 0/3 (0%) |
| Phase 5: ツール設定API | **大部分完了** | 6/8 (75%) |
| Phase 6: モジュール拡張 | **完了** | 2/2 (100%) |
| Phase 7: OAuth トークンリフレッシュ | **完了** | Google + Microsoft |

---

## Sprint-006 への引き継ぎ候補

| 優先度 | タスク | 備考 |
|--------|--------|------|
| 高 | 仕様書の実装追従更新 (D16-002 + BL-010〜014) | 仕様と実装の乖離が拡大中 |
| 中 | CIにtools.json検証追加 (S5-076) | Go出力との差分チェック自動化 |
| 中 | E2Eテスト設計 (D16-006) | 品質保証の基盤 |
| 低 | UI要件定義 (S5-060〜062) | 実装が先行しており緊急度低 |
| 低 | 未使用Goモジュール削除 (S5-077) | クリーンアップ |
| 保留 | 切断時ツール設定クリーンアップ (BL-002) | DB migration前提 |

---

## 参考

- [day016-worklog.md](./day016-worklog.md) - 作業ログ
- [day016-plan.md](./day016-plan.md) - 計画
- [day015-backlog.md](./day015-backlog.md) - 前日バックログ
- [sprint005.md](../sprint/sprint005.md) - スプリント計画
