# MCPist デプロイ戦略

## 概要

MCPサーバーのデプロイ戦略。ホットスタンバイ、CI/CD、シークレット管理を含む。

関連ドキュメント:
- [dsn-infrastructure.md](./dsn-infrastructure.md) - インフラ全体構成
- [dsn-load-management.md](./dsn-load-management.md) - 負荷対策設計

---

## ホットスタンバイ + 同時デプロイ

### 概念

```
Primary (Koyeb)  = 通常時のトラフィック先
Standby (Fly.io) = 常時稼働、障害時に自動切替

両方とも本番環境（同一DBに接続）
両方とも常にアクティブ（ホットスタンバイ）
```

### デプロイ戦略

MCPサーバーはステートレスのため、新旧バージョンが一時的に混在しても問題ない。

```
GitHub Push
    │
    ├─→ Koyeb: 自動デプロイ
    └─→ Fly.io: 自動デプロイ

デプロイ中（数分間）:
    Koyeb: 新版
    Fly.io: 旧版（または新版）

    → ユーザーは気づかない（リクエスト単位で完結）
    → ホットスタンバイは常に有効
```

### なぜ混在しても問題ないか

1. **ステートレス**: 各リクエストは独立、セッション状態なし
2. **同一DB**: どちらのインスタンスも同じDBを参照
3. **後方互換**: 破壊的変更時は別途対応（段階リリース等）

### ロールバック

```
問題発生時:
    1. git revert でコミット
    2. 再度両環境にデプロイ

    または

    1. Cloudflare で片方を一時的に除外
    2. 問題のある環境を修正
```

### ステージングとの違い

| 環境 | 目的 | DB | Phase 1 |
|------|------|-----|---------|
| Primary/Standby | 本番（ホットスタンバイ） | 本番DB | ○ |
| Staging | 開発中検証 | テストDB | 後回し |

**Phase 1ではStagingは不要**。必要になったら別途構築。

---

## CI/CD

### GitHub Actions

```yaml
name: Deploy

on:
  push:
    branches: [main]

jobs:
  deploy-koyeb:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - name: Deploy to Koyeb
        # Koyebは自動デプロイ（GitHub連携）

  deploy-flyio:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: superfly/flyctl-actions/setup-flyctl@master
      - run: flyctl deploy --remote-only
        env:
          FLY_API_TOKEN: ${{ secrets.FLY_API_TOKEN }}
```

### 同一Dockerイメージ

```dockerfile
FROM golang:1.22-alpine AS builder
WORKDIR /app
COPY . .
RUN go build -o server ./cmd/server

FROM alpine:latest
COPY --from=builder /app/server /server
CMD ["/server"]
```

両環境に同一イメージをデプロイ。環境変数のみ異なる。

---

## シークレット管理

### 方針

**GitHub Secretsを唯一の情報源（Single Source of Truth）として使用**。

```
ローカルでシークレットを管理しない

理由:
- 鍵紛失リスク（PCクラッシュ等）
- DevOpsの責任が重すぎる
- 放置運用に適さない

代わりに:
- GitHub SecretsでCIが各サービスに配布
- ローカル開発はシークレット検証をスキップ
```

### シークレット一覧

| シークレット | 用途 | 配布先 |
|--------------|------|--------|
| `GATEWAY_SECRET` | Worker→オリジン認証 | Worker, Koyeb, Fly.io |
| `FLY_API_TOKEN` | Fly.ioデプロイ | CI/CD |
| `CLOUDFLARE_API_TOKEN` | Terraform適用 | CI/CD |

### GitHub Secrets設定

```
GitHub Repository Settings → Secrets and variables → Actions

GATEWAY_SECRET: (ランダム文字列、初回のみ生成)
FLY_API_TOKEN: (Fly.io dashboardから取得)
CLOUDFLARE_API_TOKEN: (Cloudflare dashboardから取得)
```

### ローカル開発

**シークレット検証はスキップ**。

```go
// GatewayMiddleware
func GatewayMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // 開発環境ではスキップ
        if os.Getenv("ENVIRONMENT") == "development" {
            next.ServeHTTP(w, r)
            return
        }

        // 本番環境では検証
        secret := r.Header.Get("X-Gateway-Secret")
        if secret != os.Getenv("GATEWAY_SECRET") {
            http.Error(w, "Forbidden", http.StatusForbidden)
            return
        }
        next.ServeHTTP(w, r)
    })
}
```

ローカル開発時:
```bash
ENVIRONMENT=development go run ./cmd/server
```

---

## デプロイ後検証

### 検証フロー

**シークレット検証が正しく動作しているかをデプロイ後に自動検証**。

```yaml
# .github/workflows/deploy.yml

name: Deploy

on:
  push:
    branches: [main]

jobs:
  deploy-koyeb:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - name: Deploy to Koyeb
        # Koyebは自動デプロイ（GitHub連携）

  deploy-flyio:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: superfly/flyctl-actions/setup-flyctl@master
      - run: flyctl deploy --remote-only
        env:
          FLY_API_TOKEN: ${{ secrets.FLY_API_TOKEN }}

  deploy-worker:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - name: Deploy Cloudflare Worker
        uses: cloudflare/wrangler-action@v3
        with:
          apiToken: ${{ secrets.CLOUDFLARE_API_TOKEN }}
          secrets: |
            GATEWAY_SECRET
        env:
          GATEWAY_SECRET: ${{ secrets.GATEWAY_SECRET }}

  # デプロイ後検証
  verify:
    needs: [deploy-koyeb, deploy-flyio, deploy-worker]
    runs-on: ubuntu-latest
    steps:
      - name: Wait for deployments to stabilize
        run: sleep 60

      # 各オリジンの詳細ヘルスチェック（GATEWAY_SECRET付き）
      - name: Health check Koyeb origin
        run: |
          RESPONSE=$(curl -s -w "\n%{http_code}" \
            -H "X-Gateway-Secret: ${{ secrets.GATEWAY_SECRET }}" \
            https://mcpist.koyeb.app/health)
          BODY=$(echo "$RESPONSE" | head -n -1)
          STATUS=$(echo "$RESPONSE" | tail -n 1)
          echo "Koyeb: $STATUS"
          echo "$BODY" | jq .
          if [ "$STATUS" != "200" ]; then
            echo "ERROR: Koyeb origin unhealthy"
            exit 1
          fi

      - name: Health check Fly.io origin
        run: |
          RESPONSE=$(curl -s -w "\n%{http_code}" \
            -H "X-Gateway-Secret: ${{ secrets.GATEWAY_SECRET }}" \
            https://mcpist.fly.dev/health)
          BODY=$(echo "$RESPONSE" | head -n -1)
          STATUS=$(echo "$RESPONSE" | tail -n 1)
          echo "Fly.io: $STATUS"
          echo "$BODY" | jq .
          if [ "$STATUS" != "200" ]; then
            echo "ERROR: Fly.io origin unhealthy"
            exit 1
          fi

      # Worker経由の疎通確認
      - name: Verify Worker route
        run: |
          STATUS=$(curl -s -o /dev/null -w "%{http_code}" \
            https://mcp.mcpist.com/health)
          echo "Worker route: $STATUS"
          if [ "$STATUS" != "200" ]; then
            echo "ERROR: Worker route failed"
            exit 1
          fi

      # セキュリティ検証（直接アクセスがブロックされるか）
      - name: Verify direct access blocked (no secret)
        run: |
          # シークレットなしでの直接アクセスは403
          STATUS=$(curl -s -o /dev/null -w "%{http_code}" \
            https://mcpist.koyeb.app/api/test)
          if [ "$STATUS" != "403" ]; then
            echo "ERROR: Direct access without secret should be blocked"
            exit 1
          fi
          echo "Security check passed: direct access blocked"

      - name: All verifications passed
        run: echo "All health and security checks passed!"
```

### 検証内容

| テスト | 方法 | 期待結果 | 意味 |
|--------|------|----------|------|
| Koyeb + SECRET | 直接 + ヘッダー | 200 + JSON | オリジン正常動作 |
| Fly.io + SECRET | 直接 + ヘッダー | 200 + JSON | オリジン正常動作 |
| Worker経由 | mcp.mcpist.com | 200 | LBルート正常 |
| 直接（SECRETなし） | 直接アクセス | 403 | セキュリティ有効 |

### 失敗時

```
verifyジョブが失敗
    ↓
GitHub Actions通知（メール/Slack等）
    ↓
手動で確認・修正
```

**検証失敗 = オリジン障害 or セキュリティ設定に問題あり**。即座に対応が必要。

---

## Terraform（IaC）

### 目的

- CI/CDの柔軟性確保
- Cloudflare設定のコード管理
- 障害時のフォールバック選択肢

### Cloudflare Terraform

```hcl
# cloudflare.tf

terraform {
  required_providers {
    cloudflare = {
      source  = "cloudflare/cloudflare"
      version = "~> 4.0"
    }
  }
}

variable "cloudflare_api_token" {
  sensitive = true
}

variable "zone_id" {}
variable "domain" {}

# Load Balancer Pool
resource "cloudflare_load_balancer_pool" "mcp_servers" {
  name = "mcp-servers"

  origins {
    name    = "koyeb-primary"
    address = "mcpist.koyeb.app"
    enabled = true
    weight  = 1.0
  }

  origins {
    name    = "flyio-standby"
    address = "mcpist.fly.dev"
    enabled = true
    weight  = 0.0  # Standby: トラフィックは通常Primaryへ
  }

  minimum_origins = 1

  monitor = cloudflare_load_balancer_monitor.health.id
}

# Health Check Monitor
resource "cloudflare_load_balancer_monitor" "health" {
  type           = "http"
  expected_body  = "ok"
  expected_codes = "200"
  method         = "GET"
  path           = "/health"
  interval       = 30
  retries        = 2
  timeout        = 5
}

# Load Balancer
resource "cloudflare_load_balancer" "mcp" {
  zone_id          = var.zone_id
  name             = "mcp.${var.domain}"
  fallback_pool_id = cloudflare_load_balancer_pool.mcp_servers.id
  default_pool_ids = [cloudflare_load_balancer_pool.mcp_servers.id]
  proxied          = true

  # Failover設定
  steering_policy = "off"  # 最初の健全なオリジンを使用
}
```

### GitHub Actions統合

```yaml
# .github/workflows/terraform.yml

name: Terraform

on:
  push:
    paths:
      - 'terraform/**'
    branches: [main]

jobs:
  apply:
    runs-on: ubuntu-latest
    defaults:
      run:
        working-directory: terraform
    steps:
      - uses: actions/checkout@v4

      - uses: hashicorp/setup-terraform@v3
        with:
          terraform_version: "1.7"

      - name: Terraform Init
        run: terraform init
        env:
          TF_VAR_cloudflare_api_token: ${{ secrets.CLOUDFLARE_API_TOKEN }}

      - name: Terraform Apply
        run: terraform apply -auto-approve
        env:
          TF_VAR_cloudflare_api_token: ${{ secrets.CLOUDFLARE_API_TOKEN }}
          TF_VAR_zone_id: ${{ secrets.CLOUDFLARE_ZONE_ID }}
          TF_VAR_domain: "mcpist.com"
```

---

## 環境変数

### 共通

| 変数 | 値 |
|------|-----|
| `SUPABASE_URL` | https://xxx.supabase.co |
| `SUPABASE_KEY` | (共通) |
| `STRIPE_SECRET_KEY` | (共通) |

### 環境別

| 変数 | Koyeb | Fly.io |
|------|-------|--------|
| `ENVIRONMENT` | production-primary | production-standby |
| `LOG_PREFIX` | [koyeb] | [flyio] |

---

## Phase 1 スコープ

### 実装する

- [ ] Fly.io デプロイ
- [ ] GitHub Actions CI/CD
- [ ] デプロイ後検証
- [ ] Cloudflare Terraform化

### 実装しない

- Staging環境
- Blue/Greenデプロイ
- カナリアリリース

---

## 関連ドキュメント

- [dsn-infrastructure.md](./dsn-infrastructure.md) - インフラ全体構成
- [dsn-load-management.md](./dsn-load-management.md) - 負荷対策設計
