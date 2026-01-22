#!/bin/bash
# MCPist - Deploy Secrets to Production Services
#
# Prerequisites:
#   - vercel CLI: npm i -g vercel
#   - koyeb CLI: https://www.koyeb.com/docs/cli/installation
#   - wrangler CLI: npm i -g wrangler
#   - supabase CLI: npm i -g supabase
#
# Usage:
#   # Set environment variables first
#   export SUPABASE_SERVICE_ROLE_KEY=xxx
#   export INTERNAL_SERVICE_KEY=xxx
#
#   # Run script
#   ./scripts/deploy-secrets.sh

set -e

echo "=== MCPist Secret Deployment ==="

# Check required environment variables
if [ -z "$SUPABASE_SERVICE_ROLE_KEY" ]; then
  echo "Error: SUPABASE_SERVICE_ROLE_KEY is not set"
  exit 1
fi

if [ -z "$INTERNAL_SERVICE_KEY" ]; then
  echo "Error: INTERNAL_SERVICE_KEY is not set"
  exit 1
fi

# Production URLs
SUPABASE_URL="https://xstfrjvgpqxvyuochtss.supabase.co"
CONSOLE_URL="https://console.mcpist.app"
SERVER_URL="https://mcp.mcpist.app"
WORKER_URL="https://api.mcpist.app"

echo ""
echo "--- Vercel (Console) ---"
if command -v vercel &> /dev/null; then
  vercel env rm NEXT_PUBLIC_SUPABASE_URL production -y 2>/dev/null || true
  echo "$SUPABASE_URL" | vercel env add NEXT_PUBLIC_SUPABASE_URL production

  vercel env rm SUPABASE_SERVICE_ROLE_KEY production -y 2>/dev/null || true
  echo "$SUPABASE_SERVICE_ROLE_KEY" | vercel env add SUPABASE_SERVICE_ROLE_KEY production

  vercel env rm NEXT_PUBLIC_APP_URL production -y 2>/dev/null || true
  echo "$CONSOLE_URL" | vercel env add NEXT_PUBLIC_APP_URL production

  vercel env rm INTERNAL_SERVICE_KEY production -y 2>/dev/null || true
  echo "$INTERNAL_SERVICE_KEY" | vercel env add INTERNAL_SERVICE_KEY production

  echo "Vercel secrets updated"
else
  echo "Skipped: vercel CLI not found"
fi

echo ""
echo "--- Koyeb (Server) ---"
if command -v koyeb &> /dev/null; then
  koyeb secrets create SUPABASE_URL --value "$SUPABASE_URL" 2>/dev/null || \
    koyeb secrets update SUPABASE_URL --value "$SUPABASE_URL"

  koyeb secrets create SUPABASE_SERVICE_ROLE_KEY --value "$SUPABASE_SERVICE_ROLE_KEY" 2>/dev/null || \
    koyeb secrets update SUPABASE_SERVICE_ROLE_KEY --value "$SUPABASE_SERVICE_ROLE_KEY"

  koyeb secrets create CONSOLE_URL --value "$CONSOLE_URL" 2>/dev/null || \
    koyeb secrets update CONSOLE_URL --value "$CONSOLE_URL"

  koyeb secrets create VAULT_URL --value "${CONSOLE_URL}/api" 2>/dev/null || \
    koyeb secrets update VAULT_URL --value "${CONSOLE_URL}/api"

  koyeb secrets create INTERNAL_SERVICE_KEY --value "$INTERNAL_SERVICE_KEY" 2>/dev/null || \
    koyeb secrets update INTERNAL_SERVICE_KEY --value "$INTERNAL_SERVICE_KEY"

  echo "Koyeb secrets updated"
else
  echo "Skipped: koyeb CLI not found"
fi

echo ""
echo "--- Cloudflare (Worker) ---"
if command -v wrangler &> /dev/null; then
  echo "$SUPABASE_URL" | wrangler secret put SUPABASE_URL
  echo "$SUPABASE_SERVICE_ROLE_KEY" | wrangler secret put SUPABASE_SERVICE_ROLE_KEY
  echo "$SERVER_URL" | wrangler secret put SERVER_URL
  echo "$INTERNAL_SERVICE_KEY" | wrangler secret put INTERNAL_SERVICE_KEY

  echo "Cloudflare secrets updated"
else
  echo "Skipped: wrangler CLI not found"
fi

echo ""
echo "=== Done ==="
