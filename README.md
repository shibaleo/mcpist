# MCPist

MCP (Model Context Protocol) gateway service — Connect AI assistants to external tools through a single endpoint.

## Usage

Add to your MCP client configuration (Claude Code, Cursor, etc.):

```json
{
  "mcpServers": {
    "mcpist": {
      "url": "https://mcp.mcpist.app/v1/mcp",
      "headers": {
        "Authorization": "Bearer <your-api-key>"
      }
    }
  }
}
```

Get your API key at [mcpist.app](https://mcpist.app).

## Supported Modules

Notion, GitHub, Jira, Confluence, Google Workspace (Sheets, Docs, Drive, Calendar, Tasks), Todoist, TickTick, Microsoft Todo, Asana, Trello, Airtable, Dropbox, PostgreSQL, Grafana, and more.

## Architecture

```
Console (Next.js / Vercel)
    ↓
Worker (Cloudflare Workers) — Auth + Routing
    ↓
Server (Go / Render) — MCP Handler + REST API
    ↓
PostgreSQL
```

### Authentication

1. **User auth**: Clerk (JWT) via Console
2. **API keys**: Ed25519 JWT issued by Server
3. **Gateway auth**: Worker signs short-lived JWT for Server-to-Server trust

## Tech Stack

| Layer | Technology |
|-------|------------|
| Frontend | Next.js 16, React 19, Tailwind CSS, Radix UI |
| API Gateway | Cloudflare Workers, Hono, TypeScript |
| Backend | Go 1.24, GORM |
| Database | PostgreSQL |
| Auth | Clerk, Ed25519 JWT |
| Billing | Stripe |
| Observability | Grafana Loki |
| Monorepo | pnpm workspaces, Turborepo |

---

## Development

### Prerequisites

- Node.js 20+
- pnpm 9
- Go 1.24+
- Docker Desktop

### Setup

```bash
pnpm install
cp .env.example .env.local   # Edit with your values
pnpm env:sync
```

### Run

```bash
pnpm dev
```

Starts PostgreSQL (Docker), Console, Server, and Worker concurrently.

| Service | URL | Description |
|---------|-----|-------------|
| PostgreSQL | localhost:57432 | Database (Docker) |
| Console | http://localhost:3000 | Web UI |
| Server | http://localhost:8089 | MCP Server |
| Worker | http://localhost:8787 | API Gateway |

```bash
pnpm db:down    # Stop PostgreSQL container
```

### Scripts

| Command | Description |
|---------|-------------|
| `pnpm dev` | Start DB + all apps |
| `pnpm db:up` | Start PostgreSQL only |
| `pnpm db:down` | Stop PostgreSQL |
| `pnpm dev:console` | Console only |
| `pnpm dev:server` | Go Server only (Air hot reload) |
| `pnpm dev:worker` | Worker only (Wrangler) |
| `pnpm build` | Build all apps (Turbo) |
| `pnpm lint` | Lint all apps |
| `pnpm test` | Run tests |
| `pnpm env:sync` | Distribute .env.local to each app |
| `pnpm erd:build` | Generate ER diagram from schema |

### Project Structure

```
mcpist/
├── apps/
│   ├── console/        # Web UI
│   ├── server/         # MCP Server + REST API
│   └── worker/         # API Gateway
├── database/
│   └── migrations/     # PostgreSQL schema
├── docs/               # Specifications & design docs
├── scripts/            # Dev utilities
└── .github/workflows/  # CI
```

### Deployment

| App | Platform | Domain |
|-----|----------|--------|
| Console | Vercel | mcpist.app |
| Worker | Cloudflare Workers | mcp.mcpist.app |
| Server | Render | mcpist.onrender.com |
| Database | Neon | — |

### API Specification

OpenAPI 3.1 spec: `GET /openapi.json`
Source: `apps/worker/src/openapi.yaml`
