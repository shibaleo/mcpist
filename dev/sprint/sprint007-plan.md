# Sprint 007 計画書

## 基本情報

| 項目 | 値 |
|------|-----|
| スプリント番号 | SPRINT-007 |
| 期間 | 2026-02-03 〜 2026-02-09 (7日間) |
| マイルストーン | M6: Observability・セキュリティ・仕様整備 |
| 前提 | Sprint-006 完了（18モジュール248ツール、Stripe Phase 1、prompts MCP） |
| 状態 | 計画中 |

---

## Sprint 目標

**Observability 基盤の実装、セキュリティ設計の文書化、仕様書の実装追従更新**

Sprint-006 でモジュール数が急増（8→18）し、ツール数も倍増（115→248）した。運用可視化とセキュリティ対策を整備し、仕様書を実装に追従させることで、保守性と信頼性を向上させる。

---

## 背景

### Sprint-006 で蓄積した技術的負債

| 項目 | 状態 | リスク |
|------|------|--------|
| Observability 未整備 | ログ出力のみ | 本番障害時の調査が困難 |
| セキュリティ設計書なし | 実装に散在 | 監査対応・新規メンバーへの説明が困難 |
| 仕様書の乖離 | 5項目未修正 | 仕様書が信頼できない |

### 優先度の根拠

1. **Observability**: 18モジュール・248ツールの運用には可視化が必須
2. **セキュリティ**: OAuth 1.0a/2.0、PKCE、多層防御の設計を文書化
3. **仕様書整備**: 実装と乖離した仕様書は混乱の元

---

## タスク一覧

### Phase 1: Observability 実装

現在の Loki + X-Request-ID 基盤を拡張し、運用に必要な可視化を実現する。

| ID | タスク | 成果物 | 備考 |
|----|--------|--------|------|
| S7-001 | Observability 設計書作成 | dsn-observability.md | ログ構造、メトリクス、トレーシング設計 |
| S7-002 | 構造化ログの統一 | Go Server | JSON 形式、共通フィールド（request_id, user_id, module, tool） |
| S7-003 | ツール実行ログに user_id 追加 | Go Server | BL-019 対応 |
| S7-004 | invalid_gateway_secret ログ実装 | Go Server | BL-020 対応、セキュリティイベント |
| S7-005 | エラー分類とログレベル整理 | Go Server | INFO/WARN/ERROR の基準明確化 |
| S7-006 | Grafana ダッシュボード設計 | dsn-observability.md | 主要メトリクスの可視化計画 |

### Phase 2: セキュリティ設計書作成

実装済みのセキュリティ対策を文書化し、監査可能な状態にする。

| ID | タスク | 成果物 | 備考 |
|----|--------|--------|------|
| S7-010 | セキュリティ設計書作成 | dsn-security.md | BL-080 対応 |
| S7-011 | 認証・認可フロー整理 | dsn-security.md | Supabase Auth, JWT, Gateway Secret |
| S7-012 | OAuth セキュリティ整理 | dsn-security.md | OAuth 2.0, OAuth 1.0a, PKCE, state パラメータ |
| S7-013 | データ保護整理 | dsn-security.md | Supabase Vault, RLS, 暗号化 |
| S7-014 | SSRF 対策整理 | dsn-security.md | PostgreSQL localhost 禁止、外部 API 呼び出し制限 |
| S7-015 | セキュリティチェックリスト作成 | dsn-security.md | 新規モジュール追加時の確認項目 |

### Phase 3: 仕様書の実装追従更新

Sprint-006 で未着手だった仕様書整備を完了する。

| ID | タスク | 対象ファイル | 備考 |
|----|--------|-------------|------|
| S7-020 | Rate Limit 記述の更新 | spc-dsn.md | BL-010: 「将来実装予定」に変更 |
| S7-021 | JWT `aud` チェック要件整理 | spc-itf.md | BL-011: 現状と方針を明記 |
| S7-022 | MCP 拡張エラーコード整理 | spc-itf.md | BL-012: JSON-RPC 標準コードのみに更新 |
| S7-023 | Console API 設計更新 | spc-itf.md | BL-013: REST API → Supabase RPC 方式 |
| S7-024 | PSP Webhook 仕様整理 | spc-itf.md | BL-014: Phase 1 実装に合わせて更新 |
| S7-025 | credentials JSON 構造整理 | dtl-itr-MOD-TVL.md | BL-090: 仕様を実装に合わせて更新 |
| S7-026 | spec-impl-compare.md 更新 | spec-impl-compare.md | D16-002: 差分解消確認 |

### Phase 4: 機能実装（Observability 関連）

| ID | タスク | 備考 | 状態 |
|----|--------|------|------|
| S7-030 | usage_stats 参照 API 実装 | BL-017: Console で使用量表示 | 未着手 |
| S7-031 | enabled_modules 参照 API 完成 | BL-015: 残作業完了 | 進行中 |

---

## 完了条件

### Phase 1: Observability
- [ ] dsn-observability.md が存在し、ログ構造・メトリクス・ダッシュボード設計が記載
- [ ] Go Server のログが JSON 形式で統一され、request_id, user_id, module, tool が含まれる
- [ ] invalid_gateway_secret がログに記録される

### Phase 2: セキュリティ
- [ ] dsn-security.md が存在し、認証・認可・データ保護・SSRF 対策が記載
- [ ] 新規モジュール追加時のセキュリティチェックリストが存在

### Phase 3: 仕様書
- [ ] BL-010〜014, BL-090 の6項目が全て resolved
- [ ] spec-impl-compare.md の未解決項目がゼロ

### Phase 4: 機能実装
- [ ] Console で使用量統計が表示できる（usage_stats API）

---

## タイムライン（目安）

| 日 | タスク |
|----|--------|
| DAY023 | Phase 1: Observability 設計書・構造化ログ (S7-001〜003) |
| DAY024 | Phase 1: Observability 残り (S7-004〜006) |
| DAY025 | Phase 2: セキュリティ設計書 (S7-010〜012) |
| DAY026 | Phase 2: セキュリティ設計書 (S7-013〜015) |
| DAY027 | Phase 3: 仕様書更新 (S7-020〜023) |
| DAY028 | Phase 3: 仕様書更新 (S7-024〜026) |
| DAY029 | Phase 4: 機能実装・バッファ |

---

## 次Sprint への引き継ぎ候補

本 Sprint で対応しない項目（優先度低または時間不足の場合）:

### テスト基盤
| ID | タスク | 備考 |
|----|--------|------|
| S6-020 | E2E テスト設計書作成 | tst-e2e.md |
| S6-021 | Go Server ユニットテスト | *_test.go |
| D16-006 | E2Eテスト設計 | OAuth認可フロー等 |

### CI/CD 整備
| ID | タスク | 備考 |
|----|--------|------|
| S6-030 | tools.json 検証 CI | GitHub Actions |
| S6-031 | Go lint + test CI | GitHub Actions |
| S6-032 | Console lint + build CI | GitHub Actions |

### 開発基盤
| ID | タスク | 備考 |
|----|--------|------|
| BL-070 | database.types.ts 自動生成フロー整備 | CI/CD組み込み |
| BL-085 | ユーザートークン保管方式の見直し | Supabase Vault は運営用 |

---

## 参考

- [sprint006.md](./sprint006.md) - Sprint 006 計画書
- [sprint006-review.md](./sprint006-review.md) - Sprint 006 レビュー
- [sprint006-backlog.md](./sprint006-backlog.md) - Sprint 006 残課題
- [spec-impl-compare.md](../../docs/spec-impl-compare.md) - 仕様・実装差分
