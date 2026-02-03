# DAY024 計画

## 日付

2026-02-04

---

## 概要

Sprint-007 2日目。DAY023 で完了した設計書（dsn-observability.md, dsn-security.md）に基づき、Go Server の構造化ログ実装（S7-002〜005）を行う。

---

## DAY023 からの引き継ぎ

| 項目 | 状態 |
|------|------|
| Phase 1: Observability 設計書 (S7-001) | ✅ 完了 |
| Phase 1: ダッシュボード設計 (S7-006) | ✅ 設計完了（Grafana UI 設定は未着手） |
| Phase 2: セキュリティ設計書 (S7-010〜015) | ✅ 全完了 |
| Phase 1: 構造化ログ実装 (S7-002〜005) | ❌ 未着手 |
| Phase 3: 仕様書更新 (S7-020〜026) | ❌ 未着手 |

---

## 本日のタスク

### Phase 1: Go Server 構造化ログ実装（優先度：高）

dsn-observability.md Section 11 の実装タスク一覧に基づく。

#### S7-005: ラベル・データフィールド整理

ラベルカーディナリティ対策を先に行い、以降の変更と衝突しないようにする。

| ID | タスク | 対象ファイル | 備考 | 状態 |
|----|--------|-------------|------|------|
| D24-001 | `LogToolCall` ラベルから `tool` を削除、データフィールドに移動 | `internal/observability/loki.go` | S7-005-1 | 未着手 |
| D24-002 | `LogSecurityEvent` ラベルから `event` を削除、データフィールドに移動 | `internal/observability/loki.go` | S7-005-2 | 未着手 |

#### S7-003: user_id 追加

| ID | タスク | 対象ファイル | 備考 | 状態 |
|----|--------|-------------|------|------|
| D24-003 | `LogToolCall` に `userID` パラメータ追加 | `internal/observability/loki.go` | S7-003-1 | 未着手 |
| D24-004 | `modules.Run` から `userID` を渡す | `internal/modules/modules.go` | S7-003-2 | 未着手 |
| D24-005 | `modules.Batch` から `userID` を渡す | `internal/modules/modules.go` | S7-003-3 | 未着手 |

#### S7-004: invalid_gateway_secret ログ

| ID | タスク | 対象ファイル | 備考 | 状態 |
|----|--------|-------------|------|------|
| D24-006 | Gateway Secret 検証失敗時に `LogSecurityEvent` 呼び出し追加 | `internal/middleware/authz.go` | S7-004-1 | 未着手 |

#### S7-002: 不要な log.Printf の削除

| ID | タスク | 対象ファイル | 備考 | 状態 |
|----|--------|-------------|------|------|
| D24-007 | `LogToolCall` 内の `log.Printf` を削除 | `internal/observability/loki.go` | S7-002-1 | 未着手 |
| D24-008 | `middleware/authz.go` の認証ログを `LogSecurityEvent` に統合 | `internal/middleware/authz.go` | S7-002-2 | 未着手 |
| D24-009 | 各モジュール内の不要な `log.Printf` を削除 | `internal/modules/*.go` | S7-002-3 | 未着手 |

#### ビルド・動作確認

| ID | タスク | 備考 | 状態 |
|----|--------|------|------|
| D24-010 | Go ビルド確認 | `go build ./cmd/server` | 未着手 |
| D24-011 | ローカル動作確認 | Loki Push のログ構造確認 | 未着手 |

---

## 実装方針

### 変更順序

1. **ラベル整理 (D24-001, 002)** を先に → ログ構造の基盤を確定
2. **user_id 追加 (D24-003〜005)** → `LogToolCall` のシグネチャ変更
3. **異常イベントログ (D24-006)** → `LogSecurityEvent` の呼び出し追加
4. **不要ログ削除 (D24-007〜009)** → クリーンアップ

この順序により、シグネチャ変更の影響を最小化する。

### 設計書の参照箇所

| タスク | dsn-observability.md |
|--------|---------------------|
| ラベル設計 | Section 3.1 |
| データフィールド | Section 3.2 |
| user_id 追加 | Section 5 |
| 異常イベント | Section 6 |
| ログレベル | Section 7 |
| 構造化ログ統一 | Section 4 |

---

## 完了条件

- [ ] `LogToolCall` のラベルが `app`, `module`, `status` のみ（`tool` はデータフィールド）
- [ ] `LogSecurityEvent` のラベルが `app`, `type`, `level` のみ（`event` はデータフィールド）
- [ ] `LogToolCall` に `userID` パラメータがあり、ログデータに `user_id` が含まれる
- [ ] Gateway Secret 検証失敗時に `LogSecurityEvent` が呼ばれる
- [ ] 運用ログは Loki Push のみ（`log.Printf` は起動・初期化・Loki失敗に限定）
- [ ] `go build ./cmd/server` が成功する

---

## 参考

- [dsn-observability.md](../../docs/003_design/observability/dsn-observability.md) - Observability 設計書
- [dsn-security.md](../../docs/003_design/security/dsn-security.md) - セキュリティ設計書
- [day023-worklog.md](./day023-worklog.md) - DAY023 作業ログ
- [sprint007-plan.md](../sprint/sprint007-plan.md) - Sprint 007 計画
