# go-mcp-dev 実装計画

## 概要

MCPサーバーをGo言語で実装し、Fly.ioにホストするシングルテナント用プロジェクト。認証は固定シークレットで処理し、軽量・高速を重視。

## アーキテクチャ

```
[Claude Code / LLM]
    ↓ Bearer <INTERNAL_SECRET>
[Go MCP Server on Fly.io] ← 固定シークレット検証 + SSE
    ↓ 各種APIトークン（環境変数）
[External APIs: Supabase Management, Notion, GitHub, Jira, Confluence]
```

## 設計方針

- **シングルテナント**: Cloudflare Worker不要、直接Go Serverにアクセス
- **SSE直接実装**: Go標準ライブラリで実装、外部依存なし
- **スキーマハードコード**: 安定性・速度重視
- **JSON-RPC 2.0完全準拠**: エラーコード含め硬い実装
- **モジュール粒度**: 極限まで細かく、全API操作を網羅

## 通信方式

- LLM ↔ Go Server: SSE (Server-Sent Events) over HTTP
- Bearer token認証: `Authorization: Bearer <INTERNAL_SECRET>`
- 認証失敗時: 401返却のみ（レート制限なし）

## エンドポイント

| メソッド | パス | 説明 |
|---------|------|------|
| GET | /health | ヘルスチェック |
| POST | /mcp | JSON-RPC over SSE |

---

## インフラ決定事項

### ホスティング変更履歴

**初期決定（2025-01-09）**: Fly.io（~$2/月）
**変更（2026-01-10）**: Koyeb Free Tier（$0/月）に移行

**移行理由**: Round 0完了後、開発ツールに$2/月の課金は不要と判断。Koyebは無料枠でも常時稼働可能（1時間アイドルでスリープするが、GitHub Actionsで45分ごとにpingして回避）。

### 現在のホスティング

| コンポーネント | サービス | 月額 | 理由 |
|---------------|----------|------|------|
| Go MCP Server | **Koyeb** | $0 | 無料枠、GitHub連携、自動デプロイ |
| ドメイン | Cloudflare | $0 | shibaleo-dev.mcpist.app |

**Cloudflare Workerは不採用**: シングルテナントのため中間レイヤー不要。

### IaC方針

**Terraformは不採用**。1サービスのみのため、deploy.sh + Koyeb CLIで管理。

### プロジェクト構成

```
go-mcp-dev/
├── .github/
│   └── workflows/
│       ├── ci.yml              # CI/CD設定（テスト + Koyeb再デプロイ）
│       └── ping.yml            # 45分ごとのヘルスチェック（スリープ回避）
├── deploy.sh                   # Koyeb CLIデプロイスクリプト
├── Dockerfile                  # マルチステージビルド
├── docker-compose.yml          # ローカル開発用
├── Makefile                    # ローカル開発コマンド
├── .env.development            # ローカル環境変数
├── .env.production             # 本番シークレット（.gitignore）
├── .env.example                # テンプレート（Git管理）
├── cmd/
│   ├── server/
│   │   └── main.go
│   └── version-check/
│       └── main.go             # APIバージョン検証ツール
├── internal/
│   ├── auth/                   # Bearer token検証
│   ├── mcp/                    # JSON-RPC 2.0 + SSE
│   ├── observability/          # Loki送信
│   └── modules/                # supabase, github, notion, jira, confluence
├── go.mod
└── README.md
```

---

## Docker構成

### マルチステージビルド

ビルドステージ（golang:alpine）と実行ステージ（golang:alpine）を分離。本番イメージはalpineベースでデバッグ可能性を維持。

### Dockerfile

```dockerfile
# ビルドステージ
FROM golang:1.22-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o server ./cmd/server

# 本番ステージ
FROM alpine:3.19
RUN apk add --no-cache ca-certificates
COPY --from=builder /app/server /server
EXPOSE 8080
CMD ["/server"]
```

### 環境別イメージ

| 環境 | ベースイメージ | 用途 |
|------|---------------|------|
| ローカル開発 | golang:1.22-alpine (builder) | `go run`で実行 |
| GitHub Actions | golang:1.22-alpine | テスト・ビルド |
| 本番 (Koyeb) | alpine:3.19 | コンパイル済みバイナリ + デバッグ可能 |

### alpine選定理由（distrolessから変更）

- **デバッグ可能**: Koyebダッシュボードでログ確認可能
- **CA証明書**: 標準パッケージで確実に入る
- **サイズ**: ~5MB + バイナリ（distrolessとの差は誤差）
- **リスク削減**: 「取らなくていいリスク」は排除する方針

---

## ローカル開発

### 方針

Dockerコンテナ内で開発。OS依存なし、環境統一。

### docker-compose.yml

```yaml
services:
  app:
    build:
      context: .
      target: builder    # alpineではなくbuilderステージを使用
    volumes:
      - .:/app
    ports:
      - "8080:8080"
    command: go run ./cmd/server
    env_file:
      - .env.development
```

### 開発コマンド

```bash
# サーバー起動
docker-compose up

# 別ターミナルで動作確認
curl http://localhost:8080/health

# テスト実行
docker-compose run app go test ./...
```

### ログレベル

- **debug**: 詳細なデバッグ情報
- **info**: 通常の動作ログ

### Makefile

```makefile
.PHONY: dev test

# ローカル開発
dev:
	docker-compose up

# テスト
test:
	docker-compose run app go test ./...
```

### deploy.sh

```bash
#!/bin/bash
# .env.productionから環境変数を読み取り、Koyebにデプロイ

ENV_FILE=".env.production"
APP_NAME="go-mcp-dev"

# 環境変数をフラグに変換
ENV_FLAGS=""
while IFS='=' read -r key value; do
  [[ -z "$key" || "$key" =~ ^# ]] && continue
  ENV_FLAGS="$ENV_FLAGS --env $key=$value"
done < "$ENV_FILE"

# デプロイ
koyeb app create "$APP_NAME" 2>/dev/null || true
koyeb service create "$APP_NAME" \
  --app "$APP_NAME" \
  --git github.com/shibaleo/go-mcp-dev \
  --git-branch main \
  --git-builder docker \
  --instance-type free \
  --ports 8080:http \
  --routes /:8080 \
  $ENV_FLAGS
```

### .env.example

```bash
INTERNAL_SECRET=
SUPABASE_ACCESS_TOKEN=
SUPABASE_PROJECT_REF=
NOTION_TOKEN=
GITHUB_TOKEN=
JIRA_EMAIL=
JIRA_API_TOKEN=
CONFLUENCE_EMAIL=
CONFLUENCE_API_TOKEN=
GRAFANA_LOKI_URL=
GRAFANA_LOKI_USER=
GRAFANA_LOKI_API_KEY=
```

---

## CI/CD

### ブランチ戦略

```
feat-xxx → PR → main (Squash Merge)
                  ↓
              CI テスト
                  ↓ 通過
              Koyeb再デプロイ
                  ↓
              ヘルスチェック
```

- **feat-xxx**: 機能追加・修正ブランチ
- **main**: 本番ブランチ（Squash Mergeで履歴クリーン）
- **Conventional Commits**: 強制なし、PRタイトルで気をつける程度

### GitHub Branch Protection設定

- ☑ Require status checks to pass before merging
- ☑ Require branches to be up to date before merging（PR状態=マージ後main保証）
- ☑ Require pull request reviews before merging（オプション）

### GitHub Actions

**ci.yml** - テストと自動デプロイ:

```yaml
name: CI/CD

on:
  pull_request:
    branches: [main]
  push:
    branches: [main]

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: '1.22'
      - name: Build
        run: go build ./...
      - name: Test
        run: go test -v ./...

  deploy:
    if: github.event_name == 'push' && github.ref == 'refs/heads/main'
    needs: test
    runs-on: ubuntu-latest
    steps:
      - name: Trigger Koyeb Redeploy
        run: |
          curl -X POST "https://app.koyeb.com/v1/services/${{ secrets.KOYEB_SERVICE_ID }}/redeploy" \
            -H "Authorization: Bearer ${{ secrets.KOYEB_API_TOKEN }}"
      - name: Wait for deployment
        run: sleep 60
      - name: Health Check
        run: curl -f https://shibaleo-dev.mcpist.app/health || exit 1
```

**ping.yml** - スリープ回避:

```yaml
name: Keep Alive Ping

on:
  schedule:
    - cron: '*/45 * * * *'  # 45分ごと
  workflow_dispatch:

jobs:
  ping:
    runs-on: ubuntu-latest
    steps:
      - name: Health Check Ping
        run: curl -f https://shibaleo-dev.mcpist.app/health || exit 1
```

### テスト戦略

| レイヤー | 対象 | 実行タイミング |
|---------|------|---------------|
| 単体テスト | JSON-RPC 2.0パーサー、認証ミドルウェア、エラーコード変換 | PR時 |
| バージョンチェック | 各外部APIのバージョン照合 | PR時 |
| ヘルスチェック | デプロイ後の疎通確認 | デプロイ後 |

### バージョン管理

各モジュールが対応APIバージョンを宣言。CIでAPIの現行バージョンと照合し、乖離があればビルド失敗。semverなし、日付のみ。

```go
// internal/modules/github/module.go
var ModuleInfo = module.Info{
    Name:       "github",
    APIVersion: "2022-11-28",  // GitHub API version header
    TestedAt:   "2025-01-09",  // 最終動作確認日
}
```

### ロールバック

Koyebダッシュボードから前バージョンにロールバック可能。または手動で前コミットをデプロイ。

---

## オブザーバビリティ

### 方針

Grafana Cloud Lokiに直接Push。マイグレーション不要、外部エージェント不要。

### 実装

```go
// ツール呼び出し時にLokiへ送信
func (m *Module) CallTool(name string, params any) (any, error) {
    start := time.Now()
    result, err := m.execute(name, params)

    // Lokiに送信
    loki.Push(map[string]string{
        "app":    "go-mcp-dev",
        "module": m.Name,
        "tool":   name,
    }, map[string]any{
        "duration_ms": time.Since(start).Milliseconds(),
        "status":      statusFromErr(err),
    })

    return result, err
}
```

### レート制限監視

各サービスのAPIレート制限をメトリクスで追跡。95%に達したら警告をレスポンスに含める。

```go
// レスポンスヘッダーからレート制限情報を取得
// X-RateLimit-Remaining / X-RateLimit-Limit
func checkRateLimit(resp *http.Response) *Warning {
    remaining := resp.Header.Get("X-RateLimit-Remaining")
    limit := resp.Header.Get("X-RateLimit-Limit")

    usage := 1 - (remaining / limit)
    if usage >= 0.95 {
        return &Warning{
            Code:    "RATE_LIMIT_WARNING",
            Message: fmt.Sprintf("%s API rate limit at %.0f%%", module, usage*100),
        }
    }
    return nil
}
```

---

## モジュールスコープ

### 含める

| モジュール | 説明 | 認証 | 粒度 |
|-----------|------|------|------|
| supabase | Management API（プロジェクト管理、SQL実行） | Access Token | 全API操作 |
| github | リポジトリ、Issue、PR、Actions、Gists等 | PAT | 全API操作 |
| notion | ページ・データベース・ブロック操作 | Integration Token | 全API操作 |
| jira | Issue/Project/Sprint/Board操作 | API Token | 全API操作 |
| confluence | Space/Page/Comment/Label操作 | API Token | 全API操作 |

### 含めない（OAuth必要）

- Google Calendar
- Microsoft Todo
- RAG（後回し）

---

## Supabase Management API モジュール

### ツール一覧（17ツール）✅ 実装済み

**Account**:
| ツール名 | 説明 | 状態 |
|---------|------|------|
| `supabase_list_organizations` | 組織一覧取得 | ✅ |
| `supabase_list_projects` | プロジェクト一覧 | ✅ |
| `supabase_get_project` | プロジェクト詳細 | ✅ |

**Database**:
| ツール名 | 説明 | 状態 |
|---------|------|------|
| `supabase_list_tables` | テーブル一覧取得 | ✅ |
| `supabase_run_query` | SQL実行 | ✅ |
| `supabase_list_migrations` | マイグレーション履歴取得 | ✅ |
| `supabase_apply_migration` | マイグレーション適用 | ✅ |

**Debugging**:
| ツール名 | 説明 | 状態 |
|---------|------|------|
| `supabase_get_logs` | ログ取得 | ✅ |
| `supabase_get_security_advisors` | セキュリティ推奨事項取得 | ✅ |
| `supabase_get_performance_advisors` | パフォーマンス推奨事項取得 | ✅ |

**Development**:
| ツール名 | 説明 | 状態 |
|---------|------|------|
| `supabase_get_project_url` | プロジェクトURL取得 | ✅ |
| `supabase_get_api_keys` | APIキー取得 | ✅ |
| `supabase_generate_typescript_types` | TypeScript型定義生成 | ✅ |

**Edge Functions**:
| ツール名 | 説明 | 状態 |
|---------|------|------|
| `supabase_list_edge_functions` | Edge Functions一覧取得 | ✅ |
| `supabase_get_edge_function` | Edge Function詳細取得 | ✅ |

**Storage**:
| ツール名 | 説明 | 状態 |
|---------|------|------|
| `supabase_list_storage_buckets` | ストレージバケット一覧取得 | ✅ |
| `supabase_get_storage_config` | ストレージ設定取得 | ✅ |

---

## 実装ステップ（Horizontal Snake方式）

Quality Reflux原理に基づき、後工程の知見を前工程に還流させる進め方を採用。

### Round 0: リスク検証（2日）

**目的**: 取るべきリスクを最速で検証し、基盤の確実性を確保する

| Day | タスク | 検証対象リスク |
|-----|--------|---------------|
| 1 | 最小HTTPサーバー + SSE + supabase_run_query 1ツール + Loki送信 | SSE実装、Loki疎通 |
| 2 | デプロイ + Claude Codeから実際に叩く | 常駐特性、外部API接続、MCP仕様解釈 |

**Round 0 成功条件（✅完了 2026-01-10）**:
- ✅ Claude CodeからSSE経由でsupabase_run_queryが動く
- ✅ その実行ログがLokiに届く
- ✅ Koyebで常駐する（GitHub Actionsで45分ごとping）
- ✅ 外部API（Supabase）に繋がる

### Round 1: 基盤深化（3日）

**目的**: Round 0の知見を還流させ、基盤を強化

| Day | タスク | 還流元 | 状態 |
|-----|--------|--------|------|
| 3 | SSE/JSON-RPCの改善、エラーハンドリング統一 | E（実運用）での挙動 | ✅完了 |
| 4 | Supabase残りツール追加 | B（モジュール）深化 | ✅完了 |
| 5 | 最小CI（GitHub Actions）+ 認証ミドルウェアテスト | D（CI/CD）骨格 | ✅完了 |

**Round 1 実績（2026-01-10）**:
- ✅ 共通HTTPクライアント作成（internal/httpclient/client.go）
- ✅ Supabaseモジュールを2ツール→17ツールに拡張
- ✅ GitHub Actions CI/CD設定済み
- ✅ 認証ミドルウェアテスト済み

### Round 2: モジュール拡張（4日）

**目的**: 検証済み基盤の上でモジュールを横展開。各追加後に即デプロイ・検証

| Day | タスク | 検証対象 | 状態 |
|-----|--------|---------|------|
| 6 | Notionモジュール + デプロイ + 実運用検証 | Notion API接続 | ✅完了 |
| 7 | GitHubモジュール + デプロイ + 実運用検証 | GitHub API接続 | ✅完了 |
| 8 | Jiraモジュール + デプロイ + 実運用検証 | Atlassian API接続 | ✅完了 |
| 9 | Confluenceモジュール + デプロイ + 実運用検証 | Atlassian API接続 | ✅完了 |

**Round 2 実績（2026-01-10）**:
- ✅ Notionモジュール作成（14ツール）
- ✅ GitHubモジュール作成（22ツール）
- ✅ Jiraモジュール作成（13ツール）
- ✅ Confluenceモジュール作成（12ツール）
- ☐ デプロイ + 実運用検証（未実施）

**Round 2 検証項目**:
- ☐ 256MBメモリで全モジュール載るか
- ☐ 複数モジュール間でエラーが波及しないか
- ☐ 各APIのレート制限の実態

### Round 3: 仕上げ（3日）

| Day | タスク | 状態 |
|-----|--------|------|
| 10 | テスト網羅（JSON-RPCパーサー、エラーコード変換） | ✅完了 |
| 11 | Branch Protection設定、バージョンチェック実装 | ✅完了 |
| 12 | ドキュメント整備、運用手順確認 | ✅完了 |

**Round 3 実績（2026-01-10）**:
- ✅ httpclientテスト追加（5テスト）
- ✅ MCPハンドラーテスト追加（ツールエラー、InvalidParams）
- ✅ 全17テスト通過
- ✅ GitHub Branch Protection設定（testステータスチェック必須）
- ✅ モジュールバージョン管理（internal/modules/module.go）
- ✅ README.md作成済み

**合計: 12日**（総日数は変わらず、リスク検証を前倒し）

---

## 品質計画（Quality Reflux）

### リスク分類

| リスク | 分類 | 検証タイミング | 対応 |
|--------|------|---------------|------|
| SSEの挙動（Claude Codeとの相性） | 検証必須 | Round 0 | 最小実装で即検証 |
| Fly.io常駐・タイムアウト | 検証必須 | Round 0 | 24時間監視 |
| Lokiへの直接Push | 検証必須 | Round 0 | 疎通確認 |
| IPv4/IPv6接続 | 検証必須 | Round 0 | IPv4割り当てで回避 |
| JSON-RPC + MCP仕様解釈 | 検証必須 | Round 0 | Claude Code実叩き |
| 256MB制限 | 後から顕在化 | Round 2 | モジュール追加時に監視 |
| 複数セッション同時接続 | 後から顕在化 | Round 2以降 | 単一動作後に検証 |
| CA証明書 | 排除済み | - | alpine採用で解消 |
| distrolessデバッグ不能 | 排除済み | - | alpine採用で解消 |
| シークレットローテーション | 運用課題 | 運用開始後 | 初期は固定 |
| ブルートフォース対策 | 運用課題 | 運用開始後 | 個人用なら後回し |

### 還流ポイント

各Roundで得た知見を前工程に還流させる具体的ポイント:

- **Round 0 → 設計**: SSE/MCP実挙動を踏まえたJSON-RPCパーサー改善
- **Round 1 → インフラ**: Fly.io特性を踏まえたfly.toml調整
- **Round 2 → モジュール設計**: 各API癖を踏まえたエラーハンドリング統一
- **Round 3 → 全体**: 運用知見を踏まえたドキュメント・テスト整備

---

## デプロイ手順

### 初回セットアップ

```bash
# 1. Fly.ioセットアップ
make setup

# 2. シークレット設定
cp .env.production.example .env.production
# .env.production を編集
make secrets

# 3. デプロイ
make deploy

# 4. GitHub Branch Protection設定（UI）
```

### 通常開発フロー

```bash
# 1. featureブランチ作成
git checkout -b feat-xxx

# 2. ローカル開発
make dev
# 別ターミナルで動作確認
curl http://localhost:8080/health

# 3. テスト
make test

# 4. コミット
git add . && git commit -m "feat: xxx"

# 5. PR作成 → CI自動実行
git push origin feat-xxx
# GitHub上でPR作成

# 6. CI通過 + mainが最新であることを確認
# 7. Squash Merge → 自動デプロイ
```

---

## コスト見積もり

| 項目 | 月額 |
|------|------|
| Koyeb (Go Server, Free tier) | $0 |
| Grafana Cloud (無料枠) | $0 |
| GitHub Actions (無料枠) | $0 |
| Cloudflare (ドメイン管理) | $0 |
| **合計** | **$0/月** |

---

## 既存プロジェクトとの関係

| 項目 | 既存 (dwhbi) | 新規 (go-mcp-dev) |
|------|-------------|----------------------|
| ホスティング | Vercel | Koyeb |
| 言語 | TypeScript (Next.js) | Go |
| 認証 | MCPトークン + OAuth + Service Role | 固定シークレット (INTERNAL_SECRET) |
| Supabase | Data API（ユーザー別Vault） | Management API（管理操作） |
| 用途 | マルチユーザーSaaS | 個人用 / シングルテナント |
| オブザーバビリティ | - | Grafana Cloud Loki |
| CI/CD | Vercel自動 | GitHub Actions + Fly.io |
| ローカル開発 | 直接実行 | Docker Compose |

**既存プロジェクトは維持**: MCPトークン管理UI、Vault管理、コンソール機能はそのまま稼働。新プロジェクトはシングルテナント用途の軽量版として並行運用。

---

*最終更新: 2026-01-10 - Round 0/1/2/3完了、全モジュール拡張（84ツール）完了、デプロイ+実運用検証のみ残*

---

## 実装済みモジュール一覧

| モジュール | ツール数 | 状態 |
|-----------|---------|------|
| Supabase | 18 | ✅ |
| Notion | 15 | ✅ |
| GitHub | 24 | ✅ |
| Jira | 14 | ✅ |
| Confluence | 13 | ✅ |
| **合計** | **84** | ✅ |
