# DAY030 作業ログ

## 日付

2026-02-15

---

## コミット一覧

| # | ハッシュ | メッセージ |
|---|---------|-----------|
| 1 | 29f787a | refactor(server): extract SSE/Inline transport from handler into middleware |

---

## 計画との差分

当初計画は Phase 1a (OAuth 2.1 Server) だったが、Claude 認可フローが原因不明で復旧したため Phase 1 を sprint010-backlog に移動。代わりにシステム構成図の更新と Go Server のリファクタリングを実施。

---

## 実施内容

### 1. Sprint 010 バックログ作成

Sprint 009 バックログを引き継ぎ、Sprint 010 の状況を反映した `sprint010-backlog.md` を新規作成。

- Phase 1 (OAuth Server) をバックログに格下げ
- Claude 認可フロー一時障害の原因調査を優先度低で追加

### 2. システム構成図 (grh-componet-interactions.canvas) 更新

#### 第1回（リファクタリング前）

| 変更 | 内容 |
|------|------|
| MOD | モジュール数の記述を削除（変動が激しいため） |
| CON | "クレジット課金" → "プラン管理" |
| AMW | "クレジット消費" → "日次制限"、SSE/Inline トランスポート + セッション管理を追加 |
| HDL | エンドポイント記述 → 責務記述に変更（JSON-RPC ルーティング、ツールフィルタリング、フォーマット変換、バッチ検証） |
| EXT | 個別サービス列挙 → "各種SaaS API" に簡略化 |
| OBS | Grafana Loki を明記 |
| DST | Token Vault の記述を DST に統合 |
| BRK | Broker コンポーネントを新規追加 |
| グループ | "MCP サーバー" グループで AMW/HDL/BRK/MOD を囲む |

#### 第2回（実コードとの照合・修正）

実装コードを全調査し、構成図の記述を実態に合わせて修正。

| 変更 | 内容 |
|------|------|
| GWY | "Rate Limit / Burst制限" 削除（Worker から削除済み、サーバー側に移行）、"Loki ログ送信" 追加 |
| AUS | "OAuth 2.1準拠" → "OAuth 2.0 (Supabase Auth)" |
| AMW | "パニックリカバリ"、"レートリミット（per-user）" 追加（recovery.go, ratelimit.go を反映） |
| MOD | "トークン取得" → "パラメータバリデーション"（トークン管理は BRK の責務） |
| DST | "課金情報 / クレジット残高" → "プラン・日次使用量"、"pgsodium TCE" 明記、"プロンプトテンプレート" 追加 |
| CON | "プロンプトテンプレート管理"、"MCP接続情報表示" 追加 |

### 3. トランスポート層リファクタリング (Go Server)

handler.go に混在していた SSE/Inline トランスポート層とビジネスロジックを分離。

#### 新規ファイル

| ファイル | 内容 |
|----------|------|
| `internal/jsonrpc/types.go` | JSON-RPC 2.0 型 (Request, Response, Error) とエラーコード定数を共通パッケージに切り出し |
| `internal/middleware/transport.go` | SSE/Inline トランスポート。`RequestProcessor` interface で handler に委譲 |

#### 変更ファイル

| ファイル | 変更内容 |
|----------|----------|
| `internal/mcp/types.go` | JSON-RPC 型を `jsonrpc` パッケージから type alias で再エクスポート |
| `internal/mcp/handler.go` | Session, sessions, mu, ServeHTTP, handleSSE, handleMessage 等を削除。`processRequest` → `ProcessRequest` に公開 (568 → 396 行, -172 行) |
| `cmd/server/main.go` | ミドルウェアチェーンに Transport を追加 |

#### 設計判断

- **循環参照回避**: `mcp` → `middleware` の import が既存のため、Request/Response/Error を `internal/jsonrpc` パッケージに切り出し
- **Transport は末端**: `func(next) http.Handler` ではなく、チェーン最後で `RequestProcessor` interface を受け取る `http.Handler`
- **Handler は http.Handler を実装しなくなる**: `ProcessRequest` メソッドのみ公開

#### ミドルウェアチェーン (変更後)

```
Before: Recovery → Authorize → RateLimit → MCPHandler (transport + logic 混在)
After:  Recovery → Authorize → RateLimit → Transport → MCPHandler (logic のみ)
```

---

## ビルド・テスト・デプロイ結果

- `go build ./...` — pass
- `go test ./...` — 全テスト pass
- Render auto-deploy — `29f787a` が live（ビルド約4分21秒）

### 本番動作検証

| テスト | 結果 | 備考 |
|--------|------|------|
| `/health` | OK | `status: ok`, `db: ok` |
| GET `/mcp` (認証なし) | 401 正常拒否 | `INVALID_GATEWAY_SECRET` |
| `get_module_schema` (MCP経由) | OK | github モジュール 26 ツール返却 |
| `run` (MCP経由, github:get_repo) | OK | Compact フォーマット適用確認 |
| `list_lists` (MCP経由, microsoft_todo) | OK | 5 リスト返却 |

リファクタリングによる機能影響なし。

---

## ステージ済みファイル

| ファイル | 種別 |
|----------|------|
| `apps/server/cmd/server/main.go` | 変更 |
| `apps/server/internal/jsonrpc/types.go` | 新規 |
| `apps/server/internal/mcp/handler.go` | 変更 |
| `apps/server/internal/mcp/types.go` | 変更 |
| `apps/server/internal/middleware/transport.go` | 新規 |

## 未ステージ

| ファイル | 内容 |
|----------|------|
| `docs/graph/grh-componet-interactions.canvas` | 構成図更新 |
| `docs/graph/grh-*.canvas` (5 ファイル) | Obsidian 編集による変更 |
| `dev/sprint/sprint010-backlog.md` | 新規 (untracked) |

---

## DAY030 サマリ

| 項目 | 内容 |
|------|------|
| テーマ | 構成図更新 + トランスポート層リファクタリング |
| 対応スプリント | Sprint 010 (計画変更: OAuth → リファクタリング) |
| コミット数 | 1 |
| handler.go | 568 → 396 行 (-30%) |
| 新規パッケージ | `internal/jsonrpc` (型の共通化)、`middleware/transport.go` (トランスポート分離) |
| 構成図修正 | 2回実施（初回: 大枠更新、2回目: 実コード照合による6箇所修正） |
| デプロイ | Render auto-deploy 成功、本番動作検証 5項目 all pass |
| 主な成果 | ミドルウェアとハンドラの責務分離、構成図の実装一致 |
