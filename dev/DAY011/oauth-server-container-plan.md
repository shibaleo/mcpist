# DAY011: Traefik統合計画

## 概要

mcpistリポジトリにTraefikリバースプロキシを導入し、`*.localhost` ドメインで各サービスにアクセスできるようにする。
これにより本番環境（`*.mcpist.app`）と同様の開発体験を実現する。

> **Note**: OAuth Server分離はDAY012で実施する。

## 目標のアーキテクチャ

```
開発環境（DAY011完了後）

                    ┌─────────────────────────────────────┐
                    │ Traefik (:80)                       │
                    │ リバースプロキシ + 自動ルーティング │
                    └─────────────────────────────────────┘
                                    │
        ┌───────────────────────────┼───────────────────────────┐
        │                           │                           │
        ▼                           ▼                           ▼
┌───────────────────┐   ┌───────────────────┐   ┌───────────────────┐
│ console.localhost │   │ mcp.localhost     │   │ api.localhost     │
│ → Console :3000   │   │ → Worker :8787    │   │ → Go Server :8089 │
└───────────────────┘   └───────────────────┘   └───────────────────┘

本番環境との対応:
┌─────────────────────┬─────────────────────┐
│ 開発 (*.localhost)  │ 本番               │
├─────────────────────┼─────────────────────┤
│ console.localhost   │ console.mcpist.app │
│ mcp.localhost       │ mcp.mcpist.app     │
│ api.localhost       │ api.mcpist.app     │
│ db.localhost:54323  │ Supabase Cloud     │
└─────────────────────┴─────────────────────┘
```

## 実装計画

### Phase 1: docker-compose.traefik.yml 作成

```yaml
services:
  traefik:
    image: traefik:v3.0
    container_name: mcpist-traefik
    command:
      - "--api.insecure=true"
      - "--api.dashboard=true"
      - "--providers.docker=true"
      - "--providers.docker.exposedbydefault=false"
      - "--entrypoints.web.address=:80"
      - "--log.level=INFO"
    ports:
      - "80:80"
      - "8080:8080"  # Traefik Dashboard
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock:ro
    networks:
      - mcpist

  console:
    build:
      context: ./apps/console
      dockerfile: Dockerfile.dev
    container_name: mcpist-console
    labels:
      - "traefik.enable=true"
      - "traefik.http.routers.console.rule=Host(`console.localhost`)"
      - "traefik.http.routers.console.entrypoints=web"
      - "traefik.http.services.console.loadbalancer.server.port=3000"
    environment:
      - PORT=3000
      - NEXT_PUBLIC_APP_URL=http://console.localhost
      - NEXT_PUBLIC_MCP_SERVER_URL=http://mcp.localhost
      - NEXT_PUBLIC_SUPABASE_URL=http://host.docker.internal:54321
      - NEXT_PUBLIC_SUPABASE_ANON_KEY=${NEXT_PUBLIC_SUPABASE_ANON_KEY}
      - SUPABASE_SERVICE_ROLE_KEY=${SUPABASE_SERVICE_ROLE_KEY}
      - INTERNAL_SERVICE_KEY=${INTERNAL_SERVICE_KEY}
    volumes:
      - ./apps/console:/app
      - /app/node_modules
      - /app/.next
    networks:
      - mcpist
    depends_on:
      - traefik
    extra_hosts:
      - "host.docker.internal:host-gateway"

  worker:
    build:
      context: ./apps/worker
      dockerfile: Dockerfile.dev
    container_name: mcpist-worker
    labels:
      - "traefik.enable=true"
      - "traefik.http.routers.worker.rule=Host(`mcp.localhost`)"
      - "traefik.http.routers.worker.entrypoints=web"
      - "traefik.http.services.worker.loadbalancer.server.port=8787"
    environment:
      - GATEWAY_SECRET=${GATEWAY_SECRET}
      - SUPABASE_ANON_KEY=${SUPABASE_ANON_KEY}
      - RENDER_URL=http://server:8089
      - SUPABASE_URL=http://host.docker.internal:54321
      - SUPABASE_JWKS_URL=http://host.docker.internal:54321/auth/v1/.well-known/jwks.json
    networks:
      - mcpist
    depends_on:
      - traefik
    extra_hosts:
      - "host.docker.internal:host-gateway"

  server:
    build:
      context: ./apps/server
      dockerfile: Dockerfile.dev
    container_name: mcpist-server
    labels:
      - "traefik.enable=true"
      - "traefik.http.routers.server.rule=Host(`api.localhost`)"
      - "traefik.http.routers.server.entrypoints=web"
      - "traefik.http.services.server.loadbalancer.server.port=8089"
    environment:
      - PORT=8089
      - SUPABASE_URL=http://host.docker.internal:54321
      - SUPABASE_SERVICE_ROLE_KEY=${SUPABASE_SERVICE_ROLE_KEY}
      - CONSOLE_URL=http://console.localhost
      - GATEWAY_SECRET=${GATEWAY_SECRET}
      - VAULT_URL=http://console:3000/api
    volumes:
      - ./apps/server:/app
    networks:
      - mcpist
    depends_on:
      - traefik
    extra_hosts:
      - "host.docker.internal:host-gateway"

networks:
  mcpist:
    driver: bridge
```

### Phase 2: 各サービスのDockerfile.dev作成

#### apps/console/Dockerfile.dev

```dockerfile
FROM node:22-alpine

WORKDIR /app

# pnpm インストール
RUN corepack enable && corepack prepare pnpm@9.15.4 --activate

# 依存関係インストール
COPY package.json pnpm-lock.yaml* ./
RUN pnpm install --frozen-lockfile

# ソースコード（volumeでマウント）
COPY . .

EXPOSE 3000

CMD ["pnpm", "dev"]
```

#### apps/worker/Dockerfile.dev

```dockerfile
FROM node:22-alpine

WORKDIR /app

# pnpm インストール
RUN corepack enable && corepack prepare pnpm@9.15.4 --activate

# 依存関係インストール
COPY package.json pnpm-lock.yaml* ./
RUN pnpm install --frozen-lockfile

COPY . .

EXPOSE 8787

# wrangler dev --local でNode.jsモードで実行
CMD ["pnpm", "dev", "--local", "--port", "8787"]
```

#### apps/server/Dockerfile.dev

```dockerfile
FROM golang:1.24-alpine

WORKDIR /app

# Air インストール（ホットリロード）
RUN go install github.com/air-verse/air@latest

# 依存関係インストール
COPY go.mod go.sum ./
RUN go mod download

# ソースコード（volumeでマウント）
COPY . .

EXPOSE 8089

CMD ["air", "-c", ".air.toml"]
```

### Phase 3: 環境変数の整理

#### .env.traefik（新規作成）

```env
# =============================================================================
# Traefik開発環境用環境変数
# =============================================================================
# docker-compose.traefik.yml で使用

# =============================================================================
# URLs (Traefik ドメイン)
# =============================================================================
NEXT_PUBLIC_APP_URL=http://console.localhost
NEXT_PUBLIC_MCP_SERVER_URL=http://mcp.localhost
CONSOLE_URL=http://console.localhost
VAULT_URL=http://console.localhost/api
MCP_SERVER_URL=http://mcp.localhost

# =============================================================================
# Supabase (ローカル - host.docker.internal経由)
# =============================================================================
SUPABASE_URL=http://host.docker.internal:54321
NEXT_PUBLIC_SUPABASE_URL=http://127.0.0.1:54321
SUPABASE_ANON_KEY=${SUPABASE_ANON_KEY}
NEXT_PUBLIC_SUPABASE_ANON_KEY=${NEXT_PUBLIC_SUPABASE_ANON_KEY}
SUPABASE_SERVICE_ROLE_KEY=${SUPABASE_SERVICE_ROLE_KEY}

# =============================================================================
# Secrets
# =============================================================================
GATEWAY_SECRET=${GATEWAY_SECRET}
INTERNAL_SERVICE_KEY=${INTERNAL_SERVICE_KEY}

# =============================================================================
# Go Server
# =============================================================================
PORT=8089
INSTANCE_ID=local-docker
INSTANCE_REGION=local
```

### Phase 4: package.json スクリプト追加

```json
{
  "scripts": {
    "dev:traefik": "supabase start && docker compose -f docker-compose.traefik.yml up -d --build",
    "stop:traefik": "docker compose -f docker-compose.traefik.yml down",
    "logs:traefik": "docker compose -f docker-compose.traefik.yml logs -f"
  }
}
```

## アクセスURL

| URL | 説明 |
|-----|------|
| http://console.localhost | Console UI |
| http://mcp.localhost | MCP Gateway (Worker) |
| http://api.localhost | Go Server |
| http://localhost:8080 | Traefik Dashboard |
| http://localhost:54323 | Supabase Studio |

## Supabaseとの連携

Supabaseはホストマシンで `supabase start` で起動し、Docker内からは `host.docker.internal:54321` でアクセスする。

```
┌─────────────────────────────────────────────────┐
│ Host Machine                                    │
│                                                 │
│   supabase start → localhost:54321              │
│                         ↑                       │
│                         │ host.docker.internal  │
│   ┌─────────────────────┼───────────────────┐   │
│   │ Docker Network      │                   │   │
│   │                     │                   │   │
│   │  console ──────────┬┘                   │   │
│   │  worker ───────────┘                    │   │
│   │  server ───────────┘                    │   │
│   └─────────────────────────────────────────┘   │
└─────────────────────────────────────────────────┘
```

## スケジュール

| タスク | 工数 |
|-------|------|
| docker-compose.traefik.yml 作成 | 1h |
| Console Dockerfile.dev 作成 | 0.5h |
| Worker Dockerfile.dev 作成 | 0.5h |
| Server Dockerfile.dev 作成 | 0.5h |
| .env.traefik 作成 | 0.5h |
| package.json スクリプト追加 | 0.25h |
| 動作確認・デバッグ | 1h |
| ドキュメント更新 | 0.5h |
| **合計** | **4.75h（約0.5日）** |

## 成果物

- `docker-compose.traefik.yml`
- `apps/console/Dockerfile.dev`
- `apps/worker/Dockerfile.dev`
- `apps/server/Dockerfile.dev`
- `.env.traefik`
- 更新された `package.json`

## 次のステップ（DAY012）

1. OAuth Server分離（`apps/oauth`）
2. `oauth.localhost` ルーティング追加
3. Console OAuth実装削除
4. E2Eテスト
