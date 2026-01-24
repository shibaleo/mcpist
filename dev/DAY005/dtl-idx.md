---
title: MCPist コア・サブコア索引
aliases:
  - dtl-idx
  - core-sub-core-index
tags:
  - MCPist
  - architecture
  - index
  - DTL
document-type: detail
document-class: DTL
created: 2026-01-14T00:00:00+09:00
updated: 2026-01-14T00:00:00+09:00
---
# MCPist コア・サブコア索引

## ドキュメント管理情報

| 項目 | 値 |
|------|-----|
| Status | `current` |
| Version | v1.0 (DAY5) |
| Note | コア9件 + 暫定コア4件 + サブコア34件 |

---

## コア機能（COR）

コア機能を変更すると複数の設計判断が破綻する。

### Tier 1: 強固（5つの理由）

| ID | コア機能 | 説明 |
|----|----------|------|
| COR-007 | Go採用 | $0制約・SSE直接実装・差別化・並行処理・サーバー側計算の5理由でGoを採用 |

### Tier 2: 強固（3つの理由）

| ID | コア機能 | 説明 |
|----|----------|------|
| COR-001 | メタツール方式 | Context Rot防止・セキュリティ・MCP制約回避のため3メタツールで84ツールを制御 |
| COR-003 | サーバー側認証設計 | クライアント非依存・ポータビリティ・Electron対応のため認証をMCPサーバー側に持つ |
| COR-004 | 決定論的オーケストレーター | 品質管理・コスト・障害点削減のためMCPistは判断せずルーティングに徹する |
| COR-006 | Next.js採用 | $0制約・SPA要件・Electron対応の3理由で管理UIにNext.jsを採用 |

### Tier 3: 強固（2つの理由）

| ID | コア機能 | 説明 |
|----|----------|------|
| COR-005 | RLS非依存認可 | ポータビリティ・多層防御のためToken Brokerが主責務、RLSは補助層 |
| COR-008 | TOON形式 | Context Rot対策・ベンダー動機欠如への対抗としてトークン最適化出力形式を採用 |
| COR-009 | モジュールCLI実装 | レジストリ平準化・人間/LLM等価性のため各モジュールをGo CLIとして実装 |

### Tier 4: 暫定（1つの理由）

| ID | コア機能 | 説明 |
|----|----------|------|
| COR-101 | $0/月コスト制約 | 個人開発ツールに課金したくないという意向からの制約 |
| COR-102 | Koyeb選択 | $0制約からの導出でMCPサーバーホスティングにKoyeb Free Tierを選択 |
| COR-103 | Vercel選択 | $0制約からの導出で管理UIホスティングにVercel Hobbyを選択 |
| COR-104 | コールドスタート対策 | Koyeb Free Tierのスリープ回避のためGitHub Actionsで定期ping |

---

## サブコア要件

コア機能を前提とした場合に、2つ以上の独立した根拠を持つ要件。

### 要件仕様（REQ-COR）

| ID | サブコア | 説明 | 前提コア |
|----|----------|------|----------|
| REQ-COR-001 | シングルユーザー・マルチアカウント | 1インスタンス=1ユーザー=N外部アカウントの設計 | COR-003, COR-004 |
| REQ-COR-002 | スコープ外事項の明確化 | 非決定論的処理・マルチテナント等を明示的にスコープ外と定義 | COR-004, COR-001 |
| REQ-COR-003 | メタツールによるコンテキスト制御 | メタツール3個+TOON形式でContext Rot防止 | COR-001, COR-008 |
| REQ-COR-004 | 外部サービス連携のモジュール化 | 各外部サービス連携をGoモジュールとして独立実装 | COR-009, COR-007 |
| REQ-COR-005 | 認証・認可の一元管理 | Token BrokerでJWT検証・トークン取得を一元化 | COR-003, COR-005 |

### システム仕様（SYS-COR）

| ID | サブコア | 説明 | 前提コア |
|----|----------|------|----------|
| SYS-COR-001 | 認証3層構造 | ユーザー認証→ツールマスク→サービス認証の3層で多層防御 | COR-003, COR-005 |
| SYS-COR-002 | Tool Sieve | 権限外ツールは存在すら見せないセキュリティ設計 | COR-001, COR-003 |
| SYS-COR-003 | Token Broker設計 | Edge Function内でuser_idフィルタ、共有/個人トークン解決 | COR-003, COR-005 |
| SYS-COR-004 | モジュールレジストリ統合 | 3プリミティブをモジュールに統合、独立層として実装しない | COR-009, COR-007, COR-001 |
| SYS-COR-005 | ステートレス設計 | MCPサーバーは状態を持たず決定論的処理のみ実行 | COR-004, COR-007 |
| SYS-COR-006 | 認証ポイント分離 | MCPサーバー認証と外部サービス認証を明確に分離 | COR-003, COR-005 |

### インフラ仕様（INF-COR）

| ID | サブコア | 説明 | 前提コア |
|----|----------|------|----------|
| INF-COR-001 | MCPサーバーDocker軽量化 | alpine+静的バイナリでKoyeb Free Tier制限に対応 | COR-007, COR-101, COR-102 |
| INF-COR-002 | Supabase統合アーキテクチャ | Auth/DB/Vault/Edge Functionsを単一プロジェクトに統合 | COR-003, COR-005, COR-101 |
| INF-COR-003 | Next.js + Vercel構成 | App Router+@supabase/ssrでSPA管理UI実装 | COR-006, COR-101 |
| INF-COR-004 | MCPサーバー↔Edge Function通信 | JWKSで検証、MCPサーバーはEdge Function経由のみ | COR-003, COR-007 |

### 設計仕様（DSN-COR）

| ID | サブコア | 説明 | 前提コア |
|----|----------|------|----------|
| DSN-COR-001 | TOON形式統一レスポンス | 全ツールがTOON形式で応答、JSON比30-40%トークン削減 | COR-008, COR-001, COR-004 |
| DSN-COR-002 | メタツール設計 | get_module_schema/call/batchの3メタツールでツール制御 | COR-001, COR-008, COR-007 |
| DSN-COR-003 | Token Brokerデータモデル | oauth_tokensテーブルでrole_id+user_idベースのトークン管理 | COR-003, COR-005 |
| DSN-COR-004 | Tool Sieveデータモデル | users/roles/role_permissionsでロールベース権限管理 | COR-001, COR-003 |
| DSN-COR-005 | 管理UI URL設計 | query parameterでモーダル状態管理するSPA設計 | COR-006, COR-003 |
| DSN-COR-006 | モジュールインターフェース | Name/Description/Tools/Executeの統一インターフェース | COR-009, COR-007 |

### テスト仕様（TST-COR）

| ID | サブコア | 説明 | 前提コア |
|----|----------|------|----------|
| TST-COR-001 | TOON/JSONL単体テスト | パーサー・変数解決・フォーマッターの高カバレッジテスト | COR-008, COR-001, COR-007 |
| TST-COR-002 | Token Broker結合テスト | トークン取得・リフレッシュ・失効の結合テスト | COR-003, COR-005 |
| TST-COR-003 | Tool Sieve結合テスト | 権限フィルタリング・整合性検証の結合テスト | COR-001, COR-003 |
| TST-COR-004 | JSONL並列実行テスト | goroutineによる並列実行・依存解決のテスト | COR-001, COR-007 |
| TST-COR-005 | 認証セキュリティテスト | JWT検証・権限昇格防止・IDOR防止のテスト | COR-003, COR-005 |
| TST-COR-006 | 管理UI E2Eテスト | Playwrightによるadmin/user権限分離E2Eテスト | COR-006, COR-003 |

### 運用仕様（OPS-COR）

| ID | サブコア | 説明 | 前提コア |
|----|----------|------|----------|
| OPS-COR-001 | 放置運用原則 | 自動リトライ・自動リフレッシュで日常的手動介入不要 | COR-003, COR-004, COR-101 |
| OPS-COR-002 | OAuthアプリ登録・トークン管理 | プロバイダ別OAuth設定と共有/個人トークン管理フロー | COR-003, COR-005 |
| OPS-COR-003 | 障害検知・アラート設計 | Grafana Cloud無料枠でCritical/Warning分類アラート | COR-003, COR-101 |
| OPS-COR-004 | セキュリティ運用 | シークレットローテーション・漏洩時対応手順 | COR-003, COR-005 |

### マニュアル仕様（MNL-COR）

| ID | サブコア | 説明 | 前提コア |
|----|----------|------|----------|
| MNL-COR-001 | マニュアル3分類体制 | admin/user/dev向けに3種類のマニュアルを整備 | COR-003, COR-009 |
| MNL-COR-002 | 初期セットアップチェックリスト | Supabase/Koyeb/Vercel/Grafanaの無料枠構成手順 | COR-003, COR-101 |
| MNL-COR-003 | モジュール追加手順 | 統一インターフェース実装とテスト作成手順 | COR-009, COR-007 |

---

## 統計

| カテゴリ | 件数 |
|----------|------|
| 強固なコア（Tier 1-3） | 9件 |
| 暫定的コア（Tier 4） | 4件 |
| サブコア要件 | 34件 |
| **合計** | **47件** |

---

## 関連ドキュメント

- [dtl-core.md](dtl-core.md) - コア機能定義（詳細）
- [dtl-req-cor.md](dtl-req-cor.md) - 要件仕様サブコア
- [dtl-sys-cor.md](dtl-sys-cor.md) - システム仕様サブコア
- [dtl-inf-cor.md](dtl-inf-cor.md) - インフラ仕様サブコア
- [dtl-dsn-cor.md](dtl-dsn-cor.md) - 設計仕様サブコア
- [dtl-tst-cor.md](dtl-tst-cor.md) - テスト仕様サブコア
- [dtl-ops-cor.md](dtl-ops-cor.md) - 運用仕様サブコア
- [dtl-mnl-cor.md](dtl-mnl-cor.md) - マニュアル仕様サブコア
