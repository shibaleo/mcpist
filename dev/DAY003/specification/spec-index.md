---
title: MCPist 仕様書インデックス（spec-index）
aliases:
  - spec-index
  - MCPist-specification-index
tags:
  - MCPist
  - specification
  - index
document-type:
  - specification
document-class: specification
created: 2026-01-12T00:00:00+09:00
updated: 2026-01-14T00:00:00+09:00
---
# MCPist 仕様書インデックス

本ドキュメントは、MCPistプロジェクトの全仕様書の索引です。各ドキュメントの役割と参照タイミングを示します。

---

## ドキュメント管理情報

| 項目 | 値 |
|------|-----|
| Status | `current` |
| Version | v1.0 |
| Note | go-mcp-dev版と同一内容（微差あり） |

---

## 仕様書の構成

| ドキュメント | 役割 | 読むタイミング | サイズ |
|-------------|------|--------------|--------|
| [spec-sys.md](spec-sys.md) | システム全体像 | **最初に読む** | ~500行 |
| [spec-dsn.md](spec-dsn.md) | 技術仕様・API設計 | 実装時に参照 | ~460行 |
| [spec-inf.md](spec-inf.md) | インフラ構成・デプロイ | デプロイ時に参照 | ~350行 |
| [spec-ops.md](spec-ops.md) | 運用手順・監視 | 運用開始時に参照 | ~540行 |

**推奨読書順序:**
1. spec-sys.md (全体像把握)
2. spec-dsn.md (技術詳細)
3. spec-inf.md + spec-ops.md (インフラ・運用)

---

## 各Specのカバー範囲

### [spec-sys.md](spec-sys.md) - システム仕様書

**カバー範囲:**
- ✅ システム全体構成図
- ✅ コンポーネント一覧と役割
- ✅ MCPサーバー本体の役割
- ✅ Token Broker (Edge Function)
- ✅ 管理UI (SvelteKit)
- ✅ Vault (Supabase)
- ✅ コンポーネント間の通信フロー
- ✅ 認証・認可の設計方針
- ✅ データフロー

**対象読者:** 開発者全員、アーキテクト、新規参加者

**特徴:** 最も高レベルな視点。コード実装の詳細は含まない。

---

### [spec-dsn.md](spec-dsn.md) - 設計仕様書

**カバー範囲:**
- ✅ MCP Protocol準拠のメタツール設計 (get_module_schema, call, batch)
- ✅ データモデル (users, accounts, oauth_tokens)
- ✅ モジュールインターフェース (Go)
- ✅ エラーコード一覧
- ✅ リトライ戦略
- ✅ セキュリティ設計 (暗号化、危険操作フラグ)
- ✅ 管理UI画面設計・APIエンドポイント

**対象読者:** 実装担当者、テスト担当者

**特徴:** コードレベルの詳細。JSON例は最小限。

---

### [spec-inf.md](spec-inf.md) - インフラ仕様書

**カバー範囲:**
- ✅ Koyeb (MCPサーバーホスティング)
- ✅ Supabase (Vault, Edge Function, 管理UI)
- ✅ Grafana Cloud (ログ集約・監視)
- ✅ 環境変数一覧
- ✅ コスト試算 ($0/月運用)
- ✅ ログ設計 (構造化JSON)
- ✅ デプロイ手順

**対象読者:** DevOps担当者、インフラ管理者

**特徴:** 外部サービス依存関係が明確。

---

### [spec-ops.md](spec-ops.md) - 運用仕様書

**カバー範囲:**
- ✅ 運用原則 (放置運用、無料枠)
- ✅ 初回セットアップ手順
- ✅ OAuth設定ガイド (各サービス別)
- ✅ 可用性目標・KPI (99%可用性、P95<3秒)
- ✅ ログ検索・監視ダッシュボード
- ✅ アラート設定
- ✅ トラブルシューティング
- ✅ Runbook (アラート別対応手順)
- ✅ ポストモーテムテンプレート
- ✅ 運用成熟ロードマップ

**対象読者:** 運用担当者、障害対応者

**特徴:** 実際の運用で必要な手順書。

---

## コンポーネント別参照マップ

「〜について知りたい」ときにどのSpecを読むべきか。

| 知りたいこと | 参照先 |
|------------|--------|
| **システム全体像** | [spec-sys.md § 1](spec-sys.md) |
| **MCPサーバーの役割** | [spec-sys.md § 2.1](spec-sys.md) |
| **メタツール (get_module_schema/call/batch)** | [spec-dsn.md § 2](spec-dsn.md) |
| **Token Brokerの仕組み** | [spec-sys.md § 2.3](spec-sys.md), [ADR-005](adr/ADR-005-no-rls-dependency.md) |
| **管理UIの機能** | [spec-sys.md § 2.4](spec-sys.md), [spec-dsn.md § 6](spec-dsn.md) |
| **Vaultのデータ構造** | [spec-dsn.md § 3](spec-dsn.md) |
| **認証・認可の仕組み** | [spec-sys.md § 4](spec-sys.md), [ADR-005](adr/ADR-005-no-rls-dependency.md) |
| **OAuth設定方法** | [spec-ops.md § 2.3](spec-ops.md) |
| **エラーコード一覧** | [spec-dsn.md § 4](spec-dsn.md) |
| **危険操作フラグ** | [spec-dsn.md § 5.3](spec-dsn.md) |
| **インフラ構成** | [spec-inf.md § 1](spec-inf.md) |
| **デプロイ手順** | [spec-inf.md § 3](spec-inf.md) |
| **ログ検索方法** | [spec-ops.md § 3.3](spec-ops.md) |
| **監視ダッシュボード** | [spec-ops.md § 6](spec-ops.md) |
| **障害対応手順** | [spec-ops.md § 9](spec-ops.md) |
| **可用性目標・KPI** | [spec-ops.md § 1.3](spec-ops.md) |
| **コスト試算** | [spec-inf.md § 2](spec-inf.md) |

---

## 役割別の推奨読書リスト

### 新規参加者 (初めてMCPistに触れる開発者)

1. [spec-sys.md](spec-sys.md) - システム全体像
2. [spec-dsn.md § 2](spec-dsn.md) - メタツール設計
3. [spec-ops.md § 2](spec-ops.md) - 初回セットアップ

### 実装担当者

1. [spec-dsn.md](spec-dsn.md) - 全体
2. [spec-sys.md § 4](spec-sys.md) - 認証フロー
3. [ADR-005](adr/ADR-005-no-rls-dependency.md) - RLS非依存設計

### インフラ・運用担当者

1. [spec-inf.md](spec-inf.md) - 全体
2. [spec-ops.md](spec-ops.md) - 全体
3. [spec-sys.md § 1](spec-sys.md) - システム構成図

### 障害対応者

1. [spec-ops.md § 9](spec-ops.md) - トラブルシューティング・Runbook
2. [spec-ops.md § 3.3](spec-ops.md) - ログ検索
3. [spec-inf.md § 5](spec-inf.md) - 環境変数一覧

---

## カバー範囲チェックリスト

### ✅ 完全にカバーされている領域

- システムアーキテクチャ (spec-sys.md)
- メタツール設計 (spec-dsn.md)
- データモデル (spec-dsn.md)
- インフラ構成 (spec-inf.md)
- 運用手順 (spec-ops.md)
- 認証・認可 (spec-sys.md, ADR-005)
- 監視・ログ (spec-inf.md, spec-ops.md)
- トラブルシューティング (spec-ops.md)

### ⚠️ 部分的にカバーされている領域

- **アーキテクチャ図**: テキストベースの説明のみ (図は未作成)
- **モジュール実装詳細**: 各モジュール(notion, github等)の詳細は別資料 (README.md参照)

### ❌ カバーされていない領域

- **C4モデル図**: Context/Container/Component/Code図 (将来作成予定)
- **ER図**: Vaultのテーブル関係図 (spec-dsn.md § 3にテキスト記載のみ)
- **シーケンス図**: ツール実行フローの詳細図 (将来作成予定)

---

## 関連ドキュメント

### 要件ドキュメント

- [要件一覧](DAY3/requirements/req-list.md) - 全要件のREQ-ID一覧
- [非機能要件](DAY3/requirements/req-nfr.md) - NFR-xxx一覧
- [スコープ外](DAY3/requirements/req-ofs.md) - OFS-xxx一覧
- [Gap分析](gap-analysis.md) - Requirements ↔ Spec対応表

### 設計判断記録

- [ADR-005](adr/ADR-005-no-rls-dependency.md) - RLS非依存設計
- [decision-log-requirements-first.md](decision-log-requirements-first.md) - Requirements First アプローチ

### モジュール詳細

- [README.md](../README.md) - 各モジュール(84ツール)の一覧

---

## 更新履歴

| 日付 | 更新内容 |
|------|---------|
| 2026-01-12 | 初版作成 (spec-index.md) |

---

## このドキュメントの保守

**更新タイミング:**
- 新しいspecファイルを追加したとき
- 既存specの構成を大きく変更したとき
- セクション番号が変わったとき

**保守担当:** プロジェクトリード
