# DAY014 バックログ

## 概要

Sprint-005の進行中バックログ。RPC実装・モジュール拡張・インフラ整備を完了し、次フェーズへ移行。

---

## 完了タスク

### インフラ・デプロイ ✅

| タスク | 完了日 |
|--------|--------|
| Render GitHub連携（auto-deploy） | 2026-01-26 |
| Koyeb GitHub連携（auto-deploy） | 2026-01-26 |
| render.yaml追加 | 2026-01-26 |
| 不要ファイル削除（.devcontainer, compose, infra） | 2026-01-26 |

### RPC関数実装 ✅

| 関数名 | 完了日 |
|--------|--------|
| list_oauth_consents | 2026-01-25 |
| revoke_oauth_consent | 2026-01-25 |
| list_all_oauth_consents | 2026-01-25 |
| sync_modules | 2026-01-26 |

### モジュール拡張 ✅

| モジュール | ツール数 | 完了日 |
|-----------|---------|--------|
| Airtable | 11 | 2026-01-26 |

---

## 残タスク

### 優先度: 高

| ID | タスク | 状態 | 備考 |
|----|--------|------|------|
| B-001 | マイグレーションpush | ⬜ | sync_modules RPC適用 |
| B-002 | Phase 2: RPC呼び出しリファクタ | ⬜ | Console/Worker/Go統一 |

### 優先度: 中

| ID | タスク | 状態 | 備考 |
|----|--------|------|------|
| B-003 | Phase 5: ツール設定API | ⬜ | tool_settingsテーブル・RPC |
| B-004 | Phase 4: UI要件定義 | ⬜ | spc-ui.md作成 |
| B-005 | next.config.ts デバッグログ削除 | ⬜ | console.log削除 |

### 優先度: 低

| ID | タスク | 状態 | 備考 |
|----|--------|------|------|
| B-006 | E2Eテスト設計 | ⬜ | OAuth認可フロー等 |
| B-007 | update_module_token RPC | ⬜ | トークンリフレッシュ用 |

---

## 技術的負債

### Supabase RPC型定義

新しいRPCを追加した場合、`database.types.ts`に手動で型定義を追加する必要あり。

### 型チェック

デプロイ前に `pnpm exec next build` を実行して型エラーを検出。

---

## デプロイ手順

### 現在の方式（GitHub連携）

mainブランチにpushするとRender/Koyebが自動デプロイ。

### マイグレーション

```bash
cd supabase
supabase db push
```

---

## 参考

- [sprint-005.md](./sprint-005.md) - 詳細タスク一覧
- [dsn-route.md](../../docs/design/dsn-route.md) - ルート設計
