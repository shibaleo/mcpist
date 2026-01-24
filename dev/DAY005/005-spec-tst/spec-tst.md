---
title: MCPist テスト仕様書（spec-tst）
aliases:
  - spec-tst
  - MCPist-test-specification
tags:
  - MCPist
  - specification
  - test
document-type:
  - specification
document-class: specification
created: 2026-01-14T00:00:00+09:00
updated: 2026-01-14T00:00:00+09:00
---
# MCPist テスト仕様書（spec-tst）

## ドキュメント管理情報

| 項目 | 値 |
|------|-----|
| Status | `current` |
| Version | v2.0 (DAY5) |
| Base | DAY4からコピー |
| Note | DAY5で詳細化予定 |

---

本ドキュメントは、MCPistの品質保証に関するテスト戦略・テスト項目を定義する。

---

## 1. テスト戦略概要

### 1.1 テストピラミッド

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
| 単体 | 関数・メソッド | ローカル、CI | go test |
| 結合 | コンポーネント間連携 | CI | go test + testcontainers |
| E2E | システム全体 | リリース前 | Playwright |

### 1.2 テスト環境

| 環境 | 用途 | データ |
|------|------|--------|
| ローカル | 開発中の単体・結合テスト | モック・テストDB |
| CI (GitHub Actions) | PR時の自動テスト | モック・テストDB |
| dev環境 | E2E・手動テスト | テストデータ |
| production | 定期ヘルスチェック | 本番データ（読み取りのみ） |

---

## 2. 単体テスト（Unit Test）

### 2.1 対象コンポーネント

#### MCPサーバー（Go）

| パッケージ | テスト対象 | 優先度 |
|-----------|-----------|--------|
| `internal/toon` | TOON パーサー・フォーマッター | 高 |
| `internal/jsonl` | JSONL パーサー、依存関係解決 | 高 |
| `internal/variable` | 変数参照 `${id.items[N].field}` 解決 | 高 |
| `internal/auth` | JWT検証、JWKS取得 | 高 |
| `internal/ratelimit` | レートリミッター | 中 |
| `modules/*` | 各モジュールのパラメータバリデーション | 中 |

#### 管理UI（TypeScript/Next.js）

| ディレクトリ | テスト対象 | 優先度 |
|-------------|-----------|--------|
| `lib/validators` | フォーム入力バリデーション | 高 |
| `lib/auth` | 認証ヘルパー関数 | 高 |
| `components/*` | UIコンポーネント（スナップショット） | 低 |

### 2.2 テストケース例

#### TOON パーサー

```go
func TestParseTOON(t *testing.T) {
    tests := []struct {
        name    string
        input   string
        want    []map[string]string
        wantErr bool
    }{
        {
            name:  "正常系: 3件のレコード",
            input: "items[3]{id,title,status}:\n  task1,買い物,notStarted\n  task2,掃除,completed\n  task3,料理,inProgress",
            want: []map[string]string{
                {"id": "task1", "title": "買い物", "status": "notStarted"},
                {"id": "task2", "title": "掃除", "status": "completed"},
                {"id": "task3", "title": "料理", "status": "inProgress"},
            },
        },
        {
            name:  "正常系: 0件",
            input: "items[0]{}:",
            want:  []map[string]string{},
        },
        {
            name:    "異常系: 件数不一致",
            input:   "items[2]{id,title}:\n  task1,買い物",
            wantErr: true,
        },
        {
            name:    "異常系: フィールド数不一致",
            input:   "items[1]{id,title,status}:\n  task1,買い物",
            wantErr: true,
        },
    }
    // ...
}
```

#### 変数参照解決

```go
func TestResolveVariable(t *testing.T) {
    tests := []struct {
        name     string
        template string
        results  map[string]string
        want     string
        wantErr  bool
    }{
        {
            name:     "正常系: items[0].title",
            template: "Title: ${issues.items[0].title}",
            results: map[string]string{
                "issues": "items[2]{id,title}:\n  1,First Issue\n  2,Second Issue",
            },
            want: "Title: First Issue",
        },
        {
            name:     "異常系: 存在しないID",
            template: "${unknown.items[0].title}",
            results:  map[string]string{},
            wantErr:  true,
        },
        {
            name:     "異常系: インデックス範囲外",
            template: "${issues.items[10].title}",
            results: map[string]string{
                "issues": "items[2]{id,title}:\n  1,First\n  2,Second",
            },
            wantErr: true,
        },
    }
    // ...
}
```

#### JWT検証

```go
func TestValidateJWT(t *testing.T) {
    tests := []struct {
        name    string
        token   string
        wantErr error
    }{
        {
            name:    "正常系: 有効なトークン",
            token:   generateValidToken(t),
            wantErr: nil,
        },
        {
            name:    "異常系: 期限切れ",
            token:   generateExpiredToken(t),
            wantErr: ErrTokenExpired,
        },
        {
            name:    "異常系: 不正な署名",
            token:   "invalid.token.here",
            wantErr: ErrInvalidSignature,
        },
        {
            name:    "異常系: 空トークン",
            token:   "",
            wantErr: ErrTokenRequired,
        },
    }
    // ...
}
```

### 2.3 カバレッジ目標

| パッケージ | 目標カバレッジ |
|-----------|---------------|
| `internal/toon` | 90%以上 |
| `internal/jsonl` | 90%以上 |
| `internal/auth` | 85%以上 |
| `modules/*` | 70%以上 |
| 全体 | 70%以上 |

---

## 3. 結合テスト（Integration Test）

### 3.1 対象

| テスト対象 | 検証内容 |
|-----------|---------|
| MCPサーバー ↔ Token Broker | トークン取得・リフレッシュ |
| MCPサーバー ↔ Tool Sieve | 権限チェック |
| MCPサーバー ↔ 外部API | モジュール別API呼び出し |
| 管理UI ↔ Supabase | 認証・データ取得 |
| Edge Function ↔ Vault | 暗号化トークン取得 |

### 3.2 テスト方式

#### 外部依存のモック化

| 依存先 | モック方法 |
|--------|-----------|
| Supabase Auth | テスト用JWTを静的生成 |
| Supabase DB | testcontainers (PostgreSQL) |
| Supabase Vault | モックEdge Function |
| 外部API (GitHub等) | httptest.Server でモック |

#### testcontainers 使用例

```go
func TestToolSieveIntegration(t *testing.T) {
    ctx := context.Background()

    // PostgreSQLコンテナ起動
    postgres, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
        ContainerRequest: testcontainers.ContainerRequest{
            Image:        "postgres:15-alpine",
            ExposedPorts: []string{"5432/tcp"},
            Env: map[string]string{
                "POSTGRES_USER":     "test",
                "POSTGRES_PASSWORD": "test",
                "POSTGRES_DB":       "mcpist_test",
            },
            WaitingFor: wait.ForListeningPort("5432/tcp"),
        },
        Started: true,
    })
    require.NoError(t, err)
    defer postgres.Terminate(ctx)

    // マイグレーション実行
    // テストデータ投入
    // Tool Sieve テスト実行
}
```

### 3.3 テストケース

#### Token Broker 結合テスト

| ケース | 入力 | 期待結果 |
|--------|------|---------|
| 有効トークン取得 | user_id, module | access_token 返却 |
| 期限切れトークンのリフレッシュ | user_id, module (expired) | 新access_token + refresh実行 |
| リフレッシュトークン失効 | user_id, module (revoked) | 401 + 再認証要求 |
| 未連携モジュール | user_id, unknown_module | 404 + 連携案内 |

#### Tool Sieve 結合テスト

| ケース | ユーザーロール | リクエストツール | 期待結果 |
|--------|--------------|----------------|---------|
| 許可されたツール | developer | github_list_issues | 許可 |
| 禁止されたツール | viewer | github_create_issue | 拒否 |
| モジュール無効化 | admin | notion_* (broken) | 拒否 + エラーメッセージ |
| ロール未割当 | (none) | any | 拒否 |

#### JSONL 並列実行テスト

| ケース | 入力 | 期待結果 |
|--------|------|---------|
| 独立タスク3件 | after なし x3 | 3件並列実行 |
| 依存チェーン | A → B → C | 順次実行 |
| 分岐依存 | A → B, A → C | A完了後 B,C 並列 |
| 循環依存検出 | A → B → A | パース時エラー |
| 依存先失敗 | A(fail) → B | B スキップ、errors に記録 |

---

## 4. 総合テスト（System Test）

### 4.1 対象シナリオ

| シナリオ | 検証内容 |
|---------|---------|
| 新規ユーザー登録フロー | 許可リスト確認 → Supabase Auth登録 → 初期ロール割当 |
| OAuth連携フロー | 管理UI → OAuth認可 → コールバック → トークン保存 |
| ツール実行フロー | LLMクライアント → MCP → Tool Sieve → Token Broker → 外部API |
| 管理者操作フロー | ロール作成 → 権限設定 → ユーザー割当 |

### 4.2 テストケース

#### 新規ユーザー登録

```
前提条件:
- 許可リストに email@example.com が登録済み

手順:
1. 管理UIにアクセス
2. Supabase Auth でログイン（email@example.com）
3. ダッシュボードにリダイレクト

検証:
- users テーブルにレコード作成
- system_role = 'user' が設定
- 初期ロールが割り当て（設定があれば）
```

#### OAuth連携（GitHub）

```
前提条件:
- admin が GitHub OAuth アプリを設定済み
- user がログイン済み

手順:
1. /oauth/connect?provider=github にアクセス
2. GitHub認可画面でApprove
3. コールバック処理

検証:
- oauth_tokens にレコード作成
- access_token, refresh_token が Vault に暗号化保存
- /tools で GitHub が「連携済み」表示
```

#### ツール実行（正常系）

```
前提条件:
- user に developer ロール割当
- GitHub 連携済み

手順:
1. LLMクライアントから JSONL 送信:
   {"module":"github","tool":"github_list_issues","params":{"owner":"org","repo":"app"}}

検証:
- Tool Sieve: 権限チェック OK
- Token Broker: トークン取得 OK
- GitHub API: 呼び出し成功
- レスポンス: TOON形式で返却
```

---

## 5. E2Eテスト（End-to-End Test）

### 5.1 ツール

| ツール | 用途 |
|--------|------|
| Playwright | ブラウザ自動操作、管理UI テスト |
| MCP Inspector | MCPプロトコルレベルのテスト |

### 5.2 テストケース

#### 管理UI E2E

| ケース | 手順 | 検証 |
|--------|------|------|
| ログイン成功 | メール入力 → パスワード入力 → ログイン | ダッシュボード表示 |
| ログイン失敗（許可リスト外） | 未登録メールでログイン | エラーメッセージ表示 |
| ロール作成 | /admin/roles → 新規作成 → 保存 | ロール一覧に表示 |
| ユーザーロール割当 | /admin/users → ユーザー選択 → ロール割当 | 更新反映 |
| OAuth連携 | /oauth/connect → 認可 → コールバック | 連携済み表示 |

#### MCP E2E

| ケース | 入力 | 検証 |
|--------|------|------|
| get_module_schema | `{"modules":["github"]}` | スキーマJSON返却 |
| call_module_tool (単発) | GitHub issue 取得 | TOON形式で返却 |
| call_module_tool (並列) | 複数モジュール同時 | 全結果返却 |
| call_module_tool (依存) | 連鎖処理 | 順序通り実行、変数解決 |
| 認証エラー | 無効JWT | 401エラー |
| 権限エラー | 禁止ツール呼び出し | 403エラー |

### 5.3 Playwright テスト例

```typescript
import { test, expect } from '@playwright/test';

test.describe('管理UI', () => {
  test('adminがロールを作成できる', async ({ page }) => {
    // ログイン
    await page.goto('/login');
    await page.fill('[data-testid="email"]', 'admin@example.com');
    await page.fill('[data-testid="password"]', 'password');
    await page.click('[data-testid="login-button"]');

    // ロール作成画面へ
    await page.goto('/admin/roles');
    await page.click('[data-testid="create-role"]');

    // フォーム入力
    await page.fill('[data-testid="role-name"]', 'test-developer');
    await page.fill('[data-testid="role-description"]', 'テスト用ロール');
    await page.check('[data-testid="module-github"]');
    await page.click('[data-testid="save-role"]');

    // 検証
    await expect(page.locator('[data-testid="role-list"]')).toContainText('test-developer');
  });

  test('userが権限のないページにアクセスできない', async ({ page }) => {
    // userでログイン
    await loginAsUser(page);

    // admin専用ページにアクセス
    await page.goto('/admin/roles');

    // リダイレクトまたはエラー
    await expect(page).toHaveURL('/dashboard');
    // または
    await expect(page.locator('[data-testid="error-message"]')).toBeVisible();
  });
});
```

---

## 6. セキュリティテスト

### 6.1 認証・認可テスト

| テスト項目 | 検証内容 | 期待結果 |
|-----------|---------|---------|
| JWT偽造 | 不正署名のJWT送信 | 401 Unauthorized |
| JWT期限切れ | 期限切れJWT送信 | 401 + リフレッシュ案内 |
| 権限昇格 | userがadmin APIを呼び出し | 403 Forbidden |
| IDOR | 他ユーザーのリソースアクセス | 403 または 404 |
| RLS回避 | SQLインジェクションでRLS回避試行 | クエリ失敗 |

### 6.2 入力検証テスト

| テスト項目 | 入力 | 期待結果 |
|-----------|------|---------|
| SQLインジェクション | `'; DROP TABLE users; --` | エスケープ処理、エラー |
| XSS | `<script>alert(1)</script>` | エスケープ処理 |
| パストラバーサル | `../../etc/passwd` | 拒否 |
| 大量データ | 100MB JSON | 413 または タイムアウト |
| 不正JSONL | 構文エラー | 400 + エラー詳細 |

### 6.3 トークン管理テスト

| テスト項目 | 検証内容 | 期待結果 |
|-----------|---------|---------|
| トークン漏洩検知 | 異常なAPI呼び出しパターン | アラート発火 |
| トークンローテーション | リフレッシュ後の旧トークン | 無効化確認 |
| Vault暗号化 | DBダンプからのトークン読み取り | 暗号化状態 |

### 6.4 権限境界テスト

| シナリオ | 操作者 | 対象 | 期待結果 |
|---------|--------|------|---------|
| 自分のプロファイル編集 | user | 自分 | 許可 |
| 他人のプロファイル編集 | user | 他user | 拒否 |
| ロール作成 | user | - | 拒否 |
| ロール作成 | admin | - | 許可 |
| システムロール変更 | admin | 他user | 許可 |
| 自分のシステムロール変更 | admin | 自分 | 拒否（最後のadmin保護） |

---

## 7. 負荷テスト

### 7.1 テスト項目

| テスト | 条件 | 目標 |
|--------|------|------|
| 同時接続 | 100並列リクエスト | 応答時間 < 5秒 |
| スループット | 1000リクエスト/分 | エラー率 < 1% |
| レートリミット | 制限超過リクエスト | 429 返却、正常復帰 |
| 長時間�kind | 1時間連続負荷 | メモリリークなし |

### 7.2 負荷テストツール

| ツール | 用途 |
|--------|------|
| k6 | HTTP負荷テスト |
| vegeta | シンプルな負荷テスト |

### 7.3 k6 テスト例

```javascript
import http from 'k6/http';
import { check, sleep } from 'k6';

export const options = {
  stages: [
    { duration: '30s', target: 20 },  // ウォームアップ
    { duration: '1m', target: 100 },  // 負荷上昇
    { duration: '2m', target: 100 },  // 維持
    { duration: '30s', target: 0 },   // クールダウン
  ],
  thresholds: {
    http_req_duration: ['p(95)<5000'],  // 95%ile < 5秒
    http_req_failed: ['rate<0.01'],     // エラー率 < 1%
  },
};

export default function () {
  const res = http.post(
    'https://mcpist.app/mcp',
    JSON.stringify({
      module: 'github',
      tool: 'github_list_issues',
      params: { owner: 'test', repo: 'test' }
    }),
    {
      headers: {
        'Content-Type': 'application/json',
        'Authorization': `Bearer ${__ENV.JWT_TOKEN}`,
      },
    }
  );

  check(res, {
    'status is 200': (r) => r.status === 200,
    'response has items': (r) => r.body.includes('items['),
  });

  sleep(1);
}
```

---

## 8. 障害テスト

### 8.1 障害シナリオ

| シナリオ | 障害内容 | 期待動作 |
|---------|---------|---------|
| Token Broker停止 | Edge Function タイムアウト | 503 + リトライ案内 |
| 外部API障害 | GitHub API 500 | エラー返却、他ツール継続 |
| DB接続断 | PostgreSQL停止 | 503 + グレースフル停止 |
| Vault接続断 | 暗号化サービス停止 | 503 + トークン取得不可 |
| ネットワーク分断 | DNS解決失敗 | タイムアウト + エラー |

### 8.2 回復テスト

| シナリオ | 検証内容 |
|---------|---------|
| MCPサーバー再起動 | 状態復元、リクエスト再開 |
| DB復旧後 | 接続プール再確立、正常動作 |
| 外部API復旧後 | リトライ成功、キャッシュクリア |

### 8.3 障害注入テスト（Chaos Engineering）

```yaml
# chaos-mesh 設定例
apiVersion: chaos-mesh.org/v1alpha1
kind: NetworkChaos
metadata:
  name: token-broker-delay
spec:
  action: delay
  mode: all
  selector:
    labelSelectors:
      app: edge-function
  delay:
    latency: "5s"
  duration: "60s"
```

---

## 9. OAuthフローテスト

### 9.1 テストシナリオ

| フェーズ | テスト内容 |
|---------|-----------|
| 認可リクエスト | state パラメータ検証、PKCE チャレンジ |
| コールバック | code 交換、トークン取得 |
| トークン保存 | Vault 暗号化、DB レコード作成 |
| トークンリフレッシュ | 期限前リフレッシュ、失敗時の再認証フロー |
| トークン失効 | 手動解除、プロバイダ側失効 |

### 9.2 プロバイダ別テスト

| プロバイダ | 特殊テスト項目 |
|-----------|---------------|
| Google (OIDC) | ID Token 検証、クレーム抽出 |
| Microsoft (OIDC) | テナント制限、スコープ確認 |
| GitHub | Organization 権限、App vs OAuth App |
| Notion | ワークスペース選択、ページ権限 |
| Jira/Confluence | サイト選択、クラウド/DC判定 |

### 9.3 エッジケース

| ケース | 検証内容 |
|--------|---------|
| ユーザーがOAuth拒否 | エラーハンドリング、UI表示 |
| state不一致 | CSRF攻撃検知、拒否 |
| code再利用 | 拒否（replay attack防止） |
| リフレッシュトークン失効 | 再認証フロー誘導 |
| スコープ不足 | 追加スコープ要求 |

---

## 10. ロールバックテスト

### 10.1 テストシナリオ

| シナリオ | 手順 | 検証 |
|---------|------|------|
| 即時ロールバック | デプロイ → Smoke Test失敗 → 自動ロールバック | 前バージョンで正常動作 |
| 手動ロールバック | 本番障害発覚 → Koyeb手動ロールバック | 1-2分で復旧 |
| マイグレーション込み | 新スキーマデプロイ → ロールバック | 後方互換性確認 |

### 10.2 検証項目

| 項目 | 確認内容 |
|------|---------|
| API互換性 | 旧バージョンAPIが動作 |
| DBスキーマ | マイグレーション後方互換 |
| トークン有効性 | 保存済みトークンが使用可能 |
| セッション | ユーザーセッション維持 |

---

## 11. DBマイグレーションテスト

### 11.1 テスト方針

| 方針 | 内容 |
|------|------|
| 後方互換性必須 | ロールバック可能なスキーマ変更のみ |
| 段階的移行 | 破壊的変更は複数リリースで段階実施 |
| データ検証 | マイグレーション前後のデータ整合性確認 |

### 11.2 テストケース

| ケース | 検証内容 |
|--------|---------|
| カラム追加 | NULL許容、デフォルト値設定 |
| カラム削除 | 事前に参照コード削除済み確認 |
| インデックス追加 | 本番データ量でのパフォーマンス |
| テーブル名変更 | エイリアス期間、完全移行 |

### 11.3 マイグレーションテスト手順

```
1. 本番DBダンプ取得（匿名化済み）
2. テスト環境にリストア
3. マイグレーション実行
4. データ整合性チェッククエリ実行
5. アプリケーション結合テスト
6. ロールバックテスト
7. 再マイグレーションテスト
```

---

## 12. 外部API互換性テスト

### 12.1 バージョン変更検知

version-notify.yml が検知した変更に対するテスト。

| テスト | 内容 |
|--------|------|
| 既存機能動作確認 | 使用中のエンドポイント全テスト |
| レスポンス形式確認 | フィールド追加/削除/型変更の検出 |
| 廃止予定確認 | Deprecation ヘッダーのチェック |

### 12.2 プロバイダ別監視項目

| プロバイダ | 監視対象 |
|-----------|---------|
| GitHub | REST API バージョン、GraphQL スキーマ |
| Notion | API バージョン（レスポンスヘッダ） |
| Microsoft Graph | API バージョン、スキーマ変更 |
| Jira/Confluence | REST API バージョン |

### 12.3 互換性テストの実行

```go
func TestGitHubAPICompatibility(t *testing.T) {
    // 現在の api_version を取得
    currentVersion := getModuleVersion("github")

    // 実際のAPIを呼び出し
    resp := callGitHubAPI("/repos/test/test/issues")

    // レスポンスヘッダからバージョン確認
    actualVersion := resp.Header.Get("X-GitHub-Api-Version")

    // バージョン一致確認
    if actualVersion != currentVersion {
        t.Logf("API version changed: %s -> %s", currentVersion, actualVersion)
    }

    // 必須フィールドの存在確認
    var issues []Issue
    json.Unmarshal(resp.Body, &issues)

    for _, issue := range issues {
        assert.NotEmpty(t, issue.Number, "number field required")
        assert.NotEmpty(t, issue.Title, "title field required")
        assert.NotEmpty(t, issue.State, "state field required")
    }
}
```

---

## 13. 定期チェック（運用時）

### 13.1 ヘルスチェック

| チェック | 頻度 | 内容 |
|---------|------|------|
| ping.yml | 毎日 | /health エンドポイント疎通 |
| Smoke Test | デプロイ後 | 基本機能動作確認 |
| Deep Health | 週1 | 全モジュール疎通確認 |

### 13.2 Deep Health チェック項目

```yaml
# deep-health.yml
checks:
  - name: supabase_auth
    endpoint: /api/auth/health
    expected_status: 200

  - name: token_broker
    endpoint: /api/tokens/health
    expected_status: 200

  - name: github_module
    test: get_module_schema
    params: { modules: ["github"] }
    expected: schema_returned

  - name: notion_module
    test: get_module_schema
    params: { modules: ["notion"] }
    expected: schema_returned
```

### 13.3 アラート条件

| 条件 | 重要度 | 対応 |
|------|--------|------|
| ヘルスチェック失敗 | Critical | 即時調査 |
| モジュール疎通失敗 | Warning | 当日中確認 |
| レスポンス遅延 (P95 > 5秒) | Warning | 監視継続 |

---

## 14. テスト自動化

### 14.1 CI パイプライン

```yaml
# ci.yml
name: CI

on:
  push:
    branches: [dev]
  pull_request:
    branches: [dev]

jobs:
  unit-test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: '1.22'
      - name: Run unit tests
        run: go test -v -race -coverprofile=coverage.out ./...
      - name: Check coverage
        run: |
          COVERAGE=$(go tool cover -func=coverage.out | grep total | awk '{print $3}' | sed 's/%//')
          if (( $(echo "$COVERAGE < 70" | bc -l) )); then
            echo "Coverage $COVERAGE% is below 70%"
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
        with:
          go-version: '1.22'
      - name: Run integration tests
        run: go test -v -tags=integration ./...
        env:
          DATABASE_URL: postgres://test:test@localhost:5432/mcpist_test

  lint:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: golangci/golangci-lint-action@v4
        with:
          version: latest
```

### 14.2 E2E パイプライン

```yaml
# e2e.yml
name: E2E Tests

on:
  workflow_dispatch:
  schedule:
    - cron: '0 0 * * 0'  # 週1回

jobs:
  e2e:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-node@v4
        with:
          node-version: '20'
      - name: Install Playwright
        run: npx playwright install --with-deps
      - name: Run E2E tests
        run: npx playwright test
        env:
          BASE_URL: https://dev.mcpist.app
          TEST_USER_EMAIL: ${{ secrets.TEST_USER_EMAIL }}
          TEST_USER_PASSWORD: ${{ secrets.TEST_USER_PASSWORD }}
```

---

## 15. テストデータ管理

### 15.1 テストデータ方針

| 環境 | データ |
|------|--------|
| 単体テスト | インラインデータ、fixtures |
| 結合テスト | Seed スクリプト |
| E2E | 専用テストアカウント |
| 負荷テスト | 匿名化本番データ |

### 15.2 テストアカウント

| アカウント | 用途 | system_role | 認証方式 |
|-----------|------|-------------|---------|
| test-admin@mcpist.app | 管理機能テスト | admin | メール + パスワード（MFAなし） |
| test-user@mcpist.app | 一般機能テスト | user | メール + パスワード |
| test-viewer@mcpist.app | 権限制限テスト | user (viewer role) | メール + パスワード |

**テスト環境での認証:**
- テストアカウントは **メール + パスワード** 認証（E2E 自動化用）
- 本番環境ではメール + パスワード欄を非表示（ソーシャルログインのみ）
- テスト環境では `NEXT_PUBLIC_SHOW_PASSWORD_LOGIN=true` でパスワード欄を表示
- E2E テストの自動化が可能（Playwright でログインフロー実行）

### 15.3 Seed スクリプト

```sql
-- test-seed.sql
-- テスト用ロール
INSERT INTO roles (id, name, description) VALUES
  ('role-admin', 'admin', '管理者ロール'),
  ('role-developer', 'developer', '開発者ロール'),
  ('role-viewer', 'viewer', '閲覧者ロール');

-- テスト用権限
INSERT INTO role_permissions (id, role_id, enabled_modules, tool_masks) VALUES
  ('perm-admin', 'role-admin', ARRAY['github', 'notion', 'jira'], '{}'),
  ('perm-developer', 'role-developer', ARRAY['github', 'notion'], '{"github_delete_repo": false}'),
  ('perm-viewer', 'role-viewer', ARRAY['github'], '{"github_create_issue": false}');

-- モジュールレジストリ
INSERT INTO module_registry (module_name, display_name, status, api_version) VALUES
  ('github', 'GitHub', 'stable', '2022-11-28'),
  ('notion', 'Notion', 'stable', '2022-06-28'),
  ('jira', 'Jira', 'stable', '3');
```

---

## 16. 関連ドキュメント

| ドキュメント | 内容 |
|-------------|------|
| [要件仕様書](spec-req.md) | 要件定義 |
| [システム仕様書](spec-sys.md) | システム全体像 |
| [設計仕様書](spec-dsn.md) | 詳細設計 |
| [インフラ仕様書](spec-inf.md) | インフラ構成 |
| [運用仕様書](spec-ops.md) | 運用設計 |
