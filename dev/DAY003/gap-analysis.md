---
title: MCPist Requirements vs Spec Gap分析
aliases:
  - gap-analysis
  - requirements-spec-gap
tags:
  - MCPist
  - analysis
  - requirements
  - specification
document-type:
  - analysis
document-class: analysis
created: 2026-01-12T00:00:00+09:00
updated: 2026-01-14T00:00:00+09:00
---
# Requirements vs Spec Gap分析

主要コンポーネントごとに Requirements ↔ Spec の対応を確認し、過剰な機能や不足を洗い出す。

---

## 1. MCPサーバー本体

### 1.1 メタツール設計

| 要件 | Spec参照 | 状態 |
|------|----------|------|
| REQ-039: 必要最低限の情報だけがコンテキストに含まれる | [spec-dsn.md § 2.1 get_module_schema](spec-dsn.md) 配列対応 | ✅ 対応済み |
| REQ-040: タスク実行時に不要な情報参照が発生しない | [spec-dsn.md § 2.1 get_module_schema](spec-dsn.md) 選択的スキーマ取得 | ✅ 対応済み |
| NFR-035: Context Rot防止 | [spec-dsn.md § 2.1 get_module_schema](spec-dsn.md) | ✅ 対応済み（今回追加） |
| NFR-036: 選択的スキーマ取得 | [spec-dsn.md § 2.1 get_module_schema](spec-dsn.md) 複数モジュール同時取得 | ✅ 対応済み（今回追加） |

**Gap:** なし

### 1.2 並列実行機能

| 要件 | Spec参照 | 状態 |
|------|----------|------|
| REQ-002: 依存関係のないタスクは並列実行される | [spec-dsn.md § 2.3 batch](spec-dsn.md) `after`フィールド | ✅ 対応済み |
| REQ-003: タスク完了結果だけを返す | [spec-dsn.md § 2.3 batch](spec-dsn.md) `output: true` | ✅ 対応済み |
| REQ-010: 軽量タスクは並列処理され待ち時間を最小化 | [spec-dsn.md § 2.3 batch](spec-dsn.md) 並列実行 | ✅ 対応済み |
| REQ-011: 大量の単純・軽量タスクを並列で処理 | [spec-dsn.md § 2.3 batch](spec-dsn.md) JSONL形式 | ✅ 対応済み |
| REQ-022: 調査・生成・準備タスクは可能な限り並列実行 | [spec-dsn.md § 2.3 batch](spec-dsn.md) | ✅ 対応済み |
| NFR-023: レスポンス時間 P95 < 3秒 | [spec-ops.md § 1.3](spec-ops.md) 可用性目標 | ✅ 対応済み |
| NFR-024: レスポンス時間 P99 < 5秒 | [spec-ops.md § 1.3](spec-ops.md) 可用性目標 | ✅ 対応済み |

**Gap:** なし

---

## 2. Token Broker

### 2.1 認証・トークン管理

| 要件 | Spec参照 | 状態 |
|------|----------|------|
| REQ-001: 個人用アカウントは他アカウントと物理的・論理的に分離 | [spec-sys.md § 2.3](spec-sys.md), [ADR-005](adr/ADR-005-no-rls-dependency.md) | ✅ 対応済み |
| REQ-013: ユーザー操作なしで継続実行（自動リフレッシュ） | [spec-sys.md § 2.3.2](spec-sys.md) トークン照会フロー | ✅ 対応済み |
| NFR-003: 自動認証リフレッシュ | [spec-sys.md § 2.3.2](spec-sys.md), [spec-sys.md § 3.2](spec-sys.md) | ✅ 対応済み |
| REQ-023: 他人のデータ侵害につながる設計は避けられている | [ADR-005](adr/ADR-005-no-rls-dependency.md) 多層防御 | ✅ 対応済み |
| REQ-024: 権限・データ境界が明確に分離 | [ADR-005](adr/ADR-005-no-rls-dependency.md) Edge Function認可 | ✅ 対応済み |
| REQ-025: デフォルトが安全側の実装 | [ADR-005](adr/ADR-005-no-rls-dependency.md) RLS保険 | ✅ 対応済み |

**Gap:** なし

---

## 3. 管理UI

### 3.1 マルチアカウント管理

| 要件 | Spec参照 | 状態 |
|------|----------|------|
| NFR-001: マルチアカウント管理 | [spec-sys.md § 2.4](spec-sys.md) 機能一覧 | ✅ 対応済み |
| NFR-002: アカウント別認証ストア | [spec-dsn.md § 3.1](spec-dsn.md) accounts/oauth_tokens | ✅ 対応済み |
| REQ-041: アカウント切り替え操作は最小限 | [spec-sys.md § 2.4](spec-sys.md) アカウント管理画面 | ✅ 対応済み |

### 3.2 初回セットアップ

| 要件 | Spec参照 | 状態 |
|------|----------|------|
| NFR-004: 初回セットアップが30分以内 | [spec-ops.md § 2](spec-ops.md) 初回セットアップ | ✅ 対応済み |
| NFR-005: OAuth認可フローが明確 | [spec-ops.md § 2.3](spec-ops.md) OAuthアプリ登録ガイド | ✅ 対応済み |
| NFR-006: トークン管理UIが提供される | [spec-sys.md § 2.4.3](spec-sys.md) トークン登録・状態表示 | ✅ 対応済み |

**Gap:** なし

---

## 4. 監視・運用

### 4.1 ログ・可観測性

| 要件 | Spec参照 | 状態 |
|------|----------|------|
| NFR-007: 構造化ログ（JSON形式） | [spec-inf.md § 4.3](spec-inf.md) ログ設計 | ✅ 対応済み |
| NFR-008: ログレベル制御（ERROR/WARN/INFO/DEBUG） | [spec-inf.md § 4.3](spec-inf.md) | ✅ 対応済み |
| NFR-009: ログ集約（Grafana Cloud Loki） | [spec-inf.md § 4.3](spec-inf.md), [spec-ops.md § 3.3](spec-ops.md) | ✅ 対応済み |
| NFR-010: ログ検索機能 | [spec-ops.md § 3.3](spec-ops.md) LogQLクエリ例 | ✅ 対応済み |
| NFR-011: アラート設定（Critical/Warning） | [spec-ops.md § 6](spec-ops.md) 監視・アラート | ✅ 対応済み |
| NFR-012: Grafana Cloudダッシュボード | [spec-ops.md § 6.3](spec-ops.md) | ✅ 対応済み |
| NFR-013: メトリクス収集（可用性、エラー率、レスポンス時間） | [spec-ops.md § 1.3](spec-ops.md) 可用性目標・KPI | ✅ 対応済み |
| NFR-014: トレーシング（リクエストID追跡） | [spec-inf.md § 4.3](spec-inf.md) 構造化ログ | ✅ 対応済み |

### 4.2 運用性

| 要件 | Spec参照 | 状態 |
|------|----------|------|
| NFR-027: 無料枠で運用可能 | [spec-inf.md § 2](spec-inf.md) コスト試算 | ✅ 対応済み |
| NFR-028: 放置運用が可能（手動介入最小） | [spec-ops.md § 1.1](spec-ops.md) 運用原則 | ✅ 対応済み |
| NFR-029: 障害対応手順の文書化 | [spec-ops.md § 9.2](spec-ops.md) Runbook | ✅ 対応済み（今回追加） |
| NFR-030: ポストモーテムの習慣化 | [spec-ops.md § 9.4](spec-ops.md) ポストモーテム | ✅ 対応済み（今回追加） |
| REQ-026: 基本的に放置運用が可能 | [spec-ops.md § 1.1](spec-ops.md) | ✅ 対応済み |
| REQ-027: 障害対応が頻発しない | [spec-ops.md § 1.3](spec-ops.md) 可用性 99% | ✅ 対応済み |
| REQ-028: 長期間触らなくても再開可能 | [spec-ops.md § 1.1](spec-ops.md) ステートレス設計 | ✅ 対応済み |

**Gap:** なし

---

## 5. セキュリティ

### 5.1 認証・認可

| 要件 | Spec参照 | 状態 |
|------|----------|------|
| NFR-015: JWT認証（MCPサーバー） | [spec-sys.md § 4.2](spec-sys.md) MCPサーバー認証 | ✅ 対応済み |
| NFR-016: OAuth 2.0/2.1（外部サービス） | [spec-sys.md § 2.4.2](spec-sys.md) タイプB認証 | ✅ 対応済み |
| NFR-017: トークン暗号化（AES-256-GCM） | [spec-dsn.md § 5.2](spec-dsn.md) Supabase Vault | ✅ 対応済み |
| NFR-018: 多層防御（JWT + Token Broker + RLS） | [ADR-005](adr/ADR-005-no-rls-dependency.md) | ✅ 対応済み（今回追加） |
| NFR-019: 危険操作の明示（dangerousフラグ） | [spec-dsn.md § 5.3](spec-dsn.md) 危険操作フラグ | ✅ 対応済み |

### 5.2 データ保護

| 要件 | Spec参照 | 状態 |
|------|----------|------|
| REQ-015: 業務データは原則ローカルPC内で完結 | [OFS-002](DAY3/requirements/req-ofs.md) スコープ外（クライアント責務） | ⚠️ MCPist責務外 |
| REQ-017: データ保存先・持ち出し範囲を明示的に制御 | [OFS-002](DAY3/requirements/req-ofs.md) スコープ外 | ⚠️ MCPist責務外 |
| REQ-018: ログ・中間生成物もローカル管理 | [OFS-002](DAY3/requirements/req-ofs.md) スコープ外 | ⚠️ MCPist責務外 |

**Gap:** REQ-015, REQ-017, REQ-018 はLLMクライアント側の責務として明示済み

---

## 6. パフォーマンス

| 要件 | Spec参照 | 状態 |
|------|----------|------|
| NFR-023: レスポンス時間 P95 < 3秒 | [spec-ops.md § 1.3](spec-ops.md) | ✅ 対応済み（今回追加） |
| NFR-024: レスポンス時間 P99 < 5秒 | [spec-ops.md § 1.3](spec-ops.md) | ✅ 対応済み（今回追加） |
| NFR-025: 並列実行数制限なし | [spec-dsn.md § 2.3](spec-dsn.md) batch | ✅ 対応済み |
| NFR-026: コールドスタート対策 | [spec-ops.md § 3.2](spec-ops.md) ヘルスチェック | ✅ 対応済み |

**Gap:** なし

---

## 7. 今回のセッションで追加されたSpec項目

| 追加項目 | 対応要件 | 評価 |
|----------|----------|------|
| [spec-ops.md § 1.3](spec-ops.md) 可用性目標・KPI | NFR-013, NFR-023, NFR-024, NFR-027 | ✅ 要件を満たす |
| [spec-ops.md § 9.2](spec-ops.md) Runbook | NFR-029 | ✅ 要件を満たす |
| [spec-ops.md § 9.4](spec-ops.md) ポストモーテム | NFR-030 | ✅ 要件を満たす |
| [spec-ops.md § 10](spec-ops.md) 運用成熟ロードマップ | - | ✅ 将来拡張の指針 |
| [ADR-005](adr/ADR-005-no-rls-dependency.md) RLS非依存設計 | REQ-023, REQ-024, REQ-025, NFR-018 | ✅ 要件を満たす |
| [spec-dsn.md § 2.1](spec-dsn.md) get_module_schema配列対応 | REQ-039, REQ-040, NFR-035, NFR-036 | ✅ 要件を満たす（今回追加） |
| [req-nfr.md NFR-035/036](DAY3/requirements/req-nfr.md) Context Rot防止 | - | ✅ メタツール設計根拠を明示（今回追加） |

---

## 8. 未対応Gap

**なし。** すべての要件がSpecでカバーされている、またはスコープ外として明示済み。

### 8.1 Requirementsから明示的にスコープ外へ移動済み

- REQ-015, REQ-017, REQ-018: ローカルデータ管理 → [req-ofs.md OFS-002](DAY3/requirements/req-ofs.md)
- REQ-064〜REQ-076: 通知・UX設計 → [req-ofs.md OFS-001](DAY3/requirements/req-ofs.md)
- 途中経過通知（SSE等）: REST原則に反するため対応外。MCPサーバーはステートレス設計を維持

---

## 9. 結論

**全体的に整合性は極めて高い。** 主要な要件はすべてSpecで実現されている。

**今回のセッションで追加されたSpec:**
- 運用・監視関連（可用性目標、Runbook、ポストモーテム）
- セキュリティ設計決定（ADR-005）
- パフォーマンス改善（get_module_schema配列対応）

**すべて要件を満たしており、Gapは解消されている。**

---

## 関連ドキュメント

- [要件一覧](DAY3/requirements/req-list.md)
- [非機能要件](DAY3/requirements/req-nfr.md)
- [スコープ外](DAY3/requirements/req-ofs.md)
- [システム仕様書](spec-sys.md)
- [設計仕様書](spec-dsn.md)
- [インフラ仕様書](spec-inf.md)
- [運用仕様書](spec-ops.md)
- [ADR-005](adr/ADR-005-no-rls-dependency.md)
