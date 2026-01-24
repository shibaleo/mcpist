---
title: MCPist コンセプトペーパー
aliases:
  - concept-paper
  - day0-concept
tags:
  - MCPist
  - concept
  - DAY0
document-type: concept
document-class: foundation
created: 2026-01-09T00:00:00+09:00
updated: 2026-01-14T00:00:00+09:00
---
# MCPist コンセプトペーパー

## ドキュメント管理情報

| 項目 | 値 |
|------|-----|
| Status | `current` |
| Version | v1.0 (DAY5補完) |
| Note | DAY0時点で書かれるべきだったコンセプト（DAY5で復元） |

---

## 背景

### 研究課題

MCPistは、より深い研究課題から生まれた：

> **認知資源の管理、通知管理、注意資源の管理**

### 問題認識: Information Flood → Context Rot

```
【歴史的連続性】

1970s: Information Flood（情報洪水）の認識
  ↓ 個人の解決策として
2000s: PKM（Personal Knowledge Management）の台頭
  ↓ LLMの登場で新たな次元が加わる
2020s: Context Rot（LLMの認知に対する情報洪水）
  ↓ 個人の解決策として
2025-: PCM（Personal Context Management）の必要性
```

**Information FloodはLLMにとってのContext Rotと同型の問題である。**

- Information Flood = 人間の認知に対する情報過負荷
- Context Rot = LLMの認知に対する情報過負荷

---

## ミッション

> **A quiet game changer for people who already know too much.**

MCPistは：
- 自前でOAuthトークンを管理できるパワーユーザーが
- 自分の複数アカウントを単一のMCPサーバー経由で
- 並列的に利用するためのセルフホスト型ツール

---

## コア仮説

### 仮説1: メタツールによるContext Rot防止

```
【従来のMCP】
initialize → tools/list → 全100ツール公開
→ LLMが100ツールから選択を推論（コンテキスト消費大）

【MCPist】
initialize → tools/list → メタツール3個のみ公開
get_module_schema("notion") → 該当アカウントの15ツールのみ返却
→ LLMは事前フィルター済みのツールから選択
```

**検証方法:** プロトタイプで実際にClaude Codeから呼び出し、コンテキスト消費を比較

### 仮説2: シングルテナント設計によるセキュリティ

```
シングルテナント・シングルユーザー・マルチアカウント

テナント: 自分（1）
  └─ ユーザー: 自分（1）
       └─ アカウント1（Notion個人）
       └─ アカウント2（GitHub仕事用）
       └─ アカウント3（Jira副業用）
```

**根拠:** Token Exchangerパターンへの批判（Trust Boundary崩壊回避）

**【DAY5更新】再評価によりコアから除外:**
MCPistの存在意義が進化（パワーユーザー向け開発ツール → 社内AIインフラのデザインパターン提示）。
会社が自社社員のトークンを預かるSaaSモデルという新たな視点が生まれ、テナンシーモデルは「実装選択」であり「コア機能」ではないと判断。

### 仮説3: 決定論的オーケストレーター

MCPistは「判断」しない：
- ツール選択の判断 → ユーザーLLM
- 結果の解釈 → ユーザーLLM
- MCPistは指定された通りに実行するだけ

**根拠:** Designative Liability（設計的責任）の回避

---

## 既存プロジェクトとの関係

| 項目 | 既存 (dwhbi) | 新規 (go-mcp-dev) |
|------|-------------|----------------------|
| ホスティング | Vercel | Koyeb |
| 言語 | TypeScript (Next.js) | Go |
| 認証 | MCPトークン + OAuth + Service Role | 固定シークレット |
| Supabase | Data API（ユーザー別Vault） | Management API（管理操作） |
| 用途 | マルチユーザーSaaS | 個人用 / シングルテナント |

**既存プロジェクトは維持:** MCPトークン管理UI、Vault管理、コンソール機能はそのまま稼働。
新プロジェクトはシングルテナント用途の軽量版として並行運用。

---

## 成功の定義

1. **技術的成功:** Claude CodeからMCPサーバー経由で外部APIを操作できる
2. **Context Rot対策:** メタツールパターンでコンテキスト消費を抑制できる
3. **運用的成功:** $0/月で安定稼働できる
4. **設計的成功:** シングルテナント設計でTrust Boundary維持

---

## 関連ドキュメント

- [prototype-review.md](./prototype-review.md) - プロトタイプ判断根拠・リスク検証
- [go-mcp-dev-plan.md](../DAY1/go-mcp-dev-plan.md) - 実装計画
- [MCPist-Design-Philosophy.md](../DAY2/MCPist-Design-Philosophy.md) - 設計哲学
- [dtl-core.md](../DAY5/dtl-core.md) - コア機能定義
