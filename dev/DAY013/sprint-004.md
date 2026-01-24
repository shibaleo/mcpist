# Sprint 004: コア機能設計 - コンポーネント連携仕様

## 基本情報

| 項目 | 値 |
|------|-----|
| スプリント番号 | SPRINT-004 |
| 期間 | 2026-01-24 〜 |
| マイルストーン | M4: コア機能完成 |
| 目標 | システムコンポーネント間連携仕様の策定、詳細要求仕様の洗い出し |
| 状態 | 進行中 |
| 前提 | Sprint-003 完了（本番デプロイ済み、OAuth認可フロー完成） |

---

## スプリント目標

1. **システムコンポーネント間の連携仕様を固める**
2. **コア機能の詳細要求仕様を洗い出す**
3. **API仕様・データフロー・シーケンス図を作成する**

---

## 前提: Sprint-003の成果

### 本番稼働中のサービス

| サービス | URL | 状態 |
|---------|-----|------|
| Console | https://dev.mcpist.app | ✅ 稼働中 |
| MCP API | https://mcp.dev.mcpist.app | ✅ 稼働中 |
| Server (Primary) | Render | ✅ 稼働中 |
| Server (Secondary) | Koyeb | ✅ 稼働中 |

### 確認済みの接続

| クライアント | 認証方式 | 状態 |
|-------------|---------|------|
| Claude.ai | OAuth 2.0 | ✅ 成功 |
| ChatGPT Desktop | OAuth 2.0 | ✅ 成功 |
| Claude Code | APIキー | ✅ 成功 |

---

## Phase 1: システムコンポーネント間連携仕様

### 1.1 コンポーネント一覧

| コンポーネント | 技術スタック | 役割 |
|---------------|-------------|------|
| Console | Next.js (Vercel) | 管理UI、OAuth callback |
| Worker | Cloudflare Workers | API Gateway、認証、ルーティング |
| Server | Go (Render/Koyeb) | MCP Server、ツール実行 |
| Supabase | PostgreSQL + Auth + Vault | DB、認証、シークレット管理 |

### 1.2 連携フロー図（作成対象）

| ID | 図 | 説明 | 状態 | 参照 |
|----|---|------|------|------|
| D-001 | シーケンス図: OAuth認可フロー | MCPクライアント → Worker → Console → Supabase | ✅ | itr-clo.md, itr-aus.md |
| D-002 | シーケンス図: APIキー認証フロー | MCPクライアント → Worker → Server | ✅ | itr-clk.md, itr-gwy.md, itr-tvl.md |
| D-003 | シーケンス図: ツール実行フロー | Worker → Server → Vault → 外部API | ✅ | itr-srv.md, itr-hdl.md, itr-mod.md |
| D-004 | シーケンス図: シークレット登録フロー | Console → 外部OAuth → Vault | ✅ | itr-con.md, itr-eas.md, itr-tvl.md |
| D-005 | データフロー図: 全体俯瞰 | 全コンポーネント間のデータの流れ | ✅ | spc-itr.md |

### 1.3 API仕様（作成対象）

| ID | API | コンポーネント | 状態 | 参照 |
|----|-----|---------------|------|------|
| A-001 | Worker内部API仕様 | Worker → Server 間 | ✅ | itr-gwy.md, itr-amw.md |
| A-002 | Server RPC仕様 | Server → Supabase 間 | 🔄 | itr-dst.md (draft) |
| A-003 | Console API仕様 | Console ↔ Worker 間 | 🔄 | itr-con.md (draft) |
| A-004 | Vault アクセス仕様 | Server → Vault 間 | ✅ | itf-tvl.md, itr-tvl.md |

### タスク

| ID | タスク | 成果物 | 状態 | 備考 |
|----|--------|--------|------|------|
| T-001 | コンポーネント間通信の現状整理 | 連携マトリクス | ✅ | dtl-spcフォルダ全20ファイル作成済み |
| T-002 | OAuth認可フロー シーケンス図作成 | D-001 | ✅ | itr-clo.md, itr-aus.md にmermaid図あり |
| T-003 | ツール実行フロー シーケンス図作成 | D-003 | ✅ | itr-srv.md にmermaid図あり |
| T-004 | Worker-Server間API仕様書作成 | A-001 | ✅ | itr-gwy.md, itr-amw.md にヘッダー仕様あり |
| T-005 | Vault アクセスパターン設計 | A-004 | ✅ | itf-tvl.md に詳細API仕様あり |

### 1.4 仕様書ステータス

#### interaction/ フォルダ - itr-xxx（コンポーネント別インタラクション仕様）

| ファイル | Status | Version | 備考 |
|---------|--------|---------|------|
| itr-clo.md | **reviewed** | v2.0 | MCP Client OAuth2.0（実装範囲外） |
| itr-clk.md | **reviewed** | v2.0 | MCP Client API KEY（実装範囲外） |
| itr-gwy.md | **reviewed** | v2.0 | API Gateway |
| itr-aus.md | **reviewed** | v2.0 | Auth Server |
| itr-ssm.md | **reviewed** | v2.0 | Session Manager |
| itr-dst.md | **reviewed** | v2.0 | Data Store |
| itr-tvl.md | **reviewed** | v2.1 | Token Vault |
| itr-amw.md | **reviewed** | v2.0 | Auth Middleware |
| itr-hdl.md | **reviewed** | v3.1 | MCP Handler（REG統合版） |
| itr-mod.md | **reviewed** | v2.1 | Modules |
| itr-con.md | **reviewed** | v2.0 | User Console |
| itr-srv.md | **reviewed** | v3.0 | MCP Server |
| itr-idp.md | **reviewed** | v2.0 | Identity Provider（実装範囲外） |
| itr-eas.md | **reviewed** | v2.0 | External Auth Server（実装範囲外） |
| itr-ext.md | **reviewed** | v2.0 | External Service API（実装範囲外） |
| itr-psp.md | **reviewed** | v2.0 | Payment Service Provider（実装範囲外） |
| itr-reg.md | **deprecated** | v2.0 | HDLに統合済み |

**サマリー:** reviewed: 16 / deprecated: 1 ✅

#### interaction/ フォルダ - dtl-itr-XXX-YYY（インタラクション詳細仕様）

| ID | ファイル | Status | 内容 |
|----|---------|--------|------|
| ITR-REL-001 | dtl-itr-CLO-GWY.md | draft | CLO - GWY MCP通信 |
| ITR-REL-002 | dtl-itr-AUS-CLO.md | draft | CLO - AUS OAuth認可 |
| ITR-REL-003 | dtl-itr-CLK-GWY.md | draft | CLK - GWY MCP通信 |
| ITR-REL-004 | dtl-itr-AUS-GWY.md | draft | GWY - AUS トークン検証 |
| ITR-REL-005 | dtl-itr-GWY-TVL.md | draft | GWY - TVL APIキー検証 |
| ITR-REL-006 | dtl-itr-AMW-GWY.md | draft | GWY - AMW リクエスト転送 |
| ITR-REL-007 | dtl-itr-AMW-HDL.md | draft | AMW - HDL 認証済みリクエスト |
| ITR-REL-008 | dtl-itr-DST-HDL.md | draft | HDL - DST ユーザー設定取得 |
| ITR-REL-009 | dtl-itr-HDL-MOD.md | draft | HDL - MOD プリミティブ操作委譲 |
| ITR-REL-010 | dtl-itr-MOD-TVL.md | draft | MOD - TVL トークン取得 |
| ITR-REL-011 | dtl-itr-DST-MOD.md | draft | MOD - DST クレジット消費 |
| ITR-REL-012 | dtl-itr-EXT-MOD.md | draft | MOD - EXT API呼び出し |
| ITR-REL-013 | dtl-itr-EAS-TVL.md | draft | TVL - EAS トークンリフレッシュ |
| ITR-REL-014 | dtl-itr-CON-SSM.md | draft | CON - SSM ソーシャルログイン |
| ITR-REL-015 | dtl-itr-CON-TVL.md | draft | CON - TVL トークン登録 |
| ITR-REL-016 | dtl-itr-CON-DST.md | draft | CON - DST ツール設定登録 |
| ITR-REL-017 | dtl-itr-CON-PSP.md | draft | CON - PSP 決済 |
| ITR-REL-018 | dtl-itr-CON-EAS.md | draft | CON - EAS 認可フロー |
| ITR-REL-019 | dtl-itr-IDP-SSM.md | draft | SSM - IDP ソーシャルログイン |
| ITR-REL-020 | dtl-itr-AUS-SSM.md | draft | SSM - AUS ユーザー認証連携 |
| ITR-REL-021 | dtl-itr-DST-PSP.md | draft | PSP - DST 有料クレジット情報 |

**サマリー:** draft: 21 ✅（全21件作成完了）

#### dtl-spc/ フォルダ

| ファイル | Status | Version | 備考 |
|---------|--------|---------|------|
| idx-ept.md | draft | v1.0 | エンドポイント一覧 |
| dtl-spc-hdl.md | draft | v1.0 | MCP Handler詳細仕様 |
| itf-tvl.md | draft | v1.1 | Token Vault API仕様 |
| dtl-spc-credit-model.md | draft | v1.0 | クレジットモデル詳細仕様 |
| itf-mod.md | - | - | Modules API仕様（未確認） |

**サマリー:** draft: 4

#### test/ フォルダ

| ファイル | Status | Version | 備考 |
|---------|--------|---------|------|
| tst-policy.md | draft | v1.0 | テスト方針書（9コンポーネント） |
| tst-tvl.md | draft | v1.1 | Token Vault テスト手順書 |
| tst-mod-notion.md | - | - | Notion Module 統合テスト結果 |
| tst-oauth-mock-server.md | - | - | OAuth mockサーバー手順書 |

#### graph/ フォルダ

| ファイル | 備考 |
|---------|------|
| grah-componet-interactions.canvas | コンポーネント間連携図（15ノード、21エッジ） |

#### 全体サマリー

| Status | 件数 |
|--------|------|
| reviewed（itr-xxx） | 16 |
| draft（dtl-itr-XXX-YYY） | 21 |
| draft（dtl-spc/） | 4 |
| draft（test/） | 2 |
| deprecated | 1 |
| **合計** | **44** |

### 1.5 完了した作業

| 作業 | 成果物 | 状態 |
|------|--------|------|
| itr-xxx全ファイルをreviewedに更新 | 16ファイル | ✅ |
| dtl-itr-XXX-YYY全21ファイル作成 | 21ファイル | ✅ |
| itr-xxxからdtl-itr-XXX-YYYへの詳細転記 | - | ✅ |
| idx-itr-rel.md（ITR-REL ID一覧）作成 | 1ファイル | ✅ |
| spc-sys.mdからModule Registry削除 | - | ✅ |
| grah-componet-interactions.canvas更新 | 15ノード、21エッジ | ✅ |
| tst-policy.md（テスト方針書）作成 | 9コンポーネント対応 | ✅ |

### 1.6 次のレビュー対象

| 優先度 | ファイル | 内容 | レビュー観点 |
|--------|---------|------|-------------|
| 高 | dtl-itr-XXX-YYY（21件） | インタラクション詳細 | draft → reviewed |
| 中 | dtl-spc-hdl.md | MCP Handler詳細 | 実装詳細 |
| 中 | itf-tvl.md | Token Vault API | API仕様 |
| 中 | dtl-spc-credit-model.md | クレジットモデル | ビジネスロジック |
| 低 | idx-ept.md | エンドポイント一覧 | インデックス |

---

## Phase 2: 詳細要求仕様の洗い出し

### 2.1 ユーザー登録

| ID | 要求 | 優先度 | 状態 |
|----|------|--------|------|
| R-001 | Supabase Auth によるログイン（Email/OAuth） | 高 | ✅ 実装済み |
| R-002 | ユーザープロファイル自動作成 | 高 | 🔄 要確認 |
| R-003 | 初回ログイン時のオンボーディング | 中 | ⬜ |
| R-004 | 利用規約同意フロー | 中 | ⬜ |
| R-005 | アカウント削除機能 | 低 | ⬜ |

### 2.2 シークレット登録

| ID | 要求 | 優先度 | 状態 |
|----|------|--------|------|
| R-010 | 外部サービスOAuth接続 | 高 | 🔄 要確認 |
| R-011 | アクセストークンのVault保存 | 高 | 🔄 要確認 |
| R-012 | リフレッシュトークンの自動更新 | 高 | ⬜ |
| R-013 | 接続状態の表示 | 中 | ⬜ |
| R-014 | 接続解除機能 | 中 | ⬜ |
| R-015 | 複数アカウント対応（同一サービス） | 低 | ⬜ |

### 2.3 機能呼び出し（ツール実行）

| ID | 要求 | 優先度 | 状態 |
|----|------|--------|------|
| R-020 | ユーザーコンテキストの伝播 | 高 | ⬜ |
| R-021 | Vaultからのトークン取得 | 高 | ⬜ |
| R-022 | 外部API呼び出し | 高 | ✅ 実装済み（固定トークン） |
| R-023 | トークン期限切れ時のリフレッシュ | 高 | ⬜ |
| R-024 | エラーハンドリング（外部API障害） | 中 | ⬜ |
| R-025 | レート制限（外部API側） | 中 | ⬜ |

### 2.4 クレジットシステム

| ID | 要求 | 優先度 | 状態 |
|----|------|--------|------|
| R-030 | リクエスト数カウント | 高 | ⬜ |
| R-031 | 無料枠の設定 | 高 | ⬜ |
| R-032 | 残高表示 | 中 | ⬜ |
| R-033 | 残高不足時の制限 | 中 | ⬜ |
| R-034 | クレジット購入（Stripe連携） | 低 | ⬜ |
| R-035 | 利用履歴表示 | 低 | ⬜ |

### タスク

| ID | タスク | 成果物 | 状態 |
|----|--------|--------|------|
| T-006 | 現状実装の棚卸し | 実装状況マトリクス | ⬜ |
| T-007 | 要求仕様のギャップ分析 | 未実装機能リスト | ⬜ |
| T-008 | 優先度付けと依存関係整理 | 実装順序表 | ⬜ |
| T-009 | 各要求の受け入れ条件定義 | AC一覧 | ⬜ |

---

## Phase 3: データモデル設計 ✅

### 3.1 テーブル設計 ✅

テーブル仕様書およびER図を作成完了。

| ドキュメント | 内容 | 状態 |
|-------------|------|------|
| spc-tbl.md | テーブル仕様書（役割・インタラクション） | ✅ 更新済み |
| dsn-tbl.md | テーブル設計書（ER図・リレーション） | ✅ 新規作成 |
| dtl-dsn-tbl.md | テーブル詳細設計書（列定義・RLS） | ✅ 新規作成 |
| grh-table-design.canvas | ER図（Obsidian Canvas） | ✅ 新規作成 |

### 3.2 テーブル一覧（確定）

| テーブル | スキーマ | 用途 | 状態 |
|---------|---------|------|------|
| `auth.users` | auth | ユーザー認証 | ✅ Supabase管理 |
| `vault.secrets` | vault | 暗号化トークン保存 | ✅ Supabase管理 |
| `mcpist.users` | mcpist | ユーザー情報、account_status | ✅ 設計完了 |
| `mcpist.api_keys` | mcpist | APIキー管理（SHA-256ハッシュ） | ✅ 設計完了 |
| `mcpist.credits` | mcpist | free_credits, paid_credits残高 | ✅ 設計完了 |
| `mcpist.credit_transactions` | mcpist | クレジット増減履歴 | ✅ 設計完了 |
| `mcpist.modules` | mcpist | モジュール定義（マスタ） | ✅ 設計完了 |
| `mcpist.module_settings` | mcpist | ユーザー×モジュール有効/無効 | ✅ 設計完了 |
| `mcpist.tool_settings` | mcpist | ユーザー×ツール有効/無効 | ✅ 設計完了 |
| `mcpist.prompts` | mcpist | ユーザー定義プロンプト | ✅ 設計完了 |
| `mcpist.processed_webhook_events` | mcpist | PSP Webhook冪等性 | ✅ 設計完了 |

### 3.3 コンポーネント別Supabaseキー

| コンポーネント | Supabaseキー | 用途 |
|---------------|-------------|------|
| User Console (Frontend) | anon key | ユーザー操作（RLS適用） |
| User Console (API Routes) | service_role key | Webhook処理、管理操作 |
| Cloudflare Worker | - | API Gateway（認証・ルーティングのみ） |
| MCP Server (Go) | service_role key | ツール実行、クレジット消費 |

### タスク

| ID | タスク | 成果物 | 状態 |
|----|--------|--------|------|
| T-010 | 既存テーブル構造の確認 | ER図（現状） | ✅ |
| T-011 | 新規テーブル設計 | dsn-tbl.md, dtl-dsn-tbl.md | ✅ |
| T-012 | RPC関数一覧の整理 | dtl-dsn-tbl.md内に記載 | ✅ |
| T-013 | RLSポリシー設計 | dtl-dsn-tbl.md内に記載 | ✅ |
| T-014 | コンポーネント別キー整理 | dtl-dsn-tbl.md内に記載 | ✅ |

---

## 完了条件

### Phase 1: コンポーネント連携仕様 ✅
- [x] 全コンポーネント間の通信パターンが文書化されている（dtl-spc/itr-*.md 全20ファイル）
- [x] 主要フローのシーケンス図が作成されている（mermaid形式）
- [x] Worker-Server間API仕様が定義されている（itr-gwy.md, itr-amw.md）

### Phase 2: 詳細要求仕様 ⬜
- [ ] コア機能の要求仕様が一覧化されている
- [ ] 各要求の優先度と依存関係が明確になっている
- [ ] 受け入れ条件（AC）が定義されている

### Phase 3: データモデル ✅
- [x] 既存テーブルの構造が文書化されている（dsn-tbl.md）
- [x] 新規テーブルの設計が完了している（dtl-dsn-tbl.md）
- [x] RPC関数の仕様が整理されている（dtl-dsn-tbl.md）
- [x] RLSポリシーが設計されている（dtl-dsn-tbl.md）
- [x] ER図が作成されている（grh-table-design.canvas）

---

## 成果物一覧

### Phase 1 完了分

| カテゴリ | ファイル | 説明 | 状態 |
|---------|---------|------|------|
| 仕様書 | itr-xxx.md（16件） | コンポーネント別インタラクション仕様 | ✅ reviewed |
| 仕様書 | dtl-itr-XXX-YYY.md（21件） | インタラクション詳細仕様 | ✅ draft |
| 仕様書 | idx-itr-rel.md | インタラクション関係ID一覧 | ✅ reviewed |
| 仕様書 | spc-sys.md | システム仕様書（REG削除） | ✅ 更新済み |
| 図 | grah-componet-interactions.canvas | コンポーネント間連携図 | ✅ 更新済み |
| テスト | tst-policy.md | テスト方針書（9コンポーネント） | ✅ draft |

### Phase 3 完了分

| カテゴリ | ファイル | 説明 | 状態 |
|---------|---------|------|------|
| 仕様書 | spc-tbl.md | テーブル仕様書（更新） | ✅ |
| 設計書 | dsn-tbl.md | テーブル設計書 | ✅ 新規作成 |
| 設計書 | dtl-dsn-tbl.md | テーブル詳細設計書 | ✅ 新規作成 |
| 図 | grh-table-design.canvas | ER図（Obsidian Canvas） | ✅ 新規作成 |

### Phase 2 予定分

| カテゴリ | ファイル | 説明 | 状態 |
|---------|---------|------|------|
| 仕様書 | `spc-req.md` | 詳細要求仕様 | ⬜ |

---

## バックログ（Sprint-005以降）

### 優先度: 高

| タスク | 備考 |
|--------|------|
| コア機能実装（Phase 2-4の要求） | 設計に基づく実装 |
| CI/CDパイプライン構築 | GitHub Actions |
| サービストークン登録の排他制御 | 1ユーザー×1サービスにつき、有効なトークンタイプ（oauth2/api_key）は1種類のみ。接続解除しない限り他方のタイプは登録不可。Console UIで制御 |

### 優先度: 中

| タスク | 備考 |
|--------|------|
| Stg/Prd環境構築 | Blue-Green方式 |
| セキュリティ強化 | レート制限、監査ログ |
| オブザーバビリティ | ログ、メトリクス、トレース |

### 優先度: 低

| タスク | 備考 |
|--------|------|
| 追加モジュール（Slack/Linear） | 新機能 |
| E2Eテスト（Playwright） | CI統合 |
| 技術的負債解消 | デバッグログ削除等 |

---

## 関連ドキュメント

| ドキュメント | 内容 |
|-------------|------|
| [Sprint-003 レビュー](../DAY012/review-003.md) | 本番デプロイ & OAuth認可フロー |
| [Sprint-003](../DAY012/sprint-003.md) | APIキー認証 & 本番デプロイ準備 |
| [spc-dmn.md](../../mcpist/docs/specification/spc-dmn.md) | ドメイン仕様書 |
| [spc-dpl.md](../../mcpist/docs/specification/spc-dpl.md) | デプロイ仕様書 |
