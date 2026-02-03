# Sprint 006 バックログ

Sprint-006 完了時点での残課題一覧。次スプリントへの引き継ぎ対象。

**更新日:** 2026-02-03

---

## 仕様書整備

| ID | タスク | 備考 | 状態 |
|----|--------|------|------|
| BL-010 | Rate Limit記述の更新 | 実装では削除済み。「将来実装予定」に変更 | 未着手 |
| BL-011 | JWT `aud` チェック要件整理 | 実装では明示チェックなし | 未着手 |
| BL-012 | MCP 拡張エラーコード整理 | JSON-RPC 標準コードのみに更新 | 未着手 |
| BL-013 | Console API 設計更新 | REST API → Supabase RPC 方式 | 未着手 |
| BL-014 | PSP Webhook 仕様整理 | Phase 1 実装に合わせて更新 | 未着手 |
| BL-090 | credentials JSON構造の乖離解消 | 仕様ではネスト、実装ではフラット | 未着手 |
| D16-002 | 仕様書の実装追従更新 | spec-impl-compare.md の差分を仕様書に反映 | 未着手 |

---

## 設計書作成

| ID | タスク | 成果物 | 状態 |
|----|--------|--------|------|
| D19-005 | Observability 設計書作成 | dsn-observability.md | 未着手 |
| BL-080 | セキュリティ設計書作成 | dsn-security.md | 未着手 |

---

## 機能実装

| ID | タスク | 備考 | 状態 |
|----|--------|------|------|
| BL-015 | enabled_modules 参照API実装 | Console ツール設定で一部実装済み | 進行中 |
| BL-017 | usage_stats 参照API実装 | | 未着手 |
| BL-019 | ツール実行ログにuser_id追加 | | 未着手 |
| BL-020 | invalid_gateway_secretログ実装 | | 未着手 |
| BL-077 | Google Calendar 日本の祝日対応 | 祝日カレンダー取得・表示機能 | 未着手 |

---

## テスト基盤構築

| ID | タスク | 成果物 | 状態 |
|----|--------|--------|------|
| S6-020 | E2E テスト設計書作成 | tst-e2e.md | 未着手 |
| S6-021 | Go Server ユニットテスト | *_test.go | 未着手 |
| S6-022 | Batch 権限チェックのテスト | handler_test.go | 未着手 |
| S6-023 | Console ビルドテスト | CI workflow | 未着手 |
| D16-006 | E2Eテスト設計 | OAuth認可フロー等 | 未着手 |

---

## CI/CD 整備

| ID | タスク | 成果物 | 状態 |
|----|--------|--------|------|
| S6-030 | tools.json 検証 CI | GitHub Actions | 未着手 |
| S6-031 | Go lint + test CI | GitHub Actions | 未着手 |
| S6-032 | Console lint + build CI | GitHub Actions | 未着手 |
| S5-076 | CIにtools.json検証追加 | | 未着手 |

---

## 開発基盤

| ID | タスク | 備考 | 状態 |
|----|--------|------|------|
| S5-077 | 未使用Goモジュール削除 | クリーンアップ | 未着手 |
| BL-070 | database.types.ts 自動生成フロー整備 | CI/CD組み込み検討 | 未着手 |
| BL-085 | ユーザートークン保管方式の見直し | Supabase Vault は運営用 | 未着手 |
| BL-086 | 環境変数の Supabase Vault 移行 | 運営シークレット（Stripe API Key等） | 未着手 |

---

## UI/UX

| ID | タスク | 備考 | 状態 |
|----|--------|------|------|
| S5-060 | UI要求仕様書作成 (spc-ui.md) | 画面一覧・機能要件 | 未着手 |
| S5-061 | ユーザーフロー図作成 | 主要フローの可視化 | 未着手 |
| S5-062 | 画面遷移図作成 | 認証後のナビゲーション | 未着手 |
| BL-081 | UX研究 | ユーザーフロー最適化 | 未着手 |
| BL-082 | UI研究 | デザインシステム検討 | 未着手 |
| BL-083 | ブランディング・ロゴ作成 | | 未着手 |
| BL-084 | ソーシャルログイン拡充 | GitHub, Apple など | 未着手 |

---

## 将来検討

| ID | タスク | 備考 | 状態 |
|----|--------|------|------|
| BL-002 | 切断時のツール設定クリーンアップ | DB migration 前提 | 保留 |
| BL-030 | Stg/Prd 環境構築 | Blue-Green 方式 | 未着手 |
| BL-032 | 追加モジュール（Slack/Linear） | 需要に応じて | 未着手 |
| BL-060 | RFC 8707 Resource Indicators 対応 | OAuth 2.0 拡張 | 未着手 |
| BL-NEW | 管理者画面でツールバッジ表示期間/対象の管理 | tools.jsonではなく管理側で制御 | 未着手 |

---

## Sprint 006 完了タスク（参考）

| ID | 内容 | 備考 |
|----|------|------|
| CORE-001 | Google Tasks MCP実装 | 9ツール |
| CORE-002 | prompts MCP実装 | `prompts/list`, `prompts/get` ハンドラ |
| CORE-003 | Console プロンプト管理UI | description追加、楽観的更新 |
| CORE-004 | チャットUIからテンプレ実行 | Claude Web等で動作確認 |
| BL-016 | user_prompts 管理UI実装 | Console で完全実装 |
| BL-076 | Microsoft To Do モジュール実装 | 8ツール |
| D22-001〜012 | 10モジュール・130+ツール実装 | Todoist, Trello, GitHub, Asana, Google Docs/Drive/Apps Script, PostgreSQL |

---

## 廃止タスク

| ID | 内容 | 理由 |
|----|------|------|
| ~~CORE-005~~ | resources MCP実装 | 実装しない方針 |
| ~~CORE-006~~ | resources/list 動作確認 | 実装しない方針 |
| ~~CORE-007~~ | profile リソース実装 | 実装しない方針 |
| ~~CORE-008~~ | tasks リソース実装 | 実装しない方針 |
| ~~CORE-009~~ | Claude Code E2E | 実装しない方針 |

---

## 参考

- [sprint006.md](./sprint006.md) - Sprint 006 計画書・完了報告
- [day022-backlog.md](../workdir/day022-backlog.md) - 統合元（参照不要）
