# DAY020 バックログ

## 日付

2026-01-31

---

## バックログ概要

| カテゴリ | 件数 | 優先度: 高 | 優先度: 中 | 優先度: 低 |
|----------|------|------------|------------|------------|
| RPC変更対応 | 5 | 3 | 2 | 0 |
| 仕様書整備 | 4 | 4 | 0 | 0 |
| 設計書作成 | 2 | 1 | 1 | 0 |
| 機能実装 | 5 | 0 | 2 | 3 |
| 開発基盤 | 3 | 0 | 1 | 2 |
| UI/UX | 5 | 0 | 0 | 5 |
| **合計** | **24** | **8** | **6** | **10** |

---

## 優先度: 高（本日対応）

### RPC変更に伴うコード更新

| ID | 内容 | 備考 |
|----|------|------|
| BL-087 | Console: RPC名変更対応 | `add_user_credits`, `complete_user_onboarding` |
| BL-088 | MCP Server: RPC名変更対応 | `consume_user_credits` |
| BL-089 | database.types.ts 再生成 | 新RPC（prompts系）のシグネチャ追加 |

### 仕様書整備（spc-itf.md）

| ID | 内容 | 備考 |
|----|------|------|
| BL-011 | JWT `aud` チェック要件整理 | 実装では明示チェックなし |
| BL-012 | MCP 拡張エラーコード整理 | JSON-RPC 標準コードのみに更新 |
| BL-013 | Console API 設計更新 | REST API → Supabase RPC 方式 |
| BL-014 | PSP Webhook 仕様整理 | Phase 1 実装に合わせて更新 |

### 設計書作成

| ID | 内容 | 備考 |
|----|------|------|
| D19-005 | Observability 設計書作成 | dsn-observability.md |

---

## 優先度: 中

### RPC変更対応

| ID | 内容 | 備考 |
|----|------|------|
| BL-090 | Prompts管理UI実装 | `list_my_prompts`, `upsert_my_prompt`, `delete_my_prompt` |
| BL-091 | Gateway: lookup_user_by_key_hash 確認 | 呼び出し元がGatewayであることを確認 |

### 設計書作成

| ID | 内容 | 備考 |
|----|------|------|
| BL-080 | セキュリティ設計書作成 | dsn-security.md |

### 開発基盤

| ID | 内容 | 備考 |
|----|------|------|
| BL-086 | 環境変数の Supabase Vault 移行 | 運営シークレット（Stripe API Key等）を Vault で管理 |

### 機能実装

| ID | 内容 | 備考 |
|----|------|------|
| BL-016 | user_prompts 管理UI実装 | BL-090 と統合可 |
| BL-017 | usage_stats 参照API実装 | |

---

## 優先度: 低（バックログ）

### 機能実装

| ID | 内容 | 備考 |
|----|------|------|
| BL-015 | enabled_modules 参照API実装 | Console ツール設定で一部実装済み |
| BL-019 | ツール実行ログにuser_id追加 | |
| BL-020 | invalid_gateway_secretログ実装 | |

### 開発基盤

| ID | 内容 | 備考 |
|----|------|------|
| BL-070 | database.types.ts 自動生成フロー整備 | CI/CD組み込み検討 |
| BL-085 | ユーザートークン保管方式の見直し | Supabase Vault は運営用。ユーザートークンは別検討 |

### UI/UX

| ID | 内容 | 備考 |
|----|------|------|
| BL-081 | UX研究 | ユーザーフロー最適化 |
| BL-082 | UI研究 | デザインシステム検討 |
| BL-083 | ブランディング・ロゴ作成 | |
| BL-084 | ソーシャルログイン拡充 | GitHub, Apple など追加検討 |

### Sprint-005 残タスク

| ID | 内容 | 備考 |
|----|------|------|
| S5-060 | UI要求仕様書作成 | 画面一覧・機能要件 |
| S5-061 | ユーザーフロー図作成 | 主要フローの可視化 |
| S5-062 | 画面遷移図作成 | 認証後のナビゲーション |
| S5-076 | CIにtools.json検証追加 | |
| S5-077 | 未使用Goモジュール削除 | |

### 将来検討

| ID | 内容 | 備考 |
|----|------|------|
| BL-060 | RFC 8707 Resource Indicators 対応 | OAuth 2.0 拡張 |

---

## DAY019 完了タスク（参考）

| ID | 内容 | 備考 |
|----|------|------|
| D19-Stripe | Stripe Phase 1 完了 | Checkout + Webhook + billing UI |
| D19-Bonus | 初回クレジット付与（Signup Bonus） | pre_active → active 遷移 |
| D19-Onboarding | オンボーディングフロー改善 | tools step 削除、残高アラート追加 |
| D19-MCP | MCPサーバーエラーメッセージ改善 | billing URL 追加 |
| D19-RPC | RPC設計・マイグレーション統合 | Canvas更新、RPCリネーム、prompts RPC作成 |
| BL-018 | クレジット付与機能（CON→DST） | ✅ 完了 |
| BL-061 | クレジット初期化をDBトリガーからアプリ層へ移行 | ✅ 完了 |

---

## 参考

- [day020-plan.md](./day020-plan.md) - 計画
- [day019-backlog.md](./day019-backlog.md) - 前日バックログ
- [day019-worklog.md](./day019-worklog.md) - 前日作業ログ
- [day019-review.md](./day019-review.md) - 前日レビュー
