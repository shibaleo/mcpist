# Sprint 008 計画書

## 基本情報

| 項目 | 値 |
|------|-----|
| スプリント番号 | SPRINT-008 |
| 期間 | 2026-02-14 〜 2026-02-20 (7日間) |
| マイルストーン | M7: ドキュメント削減・品質基盤・本番デプロイ |
| 前提 | Sprint-007 完了（20 モジュール ~280 ツール、ogen 全面移行、3 層アーキテクチャ確立、broker 集約） |
| 状態 | **完了** |

---

## Sprint 目標

**作るよりも削る。設計書を減らし、CI で品質を守る**

Sprint-007 の最大の教訓は「設計書が減ったのは良いこと」。ogen の openapi-subset.yaml と 3 層アーキテクチャにより「spec = 実装 = 設計書」が成立した今、手書き設計書の大半は負債でしかない。仕様書を「追従させる」のではなく、不要なものを削除し、残すべきものだけ最小限に維持する。

---

## 背景

### Sprint-007 の教訓（修正版）

1. **設計書を減らす努力が最重要** — 3 層分離で「spec = 実装 = 設計書」が成立。手書き設計書を維持するコストは負債
2. **自前実装より外部依存のバージョン固定** — httpclient 手書き → ogen 生成で証明済み。設計書も同じ発想で「書かずに済む」方向へ
3. **大規模リファクタの連鎖** — スコープを明確に区切る

### 現在のドキュメント状態

| カテゴリ | ファイル数 | 状態 |
|---------|-----------|------|
| 001_requirements | 7 | 安定（変更不要） |
| 002_specification | 73 | 一部乖離あり |
| 003_design | 78 | **空テンプレート多数、ogen で陳腐化** |
| 004_test | 4 | 概ね有効 |
| 005_security | 1 | 空（index のみ） |
| 006_operation | 2 | 有効 |
| graph | 9 | 不明（要確認） |
| **合計** | **174** | — |

### コードベース監査結果（Sprint 008 開始時点）

#### セキュリティ: B+

| 領域 | 状態 | リスク |
|------|------|--------|
| 認証 (JWT → Worker 委譲) | 実装済み | 低 |
| Gateway Secret 検証 | 実装済み | 低 |
| 認可 (ホワイトリスト方式) | 実装済み | 低 |
| OAuth2 トークン管理 (broker) | 実装済み、自動リフレッシュ | 低 |
| トークン暗号化 (Supabase Vault) | 委譲済み | 低 |
| パラメータバリデーション | 型・必須チェック済み | 低 |
| SQL インジェクション (pgx) | パラメータ化クエリ + DDL 制限 | 非常に低 |
| SSRF | localhost ブロック済み | 低 |
| Rate Limiting | スライディングウィンドウ 10req/s | 低 |
| クレジット制御 | 冪等消費 (request ID) | 低 |
| パニックハンドリング | ✅ 実装済み (recovery middleware) | 低 |
| CORS | `*` 全許可（意図的設計、MCP クライアントは非ブラウザ） | 低 |
| **セキュリティヘッダー** | **CSP/HSTS 等なし** | **中 → 次 Sprint** |

#### Observability: B

| 領域 | 状態 | 備考 |
|------|------|------|
| 構造化ログ (Loki) | 実装済み | JSON push、非同期、カーディナリティ管理良好 |
| Request ID 伝播 | 実装済み | Worker → Server → ログまで一貫 |
| ツール実行ログ | 実装済み | user_id, module, tool, duration_ms, status |
| セキュリティイベントログ | ✅ 実装済み | invalid_gateway_secret + permission_denied + credit 失敗 |
| ヘルスチェック | ✅ DB 接続チェック追加 | Supabase 障害時 503 返却 |
| ログレベル区分 | ✅ 実装済み | info/error/warn を level ラベルで付与 |
| 監査ログ | ✅ 実装済み | run/batch の permission denied, credit 消費失敗を Loki 送信 |
| Grafana アラート | ✅ 3 ルール設定済み | Error Rate, Security Events, Log Silence |
| Prometheus メトリクス | なし | Loki LogQL で代替可能、優先度低 |
| 分散トレーシング | なし | OTEL 依存は ogen 間接依存のみ、優先度低 |

#### 可用性: C+

| 領域 | 状態 | 深刻度 |
|------|------|--------|
| ステートレス設計 | 概ね達成（セッション・キャッシュは揮発性） | — |
| グレースフルシャットダウン | ✅ 実装済み（SIGTERM + 30s タイムアウト） | 低 |
| ヘルスチェック | ✅ DB 接続チェック追加 | 低 |
| SSE セッション管理 | 切断時クリーンアップあり、バッファ溢れでサイレントドロップ | 中 |

#### 堅牢性: C

| 領域 | 状態 | 深刻度 |
|------|------|--------|
| **Supabase 障害時** | **全ユーザーブロック、リトライなし、30s キャッシュのみ** | **Critical（次Sprint）** |
| 外部 API タイムアウト | ✅ ogen クライアントにタイムアウト設定済み | 低 |
| リトライ・バックオフ | 一切なし | 高（次Sprint） |
| サーキットブレーカー | なし | 中（次Sprint） |
| Loki 障害時 | 非同期・ベストエフォート（影響なし） | 低 |
| クレジット消費失敗時 | ログのみで実行続行（悪用リスク） | 中（次Sprint） |
| batch 部分失敗 | 依存タスクのみスキップ（正しい動作） | 低 |

#### 耐障害性: C-

| 領域 | 状態 | 深刻度 |
|------|------|--------|
| Rate Limiter | インスタンス独立（マルチインスタンスで回避可能） | 中（次Sprint） |
| パニックハンドリング | ✅ recovery middleware | 低 |
| コンテキスト伝播 | 適切に伝播 | 低 |
| Mutex 使用 | RateLimiter, UserCache, Sessions 全て適切 | 低 |

#### 可搬性: B-

| サービス | 置き換え難易度 | 見積もり |
|---------|-------------|---------|
| **Supabase (DB + Auth + Vault)** | **高** | 2-3 週間 |
| Cloudflare Workers (Gateway) | 中 | 1-2 週間 |
| Render / Koyeb (コンピュート) | 低 | < 1 日 |
| Vercel (Console) | 低〜中 | 1 週間 |
| Grafana Loki | 低 | 0（オプショナル） |

Go サーバーは Docker コンテナでコンピュート層は完全にポータブル。最大のロックインは Supabase RPC 関数群。

---

## タスク一覧

### Phase 1: 設計書の棚卸しと削減（優先度：最高）

目標：**更新ではなく削除**。設計書を減らすことで保守コストを下げる。

#### 1a. 空テンプレート・スタブの一括削除 ✅

中身のない設計書を即座に削除する。**全 15 ファイル削除済み。**

| ID | 削除対象 | 状態 |
|----|---------|------|
| S8-001 | dsn-CDS.md | ✅ 削除済み |
| S8-002 | dsn-CKV.md | ✅ 削除済み |
| S8-003 | dsn-DHB.md | ✅ 削除済み |
| S8-004 | dsn-load-management.md | ✅ 削除済み |
| S8-005 | dsn-GCL.md | ✅ 削除済み |
| S8-006 | dsn-GHA.md | ✅ 削除済み |
| S8-007 | dsn-GWY.md | ✅ 削除済み |
| S8-008 | dsn-KYB.md | ✅ 削除済み |
| S8-009 | dsn-RND.md | ✅ 削除済み |
| S8-010 | dsn-SBA.md | ✅ 削除済み |
| S8-011 | dsn-SPG.md | ✅ 削除済み |
| S8-012 | dsn-SVL.md | ✅ 削除済み |
| S8-013 | dsn-VCL.md | ✅ 削除済み |
| S8-014 | adr-b2c-focus.md | ✅ 削除済み |
| S8-015 | 005_security/index.md | ✅ 削除済み |

**削減数: 15 ファイル**

#### 1b. ogen/3層化で陳腐化した設計書の統合・削除 ✅

| ID | タスク | 状態 | 備考 |
|----|--------|------|------|
| S8-016 | dsn-module-registry.md 統合 | ✅ | Sprint 007 で削除済み |
| S8-017 | dsn-modules.md 簡略化 | ✅ | Sprint 007 で dsn-layers.md 委譲済み |
| S8-018 | dsn-layers.md Routing 更新 | ✅ | batch ApplyCompact 反映済み |
| S8-019 | graph/*.canvas 棚卸し | ✅ | 7 ファイル確認。Obsidian Canvas (JSON) のためテキスト更新不可。現状維持 |

#### 1c. 仕様書の最小限更新 ✅

| ID | タスク | 状態 | 備考 |
|----|--------|------|------|
| S8-020 | Rate Limit 記述削除 | ✅ | spc-dsn.md に Rate Limit 専用セクションなし（対応不要） |
| S8-021 | JWT `aud` チェック現状明記 | ✅ | spc-itf.md に JWT aud 記述なし（対応不要） |
| S8-022 | MCP エラーコード簡略化 | ✅ | spc-itf.md 既に最小限（対応不要） |
| S8-023 | Console API 記述削除 | ✅ | 「実装が SSoT」と明記済み（対応不要） |
| S8-024 | PSP Webhook 簡略化 | ✅ | PSP セクションなし（対応不要） |
| S8-025 | credit model 現状記述 | ✅ | running balance 方式を反映済み |

### Phase 2: セキュリティ・堅牢化（優先度：高） ✅

| ID | タスク | 状態 | 備考 |
|----|--------|------|------|
| S8-030 | panic recovery ミドルウェア追加 | ✅ | Sprint 007 末で実装済み |
| S8-031 | セキュリティヘッダー追加 | 🔜 | 次 Sprint へ繰越（Worker 側で付与が理想） |
| S8-032 | CORS 制限検討 | ✅ | 意図的な設計と確認。MCP クライアントは非ブラウザのため `*` を維持（spc-itf.md に記載済み） |
| S8-036 | グレースフルシャットダウン実装 | ✅ | Sprint 007 末で実装済み（SIGTERM + 30s タイムアウト） |
| S8-037 | ogen クライアント HTTP タイムアウト設定 | ✅ | Sprint 007 末で実装済み |

### Phase 3: CI/CD 基盤（優先度：高） ✅

| ID | タスク | 状態 | 備考 |
|----|--------|------|------|
| S8-033 | Go lint + build + test CI | ✅ | golangci-lint + `go build` + `go test -race` 全 pass |
| S8-034 | tools.json 検証 CI | 🔜 | 次 Sprint へ繰越 |
| S8-035 | Console lint + build CI | ✅ | ESLint + `pnpm build` pass |

CI トリガーは `workflow_dispatch`（手動実行のみ）に変更。

### Phase 4: Observability 仕上げ（優先度：中） ✅

| ID | タスク | 状態 | 備考 |
|----|--------|------|------|
| S8-040 | ツールログに level フィールド追加 | ✅ | `LogToolCall` に info/error、`LogRequest` に info を付与 |
| S8-041 | アクセス拒否・クレジット消費の監査ログ | ✅ | `LogSecurityEvent` で run/batch の permission denied, credit 失敗を Loki 送信 |
| S8-042 | /health に DB 接続チェック追加 | ✅ | Supabase HEAD → 503 `{"status":"degraded"}` 返却 |
| S8-043 | Grafana アラートルール設定 | ✅ | MCP ツールで 3 ルール作成: Error Rate, Security Events, Log Silence |

### Phase 5: 機能実装（優先度：低） ✅

| ID | タスク | 状態 | 備考 |
|----|--------|------|------|
| S8-050 | usage_stats 参照 API 実装 | ✅ | Console が Supabase RPC (`get_my_usage`) を直接呼出。Go Server API 不要 |
| S8-051 | enabled_modules 参照 API 完成 | ✅ | Console が Supabase RPC (`get_user_context`) を直接呼出。Go Server API 不要 |

---

## 実装方針

### 作業順序

```
Day 1:   Phase 1a (空テンプレ一括削除) + Phase 1b (統合・簡略化)
Day 2:   Phase 1c (仕様書最小更新) + Phase 2 (セキュリティ堅牢化)
Day 3-4: Phase 3 (CI/CD)
Day 5:   Phase 4 (Observability)
Day 6-7: Phase 5 (機能) + バッファ
```

### 基本原則

1. **作るよりも削る** — 設計書を更新するくらいなら削除できないか考える
2. **実装が仕様** — ogen の openapi-subset.yaml と Go コードが Single Source of Truth。手書き設計書はそれを補足する場合のみ
3. **外部依存 > 自前実装** — ogen 生成コードはバージョン固定で管理。手書きコードの設計書を書く必要がない
4. **CI で守る** — 設計書ではなく自動テストで品質を担保

### Phase 1 の判断基準

設計書を残すかどうかの判断：

| 残す | 削除する |
|------|---------|
| 実装から読み取れない「Why」 | 実装を読めばわかる「What/How」 |
| 外部サービスの設定手順 | 空テンプレート・スタブ |
| DB スキーマ・RPC 契約 | ogen 生成コードの使い方 |
| ADR（判断理由の記録） | 未実装機能の詳細仕様 |

---

## 完了条件

- [x] 空テンプレート 15 ファイル削除済み
- [x] dsn-modules.md が dsn-layers.md に委譲する形に簡略化（Sprint 007 で対応済みと判明）
- [x] spc-dsn.md, spc-itf.md の未実装記述が削除済み（Sprint 007 で対応済みと判明）
- [x] panic recovery ミドルウェアが追加されている（Sprint 007 末で対応済み）
- [x] グレースフルシャットダウンが実装されている（Sprint 007 末で対応済み）
- [x] ogen クライアントに HTTP タイムアウトが設定されている（Sprint 007 末で対応済み）
- [x] GitHub Actions CI が workflow_dispatch で実行される（push/PR トリガーは無効化）
- [x] 全テスト pass (go test + console build) — CI 6 ジョブ全 pass 確認済み
- [x] ツールログに level フィールドが追加されている
- [x] Grafana にアラートルールが設定されている（MCP ツールで 3 ルール作成済み）
- [x] /health に DB 接続チェックが追加されている
- [x] アクセス拒否・クレジット消費失敗の監査ログが Loki に送信される
- [x] docs/ の .md ファイル数が削減済み（15 ファイル削除、現在 52 ファイル）

---

## リスク

| リスク | 影響 | 対策 |
|--------|------|------|
| 削除しすぎて必要な情報を失う | 復元が必要 | git で復元可能。判断基準を明確にして機械的に判断 |
| CI 構築でハマる | Phase 3-4 に着手できない | 最小構成 (build + test のみ) で始める |

---

## 参考

- [sprint007-review.md](./sprint007-review.md) - Sprint 007 レビュー
- [sprint007-backlog.md](./sprint007-backlog.md) - Sprint 007 バックログ
- [dsn-layers.md](../../docs/003_design/modules/dsn-layers.md) - 3 層アーキテクチャ設計書（SSoT）
