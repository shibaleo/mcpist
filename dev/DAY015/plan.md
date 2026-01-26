# DAY015 計画

## 日付

2026-01-26

---

## 概要

Sprint-005の継続。RPC呼び出しリファクタ、OAuth認証モジュール実装（Google Calendar, Microsoft To Do）を完了。

---

## 本日の成果

### 完了

| タスク | 備考 |
|--------|------|
| マイグレーションpush | sync_modules + update_module_token + OAuth RPC適用 |
| Phase 2: RPC呼び出しリファクタ | Console/Worker/Go Server 全て完了 (9/9) |
| Phase 7: OAuth トークンリフレッシュ | Google Calendar対応 (4/4) |
| Microsoft To Do OAuth実装 | authorize/callback + Goモジュール (11ツール) |
| services.json / tools.json更新 | Microsoft To Do追加 |

### 未着手

| タスク | 備考 |
|--------|------|
| next.config.ts デバッグログ削除 | console.log削除 |
| Phase 5: ツール設定API | tool_settingsテーブル・RPC |
| Phase 4: UI要件定義 | spc-ui.md作成 |

---

## 次のタスク候補

1. **ツール設定API** - tool_settingsテーブル・RPC実装
2. **カスタムプロンプト** - ユーザー定義プロンプト機能

---

## 参考

- [sprint-005.md](../DAY014/sprint-005.md) - スプリント詳細
- [worklog.md](./worklog.md) - 作業ログ
- [review.md](./review.md) - 振り返り
