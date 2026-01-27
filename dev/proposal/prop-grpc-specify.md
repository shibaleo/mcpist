# ~~gRPC移行構想 - 骨格定義による設計明確化~~ **REJECTED**

> **判定: Rejected (2026-01-27)**
> Python MCP SDK の pluggable transport PR ([#1936](https://github.com/modelcontextprotocol/python-sdk/pull/1936)) が「MCP仕様外」としてreject。
> MCP仕様レベルでのgRPCサポートも根本的ブロッカー (SEP-1319) が未解決。MCPistへの導入は時期尚早。

## 背景

GoogleがMCPのネイティブトランスポートとしてgRPCを提案（2026-01-19）。~~Python MCP SDKでプラグ可能なトランスポートのPRが進行中。~~ → **PRは2026-01-23にrejectされた。**

参考: [https://cloud.google.com/blog/ja/products/networking/grpc-as-a-native-transport-for-mcp](https://cloud.google.com/blog/ja/products/networking/grpc-as-a-native-transport-for-mcp)

## 提案アーキテクチャ

```
Client --[JSON-RPC/SSE]--> CF Worker --[gRPC]--> API Server
                            │                      │
                            ├ 認証                 ├ Middleware (gRPC)
                            └ 変換                 ├ Handler (gRPC)
                                                   └ Module (gRPC)
```

### レイヤー責務

- Gateway (CF Worker): JSON-RPC受信、認証、gRPCへの変換
- Handler: gRPC受信、モジュール振り分け（変換ロジック集約）
- Module: 外部API呼び出しのみ、変換ロジックなし

## メリット

- API Server内部は純粋なgRPCで統一
- 将来MCP標準がgRPCになったらGatewayの変換層を外すだけ
- GoとgRPCの相性が最高（Google製、第一級サポート）
- Protobufからのコード生成で型安全

## レスポンスフォーマット

LLM向けにはTOON/CSVを提供。Protobufのcontentフィールドに文字列として格納。

```
message ToolResponse {
  string format = 1;      // "toon", "csv", "json"
  string content = 2;     // 実際のデータ
  bool success = 3;
  string error = 4;
}
```

## 実装ツール

### protoc-gen-go-mcp (Redpanda)

.protoからMCPサーバーコードを自動生成するprotocプラグイン。

- GitHub: [https://github.com/redpanda-data/protoc-gen-go-mcp](https://github.com/redpanda-data/protoc-gen-go-mcp)
- ランタイム: mark3labs/mcp-go
- gRPC/ConnectRPC両対応

## ステータス

構想段階。MCP標準のgRPCトランスポート採用状況を注視しつつ、実装の美学を追求するため先行検討。

### 自動生成される部分

- MCP Server実装（stdio/SSEトランスポート）
- ListTools実装（.protoのアノテーションから自動生成）
- CallToolのルーティングロジック
- 型変換コード（JSON ↔ Protobuf）
- gRPCサーバー/クライアントスタブ

### 開発者が実装する部分

- .proto定義（インターフェース設計）
- 実際のビジネスロジック（外部API呼び出し等）
- TOON/CSV等へのフォーマット変換

### 実装例の比較

### 従来（手動実装）

```
// 手動でMCPプロトコル実装
func (s *Server) handleListTools(w http.ResponseWriter, r *http.Request) {
    // JSONパース
    // バリデーション
    // レスポンス組み立て
    // JSONエンコード
}

func (s *Server) handleCallTool(w http.ResponseWriter, r *http.Request) {
    // JSONパース
    // ツール名によるswitch文
    // 各ツールの実装呼び出し
    // エラーハンドリング
    // JSONエンコード
}

// 新規ツール追加時、全て手動で追加
```

### gRPC + protoc-gen-go-mcp

```
// .proto定義のみ
service NotionService {
  rpc SearchPages(SearchRequest) returns (SearchResponse) {
    option (mcp.tool) = {
      name: "search_pages"
      description: "Search Notion pages"
    };
  }
}

// 実装は実際のロジックのみ
func (s *NotionService) SearchPages(ctx context.Context, req *SearchRequest) (*SearchResponse, error) {
    results, err := s.client.Search(ctx, req.Query)
    return &SearchResponse{Pages: results}, err
}

// MCP プロトコル処理は全て自動生成
// 新規ツール追加 = .protoに定義追加 + 実装メソッド追加のみ
```

### 軽量化の効果

- コード量: 70-80%削減（ボイラープレート排除）
- バグリスク: 大幅減少（手動JSON処理が不要）
- 開発速度: 新規ツール追加が数分～数十分
- 保守性: .protoが唯一の真実の情報源
- 型安全性: コンパイル時エラー検出

### Notion実装での具体例

```
# 1. .proto定義（インターフェース）
cat > proto/notion/notion.proto <<EOF
service NotionService {
  rpc SearchPages(SearchRequest) returns (SearchResponse);
  rpc GetPage(GetPageRequest) returns (GetPageResponse);
  rpc AppendBlocks(AppendBlocksRequest) returns (AppendBlocksResponse);
}
EOF

# 2. コード生成
buf generate proto/notion
# → notion.pb.go, notion_grpc.pb.go, notion_mcp.go が生成

# 3. 実装（ビジネスロジックのみ）
cat > modules/notion/service.go <<EOF
func (s *NotionService) SearchPages(ctx context.Context, req *SearchRequest) (*SearchResponse, error) {
    return s.client.Search(ctx, req.Query, req.Limit)
}
EOF

# 完了！MCPサーバーとして動作可能
```

### モジュール拡張の容易性

新規モジュール（例: Slack）追加のステップ：

1. proto/slack/slack.proto作成（5分）
2. buf generate実行（1分）
3. modules/slack/service.go実装（外部APIロジックのみ、30分）
4. handler登録（1行、1分）

合計: 約40分で新規モジュールが本番投入可能

### 設計の美学との整合性

- Designative Liability: .protoが「意味の定義」を担う
- 分散システム: 各モジュールは独立した.proto定義
- 比較優位: 自動生成で実装コストを最小化し、設計に集中
- 単一障害点の排除: .protoさえあれば複数言語で再実装可能

## 骨格定義による設計明確化

protoc-gen-go-mcpは実装を軽量化するのではなく、インターフェースの骨格を自動生成し、設計を明確化するツールです。

### 自動生成される部分（配管工事）

- MCPプロトコル処理（JSON-RPCパース、バリデーション、エンコード）
- ListTools実装（.protoのアノテーションからツール定義を生成）
- CallToolのルーティングロジック
- 型変換コード（JSON ↔ Protobuf）
- gRPCサーバー/クライアントスタブ

### 開発者が実装する部分（ビジネスロジック）

- .proto定義（インターフェース設計）
- DB呼び出し（Supabase: トークン取得、ユーザー検証）
- 外部API呼び出し（Notion, GitHub, Jira等）
- TOON/CSV等へのフォーマット変換

### ボイラープレート削減効果

- 配管工事コード: 約300行削減（MCPプロトコル処理、ツール定義、型定義）
- ビジネスロジック: 削減なし（DB呼び出し、API呼び出しは従来通り必要）
- バグリスク: 大幅減少（手動JSON処理が不要）
- 保守性: .protoが唯一の真実の情報源
- 型安全性: コンパイル時エラー検出

### 設計の美学との整合性

- Designation Liability: .protoが「意味の定義」を担う。シンボルに意味を付与する責任を明確化。
- 分散システム: 各モジュールは独立した.proto定義を持ち、単一障害点を排除。
- 比較優位: 配管工事を自動化し、本質的な設計（インターフェース定義）に集中。
- 複数言語対応: .protoさえあればGo以外の言語でも再実装可能

---

## 調査結果 (2026-01-27)

### Python MCP SDK Pluggable Transport PR — Rejected

[PR #1936 "Pluggable transport"](https://github.com/modelcontextprotocol/python-sdk/pull/1936) は **2026-01-23 にメンテナ @maxisbey によって閉じられた**。

#### reject理由

1. **CONTRIBUTING.md違反** — 事前issueなし、メンテナ承認なしで提出
2. **MCP仕様外** — gRPCトランスポートはMCP仕様に含まれていないため、リポジトリに追加できない
3. **メンテナンス負荷** — 仕様に含まれない大規模アーキテクチャ変更は受け入れられない

> "This is a brand new transport which is not part of the MCP Specification, so we can't really add it to the repo. Ideally you implement this in a separate package if you're wanting to use it"

### MCP仕様レベルの議論状況

| Issue/SEP | 内容 | 状態 | URL |
|-----------|------|------|-----|
| #966 | Add gRPC as a Standard Transport | Closed (→ #1352へ移行) | [github](https://github.com/modelcontextprotocol/modelcontextprotocol/issues/966) |
| SEP-1352 | Add gRPC as a transport (正式SEP) | Closed/Completed (Draft段階) | [github](https://github.com/modelcontextprotocol/modelcontextprotocol/issues/1352) |
| SEP-1319 | Decouple Request Payload from JSON-RPC | **Open (根本的ブロッカー)** | [github](https://github.com/modelcontextprotocol/modelcontextprotocol/issues/1319) |
| Discussion #283 | Why not Protobuf/gRPC? | Open | [github](https://github.com/orgs/modelcontextprotocol/discussions/283) |

#### SEP-1319 が根本的ブロッカーである理由

現在のMCP仕様はJSON-RPCに密結合しており、gRPCバインディングを定義するにはリクエスト/レスポンスのデータモデルをRPCメソッド定義から切り離す必要がある。SEP-1319はこの分離を提案しているが未完了。これが解決しない限り、gRPCの標準トランスポート化は不可能。

#### SEP-1352 のコミュニティ反応

- Google Cloud のエンジニア (Kurtis Van Gent, Mark Roth, Harvey Tuch) が著者
- Spotify等の大手企業が支持を表明 (43 upvotes)
- 一方で反対意見もあり: "I'm not really in favor of gRPC'ing things just to do it" (Adrian Cole, 2025-12-24)
- Connect Protocolを代替として推す意見も (HTTP/2 trailer非対応環境への配慮)

### MCPistへの実装可能性評価

**結論: 時期尚早。投資対効果が低い。**

| 観点 | 評価 |
|------|------|
| MCP仕様準拠 | ✗ gRPCは仕様外。現行仕様は stdio + Streamable HTTP のみ |
| 根本ブロッカー | ✗ SEP-1319 (JSON-RPCデカップリング) が未解決 |
| SDK対応 | ✗ pluggable transport PR がreject済み |
| クライアント対応 | ✗ Claude Desktop, Claude Code, Cursor 等は Streamable HTTP のみ |
| CF Worker互換性 | ✗ Cloudflare Workers は gRPC をネイティブサポートしていない |
| コスト対効果 | ✗ 既存 handler.go のボイラープレートは十分コンパクト |
| 現行方式の問題 | ✗ 現行 JSON-RPC/HTTP で運用上の問題なし |

### 提案書の主張に対する反論

| 主張 | 反論 |
|------|------|
| コード量 70-80% 削減 | 過大評価。MCPプロトコル処理 (handler.go) は既にコンパクト。モジュール側ビジネスロジックは gRPC化しても不変 |
| 新規モジュール追加が40分 | 現状でも同程度 (tools.go にツール定義 + handler 実装) |
| .proto が唯一の真実の情報源 | 現状の tools.go + annotations が同じ役割を果たしている |
| 将来 MCP 標準が gRPC になったら変換層を外すだけ | その前提が成立するか不明。SEP-1319 すら未解決 |

### 通信パターンの本質的ミスマッチ (Adrian Cole の指摘)

MCPトランスポートとしてのgRPCには、仕様の成熟度以前に**プロトコル設計レベルの不適合**がある。

#### MCPの通信モデル（LSP由来）

- **双方向リクエスト** — サーバーもクライアントにリクエストを発信する（`sampling/createMessage`, `roots/list`）
- **通知** — レスポンスを期待しない fire-and-forget メッセージ（`notifications/resources/updated`）
- **長寿命セッション** — `initialize` でセッション確立、状態を維持
- **対称的メッセージング** — JSON-RPC はリクエスト/レスポンスの方向を制限しない

#### gRPCの設計思想

- **クライアント→サーバーの一方向RPC** — クライアントがメソッドを呼び、サーバーが返す。サーバーがクライアントのメソッドを呼ぶ概念がない
- **ステートレス寄り** — 各RPC呼び出しは原則独立
- **双方向ストリーミング** — あるが、1つのRPC呼び出しの中での双方向であり「サーバーがクライアントの任意のメソッドを呼ぶ」こととは異なる

#### 具体的な不適合

| MCP機能 | gRPCでの表現 | 問題 |
|---------|-------------|------|
| Server → Client リクエスト (`sampling/createMessage`) | 双方向ストリーミング内にメッセージプロトコルを再実装、またはクライアントもgRPCサーバー化 | 不自然。JSON-RPCで解決済みの問題をgRPC上で再発明 |
| 通知 (fire-and-forget) | 空レスポンスを返す unary RPC、またはストリーミング内に埋め込み | gRPCの自然な使い方ではない |
| セッション状態 | メタデータでセッションID伝搬 | gRPCに外付けの仕組みが必要 |

#### 結論: 用途による使い分け

| 用途 | gRPC適合性 | 理由 |
|------|-----------|------|
| **MCPトランスポート** (Client ↔ Server) | **不向き** | MCPは双方向メッセージングプロトコル。gRPCの一方向RPCモデルと根本的に不一致 |
| **MCPサーバー内部通信** (Gateway → Handler → Module) | **適合** | 純粋な「呼び出し側→実行側」のRPCパターン。gRPCの設計思想と完全に合致 |

MCPistの文脈では、`CF Worker → Go Server` 間や将来のモジュール分離時の内部通信にgRPCは有効だが、MCPプロトコル自体のトランスポートとしては不適合。現状Go Serverは単一プロセスでモジュールを直接呼び出しており、内部gRPC化のユースケースもまだ存在しない。

### 推奨アクション

- **ウォッチ継続**: SEP-1319 と SEP-1352 の進捗を注視
- **実装は保留**: 仕様が確定し、主要クライアントが対応してから検討
- **現行アーキテクチャで継続**: JSON-RPC/Streamable HTTP 構成は仕様準拠かつ実用的
- **内部gRPCは将来課題**: モジュールをマイクロサービス分離する段階で検討

### ソース

- [PR #1936: Pluggable transport (Closed/Rejected)](https://github.com/modelcontextprotocol/python-sdk/pull/1936)
- [Issue #966: Add gRPC as a Standard Transport](https://github.com/modelcontextprotocol/modelcontextprotocol/issues/966)
- [SEP-1352: Add gRPC as a transport](https://github.com/modelcontextprotocol/modelcontextprotocol/issues/1352)
- [SEP-1319: Decouple Request Payload from RPC Methods](https://github.com/modelcontextprotocol/modelcontextprotocol/issues/1319)
- [Discussion #283: Why not Protobuf/gRPC?](https://github.com/orgs/modelcontextprotocol/discussions/283)
- [Google Cloud Blog: gRPC as a custom transport for MCP](https://cloud.google.com/blog/products/networking/grpc-as-a-native-transport-for-mcp)