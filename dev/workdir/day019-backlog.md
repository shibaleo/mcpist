# DAY019 バックログ

## 日付

2026-01-30

---

## 新規追加

| ID | 内容 | 優先度 | 備考 |
|----|------|--------|------|
| BL-080 | セキュリティ設計書作成 | 中 | dsn-security.md |
| BL-081 | UX研究 | 低 | ユーザーフロー最適化 |
| BL-082 | UI研究 | 低 | デザインシステム検討 |
| BL-083 | ブランディング・ロゴ作成 | 低 | |
| BL-084 | ソーシャルログイン拡充 | 低 | GitHub, Apple など追加検討 |
| BL-085 | ユーザートークン保管方式の見直し | 高 | Supabase Vault は運営シークレット用。ユーザートークンは別方式を検討 |
| BL-086 | 環境変数の Supabase Vault 移行 | 中 | 運営シークレット（Stripe API Key等）を Vault で管理 |

---

## 引き継ぎ（DAY018 から）

| ID | 内容 | 状態 | 備考 |
|----|------|------|------|
| BL-011 | JWT `aud` チェック要件整理 | 未着手 | spc-itf.md 更新 |
| BL-012 | MCP 拡張エラーコード整理 | 未着手 | spc-itf.md 更新 |
| BL-013 | Console API 設計更新 | 未着手 | spc-itf.md 更新 |
| BL-014 | PSP Webhook 仕様整理 | 未着手 | spc-itf.md 更新 |
| BL-015 | enabled_modules 参照API実装 | 未着手 | Console ツール設定で一部実装済み |
| BL-016 | user_prompts 管理UI実装 | 未着手 | |
| BL-017 | usage_stats 参照API実装 | 未着手 | |
| BL-018 | クレジット付与機能（CON→DST） | ✅ 完了 | Stripe Checkout + Signup Bonus |
| BL-019 | ツール実行ログにuser_id追加 | 未着手 | |
| BL-020 | invalid_gateway_secretログ実装 | 未着手 | |
| BL-060 | RFC 8707 Resource Indicators 対応 | 未着手 | |
| BL-061 | クレジット初期化をDBトリガーからアプリ層へ移行 | ✅ 完了 | complete_onboarding RPC で実装 |
| BL-070 | database.types.ts 自動生成フロー整備 | 未着手 | CI/CD組み込み検討 |

---

## Sprint-005 残タスク

| ID | 内容 | 状態 | 備考 |
|----|------|------|------|
| S5-060 | UI要求仕様書作成 | 未着手 | 画面一覧・機能要件 |
| S5-061 | ユーザーフロー図作成 | 未着手 | 主要フローの可視化 |
| S5-062 | 画面遷移図作成 | 未着手 | 認証後のナビゲーション |
| S5-076 | CIにtools.json検証追加 | 未着手 | |
| S5-077 | 未使用Goモジュール削除 | 未着手 | |

---

## DAY019 完了タスク

| ID | 内容 | 備考 |
|----|------|------|
| D19-Stripe | Stripe Phase 1 完了 | Checkout + Webhook + billing UI |
| D19-Bonus | 初回クレジット付与（Signup Bonus） | pre_active → active 遷移 |
| D19-Onboarding | オンボーディングフロー改善 | tools step 削除、残高アラート追加 |
| D19-MCP | MCPサーバーエラーメッセージ改善 | billing URL 追加 |
| D19-RPC | RPC設計・マイグレーション統合 | Canvas更新、RPCリネーム、prompts RPC作成 |

---

## DAY020 への引き継ぎ

### RPC変更に伴うコード更新

| ID | 内容 | 優先度 | 備考 |
|----|------|--------|------|
| BL-087 | Console: RPC名変更対応 | 高 | `add_user_credits`, `complete_user_onboarding` 呼び出し箇所更新 |
| BL-088 | MCP Server: RPC名変更対応 | 高 | `consume_user_credits` 呼び出し箇所更新 |
| BL-089 | database.types.ts 再生成 | 高 | 新RPC（prompts系）のシグネチャ追加 |
| BL-090 | Prompts管理UI実装 | 中 | `list_my_prompts`, `upsert_my_prompt`, `delete_my_prompt` を使用 |
| BL-091 | Gateway: lookup_user_by_key_hash 確認 | 中 | 呼び出し元がGatewayであることを確認 |


### 優先度: 高

- BL-011〜014: spc-itf.md 更新（仕様と実装の差分解消）
- D19-005: Observability 設計書作成

### 優先度: 中

- BL-080: セキュリティ設計書作成
- S5-060〜062: UI仕様書・フロー図

### 優先度: 低

- BL-081〜084: UX/UI研究、ロゴ、ソーシャルログイン拡充
- BL-070: database.types.ts 自動生成

---

## 参考

- [day019-plan.md](./day019-plan.md) - 計画
- [day019-worklog.md](./day019-worklog.md) - 作業ログ
- [day018-backlog.md](./day018-backlog.md) - 前日バックログ
