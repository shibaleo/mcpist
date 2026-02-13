# DAY028 作業ログ

## 日付

2026-02-13

---

## コミット一覧

| # | ハッシュ | 時刻 | メッセージ |
|---|---------|------|-----------|
| 1 | `a41b01a` | 23:58 | refactor(server): consolidate OAuth2 token refresh into broker package |
| 2 | (未コミット) | — | refactor(server): separate compact format from Run(), fix batch double-conversion and variable resolution |

---

## 完了タスク

### 1. format 層分離リファクタ ✅

| ID | タスク | 状態 | 備考 |
|----|--------|------|------|
| D28-001 | `modules.Run()` に format 分岐を追加 | ✅ | DAY027 で実施 |
| D28-002 | format.go を純粋な変換関数に書き換え | ✅ | DAY027 で実施 |
| D28-003 | 全ハンドラから format 呼び出しを除去 | ✅ | DAY027 で実施 |
| D28-004 | format パラメータを toolDefinitions から除去 | ✅ | DAY027 で実施 |
| D28-005 | ビルド・テスト・動作確認 | ✅ | DAY027 で実施 |

### 2. format.go 追加 (6 モジュール) ✅

| ID | タスク | 状態 | 備考 |
|----|--------|------|------|
| D28-006 | Jira format.go | ✅ | 11 tools |
| D28-007 | Confluence format.go | ✅ | 12 tools |
| D28-008 | GitHub format.go | ✅ | 26 tools |
| D28-009 | Asana format.go | ✅ | 23 tools |
| D28-010 | Supabase format.go | ✅ | 18 tools |
| D28-011 | Grafana format.go | ✅ | 16 tools |

### 3. 残り 9 モジュール ogen 移行 ✅

| ID | タスク | 状態 | 備考 |
|----|--------|------|------|
| D28-012 | Dropbox | ✅ | Bearer (OAuth2), 15 tools |
| D28-013 | Airtable | ✅ | Bearer (PAT), 11 tools |
| D28-014 | Google Calendar | ✅ | Bearer (OAuth2), 8 tools |
| D28-015 | Google Tasks | ✅ | Bearer (OAuth2), 9 tools |
| D28-016 | Microsoft Todo | ✅ | Bearer (OAuth2), 11 tools |
| D28-017 | Google Docs | ✅ | Bearer (OAuth2), 18 tools |
| D28-018 | Google Drive | ✅ | Bearer (OAuth2), 22 tools |
| D28-019 | Google Sheets | ✅ | Bearer (OAuth2), 27 tools |
| D28-020 | Google Apps Script | ✅ | Bearer (OAuth2), 17 tools |

### 4. ツール定義テスト ✅

| ID | タスク | 状態 | 備考 |
|----|--------|------|------|
| D28-021 | 全モジュールのツール定義テスト | ✅ | Description 非空、toolHandlers と toolDefinitions の一致確認 |

### 5. OAuth2 リフレッシュ共通化 + store → broker リネーム ✅

| ID | タスク | 状態 | 備考 |
|----|--------|------|------|
| D28-022 | `internal/store/` → `internal/broker/` リネーム | ✅ | パッケージ名・ディレクトリ移動 |
| D28-023 | `OAuthRefreshConfig` テーブル + `refreshOAuthToken()` 実装 | ✅ | 6 プロバイダ × 11 モジュール対応 |
| D28-024 | `GetModuleToken()` にリフレッシュ統合 | ✅ | fetchCredentials → needsRefresh → refresh の透過処理 |
| D28-025 | 全 23 ファイルの import `store` → `broker` 更新 | ✅ | `GetTokenStore()` → `GetTokenBroker()` |
| D28-026 | 11 モジュールから refreshToken/needsRefresh 削除 | ✅ | +679/-1,306 行 |
| D28-027 | ローカルサーバーテスト | ✅ | 13 モジュール正常応答、asana/airtable でリフレッシュ動作確認 |

### 6. batch リファクタ: raw_output 廃止 + 二重変換バグ修正 ✅

| ID | タスク | 状態 | 備考 |
|----|--------|------|------|
| D28-028 | `ApplyCompact()` 公開関数追加 | ✅ | `modules.ApplyCompact(module, tool, json)` |
| D28-029 | `Run()` から compact 変換を削除 | ✅ | Run() は常に JSON を返すように |
| D28-030 | `handleRun` で compact 変換を適用 | ✅ | handler.go に format チェック追加 |
| D28-031 | `BatchCommand.RawOutput` フィールド削除 | ✅ | `raw_output` 廃止 |
| D28-032 | `Batch()` 結果構築を修正 | ✅ | `output: true` 時に params.format チェック |
| D28-033 | batch/run description に `[Response Format]` 追加 | ✅ | ja-JP / en-US 両方、`raw_output` 記述削除 |
| D28-034 | `resolveStringVariables` の JSON 配列対応 | ✅ | Run() が常に JSON を返すため、配列直接パースに対応 |
| D28-035 | ローカルサーバーテスト | ✅ | run compact/json、batch compact/json、変数参照すべて正常 |

---

## 作業詳細

### 1. OAuth2 リフレッシュ共通化

11 モジュールに散在していた `refreshToken()` / `needsRefresh()` を `broker/token.go` の `OAuthRefreshConfig` テーブル駆動に集約。

| プロバイダ | 対象モジュール |
|-----------|--------------|
| Google | Google Calendar, Tasks, Docs, Drive, Sheets, Apps Script |
| Microsoft | Microsoft Todo |
| Dropbox | Dropbox |
| Asana | Asana |
| Todoist | Todoist |
| Trello | Trello |

`GetModuleToken()` が透過的にリフレッシュを行い、更新後のトークンを DB に保存する。

### 2. batch リファクタ

**問題:**
- `Run()` 内で compact 変換 → `Batch()` の結果構築で再度 `ToCompact()` → **二重変換**
- `raw_output: true` は `Run()` で既に compact 化された結果を返すだけで**壊れていた**
- `resultStore` に compact テキストが入るため、変数参照 (`${id.results[N].field}`) の JSONPath 解決が**不可能**だった

**修正:**
- `Run()` は常に JSON を返す（compact 変換を削除）
- `ApplyCompact()` 公開関数を追加し、呼び出し側の責務に
- `handleRun` で compact 変換を適用（`format=json` なら素通し）
- `Batch()` では `output: true` 時に `params.format` をチェックして compact/JSON を選択
- `raw_output` フィールドを完全廃止（`output: true` + `params.format: "json"` が代替）
- `resolveStringVariables` が JSON 配列を直接パースできるよう修正

**レスポンス形式（修正後）:**

| ツール | `output` | `params.format` | レスポンス |
|--------|----------|----------------|-----------|
| `run` | — | なし (デフォルト) | compact (CSV/MD) |
| `run` | — | `"json"` | 生 JSON |
| `batch` | なし | — | レスポンスに含めない（内部 JSON 保持のみ） |
| `batch` | `true` | なし (デフォルト) | compact (CSV/MD) |
| `batch` | `true` | `"json"` | 生 JSON |

### 3. 変数参照の修正

`Run()` が常に JSON を返すことで、`resultStore` に生 JSON が格納されるようになった。これにより:

- JSON 配列 `[{...},{...}]` → `${id.results[N].field}` で直接インデックスアクセス可能
- 以前は compact テキスト (CSV) が格納されており、JSONPath 解決は事実上壊れていた

`resolveStringVariables` を修正し、JSON 配列と `{"results":[...]}` ラッパーの両方をサポート。

---

## 変更ファイル (未コミット分)

| ファイル | 変更内容 |
|----------|----------|
| `internal/modules/modules.go` | `Run()` から compact 削除、`ApplyCompact()` 追加、`BatchCommand.RawOutput` 削除、`Batch()` 結果構築修正、`resolveStringVariables` JSON 配列対応、run/batch description 更新 |
| `internal/mcp/handler.go` | `handleRun` に compact 変換追加 |
| `apps/console/src/lib/tools.json` | tools.json 再生成 |

---

## DAY028 サマリ

| 項目 | 内容 |
|------|------|
| ogen 移行 | **全 19/19 モジュール完了** (DAY026: 4, DAY027: 6, DAY028: 9) |
| format 層分離 | 全 19 モジュールに format.go 追加、ハンドラから format 呼び出し除去 |
| OAuth2 リフレッシュ | broker に集約、11 モジュールから重複コード削除 (+679/-1,306 行) |
| store → broker | パッケージリネーム完了 (23 ファイル) |
| batch リファクタ | raw_output 廃止、二重変換バグ修正、変数参照修正 |
| テスト | ツール定義テスト作成 (全 PASS)、ローカルサーバー 13+ モジュール動作確認 |
| コミット数 | 1 (+ 未コミット 1) |

---

## 次回の作業

1. 未コミット変更のコミット — batch リファクタ + format 層分離
2. dsn-modules.md の 3 層アーキテクチャ整合 — dsn-layers.md との整合
3. 仕様書更新 (S7-020〜026) — Sprint-007 Phase 3 残タスク
4. クレジットモデル仕様書更新 — ランニングバランス方式への更新
