# MCPist ルーティング設計書

## 概要

User Console（Next.js）のページ・APIルート設計。

---

## 設計方針

### セキュリティ

- **管理者が他ユーザーのページにアクセスする設計は採用しない**
- 全ページは自分のデータのみ表示（`auth.uid()` で制御）
- 管理者操作は `/admin` 内で専用UIを用意

### 管理者権限

- 管理者もソーシャルログインでアカウント作成（パスワード管理不要）
- admin権限はSQLでのみ付与（`raw_app_meta_data->>'role' = 'admin'`）
- UIからの誤付与を防止
- 管理者も通常ユーザーと同じページ（`/dashboard`, `/connections` 等）を使用
- `/admin` へのアクセス時のみ `(admin)/layout.tsx` でadminチェック
- サイドバーで `/admin` リンクの表示/非表示を切り替え（isAdmin判定は1箇所のみ）

### 管理者機能

- `/admin`: OAuth認可状況、システム統計
- ログ監視: Loki（外部システム）
- ユーザー監視: 実装しない

---

## ルート命名規則

| パス | 用途 | 認証 |
|------|------|------|
| `/` | ランディング | public |
| `/login` | ログイン | public |
| `/dashboard` | ダッシュボード | authenticated |
| `/connections` | 外部サービス接続（Notion, GitHub等） | authenticated |
| `/mcp` | MCP接続（APIキー・OAuth認可） | authenticated |
| `/tools` | ツール設定 | authenticated |
| `/billing` | 課金管理 | authenticated |
| `/settings` | 外観設定 | authenticated |
| `/admin` | 管理者ダッシュボード | admin |
| `/oauth/*` | OAuth関連（同意画面・コールバック） | mixed |
| `/api/*` | API Routes | mixed |
| `/.well-known/*` | OAuth メタデータ | public |

---

## ページルート

### Public（認証不要）

| パス | 用途 | 状態 |
|------|------|------|
| `/` | ランディングページ | ✅ |
| `/login` | ログイン | ✅ |

### Authenticated（認証必須）

| パス | 用途 | 状態 | 備考 |
|------|------|------|------|
| `/dashboard` | ダッシュボード | ✅ | クレジット残高・利用状況 |
| `/connections` | 外部サービス接続 | 🔄 | Notion, GitHub等のトークン管理（旧: /my/connections） |
| `/mcp` | MCP接続管理 | 🔄 | APIキー・OAuth認可管理（旧: /my/mcp-connection） |
| `/tools` | ツール設定 | 🔄 | モジュール・ツールのON/OFF（旧: /my/preferences） |
| `/billing` | 課金管理 | 🔄 | クレジット購入・履歴 |
| `/settings` | 外観設定 | 🔄 | テーマ・背景色・アクセントカラー |

### Admin（管理者のみ）

| パス | 用途 | 状態 | 備考 |
|------|------|------|------|
| `/admin` | 管理者ダッシュボード | ✅ | OAuth認可状況・システム統計 |

### OAuth（MCPクライアント向け）

| パス | 用途 | 状態 | 備考 |
|------|------|------|------|
| `/oauth/consent` | OAuth同意画面 | ✅ | MCPクライアントからのリダイレクト先 |
| `/oauth/callback` | OAuthコールバック | ✅ | 認可コード発行後のリダイレクト |

### 削除予定 / 開発専用

| パス | 理由 | 対応 |
|------|------|------|
| `/dev/mcp-client` | 開発用テストクライアント | 開発時のみ使用、本番では非表示 |
| `/dev/mcp-client/callback` | 開発用コールバック | 開発時のみ使用、本番では非表示 |

---

## APIルート

### Public

| パス | メソッド | 用途 | 状態 |
|------|---------|------|------|
| `/.well-known/oauth-authorization-server` | GET | OAuth AS メタデータ | ✅ |
| `/.well-known/oauth-protected-resource` | GET | OAuth リソースメタデータ | ✅ |

### Authenticated

| パス | メソッド | 用途 | 状態 | 備考 |
|------|---------|------|------|------|
| `/api/token-vault` | POST | サービストークン保存 | 🔄 | RPC化検討 |
| `/api/validate-token` | POST | トークン検証 | 🔄 | 用途確認必要 |

### Auth

| パス | メソッド | 用途 | 状態 |
|------|---------|------|------|
| `/auth/callback` | GET | Supabase Auth コールバック | ✅ |

---

## ルート構成

```
/                           # ランディング
/login                      # ログイン
/dashboard                  # ダッシュボード
/connections                # 外部サービス接続（Notion, GitHub等）
/mcp                        # MCP接続（APIキー・OAuth認可）
/tools                      # ツール設定
/billing                    # 課金管理
/settings                   # 外観設定
/admin                      # 管理者ダッシュボード（adminのみ）
/oauth/
  consent                   # OAuth同意画面
  callback                  # OAuthコールバック
/dev/                       # 開発用（本番非表示）
  mcp-client/               # テストMCPクライアント
```

---

## 認証フロー

### ログイン

```
/login → Supabase Auth → /auth/callback → /dashboard
```

### MCPクライアント接続（OAuth）

```
MCPクライアント → /.well-known/oauth-authorization-server
              → /oauth/consent（ユーザー同意）
              → /oauth/callback（認可コード発行）
              → MCPクライアント（トークン取得）
```

### MCPクライアント接続（APIキー）

```
ユーザー → /mcp でAPIキー生成
       → MCPクライアントにAPIキー設定
       → MCP Server へリクエスト
```

---

## 実装メモ

### Next.js Route Groups

```
app/
  (console)/              # 認証必須グループ（一般ユーザー）
    dashboard/
    connections/
    mcp/
    tools/
    billing/
    settings/
    dev/                  # 開発用（環境変数で制御）
  (admin)/                # 管理者専用グループ
    admin/
      page.tsx            # 管理者ダッシュボード
    layout.tsx            # ← adminチェックはここだけ
  login/                  # ログイン（認証不要）
  oauth/                  # OAuth関連
    consent/
    callback/
  api/                    # API Routes
  auth/                   # Supabase Auth callback
  .well-known/            # OAuth メタデータ
```

### 認証チェック

- `(console)` グループ: layout.tsxで認証チェック（isAdmin判定なし）
- `(admin)` グループ: layout.tsxでadminチェック（1箇所のみ）
- `/oauth/*`: セッション有無で挙動変更
- `/dev/*`: NODE_ENV=development で制御

---

## 変更履歴

| 日付 | 変更内容 |
|------|---------|
| 2026-01-25 | 初版作成 |
| 2026-01-25 | ルート構造簡素化（/my/* → トップレベル）、管理者設計方針追加 |
| 2026-01-25 | 管理者機能の範囲を明確化（OAuth認可状況・統計のみ、ログはLoki） |
