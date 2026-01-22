# Development Dockerfile for Console (Next.js)
# Uses Turborepo prune for clean monorepo builds

# Stage 1: Prune monorepo
FROM node:20-alpine AS pruner
RUN corepack enable && corepack prepare pnpm@9.15.4 --activate
RUN npm install -g turbo

WORKDIR /app
COPY . .
RUN turbo prune @mcpist/console --docker

# Stage 2: Install dependencies
FROM node:20-alpine AS installer
RUN corepack enable && corepack prepare pnpm@9.15.4 --activate

WORKDIR /app

# Copy pruned lockfile and package.json files
COPY --from=pruner /app/out/json/ .

# Install dependencies
RUN pnpm install

# Copy pruned source code
COPY --from=pruner /app/out/full/ .

# Stage 3: Development runner
FROM node:20-alpine AS runner
RUN corepack enable && corepack prepare pnpm@9.15.4 --activate

WORKDIR /app

# Copy installed dependencies and source
COPY --from=installer /app .

WORKDIR /app/apps/console

EXPOSE 3000

CMD ["pnpm", "dev"]
