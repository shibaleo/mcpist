# DAY020 作業ログ

## 日付

2026-01-31

---

## 作業記録

| 時刻 | タスク ID | 内容 | 備考 |
|------|-----------|------|------|
|  | RPC-001 | Go Server: RPC名変更対応 | `get_module_token` → `get_user_credential`, `update_module_token` → `upsert_user_credential` |
|  | RPC-002 | Go Server: レガシーフィールド互換対応 | `_auth_type`, `_metadata`, `_expires_at` の後方互換 |
|  | RPC-003 | Console: フィールド名標準化 | `_auth_type` → `auth_type`, `_metadata` → `metadata`, `_expires_at` → `expires_at` |
|  | OAuth-001 | Google OAuth callback修正 | `_auth_type` → `auth_type` |
|  | OAuth-002 | Microsoft OAuth callback修正 | `_auth_type` → `auth_type` |
|  | Test-001 | Notion API テスト | search, get_page_content 成功 |
|  | Test-002 | Google Calendar API テスト | list_calendars, list_events 成功 |
|  | Canvas-001 | RPC Canvas検証・更新 | マイグレーションとCanvasの差異を修正 |
|  | Canvas-002 | OAuth Consents RPC追加 | `list_my_oauth_consents`, `revoke_my_oauth_consent`, `list_all_oauth_consents` |
|  | Canvas-003 | auth テーブル追加 | `auth.oauth_consents`, `auth.oauth_clients` ノードとエッジ |
|  | Research-001 | MCP `notifications/tools/list_changed` 調査 | MCPist では不要と結論 |
|  | D20-001 | database.types.ts 再生成 | Supabase CLI で型生成 |
|  | D20-002 | Console ビルド確認 | RPC名変更後の型チェック通過 |
|  | E2E-001 | Claude Web E2E テスト | Notion search + get_page_content 成功、クレジット消費確認 |

---

## 完了タスク

- [x] Go Server: RPC名変更対応
  - `get_module_token` → `get_user_credential`
  - `update_module_token` → `upsert_user_credential`
  - レガシーフィールド（`_auth_type`, `_metadata`, `_expires_at`）の後方互換サポート
  - `GetAuthType()`, `GetMetadata()` ヘルパーメソッド追加

- [x] Console: 認証情報フィールド名標準化
  - `token-vault.ts`: `_auth_type` → `auth_type`, `_metadata` → `metadata`
  - `expires_at`: ISO文字列 → Unix timestamp (int)
  - Google/Microsoft OAuth callback: `_auth_type` → `auth_type`

- [x] MCPサーバー動作検証
  - Notion: `search`, `get_page_content` 成功
  - Google Calendar: `list_calendars`, `list_events` 成功

- [x] RPC Canvas 検証・更新
  - マイグレーションと Canvas の差異を修正
  - 不足していた RPC 追加: `get_my_prompt`, `list_my_oauth_consents`, `revoke_my_oauth_consent`, `list_all_oauth_consents`
  - `admin_emails` テーブルノード追加
  - OAuth Consents グループ追加（`auth.oauth_consents`, `auth.oauth_clients` 参照）

- [x] MCP プロトコル調査
  - `notifications/tools/list_changed` の仕様確認
  - MCPist のメタツール設計では不要と結論（ツールリストは固定）

---

## 変更ファイル概要

| カテゴリ | ファイル | 主な変更 |
|----------|----------|----------|
| Server | `internal/store/token.go` | RPC名変更、レガシーフィールド対応 |
| Console | `lib/token-vault.ts` | フィールド名標準化 |
| Console | `app/api/oauth/google/callback/route.ts` | `auth_type` 修正 |
| Console | `app/api/oauth/microsoft/callback/route.ts` | `auth_type` 修正 |
| Docs | `grh-rpc-design.canvas` | RPC/テーブル追加、エッジ追加 |

---

## 変更詳細

### Go Server: token.go

```go
// レガシーフィールド対応
type Credentials struct {
    AuthType  string `json:"auth_type,omitempty"`  // Standard
    AuthType2 string `json:"_auth_type,omitempty"` // Legacy
    // ...
    Metadata  map[string]string `json:"metadata,omitempty"`
    Metadata2 map[string]string `json:"_metadata,omitempty"` // Legacy
}

// ヘルパーメソッド
func (c *Credentials) GetAuthType() string {
    if c.AuthType != "" { return c.AuthType }
    return c.AuthType2
}
```

### Console: token-vault.ts

```typescript
// Before
credentials._auth_type = authType
credentials._expires_at = params.expiresAt.toISOString()

// After
credentials.auth_type = authType
credentials.expires_at = Math.floor(params.expiresAt.getTime() / 1000)
```

---

## Canvas 更新詳細

### 追加したノード

| ID | 種類 | 内容 |
|----|------|------|
| `rpc-get-my-prompt` | RPC | プロンプト単体取得 |
| `rpc-list-my-oauth-consents` | RPC | OAuthコンセント一覧 |
| `rpc-revoke-my-oauth-consent` | RPC | OAuthコンセント取消 |
| `rpc-list-all-oauth-consents` | RPC | 全OAuthコンセント (Admin) |
| `tbl-admin-emails` | Table | 管理者メールアドレス |
| `tbl-auth-oauth-consents` | Table | auth.oauth_consents |
| `tbl-auth-oauth-clients` | Table | auth.oauth_clients |
| `grp-oauth-consents` | Group | OAuth Consents グループ |

### 追加したエッジ

| From | To | 説明 |
|------|----|------|
| `rpc-list-my-oauth-consents` | `tbl-auth-oauth-consents` | コンセント参照 |
| `rpc-list-my-oauth-consents` | `tbl-auth-oauth-clients` | クライアント参照 |
| `rpc-revoke-my-oauth-consent` | `tbl-auth-oauth-consents` | コンセント更新 |
| `rpc-list-all-oauth-consents` | `tbl-auth-oauth-consents` | コンセント参照 |
| `rpc-list-all-oauth-consents` | `tbl-auth-oauth-clients` | クライアント参照 |
| `rpc-list-all-oauth-consents` | `tbl-auth-users` | ユーザー参照 |
| `rpc-handle-new-user` | `tbl-admin-emails` | 管理者判定 |

---

## 解決した問題

| 問題 | 原因 | 解決策 |
|------|------|--------|
| Notion 401 Unauthorized | `_auth_type` vs `auth_type` フィールド名不一致 | Go Server でレガシーフィールドサポート追加 |
| 不整合なフィールド命名 | 実装ミス（デザインパターンではない） | Console 側を標準名に修正、Server は後方互換維持 |

---

## MCP調査結果

### `notifications/tools/list_changed`

- **目的**: サーバーからクライアントにツールリスト変更を通知
- **用途**: 認証状態変更、権限変更、サービス可用性変更でツールを動的に表示/非表示

### MCPist で不要な理由

1. メタツールは固定（`get_module_schema`, `run`, `batch`）
2. モジュール/ツールの有効/無効は `get_module_schema` のレスポンス内で動的に反映
3. MCP レベルでのツールリスト変更は発生しない

---

## 未完了タスク

- [x] database.types.ts 再生成
- [x] E2E テスト（Claude Web）
- [ ] 仕様書整備（BL-011〜014）

---

## E2E テスト結果

| テスト | 結果 | 備考 |
|--------|------|------|
| Claude Web → Notion search | ✅ | クレジット消費 1 |
| Claude Web → Notion get_page_content | ✅ | クレジット消費 1 |
| get_module_schema | ✅ | クレジット消費なし（設計通り） |
| クレジット残高変化 | ✅ | 85 → 83（差分 2 = run × 2） |

---

## メモ

- Go Server のレガシーフィールド対応は、既存データとの互換性のため当面維持
- 新規保存は標準フィールド名を使用（Console 修正済み）
- Canvas は Obsidian で手動レイアウト調整済み
