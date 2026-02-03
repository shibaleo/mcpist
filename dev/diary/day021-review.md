# DAY021 振り返り（学び）

## 1. MCP 仕様の理解

### prompts/list と prompts/get の違い

- **prompts/list**: `name` + `description`（短い説明）のみ返す
- **prompts/get**: `messages` 配列で完全なコンテンツを返す

最初は list で content を返していたが、これは仕様違反。MCPクライアントは list で概要を確認し、get で詳細を取得する設計。

**教訓**: 仕様を先に読む。実装してから「動いた」で終わらせない。

---

## 2. プロンプトの書き方

### AIに正しいツール名を使わせるには

プロンプト「今日のタスクを取得して」だけでは、AIは `list_task_lists`（推測）で呼んでしまう。

**問題**: 実際のツール名は `list_lists`

**解決**: プロンプトに「まず get_module_schema で利用可能なツールを確認してください」と明記。

**教訓**: AIはスキーマを知らない状態で推測する。プロンプトで明示的にスキーマ取得を指示する。

---

## 3. 楽観的更新パターン

### UI即時反映 + 非同期保存 + 失敗時ロールバック

```typescript
// 1. 楽観的にUIを更新
setLocalState(newValue)

// 2. バックグラウンドで保存
try {
  await save(newValue)
  setPersistedState(newValue)
} catch {
  // 3. 失敗したら元に戻す
  setLocalState(oldValue)
  toast.error("保存失敗")
}
```

**教訓**: 保存ボタンを押して待つより、トグル即反映の方がUX良い。ただし失敗時のロールバック必須。

---

## 4. PostgreSQL 関数の戻り値変更

### `cannot change return type of existing function`

`CREATE OR REPLACE FUNCTION` は戻り値の型を変更できない。

**解決**: `DROP FUNCTION IF EXISTS` を先に実行してから `CREATE`。

```sql
DROP FUNCTION IF EXISTS my_function(uuid);
CREATE FUNCTION my_function(uuid) RETURNS new_type ...
```

**教訓**: 関数シグネチャを変える場合は DROP + CREATE。マイグレーションで注意。

---

## 5. モジュール追加の優先順位

### 機能追加 vs 品質向上

- **機能追加**（モジュール）: テストユーザーへの価値直結、フィードバック収集
- **品質向上**（BL-070〜073）: 地味だが重要

**結論**: 機能を先に追加し、フィードバックを得てから品質を磨く。

ただしセキュリティ（BL-071）は本番公開前に必須。並行して設計だけ進める。

---

## 6. 認証方式の選択

| 方式 | メリット | デメリット |
|------|---------|-----------|
| OAuth 2.0 | 標準的、セキュア | 実装複雑、リフレッシュ管理 |
| API Key | シンプル | ユーザーが手動発行 |
| Connection String | 直接接続 | セキュリティリスク高 |

**教訓**: 既存の OAuth 基盤があるなら流用。API Key はユーザー負担だが実装は楽。

---

## 7. コア機能の完了確認

今日の確認で、テストユーザー向けコア機能は達成済みと判明：

- 認証・認可 ✅
- MCP tools ✅
- MCP prompts ✅
- モジュール 9個 ✅
- Console UI ✅
- Claude Code / Web 連携 ✅

**教訓**: 定期的に「何ができているか」を棚卸しする。次に何をやるべきかが見える。

---

## まとめ

1. 仕様を先に読む
2. AIへの指示は明示的に（スキーマ取得を含めて）
3. 楽観的更新でUX向上
4. 機能追加でフィードバックを得てから品質向上
5. 定期的に達成状況を棚卸し
