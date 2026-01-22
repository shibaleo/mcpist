# ソーシャル認証設定手順

MCPist Consoleでソーシャルログインを有効にするための設定手順。

## 前提条件

- Supabaseプロジェクトが作成済み
- 各プロバイダーの開発者アカウントを所持

## 1. Supabase URL Configuration

Supabaseダッシュボード > Authentication > URL Configuration で以下を設定:

| 項目 | 値 |
|------|-----|
| Site URL | `https://your-production-domain.com` |
| Redirect URLs | `http://localhost:3000/auth/callback` (開発用) |
|  | `https://your-production-domain.com/auth/callback` (本番用) |

## 2. Google OAuth設定

### 2.1 Google Cloud Console設定

1. [Google Cloud Console](https://console.cloud.google.com/) にアクセス
2. プロジェクトを選択または新規作成
3. APIs & Services > Credentials に移動
4. Create Credentials > OAuth client ID を選択
5. Application type: Web application
6. 以下を設定:
   - Name: `MCPist Console`
   - Authorized JavaScript origins:
     - `http://localhost:3000` (開発用)
     - `https://your-production-domain.com` (本番用)
   - Authorized redirect URIs:
     - `https://<project-ref>.supabase.co/auth/v1/callback`
7. Client ID と Client Secret をコピー

### 2.2 Supabase設定

1. Supabaseダッシュボード > Authentication > Providers
2. Google を有効化
3. Client ID と Client Secret を入力
4. 保存

## 3. GitHub OAuth設定

### 3.1 GitHub設定

1. [GitHub Developer Settings](https://github.com/settings/developers) にアクセス
2. OAuth Apps > New OAuth App を選択
3. 以下を設定:
   - Application name: `MCPist Console`
   - Homepage URL: `https://your-production-domain.com`
   - Authorization callback URL: `https://<project-ref>.supabase.co/auth/v1/callback`
4. Register application をクリック
5. Client ID をコピー
6. Generate a new client secret をクリックし、Client Secret をコピー

### 3.2 Supabase設定

1. Supabaseダッシュボード > Authentication > Providers
2. GitHub を有効化
3. Client ID と Client Secret を入力
4. 保存

## 4. Microsoft (Azure) OAuth設定

### 4.1 Azure Portal設定

1. [Azure Portal](https://portal.azure.com/) にアクセス
2. Azure Active Directory > App registrations > New registration
3. 以下を設定:
   - Name: `MCPist Console`
   - Supported account types: Accounts in any organizational directory and personal Microsoft accounts
   - Redirect URI: Web / `https://<project-ref>.supabase.co/auth/v1/callback`
4. Register をクリック
5. Overview から Application (client) ID をコピー
6. Certificates & secrets > New client secret でシークレットを作成し、Value をコピー

### 4.2 Supabase設定

1. Supabaseダッシュボード > Authentication > Providers
2. Azure (Microsoft) を有効化
3. Client ID (Azure Application ID) と Client Secret を入力
4. Azure Tenant URL: `https://login.microsoftonline.com/common` (マルチテナント)
5. 保存

## 5. 環境変数設定

`apps/console/.env.local`:

```env
NEXT_PUBLIC_SUPABASE_URL="https://<project-ref>.supabase.co"
NEXT_PUBLIC_SUPABASE_PUBLISHABLE_KEY="<publishable-key>"
```

## 6. 動作確認

1. `npm run dev` で開発サーバー起動
2. `http://localhost:3000/login` にアクセス
3. 各ソーシャルログインボタンをクリック
4. 認証後、`/dashboard` にリダイレクトされることを確認

## トラブルシューティング

### "redirect_uri_mismatch" エラー
- Supabaseの Redirect URLs に `http://localhost:3000/auth/callback` が含まれているか確認
- プロバイダー側の Redirect URI が `https://<project-ref>.supabase.co/auth/v1/callback` になっているか確認

### ログイン後に `/login?error=auth_callback_error` にリダイレクトされる
- ブラウザの開発者ツールでネットワークエラーを確認
- Supabaseダッシュボードの Logs > Auth を確認

### プロバイダーが表示されない
- Supabaseダッシュボードでプロバイダーが有効になっているか確認
- Client ID / Secret が正しく設定されているか確認
