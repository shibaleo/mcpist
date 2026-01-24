# MCPist Devcontainer 実装状況

## 実装完了

| タスク | ステータス | 備考 |
|--------|------------|------|
| `.devcontainer/Dockerfile` | 完了 | DinD対応、Node.js/Go/wrangler/Supabase CLI/OpenTofu |
| `.devcontainer/devcontainer.json` | 完了 | ポートフォワード、VS Code拡張機能設定 |
| `.devcontainer/docker-compose.yml` | 完了 | DinD有効、環境変数設定 |
| `.devcontainer/post-create.sh` | 完了 | 依存関係インストール、Supabase起動 |
| `compose/api.yml` | 完了 | api-koyeb(:8081), api-flyio(:8082) |
| Worker LB実装 | 既存で完了 | `apps/worker/src/index.ts` に実装済み |
| `infra/modules/cloudflare/` | 完了 | KV namespace, DNS |
| `infra/modules/vercel/` | 完了 | Project, 環境変数 |
| `infra/modules/koyeb/` | 完了 | App, Service |
| `infra/modules/flyio/` | 完了 | flyctl経由でデプロイ |
| `infra/modules/supabase/` | 完了 | Project設定 |
| `infra/environments/dev/` | 完了 | ローカルbackend |
| `infra/environments/prod/` | 完了 | 全モジュール統合 |

---

## 作成されたファイル一覧

```
mcpist/
├── .devcontainer/
│   ├── Dockerfile              # 新規作成
│   ├── devcontainer.json       # 新規作成
│   ├── docker-compose.yml      # 新規作成
│   └── post-create.sh          # 新規作成
│
├── compose/
│   └── api.yml                 # 新規作成
│
├── infra/
│   ├── modules/
│   │   ├── cloudflare/main.tf  # 新規作成
│   │   ├── vercel/main.tf      # 新規作成
│   │   ├── koyeb/main.tf       # 新規作成
│   │   ├── flyio/main.tf       # 新規作成
│   │   └── supabase/main.tf    # 新規作成
│   └── environments/
│       ├── dev/main.tf         # 新規作成
│       └── prod/main.tf        # 新規作成
```

---

## 次のアクション

1. **Goサーバー改修（手動）**: `apps/server/cmd/server/main.go` に `INSTANCE_ID`, `INSTANCE_REGION` 環境変数対応を追加
2. **Devcontainer起動テスト**: VS Codeで「Reopen in Container」
3. **ネットワーク検証**: DinD内での疎通確認
4. **LB検証**: `docker stop mcpist-api-koyeb` でflyioのみに流れることを確認

---

## Goサーバー改修内容（手動対応）

`apps/server/cmd/server/main.go` を以下のように変更:

```go
// main() 関数内に追加
instanceID := os.Getenv("INSTANCE_ID")
if instanceID == "" {
    instanceID = "local"
}
instanceRegion := os.Getenv("INSTANCE_REGION")
if instanceRegion == "" {
    instanceRegion = "local"
}

log.Printf("Instance: %s (region: %s)", instanceID, instanceRegion)

// /health ハンドラーを以下に変更
http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Type", "application/json")
    w.Header().Set("X-Instance-ID", instanceID)
    w.Header().Set("X-Instance-Region", instanceRegion)
    w.WriteHeader(http.StatusOK)
    fmt.Fprintf(w, `{"status":"ok","instance":"%s","region":"%s"}`, instanceID, instanceRegion)
})
```
