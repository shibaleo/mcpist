#!/bin/bash
# MCPist - Deploy Secrets to Cloud Services
#
# Prerequisites:
#   - vercel CLI: npm i -g vercel
#   - render CLI: or set via Render Dashboard
#   - wrangler CLI: npm i -g wrangler
#
# Usage:
#   # Load from .env.local or set environment variables
#   source .env.local
#
#   # Deploy to specific environment
#   ./scripts/deploy-secrets.sh [dev|prd]

set -e

ENV="${1:-dev}"
echo "=== MCPist Secret Deployment (${ENV}) ==="

# ── Required environment variables ──
REQUIRED_VARS=(
  "DATABASE_URL"
  "GATEWAY_SIGNING_KEY"
  "CLERK_SECRET_KEY"
  "CLERK_JWKS_URL"
  "NEXT_PUBLIC_CLERK_PUBLISHABLE_KEY"
  "API_KEY_PRIVATE_KEY"
)

for var in "${REQUIRED_VARS[@]}"; do
  if [ -z "${!var}" ]; then
    echo "Error: ${var} is not set"
    exit 1
  fi
done

# ── Environment-specific URLs ──
if [ "$ENV" = "prd" ]; then
  CONSOLE_URL="https://mcpist.app"
  MCP_SERVER_URL="https://mcp.mcpist.app"
  WORKER_JWKS_URL="https://mcp.mcpist.app/.well-known/jwks.json"
  WRANGLER_ENV=""
else
  CONSOLE_URL="https://dev.mcpist.app"
  MCP_SERVER_URL="https://mcp.dev.mcpist.app"
  WORKER_JWKS_URL="https://mcp.dev.mcpist.app/.well-known/jwks.json"
  WRANGLER_ENV="--env dev"
fi

echo ""
echo "--- Vercel (Console) ---"
if command -v vercel &> /dev/null; then
  VERCEL_ENV="${ENV}"
  [ "$ENV" = "prd" ] && VERCEL_ENV="production"
  [ "$ENV" = "dev" ] && VERCEL_ENV="preview"

  declare -A VERCEL_VARS=(
    ["NEXT_PUBLIC_CLERK_PUBLISHABLE_KEY"]="$NEXT_PUBLIC_CLERK_PUBLISHABLE_KEY"
    ["CLERK_SECRET_KEY"]="$CLERK_SECRET_KEY"
    ["CLERK_JWKS_URL"]="$CLERK_JWKS_URL"
    ["NEXT_PUBLIC_APP_URL"]="$CONSOLE_URL"
    ["NEXT_PUBLIC_MCP_SERVER_URL"]="$MCP_SERVER_URL"
    ["ENVIRONMENT"]="$ENV"
  )

  for key in "${!VERCEL_VARS[@]}"; do
    vercel env rm "$key" "$VERCEL_ENV" -y 2>/dev/null || true
    echo "${VERCEL_VARS[$key]}" | vercel env add "$key" "$VERCEL_ENV"
  done

  echo "Vercel secrets updated for ${VERCEL_ENV}"
else
  echo "Skipped: vercel CLI not found"
fi

echo ""
echo "--- Cloudflare (Worker) ---"
if command -v wrangler &> /dev/null; then
  WORKER_SECRETS=(
    "SERVER_URL"
    "GATEWAY_SIGNING_KEY"
    "CLERK_JWKS_URL"
    "SERVER_JWKS_URL"
    "STRIPE_WEBHOOK_SECRET"
  )

  for secret in "${WORKER_SECRETS[@]}"; do
    if [ -n "${!secret}" ]; then
      echo "${!secret}" | wrangler secret put "$secret" $WRANGLER_ENV
    fi
  done

  # Grafana (optional)
  for secret in GRAFANA_LOKI_URL GRAFANA_LOKI_USER GRAFANA_LOKI_API_KEY; do
    if [ -n "${!secret}" ]; then
      echo "${!secret}" | wrangler secret put "$secret" $WRANGLER_ENV
    fi
  done

  echo "Cloudflare secrets updated"
else
  echo "Skipped: wrangler CLI not found"
fi

echo ""
echo "--- Render (Go Server) ---"
echo "Set the following environment variables in Render Dashboard:"
echo ""
echo "  DATABASE_URL          = ${DATABASE_URL}"
echo "  WORKER_JWKS_URL       = ${WORKER_JWKS_URL}"
echo "  API_KEY_PRIVATE_KEY   = ${API_KEY_PRIVATE_KEY}"
echo "  CLERK_JWKS_URL        = ${CLERK_JWKS_URL}"
echo "  PORT                  = 8080"
echo "  ENVIRONMENT           = ${ENV}"
echo "  INSTANCE_ID           = render-${ENV}"
echo "  INSTANCE_REGION       = oregon"
echo ""
echo "Note: Render does not have a CLI for secrets. Use the Dashboard."
echo "      https://dashboard.render.com"

echo ""
echo "=== Done ==="
