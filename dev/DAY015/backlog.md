# DAY015 バックログ

## 概要

Sprint-005継続。RPC呼び出しリファクタとツール設定APIを進行。

---

## 完了タスク

| ID | タスク | 備考 |
|----|--------|------|
| B-001 | マイグレーションpush | ✅ sync_modules + update_module_token + OAuth認可フロー適用済み |
| B-002 | database.types.ts RPC型定義追加 | ✅ 既に完了済み |
| B-003a | Console側RPC呼び出しリファクタ | ✅ 全ページRPC利用済み |
| B-003b | Worker側RPC呼び出しリファクタ | ✅ lookup_user_by_key_hash使用済み |
| B-003c | Go Server側RPC呼び出しリファクタ | ✅ get_user_context, consume_credit, get_module_token使用済み |
| B-008 | update_module_token RPC | ✅ 新規実装完了 |

---

## 残タスク

### 優先度: 中

| ID | タスク | 状態 | 備考 |
|----|--------|------|------|
| B-004 | Phase 5: ツール設定API | ⬜ | tool_settingsテーブル・RPC |
| B-005 | Phase 4: UI要件定義 | ⬜ | spc-ui.md作成 |
| B-006 | next.config.ts デバッグログ削除 | ⬜ | console.log削除 |

### 優先度: 低

| ID | タスク | 状態 | 備考 |
|----|--------|------|------|
| B-007 | E2Eテスト設計 | ⬜ | OAuth認可フロー等 |

---

## Sprint-005 Phase別進捗

| Phase | 状態 | 進捗 |
|-------|------|------|
| Phase 1: RPC関数実装 | ✅ 完了 | 17/17 (100%) |
| Phase 2: RPC呼び出しリファクタ | ✅ 完了 | 9/9 (100%) |
| Phase 3: パスルーティング設計 | ✅ 完了 | 3/3 (100%) |
| Phase 4: UI要件定義 | ⬜ 未着手 | 0/3 (0%) |
| Phase 5: ツール設定API | 🔄 進行中 | 2/8 (25%) |
| Phase 6: モジュール拡張 | ✅ 完了 | 1/1 (100%) |

---

## デプロイ手順

### GitHub連携（自動）

mainブランチにpushすると自動デプロイ:
- **Render**: Go Server
- **Koyeb**: Go Server
- **Vercel**: Console (Next.js)

### Supabase（手動）

ローカルからマイグレーションをpush:
```bash
cd supabase
supabase db push
```

※ 開発時はクラウドのSupabaseを参照しながらローカルサーバーで開発

---

## 参考

- [sprint-005.md](../DAY014/sprint-005.md) - スプリント詳細
- [dsn-route.md](../../docs/design/dsn-route.md) - ルート設計
- [dsn-rpc.md](../../docs/design/dsn-rpc.md) - RPC設計
