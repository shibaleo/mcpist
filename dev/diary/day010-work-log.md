# DAY010 作業ログ

## 基本情報

| 項目 | 値 |
|------|-----|
| 日付 | 2026-01-19 〜 2026-01-20 |
| スプリント | SPRINT-002 |
| 状態 | 完了 |

---

## 本日の成果

### 1. Devcontainer環境構築

MCPistプロジェクト用のDocker-in-Docker (DinD) 対応Devcontainer環境を構築。

**構成ファイル:**
- `.devcontainer/Dockerfile` - カスタムDevcontainerイメージ
- `.devcontainer/docker-compose.yml` - DinD設定（`privileged: true`）
- `.devcontainer/devcontainer.json` - VS Code Devcontainer設定
- `.devcontainer/post-create.sh` - 初期セットアップスクリプト

**インストールツール:**
- Go 1.24 + Air（ホットリロード）
- Node.js 22 + pnpm
- Docker CLI + Docker Compose Plugin
- Supabase CLI v2.72.7
- Wrangler（Cloudflare Workers CLI）
- OpenTofu（Terraform代替）

**対応サービス:**
- Next.js Console: `http://localhost:3000`
- Go Server: `http://localhost:8089`
- Cloudflare Worker: `http://localhost:8787`
- Supabase: `http://localhost:54321`
- LBテスト用API（Render/Koyeb模倣）: `http://localhost:8081`, `http://localhost:8082`

### 2. 改行コード問題の解決

Windows/Linux間の改行コード（CRLF/LF）問題を解決。

**対応:**
- `.gitattributes` 追加（`* text=auto eol=lf`）
- `git rm --cached -r .` + `git reset --hard` で正規化
- 全ファイルをLFに統一

### 3. Docker-in-Docker (DinD) の理解

**学んだこと:**
- DinDはDockerデーモンをイメージに含めるのではなく、コンテナ内で別のDockerデーモンを起動する仕組み
- `privileged: true` が必要
- ネストは技術的には可能だが、一般的には1層まで

### 4. インフラアーキテクチャの議論

**Render (Primary) / Koyeb (Failover) 構成:**
- p95レイテンシベースのロードバランシング
- 日本市場規模ではKubernetesは不要
- K8sは数百人規模のエンジニアチームで使うもの

**収益モデルの検討:**
- サブスクリプション → プリペイドクレジット方式に変更
- リクエスト数に応じた課金モデル

### 5. コンテナレジストリの検討

**オプション:**
- DockerHub: 無料で1つのPrivateリポジトリ
- GitHub Container Registry (ghcr.io): 無制限、GitHubと統合
- 結論: 個人開発ではローカルビルドで十分、レジストリ公開は後回し

**DockerHubプッシュ:**
- `shibaleo/mcpist-devcontainer:latest` をビルド
- プッシュ中にエラー発生（一時的な問題）

---

## 技術メモ

### DinD vs DooD

| 方式 | 説明 | 用途 |
|------|------|------|
| DinD | コンテナ内でDockerデーモンを起動 | 完全分離が必要な場合 |
| DooD | ホストのDockerソケットをマウント | 軽量だが分離が弱い |

MCPistでは**DinD**を採用（`privileged: true` + `/var/lib/docker` ボリューム）

### 改行コードの標準

| OS | 改行コード |
|----|----------|
| Linux/macOS | LF (`\n`) |
| Windows | CRLF (`\r\n`) |

Git + Devcontainerでは**LF統一**が推奨。`.gitattributes` で強制。

---

## 課題・残タスク

- [ ] DockerHubへのプッシュ完了（エラーで中断）
- [ ] GitHub Container Registryへの移行検討
- [ ] Devcontainerイメージの最適化（将来）

---

## 関連ファイル

| ファイル | 説明 |
|----------|------|
| `mcpist/.devcontainer/Dockerfile` | Devcontainerイメージ定義 |
| `mcpist/.devcontainer/docker-compose.yml` | DinD設定 |
| `mcpist/.devcontainer/devcontainer.json` | VS Code設定 |
| `mcpist/.devcontainer/post-create.sh` | 初期セットアップ |
| `mcpist/.gitattributes` | 改行コード設定 |
| `mcpist/compose/api.yml` | LBテスト用Docker Compose |

---

---

## 2026-01-20 の成果

### 1. Windows互換性の改善

**問題:** `package.json` の `sh -c` がWindowsで動作しない

**解決:**
- Go の `-C` フラグを使用: `go run -C apps/server ./cmd/server`
- クロスプラットフォーム対応

### 2. 環境変数の統合

**問題:** `.env` ファイルが各アプリに散在

**解決:**
- ルートに `.env.local` を統一配置
- `dotenv-cli` で各アプリに読み込み
- `pnpm start` で全サービス一括起動

**package.json スクリプト:**
```json
{
  "start": "supabase start && concurrently \"pnpm start:console\" \"pnpm start:server\" \"pnpm start:worker\"",
  "start:console": "dotenv -v PORT=3000 -e .env.local -- pnpm --filter @mcpist/console dev",
  "start:server": "dotenv -e .env.local -- go run -C apps/server ./cmd/server"
}
```

### 3. ポート競合の解決

**問題:** Next.js と Go Server が両方 `PORT=8089` を読み込む

**解決:**
- `dotenv-cli -v PORT=3000` で Next.js 用に上書き
- Go Server は `PORT=8089` をそのまま使用

### 4. OAuth Server 切り替え実装

**目的:** 開発環境と本番環境で同一コードを使用

**実装:**
- `apps/console/src/lib/env.ts` 作成
- `ENVIRONMENT` 変数で判定（`development` | `production`）
- 開発: カスタム OAuth 実装 (`/api/auth/*`)
- 本番: Supabase OAuth Server (`/auth/v1/*`)

**切り替え対象:**
| エンドポイント | 開発 | 本番 |
|---------------|------|------|
| authorize | `/api/auth/authorize` | `${supabaseUrl}/auth/v1/authorize` |
| token | `/api/auth/token` | `${supabaseUrl}/auth/v1/token` |

**変更ファイル:**
- `apps/console/src/lib/env.ts` - 環境判定ユーティリティ（新規）
- `apps/console/src/app/api/auth/authorize/route.ts` - リダイレクト対応
- `apps/console/src/app/api/auth/token/route.ts` - プロキシ対応
- `apps/console/src/app/oauth/consent/` - `/auth/consent` から移動

### 5. Supabase OAuth Server の理解

**発見:**
- Supabase OAuth Server はクラウド版のみ（BETA）
- ローカル OSS Supabase では未サポート
- Supabase Vault は OSS でも利用可能

**本番設定（Supabase Dashboard）:**
- Authorization URL: `/oauth/consent`
- Token URL: `/api/auth/token`

---

## 技術メモ

### dotenv-cli の使い方

| オプション | 説明 |
|-----------|------|
| `-e .env.local` | 読み込む env ファイルを指定 |
| `-v KEY=value` | 環境変数を上書き |
| `--` | 以降をコマンドとして実行 |

### 環境別 OAuth フロー

```
開発環境 (ENVIRONMENT=development)
┌─────────────┐     ┌────────────────────────┐
│ MCP Client  │────▶│ /api/auth/authorize    │
└─────────────┘     │ (カスタム実装)           │
                    └───────────┬────────────┘
                                │
                    ┌───────────▼────────────┐
                    │ /oauth/consent         │
                    └───────────┬────────────┘
                                │
                    ┌───────────▼────────────┐
                    │ /api/auth/token        │
                    │ (カスタム実装)           │
                    └────────────────────────┘

本番環境 (ENVIRONMENT=production)
┌─────────────┐     ┌────────────────────────┐
│ MCP Client  │────▶│ Supabase OAuth Server  │
└─────────────┘     │ /auth/v1/authorize     │
                    └───────────┬────────────┘
                                │
                    ┌───────────▼────────────┐
                    │ /oauth/consent         │
                    │ (Supabase設定で指定)     │
                    └───────────┬────────────┘
                                │
                    ┌───────────▼────────────┐
                    │ /auth/v1/token         │
                    │ (Supabaseが処理)        │
                    └────────────────────────┘
```

---

## 課題・残タスク

- [ ] JWT認証テスト（OAuth トークン）
- [ ] 本番環境デプロイ
- [ ] Supabase OAuth Server の設定（Dashboard）
- [x] Windows互換スクリプト
- [x] 環境変数統合
- [x] OAuth Server 切り替え実装

---

## 次回予定

- OAuth 認証フローの E2E テスト
- 本番環境へのデプロイ準備
- JWT トークン認証の検証
