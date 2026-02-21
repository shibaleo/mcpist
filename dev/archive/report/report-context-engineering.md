# Context Engineering リサーチ（2025-01-07）

## 概要

MCPを「ツール呼び出し」ではなく「コンテキスト注入のインフラ」として活用し、プロンプトエンジニアリングを代替する方向性についてのリサーチ。

## 主要な出典

### 1. Context Engineering Is Replacing Prompt Engineering

URL: [https://tao-hpu.medium.com/context-engineering-is-replacing-prompt-engineering-for-production-ai-02205fad2a7f](https://tao-hpu.medium.com/context-engineering-is-replacing-prompt-engineering-for-production-ai-02205fad2a7f)

要点: 「2025年のAIエージェント成功の最大の予測因子はモデル選択ではなくコンテキストエンジニアリング」。MCPを「AIのUSB-C」と位置づけ、Tools/Resources/Promptsの3プリミティブでコンテキストを構造化。

### 2. The Only Guide You Will Ever Need For MCP

URL: [https://www.analyticsvidhya.com/blog/2025/07/model-context-protocol-mcp-guide/](https://www.analyticsvidhya.com/blog/2025/07/model-context-protocol-mcp-guide/)

要点: 従来のプロンプトエンジニアリングは全情報を単一プロンプトに詰め込んでいたが、MCPはコンテキストをモジュール化。LLMは受動的にテキストを受け取るのではなく、標準プロトコルで能動的にデータを要求できる。

### 3. MCP公式ドキュメント - Prompts

URL: [https://modelcontextprotocol.io/docs/concepts/prompts](https://modelcontextprotocol.io/docs/concepts/prompts)

要点: 再利用可能なプロンプトテンプレートとワークフローの作成方法。Resources経由でのコンテキスト注入パターン。

### 4. Wikipedia - Model Context Protocol

URL: [https://en.wikipedia.org/wiki/Model_Context_Protocol](https://en.wikipedia.org/wiki/Model_Context_Protocol)

要点: 2024年11月Anthropicがリリース、2025年12月にLinux Foundationへ寄贈。OpenAI、Google DeepMind、Microsoftが採用。

### 5. MCP: Model Context Pitfalls (HiddenLayer)

URL: [https://hiddenlayer.com/innovation-hub/mcp-model-context-pitfalls-in-an-agentic-world/](https://hiddenlayer.com/innovation-hub/mcp-model-context-pitfalls-in-an-agentic-world/)

要点: MCPのセキュリティリスク分析。Indirect prompt injection、tool poisoning、lookalike toolsによる攻撃ベクトル。

### 6. Microsoft - Protecting against prompt injection in MCP

URL: [https://developer.microsoft.com/blog/protecting-against-indirect-injection-attacks-mcp](https://developer.microsoft.com/blog/protecting-against-indirect-injection-attacks-mcp)

要点: MicrosoftによるMCPセキュリティガイドライン。AI prompt shieldsの実装とサプライチェーンセキュリティの推奨。

---

## MCPistへの示唆

- 「プロンプトエンジニアを雇う代わりにMCPを設定する」という世界観
- LIFETRACERの個人データをコンテキストソースとして活用
- モデル横断（Claude/GPT/Gemini/ローカルLLM）で同じパーソナライズ
- セキュリティ考慮：tool poisoning対策が必要

---

## 追加リサーチ（セキュリティアーキテクチャ）

### 7. SAFE-MCP: A Security Framework for AI+MCP

URL: [https://development.tldrecap.tech/posts/2025/opensource-securitycon-na/safe-mcp-secure-agentic-systems-llm-risks/](https://development.tldrecap.tech/posts/2025/opensource-securitycon-na/safe-mcp-secure-agentic-systems-llm-risks/)

要点: MITRE ATT&CKの手法をLLM/MCP向けに適応したセキュリティフレームワーク。信頼境界をNetwork、File System、Secrets、Inter-Server Calls、Tenancyの層別に整理。Tool Poisoning、OAuth Confused Deputy、MCP Rug Pullなど具体的な攻撃ベクトルを分類。実装ガイドラインとして、ツールの署名・Allowlist・Schema Validation・Provenance Trackingを推奨。

### 8. Design Principles for LLM-based Systems with Zero Trust（BSI/ANSSI共同）

URL: [https://www.bsi.bund.de/SharedDocs/Downloads/EN/BSI/Publications/ANSSI-BSI-joint-releases/LLM-based_Systems_Zero_Trust.pdf](https://www.bsi.bund.de/SharedDocs/Downloads/EN/BSI/Publications/ANSSI-BSI-joint-releases/LLM-based_Systems_Zero_Trust.pdf)

要点: ドイツBSIとフランスANSSIによる共同発表。Zero Trust原則（Never trust, always verify）をLLMシステムに適用。入力ソースごとの検証、最小権限の原則、データ感度レベル別の分離を推奨。Trust Algorithmによる重み付け判定（ユーザー履歴、デバイス、時間等のコンテキスト考慮）。

### 9. Contextual Integrity Verification (CIV)

URL: [https://arxiv.org/abs/2508.09288](https://arxiv.org/abs/2508.09288)

要点: 2025年8月発表の論文。各トークンに暗号署名付き信頼度ラベル（SYSTEM/USER/TOOL/DOC/WEB）を付与し、transformerのattention maskで信頼境界を強制。低信頼トークンが高信頼表現に影響できない「非干渉保証」を数学的に実現。攻撃成功率0%、出力類似性93.1%維持。Llama-3-8BとMistral-7Bで検証済み。

---

## 追加リサーチ（セマンティックレイヤー）

MCPにおける「ツールレイヤー」と「セマンティックレイヤー」の区別に関する議論。主にBI/データ分析領域で進んでいる。

### 10. Cube Blog - Unlocking Universal Data Access for AI with MCP

URL: [https://cube.dev/blog/unlocking-universal-data-access-for-ai-with-anthropics-model-context](https://cube.dev/blog/unlocking-universal-data-access-for-ai-with-anthropics-model-context)

要点: セマンティックレイヤーが「what」と「how」（ビジネスデータの意味と構造）を提供し、MCPが「access」（AIエージェントの標準化されたアクセス機構）を提供するという役割分担。MCPツールがセマンティックレイヤーのAPIと直接対話し、生SQLではなくビジネス用語でデータを要求する設計を提案。

### 11. Boring Semantic Layer + MCP = 🔥

URL: [https://juhache.substack.com/p/boring-semantic-layer-mcp](https://juhache.substack.com/p/boring-semantic-layer-mcp)

要点: 「SQLの柔軟性を信頼性と交換する」というトレードオフを明確化。生テーブルをLLMに公開する代わりに、事前定義された集計を公開。LLMは意図に集中でき、SQLの正確性を気にしなくてよくなる。MCPSemanticModelクラスでlist_models, get_model, get_time_range, query_modelの4ツールを提供。

### 12. TimeXtender - MCP Servers and the Semantic Layer Gap

URL: [https://www.timextender.com/blog/product-technology/mcp-servers-and-the-semantic-layer-gap-what-data-teams-need-to-know](https://www.timextender.com/blog/product-technology/mcp-servers-and-the-semantic-layer-gap-what-data-teams-need-to-know)

要点: 「セマンティックレイヤーがMCPを技術的な好奇心から本番環境対応のエンタープライズツールに変える」。エージェントが生SQLを生成する代わりに、セマンティックレイヤーがガバナンスされた定義を公開する構造。

### 13. AtScale - Model Context Protocol (MCP)

URL: [https://www.atscale.com/glossary/model-context-protocol-mcp/](https://www.atscale.com/glossary/model-context-protocol-mcp/)

要点: MCPはメタデータ（テーブル定義、メジャー、ディメンション、階層、説明、ユーザー権限）の交換をサポート。LLMがスキーマを発見し、セマンティック精度でユーザーの質問を解釈可能に。セマンティックレイヤーとMCPが連携し、BIツールと同等のガバナンスをAIエージェントに拡張。

---

## MCPistへの追加示唆（セマンティックレイヤー）

- MCPist = ツールレイヤー（API接続の抽象化）、PKMist構想 = セマンティックレイヤー（knowledge, schedule, financeなど意味単位でデータ提供）
- 既存議論はBI/データ分析文脈が中心。PKM×財務税務という「個人の生活・事業活動全体の抽象化」は未開拓領域
- Boring Semantic Layerのアプローチ（事前定義された集計でLLMの自由度を制限し信頼性を向上）は参考になる
- ツールが何であるか（Notion/Google Calendar）を隠蔽し、LLMは「私のスケジュール」「私の財務状況」と言えばよい設計

---

## 追加リサーチ（宣言的インターフェース / Context Rot）

MCPを「ツール呼び出し」ではなく「意味呼び出し」にすべきという議論。LLMは対話に集中し、データ取得のAPI選択はバックエンドに任せるべき。

### 14. Goal-Oriented Interface (GOI): A Case for Declarative LLM-friendly Interfaces

URL: [https://arxiv.org/html/2510.04607v1](https://arxiv.org/html/2510.04607v1)

要点: 「policy（機能オーケストレーション）とmechanism（操作の実行）の分離」を提唱。従来のGUI/ツール操作ではポリシーとメカニズムが密結合しており、LLMは「何をしたいか」と「どう操作するか」の両方を担当させられている。GOIはこれを分離し、LLMは「望む結果を宣言するだけ」でよく、具体的なアクションの発行はバックエンドが担当。例：LLMが「スクロールバーを50%に設定」と宣言するだけで、インターフェースが低レベル実行を処理。「状態ベースの観察-行動ループ」から「目標状態の設定」へのシフト。

### 15. Context Rot問題と数値データ

URL: [https://writer.com/engineering/rag-mcp/](https://writer.com/engineering/rag-mcp/)

URL: [https://demiliani.com/2025/09/04/model-context-protocol-and-the-too-many-tools-problem/](https://demiliani.com/2025/09/04/model-context-protocol-and-the-too-many-tools-problem/)

URL: [https://www.arsturn.com/blog/why-your-llm-is-ignoring-your-mcp-tools-and-how-to-fix-it](https://www.arsturn.com/blog/why-your-llm-is-ignoring-your-mcp-tools-and-how-to-fix-it)

要点: ツール定義をコンテキストに詰め込みすぎると「context rot」が発生し、モデルの推論が劣化する。具体的な数値データ：

- GitHub Copilot: 128ツールのハードリミットを設定（context window汚染防止のため）
- 経験則: 40ツールを超えるとパフォーマンス低下開始、60ツールを超えると「崖のように落ちる」
- RAG-MCP手法: ツール選択精度が3倍以上向上（43.13% vs 13.62%）、プロンプトトークン50%以上削減

---

## MCPistへの追加示唆（宣言的インターフェース）

- GOI論文の「policy-mechanism分離」はMCPistの設計思想と合致：LLMは「私のスケジュール」と宣言するだけ、どのAPIを叩くかはバックエンドが決定
- description拡充はcontext rot（コンテキスト汚染）を招く。意味をワンショットで伝え、データ取得はバックエンドに委譲すべき
- チャットLLMは対話に集中させ、API接続の責務を負わせない設計が理想
- 現状のMCPエコシステムはこの問題を認識しつつも、RAG-MCPなど「ツール選択の改善」に注力。「そもそもツール選択をLLMにさせない」という根本解決の議論は少ない