---
title: MCPist インフラ仕様サブコア定義
aliases:
  - dtl-inf-cor
  - infrastructure-sub-core
tags:
  - MCPist
  - architecture
  - sub-core
  - DTL
document-type: detail
document-class: DTL
created: 2026-01-14T00:00:00+09:00
updated: 2026-01-14T00:00:00+09:00
---
# MCPist インフラ仕様サブコア定義

## ドキュメント管理情報

| 項目 | 値 |
|------|-----|
| Status | `current` |
| Version | v1.0 (DAY5) |
| Note | spec-inf.mdからサブコア要件を抽出 |

---

## 概要

本ドキュメントは、spec-inf.md（インフラ仕様書）の中で、コア機能（COR-xxx）を前提とした場合に複数の独立した根拠を持つ要件を「サブコア」として定義する。

**評価基準:**
- コア機能を前提とした場合に、2つ以上の独立した根拠を持つ
- そのコア機能が変わらない限り、変更されない

---

## サブコア要件

### INF-COR-001: MCPサーバーのDocker軽量化

**前提コア:**
- COR-007 (Go採用)
- COR-101 ($0/月コスト制約)
- COR-102 (Koyeb選択)

**独立した根拠（3つ）:**

| # | 根拠 | 由来コア |
|---|------|----------|
| 1 | Go採用 → 静的バイナリ、CGO_ENABLED=0でクロスコンパイル | COR-007 |
| 2 | $0制約 → Koyeb Free Tierの512MBメモリ制限 | COR-101 |
| 3 | Koyeb選択 → Docker必須、軽量イメージが高速デプロイに直結 | COR-102 |

**定義:**
```dockerfile
FROM golang:1.22-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o mcpist ./cmd/server

FROM alpine:3.19
RUN apk --no-cache add ca-certificates
WORKDIR /app
COPY --from=builder /app/mcpist .
EXPOSE 8088
CMD ["./mcpist"]

軽量化ポイント:
├─ golang:1.22-alpine（ビルド用）
├─ alpine:3.19（実行用、最小限）
├─ CGO_ENABLED=0（静的リンク、Cライブラリ不要）
└─ マルチステージビルド（最終イメージにビルドツール不要）
```

**spec-inf.mdでの位置:** 2. Koyeb（MCPサーバー）

---

### INF-COR-002: Supabase統合アーキテクチャ

**前提コア:**
- COR-003 (サーバー側認証設計)
- COR-005 (RLS非依存認可)

**独立した根拠（3つ）:**

| # | 根拠 | 由来コア |
|---|------|----------|
| 1 | サーバー側認証 → Supabase AuthでJWT発行、Edge FunctionでToken Broker | COR-003 |
| 2 | RLS非依存 → Edge Function内でuser_idフィルタ、RLSは補助層 | COR-005 |
| 3 | $0制約 → 単一のSupabaseプロジェクトでAuth/DB/Vault/Edge Functionsを統合 | COR-101 |

**定義:**
```
Supabase統合構成:
├─ Auth: Authサーバー（ユーザー認証、JWT発行）
├─ PostgreSQL: Tool Sieve（ロール権限管理、RLS）
├─ Vault: Token Broker（トークン暗号化保存）
├─ Edge Functions: Tool Sieve, Token Broker（RPCエンドポイント）
└─ Dashboard: 管理者用（SQLエディタ、許可リスト登録、DB管理）

Edge Functions:
├─ tool-sieve: ロール権限に基づくツールフィルタリング
├─ token-exchanger: トークン取得・リフレッシュ
└─ oauth-callback: OAuth認可コールバック
```

**spec-inf.mdでの位置:** 3. Supabase

---

### INF-COR-003: Next.js + Vercel構成

**前提コア:**
- COR-006 (Next.js採用)
- COR-101 ($0/月コスト制約)

**独立した根拠（2つ）:**

| # | 根拠 | 由来コア |
|---|------|----------|
| 1 | Next.js採用 → App Router、@supabase/ssr連携、middleware.ts | COR-006 |
| 2 | $0制約 → Vercel Hobby Planで無料ホスティング | COR-101 |

**定義:**
```
管理UI構成:
├─ フレームワーク: Next.js 14 (App Router)
├─ スタイル: Tailwind CSS
├─ 認証: Supabase Auth (@supabase/ssr)
├─ デプロイ: Vercel

アーキテクチャ:
├─ middleware.ts: 認証チェック、system_role判定、ルート保護
├─ Route Handlers: SERVICE_ROLE_KEYでDB操作、system_roleアクセス制御
└─ Client Components: ANON_KEYでSupabase Auth、RLS適用データ取得

ルート保護:
├─ /admin/*: admin限定
├─ /settings: admin, user
└─ /oauth/connect: admin, user
```

**spec-inf.mdでの位置:** 4. Vercel（管理UI）

---

### INF-COR-004: MCPサーバーとEdge Functionの通信設計

**前提コア:**
- COR-003 (サーバー側認証設計)
- COR-007 (Go採用)

**独立した根拠（2つ）:**

| # | 根拠 | 由来コア |
|---|------|----------|
| 1 | サーバー側認証 → MCPサーバーがEdge Function経由でのみSupabaseにアクセス | COR-003 |
| 2 | Go採用 → JWT検証はJWKSエンドポイント利用（秘密鍵不要） | COR-007 |

**定義:**
```
環境変数:
├─ TOKEN_BROKER_URL: Edge FunctionのエンドポイントURL
└─ TOKEN_BROKER_KEY: Supabase ANON KEY（MCPサーバー↔Edge Function認証）

JWT検証:
├─ JWKSエンドポイント: https://<project>.supabase.co/auth/v1/jwks
├─ 公開鍵暗号方式（RS256等）
└─ MCPサーバー側に秘密鍵不要

通信原則:
├─ MCPサーバーはSupabaseに直接アクセスしない
├─ トークン暗号化保存はSupabase Vaultで実施
└─ トークン取得・リフレッシュはEdge Functionで実施
```

**spec-inf.mdでの位置:** 2.2 環境変数

---

## 非サブコア要件（単一根拠）

以下は重要な要件だが、コア機能からの導出が単一であるためサブコアとしない。

| 要件 | 根拠数 | 理由 |
|------|--------|------|
| Grafana Cloud可観測性 | 1 | $0制約からの単一導出 |
| Cloudflare DNS | 1 | $0制約からの単一導出 |
| GitHub Actions CI/CD | 1 | 運用効率化からの単一導出 |
| dev/mainブランチ戦略 | 1 | 品質管理からの単一導出 |
| 自動ロールバック | 1 | 運用安全性からの単一導出 |
| Supabase日次バックアップ | 1 | Supabase標準機能からの単一導出 |

---

## サブコアマトリックス

| ID | サブコア要件 | 前提コア | 根拠数 |
|----|--------------|----------|--------|
| INF-COR-001 | MCPサーバーDocker軽量化 | COR-007, COR-101, COR-102 | 3 |
| INF-COR-002 | Supabase統合アーキテクチャ | COR-003, COR-005, COR-101 | 3 |
| INF-COR-003 | Next.js + Vercel構成 | COR-006, COR-101 | 2 |
| INF-COR-004 | MCPサーバー↔Edge Function通信 | COR-003, COR-007 | 2 |

---

## 関連ドキュメント

- [dtl-core.md](dtl-core.md) - コア機能定義
- [spec-inf.md](../DAY4/spec-inf.md) - インフラ仕様書
