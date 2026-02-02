# DAY022 バックログ

## 日付

2026-02-02

---

## DAY022 完了タスク

| ID | 内容 | 備考 |
|----|------|------|
| D22-001 | Trello MCP モジュール実装 | 17ツール（ボード、リスト、カード、チェックリスト操作） |
| D22-002 | Trello Console UI 追加 | API Key + Token 入力ダイアログ |
| D22-003 | Trello トークン検証 API 追加 | validate-token route に Trello 対応追加 |
| D22-004 | PKCE 認証エラー修正 | skipBrowserRedirect 削除 |

---

## DAY022 やり残し

### 仕様と実装の乖離

| ID | 内容 | 備考 |
|----|------|------|
| BL-090 | credentials JSON構造が仕様と不一致 | 下記詳細参照 |

#### BL-090 詳細: credentials 構造の乖離

**仕様（dtl-itr-MOD-TVL.md）**:
```json
{
  "auth_type": "oauth2",
  "credentials": {
    "access_token": "...",
    "refresh_token": "...",
    "expires_at": 1706140800
  }
}
```

**実際のDB（user_credentials.credentials カラム）**:
```json
{"auth_type": "oauth2", "expires_at": 1769..., "access_token": "...", ...}
```

**差異**:
1. 仕様では `credentials` オブジェクト内にネストされるべき `access_token`, `refresh_token`, `expires_at` がフラットに格納されている
2. `auth_type` も credentials と同レベルに混在

**対応方針**（要検討）:
- A) 仕様を実装に合わせて更新
- B) 実装を仕様に合わせてマイグレーション
- C) Server側で両形式をサポート（後方互換）

---

## 既存バックログ（day020-backlog.md より引き継ぎ）

### 仕様書整備

| ID | 内容 | 備考 |
|----|------|------|
| BL-010 | Rate Limit記述の更新 | 実装では削除済み。「将来実装予定」に変更 |
| BL-011 | JWT `aud` チェック要件整理 | 実装では明示チェックなし |
| BL-012 | MCP 拡張エラーコード整理 | JSON-RPC 標準コードのみに更新 |
| BL-013 | Console API 設計更新 | REST API → Supabase RPC 方式 |
| BL-014 | PSP Webhook 仕様整理 | Phase 1 実装に合わせて更新 |
| D16-002 | 仕様書の実装追従更新 | spec-impl-compare.md の差分を仕様書に反映 |

### 機能実装

| ID | 内容 | 備考 |
|----|------|------|
| BL-015 | enabled_modules 参照API実装 | Console ツール設定で一部実装済み |
| BL-016 | user_prompts 管理UI実装 | CORE-003 と統合 |
| BL-017 | usage_stats 参照API実装 | |
| BL-019 | ツール実行ログにuser_id追加 | |
| BL-020 | invalid_gateway_secretログ実装 | |

### MCP Primitives 実装

| ID | 項目 | 説明 | 状態 |
|----|------|------|------|
| CORE-001 | Google Tasks MCP実装 | google_tasks モジュール追加 | ✅ |
| CORE-002 | prompts MCP実装 | `prompts/list`, `prompts/get` ハンドラ | ❌ |
| CORE-003 | Console プロンプト管理UI | ユーザーがカスタムプロンプトを定義可能 | ❌ |
| CORE-004 | チャットUIからテンプレ実行 | Claude Web等でプロンプト選択・実行 | ❌ |
| CORE-005 | resources MCP実装 | `resources/list`, `resources/read` ハンドラ | ❌ |
| CORE-006 | resources/list 動作確認 | Grafana or サーバーログで呼び出し確認 | ❌ |
| CORE-007 | profile リソース実装 | `mcpist://profile` - ユーザープロフィール | ❌ |
| CORE-008 | tasks リソース実装 | `mcpist://tasks` - タスク一覧（MS Todo + Google Tasks） | ❌ |
| CORE-009 | Claude Code E2E | ユーザーが `@` でリソース選択・実行 | ❌ |

### 設計書作成

| ID | 内容 | 備考 |
|----|------|------|
| D19-005 | Observability 設計書作成 | dsn-observability.md |
| BL-080 | セキュリティ設計書作成 | dsn-security.md |

### 開発基盤

| ID | 内容 | 備考 |
|----|------|------|
| BL-070 | database.types.ts 自動生成フロー整備 | CI/CD組み込み検討 |
| BL-085 | ユーザートークン保管方式の見直し | Supabase Vault は運営用 |
| BL-086 | 環境変数の Supabase Vault 移行 | 運営シークレット（Stripe API Key等） |
| S5-076 | CIにtools.json検証追加 | |
| S5-077 | 未使用Goモジュール削除 | |

### UI/UX

| ID | 内容 | 備考 |
|----|------|------|
| S5-060 | UI要求仕様書作成 | 画面一覧・機能要件 |
| S5-061 | ユーザーフロー図作成 | 主要フローの可視化 |
| S5-062 | 画面遷移図作成 | 認証後のナビゲーション |
| BL-081 | UX研究 | ユーザーフロー最適化 |
| BL-082 | UI研究 | デザインシステム検討 |
| BL-083 | ブランディング・ロゴ作成 | |
| BL-084 | ソーシャルログイン拡充 | GitHub, Apple など |

### 将来検討

| ID | 内容 | 備考 |
|----|------|------|
| BL-060 | RFC 8707 Resource Indicators 対応 | OAuth 2.0 拡張 |
| BL-NEW | 管理者画面でツールバッジ表示期間/対象の管理 | tools.jsonではなく管理側で制御 |
| D16-006 | E2Eテスト設計 | OAuth認可フロー等 |

---

## 参考

- [day022-plan.md](./day022-plan.md) - 本日計画
- [day020-backlog.md](./day020-backlog.md) - 前回バックログ
- [backlog-open-tasks.md](./backlog-open-tasks.md) - 全体バックログ
- [dtl-itr-MOD-TVL.md](../../docs/002_specification/interaction/dtl-itr-MOD-TVL.md) - credentials 仕様
