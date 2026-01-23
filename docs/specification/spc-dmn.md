# MCPist ドメイン仕様書（spc-dmn）

## ドキュメント管理情報

| 項目 | 値 |
|------|-----|
| Status | `approved` |
| Version | v1.1 (Sprint-003) |
| Note | Domain Specification - サブドメイン分離方式 |

---

## 概要

本ドキュメントは、MCPistで使用するドメインとURL構成を定義する。

**設計方針:**
- サブドメインでコンポーネント（Console / MCP API）を分離
- プロキシ不要、各サービスが直接応答
- MCPレイテンシ最小化を優先

---

## ドメイン構成

### ベースドメイン

| ドメイン | 用途 | 管理 |
|---------|------|------|
| mcpist.app | MCPistサービス | Cloudflare |

### 環境別サブドメイン

| 環境 | Console (Vercel) | MCP API (Worker) |
|------|------------------|------------------|
| Dev | dev.mcpist.app | mcp.dev.mcpist.app |
| Stg | stg.mcpist.app | mcp.stg.mcpist.app |
| Prd | mcpist.app | mcp.mcpist.app |

---

## URL構成

### サブドメイン分離

各コンポーネントは独立したサブドメインで公開。プロキシ不要。

| コンポーネント | 役割 | インフラ |
|--------------|------|---------|
| Console | 管理UI | Vercel |
| MCP API | MCPサーバーゲートウェイ | Cloudflare Workers |

### 環境別URL一覧

| 環境 | Console | MCP API |
|------|---------|---------|
| Dev | https://dev.mcpist.app | https://mcp.dev.mcpist.app |
| Stg | https://stg.mcpist.app | https://mcp.stg.mcpist.app |
| Prd | https://mcpist.app | https://mcp.mcpist.app |

---

## DNS設定

### Cloudflare DNS

| サブドメイン | レコードタイプ | 値 | Proxy |
|-------------|--------------|-----|-------|
| dev | CNAME | Vercel | Off |
| stg | CNAME | Vercel | Off |
| @ (apex) | A/CNAME | Vercel | Off |
| mcp.dev | CNAME | Cloudflare Workers | On |
| mcp.stg | CNAME | Cloudflare Workers | On |
| mcp | CNAME | Cloudflare Workers | On |

**注意**: ConsoleはVercel直接、MCP APIはCloudflare Workers経由。

### ルーティング構成

```
Console (dev.mcpist.app等)
    └── Vercel直接応答

MCP API (mcp.dev.mcpist.app等)
    └── Cloudflare Worker → 認証 → Render/Koyeb
```

---

## OAuth関連エンドポイント

### OAuth Protected Resource Metadata (RFC 9728)

| 環境 | URL |
|------|-----|
| Dev | https://mcp.dev.mcpist.app/.well-known/oauth-protected-resource |
| Stg | https://mcp.stg.mcpist.app/.well-known/oauth-protected-resource |
| Prd | https://mcp.mcpist.app/.well-known/oauth-protected-resource |

### OAuth Consent Page

| 環境 | URL |
|------|-----|
| Dev | https://dev.mcpist.app/oauth/consent |
| Stg | https://stg.mcpist.app/oauth/consent |
| Prd | https://mcpist.app/oauth/consent |

---

## MCP接続URL

MCPクライアント（Claude.ai, ChatGPT, Claude Code等）が接続するURL。

| 環境 | URL | 認証方式 |
|------|-----|---------|
| Dev | https://mcp.dev.mcpist.app | OAuth 2.0 / APIキー |
| Stg | https://mcp.stg.mcpist.app | OAuth 2.0 / APIキー |
| Prd | https://mcp.mcpist.app | OAuth 2.0 / APIキー |

---

## 内部サービスURL

外部公開しないサービスのURL。

| サービス | 環境 | URL |
|---------|------|-----|
| MCP Server (Primary) | 各環境 | Render内部URL |
| MCP Server (Secondary) | 各環境 | Koyeb内部URL |
| Auth + DB Backend | 各環境 | Supabase URL |

---

## SSL/TLS

| サービス | 証明書管理 |
|---------|----------|
| Cloudflare Workers | Cloudflare自動管理 |
| Supabase | Supabase自動管理 |
| Vercel | Vercel自動管理 |
| Render | Render自動管理 |
| Koyeb | Koyeb自動管理 |

---

## 関連ドキュメント

| ドキュメント | 内容 |
|-------------|------|
| [spc-inf.md](./spc-inf.md) | インフラ仕様書（コンポーネント配置） |
| [spc-dpl.md](./spc-dpl.md) | デプロイ仕様書（環境構成） |
