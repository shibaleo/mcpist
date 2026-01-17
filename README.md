# MCPist

MCP (Model Context Protocol) gateway service - Connect AI assistants to external tools.

## Project Structure

```
mcpist/
├── apps/
│   ├── server/         # MCP Server (Go)
│   ├── console/        # User Console (Next.js)
│   └── worker/         # API Gateway (Cloudflare Worker)
├── packages/           # Shared packages
├── supabase/           # Supabase migrations
├── docs/               # Documentation
└── .github/            # GitHub Actions
```

## Prerequisites

- Node.js 20+
- pnpm 9+
- Go 1.23+
- Docker (for local Supabase)
- Supabase CLI

## Getting Started

### 1. Install dependencies

```bash
pnpm install
```

### 2. Start Supabase locally

```bash
# Install Supabase CLI if not installed
# https://supabase.com/docs/guides/cli

# Start local Supabase
supabase start

# Apply migrations
supabase db reset
```

### 3. Set up environment variables

```bash
cp .env.example .env.local
# Edit .env.local with your Supabase credentials
```

### 4. Start development servers

```bash
# Start all apps
pnpm dev

# Or start individual apps
pnpm --filter @mcpist/console dev
cd apps/server && make run
```

## Development

### Available Scripts

| Command | Description |
|---------|-------------|
| `pnpm dev` | Start all apps in development mode |
| `pnpm build` | Build all apps |
| `pnpm lint` | Lint all apps |
| `pnpm test` | Run tests for all apps |
| `pnpm clean` | Clean build artifacts |

### Server (Go)

```bash
cd apps/server
make run      # Start server
make test     # Run tests
make lint     # Run linter
make build    # Build binary
```

### Console (Next.js)

```bash
cd apps/console
pnpm dev      # Start dev server
pnpm build    # Build for production
pnpm lint     # Run ESLint
```

### Worker (Cloudflare)

```bash
cd apps/worker
pnpm dev      # Start local dev server
pnpm build    # Build (dry-run)
pnpm deploy   # Deploy to Cloudflare
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
