# ADR-004: Token Exchangerパターンの採用

## ステータス

**Accepted** (2025-01-11)

## コンテキスト

MCP認証仕様（2025年11月）はエンタープライズIdP（Okta, Azure AD等）を前提としており、個人開発者向けのパターンが今後整備される可能性は低い。

この空白を埋める方法として、**Token Exchanger**パターンを検討した。

### Token Exchangerパターンとは

```
【Token Exchanger構造】
MCP Client → MCP Server → Token Exchanger → 外部サービス
                              ↓
                    ユーザーのトークンを保持
                    代理でAPI呼び出し
```

このパターンでは：
- ユーザーが外部サービス（Notion, GitHub等）のトークンをToken Exchangerに預ける
- MCP ServerはToken Exchanger経由でAPIを呼び出す
- ユーザーはOAuth認可フローを1回行うだけでよい

**メリット**:
- 参入障壁の劇的な低下
- 開発者体験の向上
- OAuth実装の集約

## 決定

MCPistは**Token Exchangerパターンを採用する。

ユーザー自身がTokenExchangerへトークンを入力(UI経由)し、MCPistに渡す構造を採用する。

```
【Token Exchanger構造】
MCP Client → MCP Server → Token Exchanger → 外部サービス
                              ↓
                    ユーザーのトークンを保持
                    代理でAPI呼び出し
```

## 理由

### 1. Trust Boundary崩壊の適用外

マルチテナント向けMCPサーバーの仕組みにおいて，Token Exchangerパターンの最大の問題は**Trust Boundaryの崩壊**である。

> Token Exchangerは「ユーザーの外部サービス権限を代理行使できる」構造を作り、実行主体・権限主体・利用主体をすべて同時に担う。**これはOAuthが20年かけて避けてきた設計**である。

具体的なリスク：
- Token Exchanger/MCP Serverの侵害 → 全ユーザー全サービス即死
- 権限スコープの過剰要求
- トークンの二次流出
- 監査不能（「誰が、どの操作をしたか」が不明確）

**しかし，MCPistは1ユーザ前提なので，TokenExchangerパターンを採用することができる．**

### 2. MCPの「文脈限定的」思想との整合性

MCPの思想は「ツールは最小権限で、文脈限定的に呼び出される」こと。

Token Exchangerは恒久的なユーザー権限を文脈非依存・MCP Server主導で扱うことになる。これは**MCPの「Contextual Tool Invocation」哲学と逆行**している。

### 3. シングルテナント設計との一貫性

MCPistはシングルテナント・シングルユーザー・マルチアカウント構造を採用している（ADR-0002参照）。

Token Exchangerパターンは本質的にシングルテナント・シングルユーザー向けの設計である。シングルテナントであれば、ユーザー自身がTokenExchnagerでトークンを管理すればよく、Trust Boundaryの制約が適用されない．

### 4. 「統合プラットフォーム」としての責任を負わない

Token Exchangerパターンが批判されるのは以下の場合のみ：

1. **統合プラットフォーム**として明確なToSのもとで運営されない場合
2. **強制的なスコープ制限**と**操作ログの完全可視化**が担保されない場合
3. 運営主体が十分な**セキュリティ監査・コンプライアンス**対応能力を持たない場合

MCPistはこれらの責任を負う「Zapier/Workato的プラットフォーム」ではない。個人のための個人ツールである。

## 影響

### 採用するもの

- TokenExcahngerによる認証情報管理
- ユーザー自身によるOAuthアプリ登録・トークン取得
- セルフホスト前提のデプロイモデル
- トークン保存・管理機能
- OAuth認可フローの代行
- ユーザー登録・認証UI

### 採用しないもの
- MCPクライアントにユーザの認証情報を渡す
- [エンタープライズIdPによるクライアント側認証(公式仕様)](https://modelcontextprotocol.io/specification/2025-11-25)

### ユーザーに求めるもの

- 外部サービスのOAuthアプリ登録能力
- トークンの安全な管理能力

これはMCPistの**ターゲットユーザー定義**と整合している：

> パワーユーザーが、自前でOAuthトークンを管理できる

## 代替案
### 案A: MCP Client側でのトークン管理

**却下理由**: MCP Clientが「巨大IdP」になる問題。MCPist側で解決すべきではないが、問題の先送りに過ぎない。

### 案B: Per-call ephemeral token（理想的だが未実装）

**将来検討**: ツール呼び出し単位で失効する短命トークン。MCPエコシステム全体での標準化が必要。現時点では誰も実装していない。

### 案C: 環境変数による自己管理

**却下理由**: 登録サービス増加に比例して官許変数が増加するため，登録が煩雑

### 案D: Token Exchanger（採用）

**採用理由**: Trust Boundary崩壊、MCPの思想との矛盾、統合プラットフォームとしての責任を負わないため，あえてアンチパターンを採用できる。


## 結論

Token Exchangerパターンは、MCP認証仕様の空白を埋める現実解として魅力的だが、公式仕様として**Trust Boundaryの崩壊**という致命的なリスクを伴う。

MCPistは**「個人のための個人ツール」**であり、統合プラットフォームとしての責任を負う設計ではないため，あえてアンチパターンを採用することができる．

ユーザー自身がトークンを管理する構造は、参入障壁を上げるが、Trust Boundaryを維持し、責任の所在を明確にする。これはMCPistのターゲットユーザー（パワーユーザー）にとって受け入れ可能なトレードオフである。

> A quiet game changer for people who already know too much.

「know too much」な人々は、トークン管理の重要性を理解している。彼らにとって、これは負担ではなく、当然の前提である。

### 参考情報
## 公式・一次情報

- [Model Context Protocol Specification 2025-11-25](https://modelcontextprotocol.io/specification/2025-11-25) - 公式仕様

## 解説記事（推奨）

- [Client Registration and Enterprise Management in the November 2025 MCP Authorization Spec](https://aaronparecki.com/2025/11/25/1/mcp-authorization-spec-update) - Aaron Parecki（OAuth専門家）による解説。**Enterprise-Managed Authorization（Cross App Access / XAA）** の詳細
    
- [MCP 2025-11-25 is here: async Tasks, better OAuth, extensions](https://workos.com/blog/mcp-2025-11-25-spec-update) - WorkOSによる解説
    

## 主なエンタープライズ機能

|機能|説明|
|---|---|
|**Cross App Access (XAA)**|エンタープライズIdPポリシーでMCP OAuth フローを制御（SEP-990）|
|**Client ID Metadata Documents**|DCR不要のクライアント登録方式|
|**OAuth client-credentials**|M2M（Machine-to-Machine）認証（SEP-1046）|

特にAaron Pareckiの記事が最も詳しく、「Shadow IT問題」の解決方法を説明しています。