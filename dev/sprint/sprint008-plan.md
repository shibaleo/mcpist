# Sprint 008 計画書

## 基本情報

| 項目 | 値 |
|------|-----|
| スプリント番号 | SPRINT-008 |
| 期間 | 2026-02-14 〜 2026-02-20 (7日間) |
| マイルストーン | M7: ドキュメント削減・品質基盤・本番デプロイ |
| 前提 | Sprint-007 完了（20 モジュール ~280 ツール、ogen 全面移行、3 層アーキテクチャ確立、broker 集約） |
| 状態 | 計画中 |

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

---

## タスク一覧

### Phase 1: 設計書の棚卸しと削減（優先度：最高）

目標：**更新ではなく削除**。設計書を減らすことで保守コストを下げる。

#### 1a. 空テンプレート・スタブの一括削除

中身のない設計書を即座に削除する。

| ID | 削除対象 | 理由 |
|----|---------|------|
| S8-001 | dsn-CDS.md | Cloudflare DNS スタブ（中身なし） |
| S8-002 | dsn-CKV.md | Cloudflare KV スタブ（中身なし） |
| S8-003 | dsn-DHB.md | DockerHub スタブ（中身なし） |
| S8-004 | dsn-load-management.md | 空ファイル |
| S8-005 | dsn-GCL.md | Grafana Cloud スタブ（中身なし） |
| S8-006 | dsn-GHA.md | GitHub Actions スタブ（中身なし） |
| S8-007 | dsn-GWY.md | API Gateway スタブ（中身なし） |
| S8-008 | dsn-KYB.md | Koyeb スタブ（中身なし） |
| S8-009 | dsn-RND.md | Render スタブ（中身なし） |
| S8-010 | dsn-SBA.md | Supabase Auth スタブ（中身なし） |
| S8-011 | dsn-SPG.md | Supabase PostgreSQL スタブ（中身なし） |
| S8-012 | dsn-SVL.md | Supabase Vault スタブ（itf-tvl.md で代替） |
| S8-013 | dsn-VCL.md | Vercel Console スタブ（中身なし） |
| S8-014 | adr-b2c-focus.md | ADR スタブ（全 TBD） |
| S8-015 | 005_security/index.md | 空 index |

**削減数: 15 ファイル**

#### 1b. ogen/3層化で陳腐化した設計書の統合・削除

| ID | タスク | 対象 | 方針 |
|----|--------|------|------|
| S8-016 | dsn-module-registry.md 統合 | dsn-module-registry.md | 必要な内容を dsn-layers.md に吸収し削除 |
| S8-017 | dsn-modules.md 簡略化 | dsn-modules.md | 3 層は dsn-layers.md に委譲。モジュール固有情報（認証方式一覧、composite ツール）のみ残す |
| S8-018 | dsn-layers.md Routing 更新 | dsn-layers.md | batch リファクタ (ApplyCompact) を反映 |
| S8-019 | graph/*.canvas 棚卸し | graph/ | 実装と乖離しているものを削除 |

#### 1c. 仕様書の最小限更新

更新するのは「実装と明確に矛盾する」箇所のみ。「未実装機能の記述」は削除または「未実装」と明記。

| ID | タスク | 対象 | 方針 |
|----|--------|------|------|
| S8-020 | Rate Limit 記述削除 | spc-dsn.md | 未実装 → 記述削除（実装時に書く） |
| S8-021 | JWT `aud` チェック現状明記 | spc-itf.md | 1 行で現状を記述 |
| S8-022 | MCP エラーコード簡略化 | spc-itf.md | JSON-RPC 標準のみ。独自拡張の記述削除 |
| S8-023 | Console API 記述削除 | spc-itf.md | REST API 仕様削除（Supabase RPC は実装が仕様） |
| S8-024 | PSP Webhook 簡略化 | spc-itf.md | Phase 1 実装に合わせて削減 |
| S8-025 | credit model 現状記述 | dtl-spc-credit-model.md | running balance 方式を反映 |

### Phase 2: CI/CD 基盤（優先度：高）

設計書よりコードで品質を守る。

| ID | タスク | 成果物 | 備考 |
|----|--------|--------|------|
| S8-030 | Go build + test CI | GitHub Actions | `go build ./...` + `go test ./...` |
| S8-031 | tools.json 検証 CI | GitHub Actions | 再生成して差分なしを確認 |
| S8-032 | Console build CI | GitHub Actions | `pnpm build` pass |

### Phase 3: Observability 仕上げ（優先度：中）

| ID | タスク | 成果物 | 備考 |
|----|--------|--------|------|
| S8-040 | エラー分類とログレベル整理 | Go Server | INFO/WARN/ERROR の基準を実装で表現 |
| S8-041 | Grafana ダッシュボード構築 | Grafana | 主要メトリクス可視化 |
| S8-042 | アラートルール設定 | Grafana | エラーレート閾値アラート |

### Phase 4: 機能実装（優先度：低）

時間があれば着手。

| ID | タスク | 成果物 | 備考 |
|----|--------|--------|------|
| S8-050 | usage_stats 参照 API 実装 | Go Server + Console | 使用量表示 |
| S8-051 | enabled_modules 参照 API 完成 | Go Server | 残作業完了 |

---

## 実装方針

### 作業順序

```
Day 1:   Phase 1a (空テンプレ一括削除) + Phase 1b (統合・簡略化)
Day 2:   Phase 1c (仕様書最小更新)
Day 3-4: Phase 2 (CI/CD)
Day 5:   Phase 3 (Observability)
Day 6-7: Phase 4 (機能) + バッファ
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

- [ ] 空テンプレート 15 ファイル削除済み
- [ ] dsn-modules.md が dsn-layers.md に委譲する形に簡略化
- [ ] spc-dsn.md, spc-itf.md の未実装記述が削除済み
- [ ] GitHub Actions CI が main push で実行される
- [ ] 全テスト pass (go test + console build)
- [ ] docs/ のファイル数が 160 未満（現在 174）

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
