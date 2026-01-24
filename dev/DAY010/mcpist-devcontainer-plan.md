# MCPist Devcontainer 実装計画

## 概要

本番環境の **Cloudflare Worker + KV + ロードバランシング** を開発環境で完全に再現する構成を構築する。

---

## 本番構成

```
┌─────────────────────────────────────────────────────────────┐
│                    Cloudflare                                │
│  ┌─────────────────┐  ┌─────────────┐  ┌─────────────────┐  │
│  │     Worker      │  │     KV      │  │   DNS/Proxy     │  │
│  │  - Routing      │  │  - Health   │  │                 │  │
│  │  - Load Balance │  │    状態     │  │                 │  │
│  │  - Health Check │  │  - Config   │  │                 │  │
│  └────────┬────────┘  └──────┬──────┘  └─────────────────┘  │
└───────────┼──────────────────┼──────────────────────────────┘
            │                  │
            ▼                  │
     ┌──────┴──────┐           │
     ▼             ▼           │
  ┌──────┐     ┌──────┐        │
  │Koyeb │     │Fly.io│        │
  │ API  │     │ API  │        │
  └──┬───┘     └──┬───┘        │
     └─────┬──────┘            │
           ▼                   │
     ┌───────────┐             │
     │ Supabase  │◄────────────┘
     └───────────┘
```

---

## 開発環境構成（Devcontainer + DinD）

```
ホストOS (Windows)
└── Docker Desktop
    └── Devcontainer (DinD有効)
        │
        ├── VS Code Server + 拡張機能
        │
        ├── ツール:
        │   ├── Node.js (wrangler, next)
        │   ├── Go
        │   ├── Supabase CLI
        │   └── OpenTofu
        │
        ├── プロセス:
        │   ├── wrangler dev       (:8787)  ← Worker + KVエミュレート
        │   └── npm run dev        (:3000)  ← Next.js
        │
        └── DinDデーモン
            │
            ├── compose: api
            │   ├── api-koyeb      (:8081)
            │   └── api-flyio      (:8082)
            │
            └── compose: supabase (supabase start)
                ├── postgres       (:54321)
                ├── auth
                ├── storage
                └── studio         (:54323)
```

---

## リクエストフロー（開発環境）

```
Browser
   │
   ▼ :3000
┌─────────┐
│ Next.js │
└────┬────┘
     │
     ▼ :8787
┌─────────────────────┐
│ Worker (wrangler)   │
│                     │
│  KV: ヘルスチェック状態  │
│                     │
│  LB: 重み付け振り分け   │
│    ├─→ :8081 (50%)  │
│    └─→ :8082 (50%)  │
└──────────┬──────────┘
           │
     ┌─────┴─────┐
     ▼           ▼
┌─────────┐ ┌─────────┐
│api-koyeb│ │api-flyio│
│  :8081  │ │  :8082  │
└────┬────┘ └────┬────┘
     │           │
     └─────┬─────┘
           ▼ :54321
     ┌───────────┐
     │ Supabase  │
     └───────────┘
```

---

## 目標ディレクトリ構成

```
mcpist/
├── .devcontainer/
│   ├── devcontainer.json      # Devcontainer設定
│   ├── Dockerfile             # DinD対応イメージ
│   ├── docker-compose.yml     # Devcontainer用compose
│   └── post-create.sh         # 初期化スクリプト
│
├── compose/
│   └── api.yml                # APIコンテナ2台の定義
│
├── apps/
│   ├── console/               # Next.js (UI) ← 既存
│   ├── server/                # Go API ← 既存
│   └── worker/                # Cloudflare Worker ← 既存（拡張）
│
├── infra/                     # OpenTofu
│   ├── modules/
│   │   ├── cloudflare/
│   │   ├── vercel/
│   │   ├── koyeb/
│   │   ├── flyio/
│   │   └── supabase/
│   └── environments/
│       ├── dev/
│       └── prod/
│
└── supabase/                  # 既存
    └── migrations/
```

---

## 現在のmcpist構成との差分

| 項目 | 現状 | 目標 |
|------|------|------|
| .devcontainer/ | 未作成 | DinD対応Devcontainer |
| compose/api.yml | 未作成 | api-koyeb, api-flyio 2台構成 |
| apps/worker/ | 基本実装 | LB + ヘルスチェック + KV状態管理 |
| infra/ | 未作成 | OpenTofuモジュール群 |
| docker-compose.yml | console + server | DinD統合構成 |

---

## 実装タスク

### Phase 1: Devcontainer環境構築

| # | タスク | 詳細 |
|---|--------|------|
| 1-1 | `.devcontainer/Dockerfile` | DinD有効なベースイメージ作成 |
| 1-2 | `.devcontainer/devcontainer.json` | VS Code設定、拡張機能、ポートフォワード |
| 1-3 | `.devcontainer/docker-compose.yml` | Devcontainer起動用compose |
| 1-4 | `.devcontainer/post-create.sh` | Node.js, Go, wrangler, Supabase CLI, OpenTofuインストール |

### Phase 2: API冗長化構成

| # | タスク | 詳細 |
|---|--------|------|
| 2-1 | `compose/api.yml` | api-koyeb(:8081), api-flyio(:8082) 定義 |
| 2-2 | Goサーバー改修 | インスタンス識別レスポンス追加 |
| 2-3 | ヘルスチェックエンドポイント | `/health` エンドポイント実装 |

### Phase 3: Worker LB実装

| # | タスク | 詳細 |
|---|--------|------|
| 3-1 | KV状態管理 | ヘルス状態のKV保存/読み取り |
| 3-2 | 重み付けLB | 設定可能な振り分けロジック |
| 3-3 | ヘルスチェッカー | 定期的なバックエンド監視 |
| 3-4 | フェイルオーバー | 片系障害時の自動切り替え |

### Phase 4: OpenTofuモジュール

| # | タスク | 詳細 |
|---|--------|------|
| 4-1 | `infra/modules/cloudflare/` | KV namespace, DNS, Worker設定 |
| 4-2 | `infra/modules/vercel/` | Project, 環境変数, ドメイン |
| 4-3 | `infra/modules/koyeb/` | App, Service, 環境変数 |
| 4-4 | `infra/modules/flyio/` | App, Machine, Secrets |
| 4-5 | `infra/modules/supabase/` | Project設定 |
| 4-6 | `infra/environments/` | dev/prod環境分離 |

### Phase 5: 統合・検証

| # | タスク | 詳細 |
|---|--------|------|
| 5-1 | ネットワーク接続検証 | DinD内の疎通確認 |
| 5-2 | LB動作検証 | 重み付け分散の確認 |
| 5-3 | フェイルオーバー検証 | `docker stop api-koyeb` でflyioのみに流れることを確認 |
| 5-4 | KV状態検証 | Miniflareでのローカルエミュレート確認 |

---

## ポート構成

| サービス | ポート | 用途 |
|----------|--------|------|
| Next.js (console) | 3000 | フロントエンドUI |
| Worker (wrangler dev) | 8787 | LB + ルーティング |
| api-koyeb | 8081 | APIインスタンス1 |
| api-flyio | 8082 | APIインスタンス2 |
| Supabase API | 54321 | データベースAPI |
| Supabase Studio | 54323 | DB管理UI |

---

## インフラ管理の責務分離

| ツール | 責務 |
|--------|------|
| **OpenTofu** | 何を作るか（KV namespace、DNS、プロジェクト設定等） |
| **Wrangler** | 何を動かすか（Workerコード、デプロイ） |
| **Supabase CLI** | スキーマ管理（マイグレーション） |

---

## 開発環境で検証できること

- LBの重み付け: 複数リクエストで分散確認
- ヘルスチェック: `docker stop api-koyeb` → KV更新 → flyioのみに流れる
- フェイルオーバー: 片方落として復旧シナリオ
- KVの状態管理: Miniflareがローカルエミュレート

---

## 次のアクション

1. `.devcontainer/` ファイル群の作成
2. `compose/api.yml` の作成
3. Worker LBロジック実装
4. OpenTofuモジュール作成
5. 統合テスト
