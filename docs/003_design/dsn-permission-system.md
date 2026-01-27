# MCPist 権限システム設計

## 概要

MCPサーバーにおけるツール権限管理の詳細設計。

関連ドキュメント:
- [adr-permission-naming.md](./adr-permission-naming.md) - 命名に関するADR
- [adr-b2c-focus.md](./adr-b2c-focus.md) - B2Cフォーカスに関するADR
- [dsn-tool-sieve.md](./dsn-tool-sieve.md) - 認証・認可アーキテクチャ全体

---

## ビジネスモデル: B2C（個人課金）

本システムはB2C（個人課金）モデルに限定する。

```
MCPist（ベンダー）
  │
  │ 直接課金
  ↓
ユーザー（個人）
```

**B2Bは対象外**:
- 組織単位の課金
- 組織管理者によるツール制御
- SSO、監査ログ等のエンタープライズ機能

詳細は [adr-b2c-focus.md](./adr-b2c-focus.md) を参照。

---

## ロール定義

Phase 1以降のスケールを見据えた設計。Phase 1では開発者＝管理者＝自分でも、ロールを分離しておく。

| ロール | 権限範囲 | 操作方法 | できること |
|--------|----------|----------|------------|
| **開発者** | システム全体 | コード、CLI、DB直接 | コード変更、デプロイ、DB操作、全設定 |
| **管理者** | ユーザー監視 | 管理UI | ユーザー状態の確認・変更（suspend等） |
| **ユーザー** | 自分の課金範囲 | ユーザーUI | 課金したツールのオン/オフ |

### 重要な制約

**管理者はツールの制御ができない**

```
× 管理者が「notion:create_pageを無効化」
○ 管理者が「ユーザーAをsuspend」
```

**理由**:
- ユーザーが課金したサービスを第三者が勝手に制限できない
- 管理者の権限が明確（監視・アカウント状態管理のみ）
- スケール時に権限の混乱を防ぐ

### ベンダー内の役割分離

```
ベンダー（MCPist運営）
  ├─ 開発者（プログラマ）
  │     └─ コード変更、デプロイ、システム設定
  │
  └─ 管理者（非プログラマ）
        └─ ユーザー管理、アカウント状態変更、サポート対応
```

---

## パッケージ構成

```
internal/
  permission/
    permission.go    ← Cache, Gate, Filter（最初は1ファイル、必要に応じて分割）
  auth/
    middleware.go    ← permission.Gate()を呼び出し
  mcp/
    handler.go       ← permission.Filter()を呼び出し
```

---

## 多層防御（Defense in Depth）

権限チェックを3層で実施し、セキュリティを強化する。

```
MCPクライアント
    │
    ↓
AuthMiddleware
    ├─ JWT検証
    └─ permission.Gate() ← 【1層目】早期リジェクト
          │
          ↓
mcp.Handler
    ├─ get_module_schema
    │     └─ permission.Filter() ← 【2層目】見えるツールを制限
    │
    └─ call
          └─ permission.Gate() ← 【3層目】実行時の最終防御
```

| 層 | 関数 | タイミング | 目的 |
|----|------|-----------|------|
| 1 | Gate (Middleware) | リクエスト受信時 | 不正リクエストを早期に弾く |
| 2 | Filter | スキーマ取得時 | 許可されたツールしか見せない |
| 3 | Gate (Handler) | ツール実行時 | Middleware迂回への最終防御 |

**なぜ多層か**:
- 1層目だけ: Middleware迂回バグで突破される可能性
- 2層目だけ: スキーマを見せないだけで、直接callされたら防げない
- 3層目だけ: 無駄なリクエストがHandlerまで到達してしまう

3層全てあることで、パフォーマンス（早期リジェクト）、UX（見えないツールは呼べない）、セキュリティ（最終防御）を全て満たす。

**パフォーマンス影響**: `GetAllowedTools()`はキャッシュを使用するため、複数回呼び出しても実質的なコストは低い。

---

## モジュールアクセス制御

`get_module_schema` 自体は常に呼び出せるが、許可ツールが0件のモジュールにはアクセス不可とする。

```go
// Handler内
func handleGetModuleSchema(ctx context.Context, req Request) Response {
    userID := GetUserID(ctx)
    allowed := cache.GetAllowedTools(userID)

    // そのモジュールに許可ツールが1つもなければエラー
    moduleTools := filterByModule(allowed, req.Module)
    if len(moduleTools) == 0 {
        return ErrorResponse("no access to module: " + req.Module)
    }

    // 通常処理（Filter適用）...
}
```

**理由**:
- 全ツールを禁止されたユーザーが空のスキーマを取得できる状態を防ぐ
- 管理者がツール単位で制御すれば、モジュール単位のアクセスも自動的に制御される

---

## ユーザーアカウント状態管理

管理者がユーザー全体のアクセスを制御できるよう、アカウント状態を導入する。

```go
type UserStatus string

const (
    UserStatusActive    UserStatus = "active"
    UserStatusSuspended UserStatus = "suspended"  // 一時停止
    UserStatusDisabled  UserStatus = "disabled"   // 無効化
)
```

AuthMiddlewareでアカウント状態をチェック：

```go
func AuthMiddleware(cache *PermissionCache, next Handler) Handler {
    return func(ctx context.Context, req Request) Response {
        userID, err := validateJWT(req)
        if err != nil {
            return ErrorResponse("unauthorized")
        }

        // アカウント状態チェック
        status := cache.GetUserStatus(userID)
        if status != UserStatusActive {
            return ErrorResponse("account is " + string(status))
        }

        // 以降は通常処理...
    }
}
```

**管理UI要件**:
- ユーザーのアカウント停止/有効化ボタン
- ユーザー一覧表示（状態、課金状況の確認）

**注**: 管理者はツール単位の権限操作ができない。ユーザー監視とアカウント状態管理のみ。

---

## batchツール対応

`batch`ツールは複数のツールを一括呼び出しする。権限チェックは**All or Nothing**方式を採用。

**戦略**: 事前に全件チェックし、1つでも拒否があればbatch全体を拒否する。

```go
func GateBatch(cache *Cache, userID string, calls []CallRequest) *BatchPermissionError {
    permissions := cache.GetToolPermissions(userID)
    deniedTools := []DeniedTool{}

    for _, call := range calls {
        fullName := call.Module + ":" + call.ToolName
        perm := permissions[fullName]

        if !perm.Allowed {
            deniedTools = append(deniedTools, DeniedTool{
                Tool:   fullName,
                Reason: perm.Reason,
                Hint:   getHintForReason(perm.Reason),
            })
        }
    }

    if len(deniedTools) == 0 {
        return nil  // 全て許可
    }

    return &BatchPermissionError{
        Message:     fmt.Sprintf("%d tool(s) not permitted", len(deniedTools)),
        DeniedTools: deniedTools,
    }
}
```

**エラーレスポンス例**:
```json
{
  "error": {
    "message": "2 tool(s) not permitted",
    "denied_tools": [
      {
        "tool": "notion:create_page",
        "reason": "not_subscribed",
        "hint": "Subscribe to Notion module: https://example.com/billing"
      },
      {
        "tool": "jira:create_issue",
        "reason": "user_disabled",
        "hint": "Enable this tool in your preferences: https://example.com/my/preferences"
      }
    ]
  }
}
```

**注**: `suspended`の場合はbatch全体がAuthMiddlewareで拒否されるため、denied_toolsには含まれない。

**All or Nothingを採用した理由**:
- 予測可能: 全成功 or 全失敗
- トランザクション性: 一貫性が高い
- エラー検出: 実行前に全て検出できる
- ロールバック不要

---

## 課金モデル

**課金単位: ユーザー個人**

```
ユーザーA → Notion, Jira を購入
ユーザーB → Notion のみ購入
ユーザーC → 無料枠のみ
```

組織の概念は不要。各ユーザーが自分で課金・管理する。

---

## 権限の決定フロー

```
アカウント状態
    │
    │ active / suspended / disabled
    ↓
suspended/disabled → 全ツール拒否（reason: "suspended"）
    │
    │ active の場合
    ↓
課金状態（ユーザー個人）
    │
    │ サービス単位で購入
    ↓
┌─────────────────────────────────────┐
│ ユーザーが購入したサービス            │
│ - Notion: ○                         │
│ - Jira: ○                           │
│ - GitHub: ×（未購入）                │
└─────────────────────────────────────┘
    │
    │ 未購入 → 拒否（reason: "not_subscribed"）
    │
    │ 購入済みの場合
    ↓
┌─────────────────────────────────────┐
│ ユーザーが有効にしたツール            │
│ - notion:search: ○                  │
│ - notion:create_page: ×（自分で無効化）│
└─────────────────────────────────────┘
    │
    │ オフ → 拒否（reason: "user_disabled"）
    │
    │ オンの場合
    ↓
許可
```

**ポイント**: 管理者はツール単位の制御ができない。アカウント状態（suspend）でのみユーザー全体を制御する。

---

## 拒否理由の詳細化

拒否理由を区別し、ユーザーに適切なアクションを提示する。

### 拒否理由の種類

```go
type DenyReason string

const (
    DenyReasonNone          DenyReason = ""               // 許可
    DenyReasonSuspended     DenyReason = "suspended"      // アカウント停止中
    DenyReasonNotSubscribed DenyReason = "not_subscribed" // 課金していない
    DenyReasonUserDisabled  DenyReason = "user_disabled"  // ユーザーがオフにした
)
```

| 理由 | ユーザーへのメッセージ | アクション |
|------|----------------------|-----------|
| `suspended` | 「アカウントが停止されています」 | サポートに連絡 |
| `not_subscribed` | 「このツールは有料プランで利用可能です」 | 課金ページへ誘導 |
| `user_disabled` | 「このツールは無効化されています」 | 設定ページへ誘導 |

**注**:
- `admin_disabled` は廃止。管理者はツール単位の制御ができない。
- `quota_exceeded` はPhase 1ではサービス単位課金（月額固定）のため不要。

### 拒否理由の優先順位

**優先順位**: アカウント状態 > 課金 > ユーザー設定

```
status != active → "suspended"（他の設定に関わらず）
subscription_ok = false → "not_subscribed"
subscription_ok = true, user_enabled = false → "user_disabled"
subscription_ok = true, user_enabled = true → 許可
```

### キャッシュ構造

**設計判断**: `Allowed` と `Reason` のみを持つ（個別フラグは持たない）

```go
type ToolPermission struct {
    Allowed bool
    Reason  DenyReason  // 拒否の場合のみ意味がある
}

type CacheEntry struct {
    UserID      string
    Status      UserStatus                    // active, suspended, disabled
    Role        string                        // superuser, user, trial
    Permissions map[string]ToolPermission     // "notion:create_page" → {Allowed, Reason}
    ExpiresAt   time.Time
}
```

**理由**:
- DBフェッチ時に1回だけ `Allowed` と `Reason` を計算
- キャッシュ参照時は計算不要、そのまま返す
- 個別フラグ（subscription_ok, user_enabled）を毎回ANDする必要がない

### DBからのフェッチ（計算はDB側で行う）

```sql
SELECT
    tool_name,
    (subscription_ok AND user_enabled) AS allowed,
    CASE
        WHEN NOT subscription_ok THEN 'not_subscribed'
        WHEN NOT user_enabled THEN 'user_disabled'
        ELSE ''
    END AS reason
FROM user_tool_permissions
WHERE user_id = ?
```

**注**: `admin_enabled`カラムは不要。アカウント状態（suspended等）は別クエリでユーザー情報として取得する。

### キャッシュのフェッチ粒度

**決定**: 1ユーザーの全権限を一度に取得

| 案 | 粒度 | 評価 |
|----|------|------|
| A | ユーザー単位 | ○ シンプル、1クエリで全取得 |
| B | モジュール単位 | △ 必要な分だけだが、管理が複雑 |
| C | ツール単位 | × DBアクセス増加 |

---

## キャッシュInvalidateトリガー

| イベント | Invalidate |
|----------|------------|
| ユーザーが課金変更 | `InvalidateUser(userID)` |
| ユーザーが自分の設定変更 | `InvalidateUser(userID)` |
| 管理者がユーザーをsuspend | `InvalidateUser(targetUserID)` |

**全て `InvalidateUser(userID)` で統一**。シンプル。

**注**: 管理者はツール単位の権限変更ができないため、「管理者がユーザー権限変更」のトリガーは不要。

---

## スーパーユーザー対応

スーパーユーザーは**PermissionGateをスキップしない**。代わりにPermissionCacheが全ツールを返す。

```go
func (c *PermissionCache) GetAllowedTools(userID string) []string {
    role := c.getUserRole(userID)
    if role == "superuser" {
        return c.registry.GetAllToolNames()  // 全ツールを返す
    }
    // 通常のキャッシュ処理...
}
```

**理由**:
- 同じ経路を通ることで例外的なバイパスを作らない
- スーパーユーザーも監査ログに同じ形式で記録される
- PermissionGateのロジック変更不要

---

## ロールベースの権限管理

### 新規ユーザーの初期権限

ハードコードせず、ロール設定で対応する。

```sql
roles (
    id          UUID PRIMARY KEY,
    name        TEXT,
    is_default  BOOLEAN  -- is_default=true が新規ユーザーに付与される
)

role_tool_permissions (
    role_id     UUID,
    tool_name   TEXT,
    allowed     BOOLEAN,
    PRIMARY KEY (role_id, tool_name)
)
```

**フロー**:
1. 管理者が「default」ロールを作成
2. 「default」ロールに初期許可サービス（例: Google Calendar, Notion, GitHub）を設定
3. 新規ユーザー作成時、自動的に「default」ロールを付与

**メリット**:
- 管理者がGUIで初期権限を変更できる
- ハードコードの変更不要
- トライアルも同様に「trial」ロールで対応可能

---

## キャッシュ戦略

### Phase 1: 単一インスタンス + TTL

```go
type CacheEntry struct {
    permissions map[string]ToolPermission
    expiresAt   time.Time  // 5分後
}
```

- 5分経過 → 自動的にDBから再取得
- 権限変更 → 最大5分で反映
- 即時反映が必要な場合 → `InvalidateUser(userID)` を呼び出し

### マルチインスタンス問題（将来）

```
                    ┌─────────────────┐
                    │  Load Balancer  │
                    └────────┬────────┘
              ┌──────────────┼──────────────┐
              ↓              ↓              ↓
        ┌─────────┐    ┌─────────┐    ┌─────────┐
        │ Server1 │    │ Server2 │    │ Server3 │
        │ Cache A │    │ Cache B │    │ Cache C │
        └─────────┘    └─────────┘    └─────────┘
```

| 案 | 説明 | 複雑度 | Phase |
|----|------|--------|-------|
| A. 単一インスタンス | サーバー1台で運用 | なし | Phase 1 |
| B. 短いTTL | 5分で自動失効、最大5分のズレを許容 | 低 | Phase 1 |
| C. 共有キャッシュ（Redis） | 全インスタンスで同じキャッシュを参照 | 中 | Phase 2 |
| D. Pub/Sub通知 | 権限変更を全インスタンスに通知 | 高 | Phase 3 |

**Phase 1の方針**: 単一インスタンス + TTL 5分
- ユーザー数: 5-10人 → 1台で十分
- 権限変更頻度: 低い → 5分のズレは許容範囲

---

## Phase 1 スコープ外

以下はPhase 1では実装しない。

| 項目 | 理由 |
|------|------|
| 監査ログ | 後から非同期実行で追加可能 |
| 権限の委譲 | ユースケースなし |
| エラーメッセージi18n | 英語のみで十分（LLMが翻訳） |

---

## Usage Meter（使用量カウンター）

課金実績のカウントを担当するコンポーネント。

### 名称定義

| 名称 | 役割 |
|------|------|
| **Usage Meter** | 使用量の計測・記録を担当するコンポーネント |
| **Usage Controller** | Usage Meterを含む、使用量制御全体を統括するコントローラー |

### 配置場所: Handler（MCPプロトコルハンドラ）

```
リクエスト
    │
    ├─ 1. Auth Middleware（JWT検証、user_id抽出）
    │
    ├─ 2. Permission Gate（Tool Sieve、権限チェック）
    │
    ├─ 3. Usage Meter: Check ← 事前チェック（Quota/Credit）
    │
    ├─ 4. MCP Handler（tools/call処理）
    │
    ├─ 5. Module Registry（ルーティング）
    │
    ├─ 6. Module（ツール実行）
    │
    └─ 7. Usage Meter: Record ← 実績記録（成功時のみ）
```

### なぜHandlerか

| 候補 | 評価 | 理由 |
|------|------|------|
| Middleware | ❌ | tools/listなど課金対象外のリクエストも通過する |
| Permission Gate | ❌ | 権限チェックと使用量制御は責務が異なる |
| **Handler** | ✅ | `tools/call`の実行前後でフックできる |
| Module Registry | ❌ | batch実行時に複数回呼ばれる、カウント重複のリスク |
| Module | ❌ | 各モジュールに実装が分散、漏れのリスク |

### 実装イメージ

```go
func (h *Handler) handleToolsCall(ctx context.Context, req *ToolsCallRequest) (*ToolsCallResponse, error) {
    userID := auth.GetUserID(ctx)
    toolName := req.Module + ":" + req.ToolName

    // 1. Usage Meter: 事前チェック（Quota/Credit）
    check, err := h.usageMeter.Check(ctx, userID, toolName)
    if err != nil {
        return nil, err
    }
    if !check.Allowed {
        return nil, &UsageError{Reason: check.Reason}
    }

    // 2. ツール実行
    result, err := h.registry.ExecuteTool(ctx, req)
    if err != nil {
        return nil, err
    }

    // 3. Usage Meter: 実績記録（成功時のみ）
    if err := h.usageMeter.Record(ctx, userID, toolName); err != nil {
        // ログに記録、レスポンスは返す（課金記録失敗でユーザー体験を損なわない）
        h.logger.Error("usage record failed", "error", err)
    }

    // 4. レスポンスにusage情報を付加
    result.Usage = check.ToUsageInfo()
    return result, nil
}
```

### batch実行時の扱い

```go
func (h *Handler) handleBatch(ctx context.Context, req *BatchRequest) (*BatchResponse, error) {
    userID := auth.GetUserID(ctx)

    // batchは1リクエストとしてカウント（内部の個別ツールはカウントしない）

    // 1. Usage Meter: 事前チェック
    check, err := h.usageMeter.Check(ctx, userID, "batch")
    if err != nil {
        return nil, err
    }
    if !check.Allowed {
        return nil, &UsageError{Reason: check.Reason}
    }

    // 2. batch実行
    result, err := h.registry.ExecuteBatch(ctx, req)
    if err != nil {
        return nil, err
    }

    // 3. Usage Meter: 1回だけ記録
    if err := h.usageMeter.Record(ctx, userID, "batch"); err != nil {
        h.logger.Error("usage record failed", "error", err)
    }

    return result, nil
}
```

### Usage Meterの責務

| 責務 | 説明 |
|------|------|
| **Check** | Quota残量・Credit残高の事前確認 |
| **Record** | 使用量の記録（usageテーブル）、Credit消費（creditsテーブル） |

### 記録先

| 制御 | テーブル | 用途 |
|------|----------|------|
| Quota | `usage` | 月間使用量カウント |
| Credit | `credits`, `credit_transactions` | クレジット残高、消費履歴 |

詳細は [dsn-subscription.md](./dsn-subscription.md) を参照。

---

## 処理フロー

### call の場合

```
MCPクライアント
    │ tools/call("call", {module: "notion", tool_name: "create_page"})
    ↓
AuthMiddleware
    ├─ JWT検証 → user_id抽出
    └─ PermissionGate(cache, userID, "notion", "create_page") ← 【1層目】
          ├─ cache.GetAllowedTools(userID) → ["notion:search", "notion:get_page"]
          ├─ "notion:create_page" in allowed?
          │     └─ No → error返却 → レスポンス: "tool not permitted"
          └─ Yes → nil返却 → 次へ
                ↓
mcp.Handler (call処理)
    ├─ PermissionGate(cache, userID, "notion", "create_page") ← 【3層目】最終防御
    │     └─ キャッシュヒット → 高速判定
    └─ 許可 → call実行
```

### get_module_schema の場合

```
MCPクライアント
    │ tools/call("get_module_schema", {module: "notion"})
    ↓
AuthMiddleware
    ├─ JWT検証 → user_id抽出
    └─ PermissionGate → get_module_schemaはゲート対象外（通過）
          ↓
mcp.Handler (get_module_schema処理)
    ├─ Registry.GetSchema("notion") → 全14ツールのスキーマ
    ├─ PermissionFilter(cache, userID, "notion", allTools)
    │     └─ cache.GetAllowedTools(userID) → ["notion:search", "notion:get_page"]
    │     └─ フィルタ → 2ツールのスキーマのみ
    └─ レスポンス: フィルタ済みスキーマ
```

---

## 詳細設計（コード例）

### PermissionCache

```go
type PermissionCache struct {
    cache    sync.Map  // user_id → *CacheEntry
    db       *supabase.Client
    registry *modules.Registry
}

type CacheEntry struct {
    permissions map[string]ToolPermission
    expiresAt   time.Time
}

// GetAllowedTools - ユーザーが使えるツール一覧を返す（キャッシュ優先）
func (c *PermissionCache) GetAllowedTools(userID string) []string {
    // スーパーユーザーは全ツール
    role := c.getUserRole(userID)
    if role == "superuser" {
        return c.registry.GetAllToolNames()
    }

    // 通常ユーザーはキャッシュ
    if entry, ok := c.cache.Load(userID); ok {
        if time.Now().Before(entry.(*CacheEntry).expiresAt) {
            return getAllowedFromEntry(entry.(*CacheEntry))
        }
    }
    entry := c.fetchFromDB(userID)
    c.cache.Store(userID, entry)
    return getAllowedFromEntry(entry)
}

// InvalidateUser - 特定ユーザーのキャッシュをクリア（課金変更時等）
func (c *PermissionCache) InvalidateUser(userID string) {
    c.cache.Delete(userID)
}

// InvalidateAll - 全キャッシュをクリア
func (c *PermissionCache) InvalidateAll() {
    c.cache = sync.Map{}
}
```

### PermissionGate

```go
// PermissionGate - ツール呼び出しの許可チェック
// 許可されていない場合はerrorを返す
func PermissionGate(cache *PermissionCache, userID string, module string, toolName string) error {
    perm := cache.GetToolPermission(userID, module+":"+toolName)

    if !perm.Allowed {
        return &PermissionError{
            Tool:   module + ":" + toolName,
            Reason: perm.Reason,
            Hint:   getHintForReason(perm.Reason),
        }
    }
    return nil
}
```

### PermissionFilter

```go
// PermissionFilter - スキーマをユーザー権限でフィルタ
// 許可されたツールのスキーマのみを返す
func PermissionFilter(cache *PermissionCache, userID string, module string, tools []ToolSchema) []ToolSchema {
    var filtered []ToolSchema
    for _, tool := range tools {
        perm := cache.GetToolPermission(userID, module+":"+tool.Name)
        if perm.Allowed {
            filtered = append(filtered, tool)
        }
    }
    return filtered
}
```

### AuthMiddleware（統合版）

```go
func AuthMiddleware(cache *PermissionCache, next Handler) Handler {
    return func(ctx context.Context, req Request) Response {
        // 1. JWT検証
        userID, err := validateJWT(req)
        if err != nil {
            return ErrorResponse("unauthorized")
        }
        ctx = WithUserID(ctx, userID)

        // 2. アカウント状態チェック
        status := cache.GetUserStatus(userID)
        if status != UserStatusActive {
            return ErrorResponse("account is " + string(status))
        }

        // 3. 権限チェック（callの場合のみ）
        if req.Method == "tools/call" && req.Params.Name == "call" {
            if err := PermissionGate(cache, userID, req.Params.Module, req.Params.ToolName); err != nil {
                return ErrorResponse(err.Error())
            }
        }

        return next(ctx, req)
    }
}
```
