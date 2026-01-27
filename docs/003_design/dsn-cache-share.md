# MCPist キャッシュ共有設計

## 概要

マルチインスタンス構成（Koyeb + Fly.io）における権限キャッシュの同期戦略。

**結論**: Supabase Realtime によるイベント駆動キャッシュ無効化を採用。Redis等の共有ストレージは不要。

関連ドキュメント:
- [dsn-infrastructure.md](./dsn-infrastructure.md) - インフラ全体構成
- [dsn-permission-system.md](./dsn-permission-system.md) - 権限システム設計

---

## 課題

### マルチインスタンスでのキャッシュ不整合

```
シナリオ: 権限変更が片方のインスタンスに反映されない

1. 管理者が role_permissions を更新
2. Koyeb: キャッシュ無効化 → 新しい権限を取得
3. Fly.io: キャッシュ TTL 残り4分 → 古い権限のまま
4. 同一ユーザーがインスタンス間で異なる結果を得る
```

### DB負荷問題

```
権限チェック = 毎リクエスト必須

1,000 MAU × 10 req/日 = 10,000 req/日
↓
TTL 5分のローカルキャッシュでも、キャッシュミス時はDB読み取り発生
↓
Supabase DB転送量 2GB/月 の無料枠を圧迫
```

---

## 検討した選択肢

### A: Redis共有ストレージ

```
MCPサーバー (Koyeb) ─┐
                     ├─► Redis ─► Supabase DB
MCPサーバー (Fly.io) ─┘
```

| 項目 | 評価 |
|------|------|
| キャッシュ整合性 | ◎ 完全 |
| 追加ホスト | 必要（Upstash, Fly.io Redis等） |
| レイテンシ | Koyeb↔Redis間で増加 |
| コスト | 無料枠小（Upstash 10,000 cmd/日） |
| 複雑度 | 中 |

**問題点**: Koyeb（リージョン制限）とFly.io（東京）でRedisホスト先が難しい

### B: Fly.io一本化

```
Cloudflare Workers (LB)
    ├─ Fly.io Instance 1
    └─ Fly.io Instance 2
         │
         └─► Fly.io Redis（同一リージョン）
```

| 項目 | 評価 |
|------|------|
| キャッシュ整合性 | ◎ 完全 |
| 追加ホスト | 不要（Fly.io内） |
| レイテンシ | ◎ 低（同一リージョン） |
| ベンダー分散 | × 失われる |

**問題点**: ベンダー分散の設計原則に反する

### C: Supabase Realtime（採用）

```
┌─────────────────────────────────────────────────────────┐
│                    Supabase                              │
│                                                          │
│  role_permissions テーブル                               │
│       │                                                  │
│       │ INSERT/UPDATE/DELETE                            │
│       ↓                                                  │
│  PostgreSQL NOTIFY  ←  Realtime拡張                     │
│       │                                                  │
│       ↓                                                  │
│  Supabase Realtime サーバー                             │
│       │                                                  │
│       │ WebSocket                                        │
└───────┼─────────────────────────────────────────────────┘
        │
        ├──────────────────┐
        ↓                  ↓
   ┌─────────┐        ┌─────────┐
   │  Koyeb  │        │ Fly.io  │
   │ キャッシュ│       │ キャッシュ│
   │ 無効化   │        │ 無効化   │
   └─────────┘        └─────────┘
```

| 項目 | 評価 |
|------|------|
| キャッシュ整合性 | ○ 数百ms遅延 |
| 追加ホスト | 不要（Supabase内） |
| レイテンシ | ◎ イベント駆動 |
| コスト | ◎ 無料枠に含む |
| ベンダー分散 | ○ 維持 |

---

## 採用: Supabase Realtime

### 仕組み

1. 権限テーブル（role_permissions等）の変更をPostgreSQLがNOTIFY
2. Supabase RealtimeがWebSocket経由で全購読者にプッシュ
3. 各MCPサーバーが該当ユーザーのキャッシュを無効化
4. 次回リクエスト時にDBから最新データを取得

### テーブルのRealtime有効化

```sql
-- Supabase Dashboard または SQL で設定
ALTER PUBLICATION supabase_realtime ADD TABLE role_permissions;
ALTER PUBLICATION supabase_realtime ADD TABLE user_tool_permissions;
ALTER PUBLICATION supabase_realtime ADD TABLE users;  -- suspended等の変更
```

### MCPサーバー実装（Go）

```go
package cache

import (
    "context"
    "log"
    "sync"
    "time"

    "github.com/supabase-community/realtime-go"
)

type PermissionCache struct {
    mu    sync.RWMutex
    cache map[string]*UserPermissions  // user_id -> permissions
    ttl   time.Duration

    client  *realtime.Client
    channel *realtime.Channel
}

type UserPermissions struct {
    UserID      string
    RoleID      string
    Permissions []string
    ExpiresAt   time.Time
}

func NewPermissionCache(supabaseURL, supabaseKey string, ttl time.Duration) *PermissionCache {
    return &PermissionCache{
        cache:  make(map[string]*UserPermissions),
        ttl:    ttl,
        client: realtime.NewClient(supabaseURL, supabaseKey),
    }
}

// Start はRealtime購読を開始
func (pc *PermissionCache) Start(ctx context.Context) error {
    pc.channel = pc.client.Channel("permission-changes")

    // role_permissions の変更を購読
    pc.channel.On("postgres_changes", realtime.PostgresChangesFilter{
        Event:  "*",  // INSERT, UPDATE, DELETE
        Schema: "public",
        Table:  "role_permissions",
    }, func(payload realtime.PostgresChangesPayload) {
        roleID, ok := payload.Record["role_id"].(string)
        if ok {
            pc.invalidateByRole(roleID)
            log.Printf("Cache invalidated for role: %s", roleID)
        }
    })

    // user_tool_permissions の変更を購読
    pc.channel.On("postgres_changes", realtime.PostgresChangesFilter{
        Event:  "*",
        Schema: "public",
        Table:  "user_tool_permissions",
    }, func(payload realtime.PostgresChangesPayload) {
        userID, ok := payload.Record["user_id"].(string)
        if ok {
            pc.invalidate(userID)
            log.Printf("Cache invalidated for user: %s", userID)
        }
    })

    // users テーブルの変更（suspended等）
    pc.channel.On("postgres_changes", realtime.PostgresChangesFilter{
        Event:  "UPDATE",
        Schema: "public",
        Table:  "users",
    }, func(payload realtime.PostgresChangesPayload) {
        userID, ok := payload.Record["id"].(string)
        if ok {
            pc.invalidate(userID)
            log.Printf("Cache invalidated for user (status change): %s", userID)
        }
    })

    if err := pc.channel.Subscribe(); err != nil {
        return err
    }

    // 再接続ループ
    go pc.reconnectLoop(ctx)

    return nil
}

// reconnectLoop はWebSocket切断時に再接続
func (pc *PermissionCache) reconnectLoop(ctx context.Context) {
    for {
        select {
        case <-ctx.Done():
            return
        case <-pc.client.OnDisconnect():
            log.Println("Realtime disconnected, reconnecting...")
            for {
                if err := pc.channel.Subscribe(); err == nil {
                    log.Println("Realtime reconnected")
                    break
                }
                time.Sleep(5 * time.Second)
            }
        }
    }
}

// Get はキャッシュから権限を取得（なければDBから取得）
func (pc *PermissionCache) Get(ctx context.Context, userID string) (*UserPermissions, error) {
    pc.mu.RLock()
    perm, ok := pc.cache[userID]
    pc.mu.RUnlock()

    if ok && time.Now().Before(perm.ExpiresAt) {
        return perm, nil
    }

    // DBから取得
    perm, err := pc.fetchFromDB(ctx, userID)
    if err != nil {
        return nil, err
    }

    // キャッシュに保存
    pc.mu.Lock()
    perm.ExpiresAt = time.Now().Add(pc.ttl)
    pc.cache[userID] = perm
    pc.mu.Unlock()

    return perm, nil
}

// invalidate は特定ユーザーのキャッシュを無効化
func (pc *PermissionCache) invalidate(userID string) {
    pc.mu.Lock()
    defer pc.mu.Unlock()
    delete(pc.cache, userID)
}

// invalidateByRole は特定ロールを持つ全ユーザーのキャッシュを無効化
func (pc *PermissionCache) invalidateByRole(roleID string) {
    pc.mu.Lock()
    defer pc.mu.Unlock()

    for userID, perm := range pc.cache {
        if perm.RoleID == roleID {
            delete(pc.cache, userID)
        }
    }
}

func (pc *PermissionCache) fetchFromDB(ctx context.Context, userID string) (*UserPermissions, error) {
    // 実際のDB取得ロジック
    // ...
    return nil, nil
}

// Stop はRealtime購読を停止
func (pc *PermissionCache) Stop() {
    if pc.channel != nil {
        pc.channel.Unsubscribe()
    }
}
```

### 初期化（main.go）

```go
func main() {
    ctx := context.Background()

    // キャッシュ初期化（TTL 5分）
    cache := cache.NewPermissionCache(
        os.Getenv("SUPABASE_URL"),
        os.Getenv("SUPABASE_ANON_KEY"),
        5*time.Minute,
    )

    if err := cache.Start(ctx); err != nil {
        log.Fatalf("Failed to start permission cache: %v", err)
    }
    defer cache.Stop()

    // サーバー起動...
}
```

---

## Supabase Realtime 無料枠

| リソース | 無料枠 | MCPist使用量 |
|----------|--------|-------------|
| 同時接続 | 200 | 2（Koyeb + Fly.io） |
| メッセージ数 | 200万/月 | 低頻度（権限変更時のみ） |

**十分余裕あり**

---

## 比較まとめ

| 項目 | Redis | Fly.io一本化 | Realtime |
|------|-------|-------------|----------|
| キャッシュ整合性 | ◎ | ◎ | ○（数百ms遅延） |
| 追加ホスト | 必要 | 不要 | 不要 |
| コスト | 有料化リスク | 無料枠内 | 無料枠内 |
| ベンダー分散 | 維持 | 失う | 維持 |
| 実装複雑度 | 中 | 低 | 低 |
| DB負荷軽減 | ◎ | ◎ | ○ |

**Realtime を採用**: ベンダー分散を維持しつつ、追加ホスト不要でキャッシュ同期を実現

---

## 注意点・制限事項

### 1. イベント遅延

権限変更からキャッシュ無効化まで数百ms程度の遅延あり。

**影響**: 権限変更直後の数リクエストで古い権限が使われる可能性
**許容**: Phase 1規模では問題なし

### 2. WebSocket接続維持

サーバー起動中は常時接続。切断時は再接続ロジックが必要。

### 3. Rate Limit共有は未解決

Realtime はキャッシュ無効化のみ。Rate Limitカウンターの共有は別問題。

**対応**:
- Phase 1: Unlimited プランで Rate Limit 無効
- Phase 2: 必要に応じて Redis 導入を検討

### 4. TTLは依然必要

Realtime イベントが何らかの理由で到達しない場合の保険として、TTL（5分）は維持。

---

## Phase 1 スコープ

### 実装する

- [x] ローカルキャッシュ（TTL 5分）
- [ ] Supabase Realtime 購読
- [ ] キャッシュ無効化ロジック
- [ ] WebSocket再接続ロジック

### 実装しない

- Redis等の共有ストレージ
- Rate Limitの共有（Unlimitedプランで回避）

---

## 関連ドキュメント

- [dsn-infrastructure.md](./dsn-infrastructure.md) - インフラ全体構成
- [dsn-permission-system.md](./dsn-permission-system.md) - 権限システム設計
- [dsn-subscription.md](./dsn-subscription.md) - サブスクリプション・使用量制御設計
