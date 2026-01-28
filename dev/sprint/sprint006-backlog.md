# Sprint 006 バックログ

Sprint-006 のスコープから外したタスク。Phase 1〜2 完了後に着手予定。

---

## Phase 3: テスト基盤構築

| ID | タスク | 成果物 | 備考 |
|----|--------|--------|------|
| S6-020 | E2E テスト設計書作成 | tst-e2e.md | テスト対象フロー、テスト環境、実行方法 |
| S6-021 | Go Server ユニットテスト | *_test.go | handler, middleware, modules の主要パス |
| S6-022 | Batch 権限チェックのテスト | handler_test.go | All-or-Nothing、クレジット不足、正常系 |
| S6-023 | Console ビルドテスト | CI workflow | `pnpm next build` が CI で通ることを保証 |

---

## Phase 4: CI/CD 整備

| ID | タスク | 成果物 | 備考 |
|----|--------|--------|------|
| S6-030 | tools.json 検証 CI | GitHub Actions | Go tools-export 出力と Console tools.json の差分チェック |
| S6-031 | Go lint + test CI | GitHub Actions | golangci-lint + go test |
| S6-032 | Console lint + build CI | GitHub Actions | eslint + next build |

---

## Phase 5: コード品質改善

| ID | タスク | 備考 |
|----|--------|------|
| S6-040 | 未使用 Go モジュール削除 | S5-077 からの引き継ぎ |
| S6-041 | spec-impl-compare.md の更新 | Phase 2 完了後に差分再チェック |

---

## Phase 6: Sprint-005 残タスク

| ID | タスク | 備考 |
|----|--------|------|
| S6-050 | UI 要求仕様書作成 (spc-ui.md) | S5-060。画面一覧・機能要件 |
| S6-051 | ユーザーフロー図作成 | S5-061。主要フローの可視化 |
| S6-052 | 画面遷移図作成 | S5-062。認証後のナビゲーション |

---

## 完了条件（移動分）

### Phase 3: テスト基盤
- [ ] E2E テスト設計書が存在する
- [ ] Go Server の handler / middleware に最低限のユニットテストが存在する
- [ ] Console の `next build` が CI で自動実行される

### Phase 4: CI/CD
- [ ] PR 作成時に Go lint + test が自動実行される
- [ ] PR 作成時に Console build が自動実行される
- [ ] tools.json の整合性が CI で検証される

---

## 継続バックログ

Sprint-006 のスコープ外だが、記録しておくタスク。

| ID | タスク | 状態 | 備考 |
|----|--------|------|------|
| BL-002 | 切断時のツール設定クリーンアップ | 保留 | DB migration 前提 |
| BL-030 | Stg/Prd 環境構築 | 未着手 | Blue-Green 方式 |
| BL-032 | 追加モジュール（Slack/Linear） | 未着手 | 需要に応じて |
