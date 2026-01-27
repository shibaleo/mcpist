# マルチテナントMCP設計の現状と動向（2025年1月時点）

## 概要

MCP（Model Context Protocol）は2024年11月にAnthropicが発表したオープン標準。LLMと外部ツール・データソースの統合を標準化する。2025年にはOpenAI、Google DeepMindも採用し、事実上の業界標準となった。

マルチテナント設計については、**公式仕様には明示的な規定がなく**、コミュニティ主導で議論・実装が進んでいる段階。

---

## 用語定義

| 用語 | 定義 | 例 |
|------|------|-----|
| **テナント（Tenant）** | 組織的な責任単位・権限集約単位 | 会社A、個人事業主B |
| **ユーザー（User）** | 操作主体としての個人 | 田中さん、鈴木さん |
| **アカウント（Account）** | ユーザーに紐づく、論理的に分離された権限インスタンス | Notion個人用、GitHub仕事用 |

---

## コミュニティで議論されている設計パターン

### 1. 専用インスタンス型（Dedicated Instance）

各ユーザー/テナントが独自のMCPサーバーインスタンスを持つ。

| 項目 | 内容 |
|------|------|
| 分離レベル | 最高（完全分離） |
| コスト | 高 |
| 運用複雑度 | 高 |
| ユースケース | コンプライアンス重視、規制産業 |

### 2. マルチテナント共有型（Pooled Multi-Tenant）

単一のMCPサーバーを複数ユーザーで共有し、RBAC + RLSで分離。

| 項目 | 内容 |
|------|------|
| 分離レベル | 論理分離（DB/ポリシーレベル） |
| コスト | 低 |
| 運用複雑度 | 中（ポリシー実装が重要） |
| ユースケース | スケール重視、コスト効率 |

### 3. ハイブリッド型（Tenant-Isolated）

テナント単位でインフラ/ネームスペースを分離し、内部でRBACを適用。

| 項目 | 内容 |
|------|------|
| 分離レベル | テナント間は物理分離、テナント内は論理分離 |
| コスト | 中 |
| 運用複雑度 | 中〜高 |
| ユースケース | B2B SaaS、エンタープライズ |

---

## 認証・認可の共通パターン

コミュニティで広く推奨されているアプローチ:

### トークン戦略

- SSO/OAuth2/OIDC で認証
- 短命JWT発行（TTLを短く設定）
- クレームに `tenantId`, `userId`, `roles`, `scopes` を埋め込み
- 積極的なローテーションと失効管理

### アクセス制御

- 毎回のツール/リソース呼び出し時にトークン検証
- RBAC/ABACをサーバー側で強制
- ポリシーをコード化（OPA/Rego等）
- 最小権限の原則

### セッション管理

- サーバーはステートレスに保つ
- セッションコンテキストは外部キャッシュ（Redis等）に保存
- 接続/トレースIDでキー管理

---

## データ分離メカニズム

| レイヤー | 手法 |
|---------|------|
| DB | 行レベルセキュリティ（RLS）、カラムマスキング |
| ベクトルストア | テナント単位のネームスペース分離 |
| ファイルストア | ドキュメントACL、所有者情報付与 |
| 出力 | PII（個人識別情報）のマスキング |
| シークレット | Secrets Manager / KMS で暗号化保存 |

---

## 既存のマルチテナントMCP実装

### Sage MCP

オープンソースのマルチテナントMCPプラットフォーム。

- **バックエンド**: FastAPI（Python async）
- **フロントエンド**: React
- **DB**: PostgreSQL / Supabase対応
- **デプロイ**: Docker & Kubernetes（Helmチャート付き）

複数のMCPサーバーインスタンスを単一プラットフォームから管理。

---

## エンタープライズ要件（議論中）

Mirantis等が提唱するエンタープライズ向け要件:

- 細粒度のパーミッション制御
- 監査ログ（全操作のトレース）
- ガバナンス・コンプライアンス対応
- 電子カルテ（EHR）等の機密データアクセス

---

## MCP仕様の進化（2025年11月時点）

公式仕様に追加された関連機能:

### Tasks

サーバー側の作業を追跡する新しい抽象化。

- 状態: `working`, `input_required`, `completed`, `failed`, `cancelled`
- マルチステップ操作の管理に有用
- 結果の一定期間保持

### 能力ネゴシエーション

クライアント・サーバー間で初期化時にサポート機能を明示的に宣言。

---

## MCPistの位置づけ

### MCPistはマルチテナントではない

MCPistは当初「Pooled Multi-Tenant型」として設計を検討していたが、議論の結果**シングルテナント・シングルユーザー・マルチアカウント型**に設計変更した。

#### MCPの構造比較

| 構造 | テナント | ユーザー | アカウント | 例 |
|------|---------|---------|-----------|-----|
| **従来のMCP** | シングル | シングル | シングル | 1サービス専用MCPサーバー |
| **MCPist** | シングル | シングル | **マルチ** | 複数サービス統合MCPサーバー |
| **Pooled Multi-Tenant** | マルチ | マルチ | マルチ | Zapier的プラットフォーム |

**理由**: 他人のトークンを預かる構造（Pooled Multi-Tenant）は、Token Exchanger批判で指摘された「Trust Boundary崩壊」を引き起こす。

```
【MCPist: シングルテナント・シングルユーザー・マルチアカウント】

テナント: 自分（1）
  └─ ユーザー: 自分（1）
       └─ アカウント1（Notion個人）
       └─ アカウント2（GitHub仕事用）
       └─ アカウント3（Jira副業用）

→ 1サーバーで複数アカウントを管理 = インフラ節約
→ Trust Boundary維持（全部自分の権限）
```

MCPistの「マルチアカウント」機能は、**従来のシングルアカウントMCPサーバーを複数運用する煩雑さ・インフラオーバーヘッドを避けるための節約手段**に過ぎない。

### MCPist独自の先進的要素

| 要素 | 説明 |
|------|------|
| **メタツール + 遅延スキーマ取得** | マルチアカウント対応とContext Rot防止の両方を同時に解決 |
| **能力宣言** | `side_effect`, `idempotent`, `concurrency_safe` 等のツールメタデータ |
| **DAG/Goスクリプト実行** | LLMが生成した実行計画をサーバー側で並列実行 |
| **シングルテナント・シングルユーザー前提** | Trust Boundary維持のため、マルチテナント・マルチユーザーを明示的に採用しない |

### MCP仕様の根本的制約とMCPistの解決策

MCP標準プロトコルは**シングルアカウント前提**（1接続 = 1ユーザー = 1アカウント = 1権限セット）で設計されている。`tools/list`はinitialize時に全ツールを返す仕様であり、エンタープライズIdPでユーザーを識別しても「同一ユーザーの複数アカウント」を区別する仕組みは存在しない。

MCPistはこの制約を**メタツール設計**で回避する：

- initialize時: メタツール（`get_module_schema`, `call_module_tool`）のみ公開
- ツール呼び出し時: `get_module_schema`で認証情報を検証し、該当アカウントのツールのみ返却

この設計により、MCP仕様を逸脱せずにマルチアカウント対応を実現している。

---

## 参考資料

- [MCP Architecture - Official Specification](https://modelcontextprotocol.io/specification/2025-03-26/architecture)
- [One Year of MCP - November 2025 Spec Release](https://blog.modelcontextprotocol.io/posts/2025-11-25-first-mcp-anniversary/)
- [Building Multi-User AI Agents with an MCP Server](https://bix-tech.com/building-multi-user-ai-agents-with-an-mcp-server-architecture-security-and-a-practical-blueprint/)
- [Multi-Tenant MCP Servers: Why Centralized Management Matters](https://medium.com/@manikandan.eshwar/multi-tenant-mcp-servers-why-centralized-management-matters-a813b03b4a52)
- [Securing Model Context Protocol for Mass Enterprise Adoption - Mirantis](https://www.mirantis.com/blog/securing-model-context-protocol-for-mass-enterprise-adoption/)
- [Model Context Protocol - Wikipedia](https://en.wikipedia.org/wiki/Model_Context_Protocol)

---

*作成日: 2025年1月11日*
