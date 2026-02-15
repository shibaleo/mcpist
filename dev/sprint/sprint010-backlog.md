# Sprint 010 バックログ

Sprint 009 バックログを引き継ぎ、Sprint 010 の状況を反映。

---

## 優先度：高

### Phase 1: Cloudflare Workers OAuth 2.1 Server

Sprint 010 計画の Phase 1 全体。Claude からの認可が現状成功するようになったため、バックログに移動。

| ID | タスク | 状態 | 備考 |
|----|--------|------|------|
| S10-001 | `@cloudflare/workers-oauth-provider` 導入 | 未着手 | KV namespace `TOKEN_STORE` 追加 |
| S10-002 | OAuth 2.1 Server エンドポイント実装 | 未着手 | authorize, token, userinfo, revoke |
| S10-003 | Token storage (Workers KV) | 未着手 | auth code / access token / refresh token |
| S10-004 | User identity resolution | 未着手 | DB から user_id lookup |
| S10-005 | Consent redirect | 未着手 | Console 同意画面へ委譲 |
| S10-006 | RFC 9728 メタデータ更新 | 未着手 | authorization_servers を自前 Worker URL に |
| S10-007 | RFC 8414 メタデータ更新 | 未着手 | Supabase プロキシ → 自前メタデータ |
| S10-008 | JWT 検証ロジック更新 | 未着手 | 自前 OAuth token 検証を優先、Supabase fallback |
| S10-010 | Consent page 改修 | 未着手 | Supabase OAuth API → 自前 API |
| S10-011 | Consent API route 追加 | 未着手 | 同意時に Worker へ callback |
| S10-015 | 既存 Supabase Auth トークン互換性確認 | 未着手 | fallback で壊れないこと |
| S10-016 | Claude App 認可フロー E2E テスト | 未着手 | 認可 → トークン取得 → API call |

### Phase 2: Neon PG 移行

| ID | タスク | 状態 | 備考 |
|----|--------|------|------|
| S10-020 | Neon プロジェクト作成 | 未着手 | Oregon リージョン |
| S10-021 | スキーマ移植 (DDL) | 未着手 | 24 migration から DDL 抽出。auth.users FK 除去 |
| S10-022 | RPC 関数移植 | 未着手 | get_user_context, record_usage 等 |
| S10-023 | pgsodium TCE → pgcrypto 暗号化変更 | 未着手 | get_user_credential / upsert_user_credential |
| S10-024 | users テーブルの auth.users FK 除去 | 未着手 | UUID はそのまま維持 |
| S10-025 | pg_dump (Supabase → ファイル) | 未着手 | mcpist スキーマのデータのみ |
| S10-026 | credential 復号 + 再暗号化 | 未着手 | pgsodium 復号 → pgcrypto 暗号化 |
| S10-027 | データインポート (→ Neon) | 未着手 | |
| S10-030 | broker/db.go 新設 | 未着手 | DB 接続抽象化 |
| S10-031 | broker/user.go: Neon Data API 切替 | 未着手 | |
| S10-032 | broker/token.go: Neon Data API 切替 | 未着手 | |
| S10-033 | broker/module.go: Neon Data API 切替 | 未着手 | |
| S10-034 | HealthCheck 更新 | 未着手 | |
| S10-035 | Console DB アクセスを Neon に切替 | 未着手 | |
| S10-036 | admin.ts の service role 接続変更 | 未着手 | |
| S10-037 | Worker API Key 検証先変更 | 未着手 | |

### CI/CD 整備

| タスク | 備考 | 由来 |
|--------|------|------|
| CI トリガーを push/PR に変更 | 現在 workflow_dispatch (手動) | S008 |
| tools.json 自動生成パイプライン | Server のツール定義変更を検知 → 自動生成 | S006〜S009 |

---

## 優先度：中

### Phase 3: テスト基盤

| ID | タスク | 状態 | 備考 |
|----|--------|------|------|
| S10-040 | authz middleware ユニットテスト | 未着手 | |
| S10-041 | broker/user.go ユニットテスト | 未着手 | |
| S10-042 | broker/retry.go ユニットテスト | 未着手 | |
| S10-043 | CI トリガーを push/PR に変更 | 未着手 | |

### 堅牢性改善（残り）

| 項目 | 対策 |
|------|------|
| 分散 Rate Limiter | マルチインスタンス運用開始時に対応 |

### Stripe ogen 化

Stripe OpenAPI spec から ogen で型安全クライアント生成。Sprint 010 ではスコープ外。

### Console の DB アクセス一元化

PostgREST が残るなら不要。Neon 移行後に再検討。

### 仕様書の残課題

| タスク | 備考 | 由来 |
|--------|------|------|
| credentials JSON 構造の整理 | ogen + broker 化を反映 | S006 |
| credit model → subscription model に更新 | plans テーブル移行を反映 | S007 |

---

## 優先度：低

### Claude 認可フロー一時障害の原因調査

Claude からの OAuth 認可フローが一時的に失敗していたが、原因不明のまま復旧した。Supabase Auth 側の問題か、Claude 側の問題か、切り分けが必要。

- 再現条件が不明
- Supabase Auth がブラックボックスのためログからの追跡が困難
- 再発時に備えて調査手順を整理しておく

### SSE 改善

- ハートビート/ping-pong 追加
- メッセージバッファ溢れ時の対策

### Loki goroutine プール

goroutine 無制限生成の制御。プール + バックプレッシャー。

### Go GC 理解

`GOGC` / `GOMEMLIMIT` チューニング等。

### UI/UX

| タスク | 由来 |
|--------|------|
| ブランディング・ロゴ作成 | S006 |
| ソーシャルログイン拡充 | S006 |

### 将来検討

| タスク | 由来 |
|--------|------|
| Stg/Prd 環境構築 (Blue-Green) | S006 |
| 追加モジュール (Slack/Linear) | S006 |
| RFC 8707 Resource Indicators 対応 | S006 |

---

## 参考

- [sprint010-plan.md](../workdir/sprint010-plan.md) - Sprint 010 計画
- [sprint009-backlog.md](./sprint009-backlog.md) - Sprint 009 バックログ
