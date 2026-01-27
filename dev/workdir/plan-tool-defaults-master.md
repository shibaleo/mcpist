# 計画書: ツール定義マスタ管理（管理者設定）

## 日付

2026-01-27

---

## 背景

### 現状

- ツール定義（id, name, description, defaultEnabled, dangerous）は `tools.json` にハードコード
- Go Server は各モジュールの `tools.go` でツール定義（id, name, description, inputSchema）を持つ
- `tools.json` はConsoleのビルド資材としてコミットされ、手動管理
- `defaultEnabled` / `dangerous` を変更するにはコード変更＋デプロイが必要

### 問題

- `defaultEnabled` / `dangerous` はビジネスルールであり、デプロイなしで変えたい
- Go Server にツールを追加したとき `tools.json` を手動で同期する必要がある
- 管理者がツールのリスクレベルや初期設定を柔軟に管理できない

---

## 設計方針

### 責務分離

| 管理主体 | 内容 | 保存先 |
|----------|------|--------|
| Go Server（コード） | id, name, description, inputSchema | `tools.go` 各モジュール |
| Go Server（ビルド時） | id, name, description | `tools.json`（自動生成） |
| DB（管理者設定） | default_enabled, dangerous | `tool_defaults` テーブル |

### データフロー

```
[Go Server] 起動 / ビルド時
    │
    ├─ sync_modules RPC → modules テーブルに名前同期（既存）
    ├─ sync_tool_definitions RPC → tool_definitions テーブルにツール定義同期（新規）
    └─ tools.json 自動生成 → Console が静的参照（ビルド時）
           │
[Console 管理画面]
    │
    ├─ tool_definitions + tool_defaults を表示
    └─ 管理者が default_enabled / dangerous を変更 → tool_defaults テーブルに保存
           │
[サービス接続時]
    │
    └─ saveDefaultToolSettings → tool_defaults テーブルの値を参照して
                                  ユーザーの tool_settings に初期設定を保存
```

---

## 実装計画

### Phase 1: DB テーブル追加

#### 1-1. `tool_definitions` テーブル（新規）

Go Server から同期されるツール定義のマスタ。

```sql
CREATE TABLE mcpist.tool_definitions (
    module_id UUID NOT NULL REFERENCES mcpist.modules(id) ON DELETE CASCADE,
    tool_name TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (module_id, tool_name)
);
```

#### 1-2. `tool_defaults` テーブル（新規）

管理者が設定するツールのデフォルト値。

```sql
CREATE TABLE mcpist.tool_defaults (
    module_id UUID NOT NULL REFERENCES mcpist.modules(id) ON DELETE CASCADE,
    tool_name TEXT NOT NULL,
    default_enabled BOOLEAN NOT NULL DEFAULT true,
    dangerous BOOLEAN NOT NULL DEFAULT false,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_by UUID REFERENCES mcpist.users(id),
    PRIMARY KEY (module_id, tool_name)
);
```

**設計判断**: `tool_definitions` と `tool_defaults` を分離する理由：

- `tool_definitions` はGoサーバーが書き込む（自動同期）
- `tool_defaults` は管理者が書き込む（手動設定）
- 書き込み主体が異なるため別テーブルにする
- GoサーバーがSync時に管理者設定を上書きしてしまうリスクを排除

### Phase 2: RPC 追加

#### 2-1. `sync_tool_definitions` RPC（新規）

Go Server 起動時に呼び出す。ツール定義をDBに同期。

```sql
-- パラメータ: p_definitions JSONB[]
-- 各要素: {"module": "notion", "tool": "search", "description": "Search..."}
-- 処理:
--   1. module名からmodule_idを解決
--   2. tool_definitions に UPSERT
--   3. DBにあるがGoから来なかったツールは削除（orphan cleanup）
```

#### 2-2. `get_tool_defaults` RPC（新規）

管理者画面 / saveDefaultToolSettings から呼び出す。

```sql
-- パラメータ: p_module_name TEXT（NULLで全モジュール）
-- 戻り値: tool_definitions LEFT JOIN tool_defaults
--   module_name, tool_name, description,
--   default_enabled (COALESCE with true),
--   dangerous (COALESCE with false)
```

#### 2-3. `upsert_tool_defaults` RPC（新規）

管理者がdefault_enabled / dangerousを更新する。

```sql
-- パラメータ: p_module_name TEXT, p_tool_name TEXT, p_default_enabled BOOLEAN, p_dangerous BOOLEAN
-- 処理: tool_defaults に UPSERT、updated_by = auth.uid()
-- 権限: admin ロールのみ
```

### Phase 3: Go Server 変更

#### 3-1. `sync_tool_definitions` 呼び出し

サーバー起動時に `sync_modules` と合わせて `sync_tool_definitions` を呼び出す。

```go
// 全モジュールのツール定義を収集
var definitions []map[string]string
for name, mod := range registry {
    for _, tool := range mod.Tools() {
        definitions = append(definitions, map[string]string{
            "module":      name,
            "tool":        tool.Name,
            "description": tool.Description,
        })
    }
}
// RPC呼び出し
supabase.RPC("sync_tool_definitions", map[string]any{
    "p_definitions": definitions,
})
```

#### 3-2. tools.json 自動生成（任意）

Go Server ビルド時にtools.jsonを生成するスクリプト。Console側はDB参照に移行後、tools.jsonは不要になるが、フォールバックとして残す。

### Phase 4: Console 管理画面

#### 4-1. 管理者ツール設定ページ

`/admin/tool-defaults` に新規ページを追加。

- モジュール一覧表示
- モジュール選択 → ツール一覧表示
- 各ツールの `default_enabled` / `dangerous` をトグルで変更
- 変更は即座にDBに保存（`upsert_tool_defaults` RPC）

#### 4-2. saveDefaultToolSettings の修正

tools.json ではなく `get_tool_defaults` RPC からデフォルト値を取得するように変更。

```typescript
// 変更前: tools.json から取得
const mod = getModule(moduleName)

// 変更後: DB から取得
const { data } = await supabase.rpc("get_tool_defaults", {
  p_module_name: moduleName,
})
```

---

## 実装順序

| Step | 内容 | 依存 |
|------|------|------|
| 1 | マイグレーション: tool_definitions + tool_defaults テーブル作成 | - |
| 2 | RPC: sync_tool_definitions 実装 | Step 1 |
| 3 | RPC: get_tool_defaults + upsert_tool_defaults 実装 | Step 1 |
| 4 | Go Server: 起動時に sync_tool_definitions 呼び出し | Step 2 |
| 5 | Console: 管理画面 /admin/tool-defaults ページ作成 | Step 3 |
| 6 | Console: saveDefaultToolSettings を DB 参照に変更 | Step 3 |
| 7 | tools.json の defaultEnabled / dangerous を削除（id, name, description のみに縮小） | Step 6 |

---

## 変更ファイル一覧

| ファイル | 変更内容 |
|----------|----------|
| `supabase/migrations/XXXX_tool_definitions.sql` | テーブル作成 |
| `supabase/migrations/XXXX_rpc_tool_definitions.sql` | RPC 3本 |
| `apps/server/internal/modules/registry.go` | sync_tool_definitions 呼び出し追加 |
| `apps/console/src/app/(admin)/admin/tool-defaults/page.tsx` | 管理画面 新規 |
| `apps/console/src/lib/tool-settings.ts` | saveDefaultToolSettings を DB 参照に変更 |
| `apps/console/src/lib/tool-defaults.ts` | 管理者用 RPC ラッパー 新規 |
| `apps/console/src/lib/tools.json` | defaultEnabled / dangerous 削除 |

---

## 整合性チェック

Go Server 起動時のフロー:

```
1. sync_modules(["notion", "github", ...])     ← 既存
2. sync_tool_definitions([{module, tool, description}, ...])  ← 新規
   ├─ DBにないツール → INSERT
   ├─ 既存ツール → UPDATE (description変更時)
   └─ Goから来なかったツール → DELETE (orphan cleanup)
3. tool_defaults にエントリがないツール → default_enabled=true, dangerous=false として扱う
   （COALESCE で処理。管理者が未設定のツールは安全側デフォルト）
```

**管理者が設定変更した場合の影響範囲:**

- 新規ユーザーの接続時: 変更後の tool_defaults が適用される
- 既存ユーザー: 既に tool_settings に保存済みなので影響なし（意図通り）

---

## GPT Custom Connector との関連（参考）

GPTのカスタムコネクタ（Actions）では、OpenAPI仕様ベースでツールメタ情報を渡す仕様がある。MCPistのアプローチはこれと似ているが、MCPプロトコル経由で `get_module_schema` として動的にツール定義を返す点が異なる。管理者マスタ化により、GPTの「ツール有効/無効」管理に近い体験を実現できる。

---

## 備考

- 本計画は D16-001（デフォルトツール設定自動保存）の拡張として位置づける
- D16-001 は tools.json ハードコードのまま完了済み
- 本計画は次スプリントのバックログに追加
