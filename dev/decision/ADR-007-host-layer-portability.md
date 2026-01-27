---
title: ADR-007 Host層を含めた完全ポータビリティの実現
aliases:
  - ADR-007
  - host-layer-portability
tags:
  - MCPist
  - ADR
  - architecture
  - portability
document-type:
  - ADR
document-class: ADR
created: 2026-01-12T00:00:00+09:00
updated: 2026-01-14T00:00:00+09:00
---
# ADR-007: Host層を含めた完全ポータビリティの実現

## ステータス

提案（将来ビジョン）

## コンテキスト

MCPist v1（現在開発中のクラウド版）は、MCPサーバー層のポータビリティを実現している:

```
【現在の構成】
Claude Desktop/Cursor (Host固定)
    ↓
MCPサーバー (MCPist) ← ポータブル（クラウド/ローカル移植可能）
    ↓
Token Broker ← ポータブル（Supabase/SQLite切り替え可能）
    ↓
外部API群
```

しかし、既存のMCPクライアント（Claude Desktop, Cursor等）には以下の制約がある:

### 既存MCPクライアントの制約

| 項目 | Claude Desktop | Cursor | 制約の本質 |
|-----|---------------|--------|-----------|
| LLM | Claudeのみ | Claude/GPT | **Host固定 → ベンダーロックイン** |
| オフライン動作 | 不可 | 不可 | **クラウドLLM必須 → プライバシー/コスト問題** |
| LLM切り替え | 不可 | 限定的 | **柔軟性なし** |
| 認証管理 | クライアント側 | クライアント側 | **MCPサーバーと認証が分離 → ポータビリティ低** |

### 根本的な問題

MCP仕様は3層アーキテクチャを定義:

```
Host (LLM) → Client (MCP Client) → Server (MCP Server)
```

しかし、**Host層がベンダー固定**されているため、MCPサーバーをポータブルにしても真の独立性は得られない。

### MCPistの将来ビジョン

MCPist v2では、Host層まで含めた**完全ポータビリティ**を実現する:

```
【将来の構成: MCPist Desktop】
┌─────────────────────────────────────┐
│  MCPist Desktop                     │
│  ┌─────────┐    ┌────────────────┐  │
│  │ 軽量UI  │    │ MCPクライアント │  │
│  └────┬────┘    └───────┬────────┘  │
│       │                 │           │
│       │    ┌────────────┘           │
│       ▼    ▼                        │
│  ┌──────────────┐                   │
│  │ LLM選択      │                   │
│  │ ・Ollama     │                   │
│  │ ・Claude API │                   │
│  │ ・GPT API    │                   │
│  │ ・Gemini API │                   │
│  └──────────────┘                   │
└─────────────────────────────────────┘
            │
            ▼
     MCPサーバー（Go）
            │
            ▼
     Token Broker (SQLite) → 外部API
```

これにより以下を実現:

| 機能 | 状態 |
|-----|-----|
| LLM選択の自由 | ✅ |
| 完全オフライン（Ollama） | ✅ |
| クラウドLLM切り替え | ✅ |
| 認証ポータビリティ | ✅ |
| ベンダーロックインゼロ | ✅ |

## 検討した選択肢

### 選択肢1: 既存MCPクライアント依存（Claude Desktop/Cursor）

現在のMCPエコシステムに乗る。

**メリット:**
- 開発コストが低い（サーバーのみ実装）
- 既存のMCPクライアント資産を活用

**デメリット:**
- LLMがベンダー固定（Claude/GPT限定）
- オフライン動作不可
- 認証がクライアント側で管理される（Token Brokerの優位性が限定的）
- ユーザーの選択肢が制限される

### 選択肢2: Host層を含めた独自クライアント（MCPist Desktop）（採用）

Host層（LLM選択）まで含めた完全なエコシステムを構築。

**メリット:**
- **LLM選択の自由**: Ollama（ローカル）、Claude API、GPT API、Gemini API等を選択可能
- **完全オフライン動作**: Ollamaを使用すれば完全にオフラインで動作
- **プライバシー保護**: ローカルLLM使用時、外部にデータ送信不要
- **コスト最適化**: タスクに応じて安価なLLMに切り替え可能
- **真のベンダーロック回避**: どのベンダーにも依存しない
- **認証の一元化**: Token BrokerがローカルSQLiteで全て管理

**デメリット:**
- 開発コストが高い（Host層 + Client層 + Server層の全実装が必要）
- MCP仕様への完全準拠が必要
- UIの実装が必要

## 決定

**選択肢2（MCPist Desktop）を将来ビジョンとして採用**

ただし、**段階的リリース戦略**を取る:

### Phase 1: クラウド版MCPサーバー（v1） - 現在開発中

```
Claude Desktop/Cursor
    ↓
MCPサーバー (Koyeb)
    ↓
Token Broker (Supabase)
    ↓
外部API群
```

**目的:**
- MCPサーバー層の実装とテスト
- モジュール中心アーキテクチャの検証
- Token Brokerによる認証統合の検証
- 実ユーザーでのフィードバック収集

**制約:**
- Host固定（Claude Desktop/Cursor依存）
- クラウドホスティング必須
- オフライン動作不可

### Phase 2: MCPist Desktop（v2） - 将来実装

```
MCPist Desktop (Electron/Tauri)
├─ LLM選択UI
├─ MCPクライアント
├─ LLM抽象化層
│   ├─ Ollama
│   ├─ Claude API
│   ├─ GPT API
│   └─ Gemini API
└─ MCPサーバー（埋め込み）
    └─ Token Broker (SQLite)
```

**目的:**
- Host層のポータビリティ実現
- 完全オフライン動作
- LLM選択の自由
- ベンダーロック完全回避

## 根拠

### 1. 段階的リリースの必然性

**Phase 1が先行すべき理由:**

1. **技術的リスクの分離**
   - MCPサーバー実装の検証
   - モジュールアーキテクチャの実戦投入
   - Token Broker統合のテスト

2. **市場検証**
   - 実ユーザーからのフィードバック
   - どのモジュールが実際に使われるか
   - パフォーマンス/コストの測定

3. **開発リソースの最適化**
   - MCPist DesktopはElectron/Tauri + LLM統合で大規模
   - Phase 1で得た知見をPhase 2設計に反映

**Phase 1の価値:**
- Claude Desktop/Cursor依存でも、**メタツール方式**によるContext Rot解決は有効
- Token Brokerによる**統一認証**は既存クライアントでも価値がある
- ADR-005で設計した**RLS非依存**により、Phase 2への移植が容易

### 2. MCPist Desktopの優位性

**既存MCPクライアントとの比較:**

| 項目 | Claude Desktop | Cursor | MCPist Desktop |
|-----|---------------|--------|----------------|
| LLM | Claudeのみ | Claude/GPT | **任意（Ollama含む）** |
| オフライン | 不可 | 不可 | **可能（Ollama使用時）** |
| 認証 | クライアント側 | クライアント側 | **サーバー側（Token Broker）** |
| ポータビリティ | なし | なし | **完全** |
| コスト | 固定（Claudeサブスク） | 固定 | **最適化可能** |

### 3. ADR群との整合性

MCPist Desktopは、既存ADRの自然な帰結:

```
ADR-003（メタツール）
    ↓ Context Rot解決 → LLM推論品質向上
ADR-005（RLS非依存）
    ↓ SQLite対応設計 → ローカルアプリ化可能
ADR-006（モジュール中心）
    ↓ 認証統合 → Token Brokerの価値最大化
        ↓
    【必然的な結論】
    Host層も含めて統合すれば、完全ポータビリティ実現
```

### 4. 技術的実現可能性

**Phase 2の技術スタック候補:**

| レイヤー | 技術選択肢 | 備考 |
|---------|----------|------|
| デスクトップ | Electron / Tauri | Tauriの方が軽量 |
| UI | React / Svelte | Svelte + Tauriで高速化 |
| MCPクライアント | 自前実装（Go移植） | 既存MCPサーバーコード流用 |
| LLM抽象化 | langchaingo / 自前 | Ollama, Claude, GPT, Gemini対応 |
| MCPサーバー | 既存Go実装を埋め込み | Phase 1コードほぼそのまま |
| Token Broker | SQLite | ADR-005で既に設計済み |

## 影響

### Phase 1（v1: クラウド版）への影響

**変更なし** - 現在の開発計画通り進行。

以下を優先実装:
1. MCPプロトコルハンドラ（Tools/Resources/Prompts）
2. モジュール中心アーキテクチャ（ADR-006）
3. Token Broker統合（Supabase Vault）
4. メタツール実装（ADR-003）

### Phase 2（v2: MCPist Desktop）への影響

**新規実装が必要:**

1. **LLM抽象化層**
   ```go
   type LLMProvider interface {
       Initialize(config Config) error
       Chat(messages []Message) (Response, error)
       StreamChat(messages []Message) (<-chan Token, error)
   }

   // 実装
   - OllamaProvider
   - ClaudeProvider
   - GPTProvider
   - GeminiProvider
   ```

2. **MCPクライアント実装**
   - JSON-RPC 2.0クライアント
   - SSE接続管理
   - 埋め込みMCPサーバーとの通信

3. **デスクトップUI**
   - LLM選択画面
   - チャットUI
   - Token Broker設定画面
   - モジュール管理画面

4. **Token BrokerのSQLite版**
   - ADR-005で設計済み
   - Supabase版のロジックを移植

### ドキュメントへの影響

1. **spec-sys.md**: Phase 1（現在）とPhase 2（将来）の2部構成に
2. **README.md**: MCPist Desktopのビジョンを追記
3. **ロードマップ作成**: Phase 1 → Phase 2の移行計画

### コミュニティへの影響

**Phase 1リリース時のメッセージング:**
- 「MCPist v1はClaude Desktop/Cursor対応のMCPサーバー」
- 「v2ではHost層を含めた完全ポータビリティを実現予定」
- 「現在の実装は将来のローカルアプリ版への布石」

これにより:
- Phase 1の制約を明示しつつ、将来性をアピール
- 早期ユーザーの期待値調整
- Phase 2への期待形成

## 実装計画

### Phase 1: クラウド版（2025 Q1-Q2）

- [ ] MCPプロトコルハンドラ完成（Tools/Resources/Prompts）
- [ ] モジュール中心アーキテクチャ実装（ADR-006）
- [ ] Token Broker統合（Supabase）
- [ ] 8モジュール実装完了
- [ ] Claude Desktop/Cursorでの動作確認
- [ ] v1.0.0リリース

### Phase 2: MCPist Desktop（2025 Q3-Q4）

- [ ] 技術スタック決定（Tauri vs Electron）
- [ ] LLM抽象化層設計・実装
- [ ] Ollama統合
- [ ] MCPクライアント実装
- [ ] Token BrokerのSQLite移植
- [ ] デスクトップUI実装
- [ ] クロスプラットフォームビルド（Windows/Mac/Linux）
- [ ] v2.0.0リリース

## 参照

- [MCP Architecture - Official Documentation](https://modelcontextprotocol.io/docs/concepts/architecture)
- [spec-sys.md § 1.1 システム構成図](../spec-sys.md)
- [ADR-003: メタツール + 選択的スキーマ取得パターンの採用](../DAY2/ADR-003-meta-tool-lazy-loading.md)
- [ADR-005: RLSに依存しない認可設計](./ADR-005-no-rls-dependency.md)
- [ADR-006: モジュール中心アーキテクチャによる3プリミティブ統合](./ADR-006-module-centric-primitives.md)
- [Ollama Documentation](https://ollama.ai/docs)
- [Tauri Documentation](https://tauri.app/)
