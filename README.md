# MCPist

MCP (Model Context Protocol) gateway service - Connect AI assistants to external tools.

## Project Structure

```
mcpist/
├── apps/
│   ├── server/         # MCP Server (Go)
│   └── console/        # User Console (Next.js)
├── worker/             # API Gateway (Cloudflare Worker)
├── supabase/           # Supabase migrations & config
├── packages/           # Shared packages
└── .github/            # GitHub Actions
```

## Prerequisites

- Node.js 20+
- pnpm 9+
- Go 1.23+
- Docker Desktop
- Supabase CLI
- Air (Go Hot Reload)

## Initial Setup (First Time Only)

### 1. Install required tools

```bash
# Install pnpm (if not installed)
npm install -g pnpm

# Install Supabase CLI
# Windows (scoop)
scoop bucket add supabase https://github.com/supabase/scoop-bucket.git
scoop install supabase

# macOS
brew install supabase/tap/supabase

# Install Air (Go Hot Reload)
go install github.com/air-verse/air@latest

# Add Go bin to PATH (Windows PowerShell - run once)
[Environment]::SetEnvironmentVariable("Path", $env:Path + ";$env:USERPROFILE\go\bin", "User")
# Then restart your terminal
```

### 2. Install dependencies

```bash
pnpm install
```

### 3. Set up environment variables

```bash
cp .env.example .env
# .env will be auto-populated after `supabase start`
```

## Development

### Start development servers

```bash
# Start everything (Supabase + Console + Server)
pnpm dev

# Stop everything
pnpm stop
```

This starts:
- **Supabase** → http://localhost:54321 (API), http://localhost:54323 (Studio)
- **Console (Next.js)** → http://localhost:3000
- **Server (Go)** → http://localhost:8089

### Docker mode (Domain-based routing)

Production-like local development with `*.localhost` domains via Traefik reverse proxy.

#### Default mode (Console / Server development)

For standard application development with single API server:

```bash
pnpm dev:docker       # Start containers
pnpm stop:docker      # Stop containers
pnpm logs:docker      # View logs
```

**Endpoints:**
| URL | Service |
|-----|---------|
| http://console.localhost | Console (Next.js) |
| http://mcp.localhost | MCP Gateway (Worker) |
| http://api.localhost | API Server (Go) |
| http://localhost:8080 | Traefik Dashboard |

#### Infra mode (Multi-server development)

For infrastructure development with primary + secondary API servers:

```bash
pnpm dev:docker:infra   # Start with primary + secondary API servers
pnpm stop:docker        # Stop containers
```

**Endpoints:**
| URL | Service |
|-----|---------|
| http://console.localhost | Console (Next.js) |
| http://mcp.localhost | MCP Gateway (Worker) |
| http://api.localhost/primary/* | Primary API Server |
| http://api.localhost/secondary/* | Secondary API Server |
| http://localhost:8080 | Traefik Dashboard |

**Health check examples:**
```bash
# Default mode
curl http://api.localhost/health

# Infra mode
curl http://api.localhost/primary/health
curl http://api.localhost/secondary/health
curl http://mcp.localhost/health  # Shows backend status
```

### Available Scripts

| Command | Description |
|---------|-------------|
| `pnpm dev` | Start Supabase + Console + Server (local) |
| `pnpm stop` | Stop Supabase |
| `pnpm dev:docker` | Start with Docker (default mode) |
| `pnpm dev:docker:infra` | Start with Docker (infra mode, multi-server) |
| `pnpm stop:docker` | Stop Docker containers |
| `pnpm logs:docker` | View Docker container logs |
| `pnpm build` | Build all apps |
| `pnpm lint` | Lint all apps |
| `pnpm test` | Run tests |
| `pnpm clean` | Clean build artifacts |

### Individual Apps

```bash
# Console (Next.js)
pnpm dev:console

# Server (Go)
pnpm dev:server

# Worker (Cloudflare)
pnpm dev:worker
```

## Database

### Migrations

```bash
# Create a new migration
supabase migration new <migration_name>

# Apply migrations
supabase db reset

# Generate types (for TypeScript)
supabase gen types typescript --local > packages/database/types.ts
```

## Documentation

See [docs/specification/](docs/specification/) for detailed specifications:

| Document | Description |
|----------|-------------|
| [idx-spc.md](docs/specification/idx-spc.md) | Specification Index |
| [spc-sys.md](docs/specification/spc-sys.md) | System Specification |
| [spc-dsn.md](docs/specification/spc-dsn.md) | Design Specification |
| [spc-tbl.md](docs/specification/spc-tbl.md) | Table Specification |
| [spc-itf.md](docs/specification/spc-itf.md) | Interface Specification |
| [spc-itr.md](docs/specification/spc-itr.md) | Interaction Specification |
| [spc-inf.md](docs/specification/spc-inf.md) | Infrastructure Specification |
| [spc-sec.md](docs/specification/spc-sec.md) | Security Specification |
| [spc-tst.md](docs/specification/spc-tst.md) | Test Specification |
| [spc-ops.md](docs/specification/spc-ops.md) | Operations Specification |
| [spc-dev.md](docs/specification/spc-dev.md) | Development Plan |
