# DAY029 計画

## 日付

2026-02-14

---

## 概要

Sprint-008 初日。Sprint-007 の教訓「設計書を減らす」を実践する。空テンプレートの一括削除から始め、ogen/3層化で陳腐化した設計書を統合・簡略化する。

---

## DAY028 からの引き継ぎ

| 項目 | 状態 |
|------|------|
| batch リファクタ (raw_output 廃止) | ✅ コミット済み |
| 未使用コード削除 (httpclient, mock/auth) | ✅ コミット済み |
| dsn-layers.md (3 層設計書) | ✅ Draft 作成済み |
| 設計書 174 ファイル | ❌ 空テンプレ多数、要削減 |

---

## 本日のタスク

### 1. 空テンプレート一括削除（優先度：最高）

中身のないスタブファイル 15 件を削除。

| ID | 削除対象 | 理由 |
|----|---------|------|
| D29-001 | dsn-CDS.md, dsn-CKV.md, dsn-DHB.md | 空スタブ (Cloudflare DNS/KV, DockerHub) |
| D29-002 | dsn-load-management.md | 空ファイル |
| D29-003 | dsn-GCL.md, dsn-GHA.md, dsn-GWY.md | 空スタブ (Grafana Cloud, GitHub Actions, Gateway) |
| D29-004 | dsn-KYB.md, dsn-RND.md | 空スタブ (Koyeb, Render) |
| D29-005 | dsn-SBA.md, dsn-SPG.md, dsn-SVL.md | 空スタブ (Supabase Auth/PG/Vault) |
| D29-006 | dsn-VCL.md | 空スタブ (Vercel Console) |
| D29-007 | adr-b2c-focus.md | ADR スタブ（全 TBD） |
| D29-008 | 005_security/index.md | 空 index |

### 2. 設計書の統合・簡略化（優先度：高）

| ID | タスク | 方針 |
|----|--------|------|
| D29-009 | dsn-module-registry.md → dsn-layers.md に吸収して削除 | 必要な記述のみ移植 |
| D29-010 | dsn-modules.md 簡略化 | 3 層は dsn-layers.md に委譲。モジュール固有情報のみ残す |
| D29-011 | dsn-layers.md Routing 更新 | batch リファクタ (ApplyCompact) 反映 |
| D29-012 | graph/*.canvas 棚卸し | 実装と乖離しているものを削除 |

### 3. 仕様書の最小限削減（時間があれば）

| ID | タスク | 方針 |
|----|--------|------|
| D29-013 | spc-dsn.md Rate Limit 記述削除 | 未実装 → 削除 |
| D29-014 | spc-itf.md 未実装記述の削減 | Console API, 独自エラーコード等を削除 |

---

## 基本原則

1. **作るよりも削る** — 更新するくらいなら削除できないか考える
2. **実装が仕様** — openapi-subset.yaml と Go コードが SSoT
3. **判断基準**: 「Why」は残す、「What/How」は削除（実装を読めばわかる）

---

## 完了条件

- [ ] 空テンプレート 15 ファイル削除
- [ ] dsn-module-registry.md が dsn-layers.md に統合され削除
- [ ] dsn-modules.md が簡略化
- [ ] dsn-layers.md が batch リファクタを反映
- [ ] docs/ ファイル数が 160 未満

---

## 参考

- [sprint008-plan.md](../sprint/sprint008-plan.md) - Sprint 008 計画
- [day028-worklog.md](./day028-worklog.md) - DAY028 作業ログ
- [dsn-layers.md](../../docs/003_design/modules/dsn-layers.md) - 3 層アーキテクチャ設計書（SSoT）
