# MCPist テスト仕様書（spc-tst）

## ドキュメント管理情報

| 項目 | 値 |
|------|-----|
| Status | `draft` |
| Version | v1.0 (DAY8) |
| Note | Test Specification |

---

## 概要

本ドキュメントは、MCPistの品質保証に関するテスト戦略・テスト項目を定義する。

---

## テスト戦略概要

### テストピラミッド

```
          ┌─────────┐
          │  E2E    │  少数・高コスト・遅い
          ├─────────┤
          │ 結合    │  中程度
          ├─────────┤
          │ 単体    │  多数・低コスト・速い
          └─────────┘
```

| レベル | 対象 | 実行タイミング | ツール |
|--------|------|---------------|--------|
| 単体 | 関数・メソッド | ローカル、CI | go test, vitest |
| 結合 | コンポーネント間連携 | CI | go test + testcontainers |
| E2E | システム全体 | リリース前 | Playwright |

### テスト環境

| 環境 | 用途 | データ |
|------|------|--------|
| ローカル | 開発中の単体・結合テスト | モック・テストDB |
| CI (GitHub Actions) | PR時の自動テスト | モック・テストDB |
| production | 定期ヘルスチェック | 本番データ（読み取りのみ） |

**備考:**
- Phase 1ではステージング環境なし（本番のみ）
- ユーザー数増加に応じてステージング環境を検討

---

## 単体テスト（Unit Test）

**TBD**: 詳細設計完了後に定義

### 対象コンポーネント（予定）

- MCP Server（Go）: `internal/mcp`, `internal/modules`, `internal/auth`, `internal/entitlement`, `internal/vault`
- User Console（TypeScript/Next.js）: `lib/validators`, `lib/supabase`, `components/*`
- API Gateway（Cloudflare Worker）: `src/index.ts`

### カバレッジ目標

| 対象 | 目標カバレッジ |
|------|---------------|
| 全体 | 70%以上 |

---

## 結合テスト（Integration Test）

### 対象

| テスト対象 | 検証内容 |
|-----------|---------|
| SRV ↔ TVL | トークン取得・リフレッシュ |
| SRV ↔ ENT | 権限チェック、Quota/Credit処理 |
| SRV ↔ EXT | モジュール別外部API呼び出し |
| CON ↔ AUS | 認証・セッション管理 |
| CON ↔ ENT | ユーザー設定取得・更新 |

### 外部依存のモック化

| 依存先 | モック方法 |
|--------|-----------|
| Supabase Auth | テスト用JWTを静的生成 |
| Supabase DB | testcontainers (PostgreSQL) |
| Supabase Vault | モック関数 |
| 外部API (GitHub等) | httptest.Server でモック |

### テストケース

#### Module Registry 結合テスト

| ケース | 入力 | 期待結果 |
|--------|------|---------|
| 独立タスク3件 | after なし x3 | 3件並列実行 |
| 依存チェーン | A → B → C | 順次実行 |
| 分岐依存 | A → B, A → C | A完了後 B,C 並列 |
| 循環依存検出 | A → B → A | パース時エラー |
| 依存先失敗 | A(fail) → B | B スキップ、errors に記録 |

#### Entitlement Store 結合テスト

| ケース | ユーザー状態 | リクエスト | 期待結果 |
|--------|------------|------------|---------|
| Rate Limit内 | Free, 10 req済 | 11回目 | 許可 |
| Rate Limit超過 | Free, 30 req済 | 31回目 | 429 Too Many Requests |
| Quota内 | Free, 900 req済 | 901回目 | 許可 |
| Quota超過 | Free, 1000 req済 | 1001回目 | 403 Quota Exceeded |
| Credit消費 | Pro, Credit有効 | ツール実行 | Credit減算 |

---

## E2Eテスト（End-to-End Test）

### ツール

| ツール | 用途 |
|--------|------|
| Playwright | ブラウザ自動操作、User Console テスト |
| MCP Inspector | MCPプロトコルレベルのテスト |

### テストケース

#### User Console E2E

| ケース | 手順 | 検証 |
|--------|------|------|
| ログイン成功 | ソーシャルログイン | ダッシュボード表示 |
| OAuth連携 | /oauth/connect → 認可 → コールバック | 連携済み表示 |
| モジュール設定 | モジュール有効/無効切替 | 設定反映 |
| 使用量確認 | /dashboard | Quota/Credit表示 |

#### MCP E2E

| ケース               | 入力                    | 検証          |
| ----------------- | --------------------- | ----------- |
| get_module_schema | `{"module":"github"}` | スキーマJSON返却  |
| call (単発)         | GitHub issue 取得       | 結果返却        |
| batch (並列)        | 複数モジュール同時             | 全結果返却       |
| batch (依存)        | 連鎖処理                  | 順序通り実行、変数解決 |
| 認証エラー             | 無効JWT                 | 401エラー      |
| 権限エラー             | 無効モジュール呼び出し           | 403エラー      |

### テストアカウント

| アカウント | 用途 | 認証方式 |
|-----------|------|---------|
| test-admin@mcpist.app | 管理機能テスト | メール + パスワード |
| test-user@mcpist.app | 一般機能テスト | メール + パスワード |

**備考:**
- テスト環境では `NEXT_PUBLIC_SHOW_PASSWORD_LOGIN=true` でパスワード認証を有効化
- 本番環境ではソーシャルログインのみ

---

## セキュリティテスト

### 認証・認可テスト

| テスト項目 | 検証内容 | 期待結果 |
|-----------|---------|---------|
| JWT偽造 | 不正署名のJWT送信 | 401 Unauthorized |
| JWT期限切れ | 期限切れJWT送信 | 401 |
| 権限外モジュール | 無効化モジュール呼び出し | 403 Forbidden |
| RLS回避 | SQLインジェクションでRLS回避試行 | クエリ失敗 |

### 入力検証テスト

| テスト項目 | 入力 | 期待結果 |
|-----------|------|---------|
| SQLインジェクション | `'; DROP TABLE users; --` | エスケープ処理、エラー |
| XSS | `<script>alert(1)</script>` | エスケープ処理 |
| 大量データ | 100MB JSON | 413 または タイムアウト |
| 不正JSONL | 構文エラー | 400 + エラー詳細 |

### トークン管理テスト

| テスト項目 | 検証内容 | 期待結果 |
|-----------|---------|---------|
| トークンローテーション | リフレッシュ後の旧トークン | 無効化確認 |
| Vault暗号化 | DBダンプからのトークン読み取り | 暗号化状態 |

---

## 負荷テスト

### テスト項目

| テスト | 条件 | 目標 |
|--------|------|------|
| 同時接続 | 100並列リクエスト | 応答時間 < 5秒 |
| スループット | 500 req/min | エラー率 < 1% |
| Rate Limit | 制限超過リクエスト | 429 返却、正常復帰 |
| 長時間負荷 | 1時間連続 | メモリリークなし |

### 負荷テストツール

| ツール | 用途 |
|--------|------|
| k6 | HTTP負荷テスト |
| vegeta | シンプルな負荷テスト |

### 負荷耐性検証シナリオ（優先度別）

| シナリオ | 発生確率 | 影響度 | 優先度 |
|----------|----------|--------|--------|
| 外部API呼び出しによるリソース枯渇 | 高 | 高 | **P0** |
| KV書き込み枯渇 | 中 | 中 | P1 |
| 複数ユーザー同時Burst | 中 | 高 | P1 |
| DB接続プール枯渇 | 中 | 高 | P1 |
| フェイルオーバー中の誤差 | 低 | 低 | P3 |
| JWT検証CPU負荷 | 低 | 低 | P3 |

**P0対策（Phase 1）:**
- リクエストタイムアウト: 30秒
- ユーザー単位同時実行数制限: 50並列

---

## 障害テスト

### 障害シナリオ

| シナリオ | 障害内容 | 期待動作 |
|---------|---------|---------|
| 外部API障害 | GitHub API 500 | エラー返却、他ツール継続 |
| DB接続断 | PostgreSQL停止 | 503 + グレースフル停止 |
| Vault接続断 | 暗号化サービス停止 | 503 + トークン取得不可 |
| Primary Server停止 | Koyeb停止 | Fly.ioへフェイルオーバー |

### 回復テスト

| シナリオ | 検証内容 |
|---------|---------|
| MCPサーバー再起動 | 状態復元、リクエスト再開 |
| DB復旧後 | 接続プール再確立、正常動作 |
| 外部API復旧後 | リトライ成功 |

---

## OAuthフローテスト

### テストシナリオ

| フェーズ | テスト内容 |
|---------|-----------|
| 認可リクエスト | state パラメータ検証、PKCE チャレンジ |
| コールバック | code 交換、トークン取得 |
| トークン保存 | Vault 暗号化、DB レコード作成 |
| トークンリフレッシュ | 期限前リフレッシュ、失敗時の再認証フロー |
| トークン失効 | 手動解除、プロバイダ側失効 |

### エッジケース

| ケース | 検証内容 |
|--------|---------|
| ユーザーがOAuth拒否 | エラーハンドリング、UI表示 |
| state不一致 | CSRF攻撃検知、拒否 |
| code再利用 | 拒否（replay attack防止） |
| リフレッシュトークン失効 | 再認証フロー誘導 |

---

## CI パイプライン

### PR時（dev向け）

```yaml
# .github/workflows/ci.yml
name: CI

on:
  pull_request:
    branches: [dev]

jobs:
  lint:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: golangci/golangci-lint-action@v4

  unit-test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: '1.21'
      - name: Run unit tests
        run: go test -v -race -coverprofile=coverage.out ./...
      - name: Check coverage
        run: |
          COVERAGE=$(go tool cover -func=coverage.out | grep total | awk '{print $3}' | sed 's/%//')
          if (( $(echo "$COVERAGE < 70" | bc -l) )); then
            exit 1
          fi

  integration-test:
    runs-on: ubuntu-latest
    services:
      postgres:
        image: postgres:15-alpine
        env:
          POSTGRES_USER: test
          POSTGRES_PASSWORD: test
          POSTGRES_DB: mcpist_test
        ports:
          - 5432:5432
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
      - name: Run integration tests
        run: go test -v -tags=integration ./...
```

### E2E（週次）

```yaml
# .github/workflows/e2e.yml
name: E2E Tests

on:
  schedule:
    - cron: '0 0 * * 0'  # 週1回

jobs:
  e2e:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-node@v4
      - name: Run Playwright tests
        run: npx playwright test
```

---

## 定期チェック（運用時）

### ヘルスチェック

| チェック | 頻度 | 内容 |
|---------|------|------|
| /health | 30秒 | エンドポイント疎通（Cloudflare LB） |
| Smoke Test | デプロイ後 | 基本機能動作確認 |
| Deep Health | 週1 | 全モジュール疎通確認 |

### アラート条件

| 条件 | 重要度 | 対応 |
|------|--------|------|
| ヘルスチェック失敗 | Critical | 即時調査 |
| モジュール疎通失敗 | Warning | 当日中確認 |
| レスポンス遅延 (P95 > 5秒) | Warning | 監視継続 |

---

## 関連ドキュメント

| ドキュメント | 内容 |
|-------------|------|
| [spc-sys.md](./spc-sys.md) | システム仕様書 |
| [spc-itr.md](spc-itr.md) | インタラクション仕様書 |
| [spc-inf.md](spc-inf.md) | インフラストラクチャ仕様書 |
| [spc-ops.md](./spc-ops.md) | 運用仕様書 |
