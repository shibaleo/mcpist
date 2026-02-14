# DAY029 作業ログ

## 日付

2026-02-14

---

## コミット一覧

| # | ハッシュ | 時刻 | メッセージ |
|---|---------|------|-----------|
| 1 | 315aa10 | — | docs: remove 65 redundant/empty files and simplify remaining specs (-6800 lines) |
| 2 | (要確認) | — | fix(server): use dedicated JSON-RPC error codes for authz and credit errors |

---

## 完了タスク

### 1. 空テンプレート一括削除 ✅

| ID | タスク | 状態 | 備考 |
|----|--------|------|------|
| D29-001 | dsn-CDS.md, dsn-CKV.md, dsn-DHB.md 削除 | ✅ | 空スタブ |
| D29-002 | dsn-load-management.md 削除 | ✅ | 空ファイル |
| D29-003 | dsn-GCL.md, dsn-GHA.md, dsn-GWY.md 削除 | ✅ | 空スタブ |
| D29-004 | dsn-KYB.md, dsn-RND.md 削除 | ✅ | 空スタブ |
| D29-005 | dsn-SBA.md, dsn-SPG.md, dsn-SVL.md 削除 | ✅ | 空スタブ |
| D29-006 | dsn-VCL.md 削除 | ✅ | 空スタブ |
| D29-007 | adr-b2c-focus.md 削除 | ✅ | ADR スタブ（全 TBD） |
| D29-008 | 005_security/index.md 削除 | ✅ | 空 index、ディレクトリごと消滅 |

### 2. 設計書の統合・簡略化 ✅

| ID | タスク | 状態 | 備考 |
|----|--------|------|------|
| D29-009 | dsn-module-registry.md 削除 | ✅ | dsn-layers.md と重複/陳腐化のため吸収せず純粋に削除 |
| D29-010 | dsn-modules.md 簡略化 | ✅ | 287→66 行。「Why」(Design Decisions, Composite Tool Design) のみ残し、コード例・ファイル構造を削除 |
| D29-011 | dsn-layers.md Routing 更新 | ✅ | `applyFormat` → `ApplyCompact`、呼び出し元を `modules.Run()` → `handler.go` に修正 |
| D29-012 | canvas 棚卸し | ✅ | 9 ファイル中 2 ファイル削除 (grh-observability, grh-rpc-componets-interaction) |

### 3. 仕様書の最小限削減 ✅

| ID | タスク | 状態 | 備考 |
|----|--------|------|------|
| D29-013 | spc-dsn.md Rate Limit 記述削除 | ✅ | 未実装の Rate Limit / Burst 記述を削除 |
| D29-014 | spc-itf.md 未実装記述の削減 | ✅ | `raw_output` フィールド削除、PSP Webhook セクション削除 |

### 4. 仕様書構造の抜本的整理（計画外・追加実施） ✅

| ID | タスク | 状態 | 備考 |
|----|--------|------|------|
| D29-015 | interaction/ ディレクトリ一括削除 | ✅ | 42 ファイル。spc-itf.md が SSoT であり全て重複 |
| D29-016 | details/ 重複ファイル削除 | ✅ | 4 ファイル (dtl-spc-hdl, idx-ept, itf-mod, itf-tvl)。dtl-spc-credit-model.md は「Why」を含むため維持 |
| D29-017 | index.md 重複削除 + idx-spc.md 更新 | ✅ | インデックス重複を解消、削除済みリンクを修正 |
| D29-018 | spc-itf.md / dtl-spc-credit-model.md リンク修正 | ✅ | 削除済みファイルへの参照を除去 |

---

## 作業詳細

### 判断: interaction/ の削除

Sprint 初期に作成したコンポーネント連携仕様 (itr-*.md 16 件 + dtl-itr-*.md 25 件 + spc-itr.md) は、spc-itf.md にまとめた時点で SSoT が移動していた。ペアごとの詳細ファイルは spc-itf.md の部分コピーになっており、片方だけ更新されるリスクがあった。

### 判断: dsn-module-registry.md の扱い

計画では「dsn-layers.md に吸収して削除」だったが、内容を精査した結果:
- System Architecture の図: 実装が SSoT → 吸収不要
- Module Interface / Meta Tools: dsn-layers.md に概念記述済み → 重複
- Migration Path / Legacy: ogen 移行完了で陳腐化
- raw_output: DAY028 で廃止済み

結論: 吸収すべき内容がないため純粋に削除。

### 判断: canvas の棚卸し

9 ファイルを個別に分析し、7 ファイル維持・2 ファイル削除と判断:
- **削除**: grh-observability (陳腐化)、grh-rpc-componets-interaction (grh-rpc-design と重複)
- **維持**: handler-callgraph, infrastructure, user-analysis, rpc-design, table-design, componet-interactions, deployment

---

## 変更サマリ

| カテゴリ | 削除数 | 内訳 |
|----------|--------|------|
| 空テンプレート (003_design) | 15 | dsn-*.md, adr-*.md, index.md |
| 設計書統合 (003_design) | 1 | dsn-module-registry.md |
| canvas 棚卸し (graph) | 2 | observability, rpc-componets-interaction |
| 仕様書構造整理 (002_specification) | 47 | interaction/ 42 + details/ 4 + index.md 1 |
| **合計削除** | **65** | |
| 更新 | 6 | dsn-layers.md, dsn-modules.md, spc-dsn.md, spc-itf.md, idx-spc.md, dtl-spc-credit-model.md |

**docs/ ソースファイル数: 108 → 61 (-44%)**

### 5. 仕様書の実装追従更新 (Phase 1c) ✅

| ID | タスク | 状態 | 備考 |
|----|--------|------|------|
| D29-019 | S8-022: エラーコードポリシー策定 | ✅ | spc-itf.md にエラー体系セクション追加。LLM 対応可能性で 2 カテゴリに分類 |
| D29-020 | S8-023: Console API 簡略化 | ✅ | 実装が SSoT の 1 行に簡略化 |
| D29-021 | S8-025: クレジットモデル仕様書更新 | ✅ | Running Balance パターンの Why 追加、サブスク併存方針追加 |
| D29-022 | spc-itf.md トークンリフレッシュ更新 | ✅ | broker 移行を反映（6 プロバイダ 11 モジュール） |

### 6. JSON-RPC エラーコード実装改善 ✅

| ID | タスク | 状態 | 備考 |
|----|--------|------|------|
| D29-023 | カスタムエラーコード追加 | ✅ | `-32001 PermissionDenied`, `-32002 InsufficientCredit` を types.go に追加 |
| D29-024 | handler.go エラーコード修正 | ✅ | `InvalidRequest` の誤用を修正。authErrorToRPC() ヘルパー追加 |
| D29-025 | spc-itf.md エラーコード対応表追加 | ✅ | JSON-RPC コード 6 種の一覧を追記 |

---

## 未テスト項目

以下は実装変更済みだがローカルテスト未実施。次回テストする。

| 項目 | 対象ファイル | テスト方法 |
|------|-------------|-----------|
| PermissionDenied (-32001) が返ること | handler.go, types.go | 未有効化モジュールで run/batch を実行 |
| InsufficientCredit (-32002) が返ること | handler.go, types.go | クレジット 0 のユーザーで run/batch を実行 |
| batch サイズ超過で InvalidParams (-32602) が返ること | handler.go | 11 コマンドの batch を送信 |
| authErrorToRPC の分岐 | handler.go | CanAccessTool の各エラーケース |

---

## DAY029 サマリ

| 項目 | 内容 |
|------|------|
| テーマ | ドキュメント大幅削減 + エラーコード体系の整理 |
| 削除ファイル数 | 65 ファイル |
| 削除行数 | 約 6,800 行 |
| docs/ ファイル数 | 108 → 61 |
| 実装変更 | JSON-RPC カスタムエラーコード追加 (-32001, -32002) |
| 計画タスク | 14/14 完了 |
| 計画外タスク | 11 追加完了 |

---

## 次回の作業

1. 未コミット docs 変更のコミット
2. 未テスト項目のローカルテスト
3. Sprint-008 Phase 2 着手 (S8-030 panic recovery, S8-036 graceful shutdown, S8-037 ogen タイムアウト)
