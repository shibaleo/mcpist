# Sprint 011 計画書

## 基本情報

| 項目 | 値 |
|------|-----|
| スプリント番号 | SPRINT-011 |
| 期間 | 2026-02-22 〜 2026-02-28 (7日間) |
| マイルストーン | M10: セキュリティ強化 + テスト基盤 |
| 前提 | Sprint 010 完了 (Supabase 除去、GORM 移行、ogen 自動生成、OAuth 2.0 改善) |

---

## Sprint 目標

**セキュリティ上の既知リスクを解消し、テスト基盤を整備する**

Sprint 010 で大規模アーキテクチャ刷新を完了したが、セキュリティ調査 (day034-issue_report) で検出された Critical/High リスク 3 件が未修正のまま残っている。Sprint 011 ではこれらを最優先で解消する。

---

## タスク一覧

### Phase 1: セキュリティ修正 (優先度: 最高)

#### 1A. API キー失効の実効性確保 (Critical)

**背景:** API キー JWT の署名検証のみで、DB 上の失効状態を参照していない。revoke 後もキーが使い続けられるリスク。

| ID | タスク | 変更対象 | 備考 |
|----|--------|---------|------|
| S11-001 | JWT に `key_id` クレームを追加 | `auth/keys.go` | DB の `api_keys.id` を含める |
| S11-002 | API キー認証時に DB 照合を追加 | `ogenserver/security.go` | key_id で存在確認、revoke 済みなら 401 |
| S11-003 | Worker 側にも DB 照合を追加 | `apps/worker/src/auth.ts` | Worker が JWT を検証するフロー |
| S11-004 | JWT `exp` クレームをデフォルト設定 | `ogenserver/api_keys.go` | デフォルト 90 日。無期限発行を禁止 |
| S11-005 | Console API キー生成 UI に有効期限表示 | `apps/console/` | 期限の表示 + 期限切れ警告 |

#### 1B. OAuth state の真正性検証 (High)

**背景:** `state` 生成時に `nonce` を作成するが、callback 側で検証していない。OAuth CSRF / セッション固定のリスク。

| ID | タスク | 変更対象 | 備考 |
|----|--------|---------|------|
| S11-010 | 共通ユーティリティ `lib/oauth/state.ts` 作成 | `apps/console/src/lib/oauth/state.ts` (新規) | `generateState(secret)` / `verifyState(state, secret)` |
| S11-011 | state を HMAC-SHA256 署名付きに変更 | 全 authorize ルート (11 プロバイダ) | `state = base64url(payload + HMAC)` |
| S11-012 | callback で state の HMAC を検証 | 全 callback ルート (11 プロバイダ) | 改ざん/リプレイ検出 |

対象プロバイダ: Google, Microsoft, Notion, Atlassian, Asana, Todoist, Airtable, TickTick, Trello, Dropbox, GitHub

#### 1C. 資格情報の暗号化保存 (High)

**背景:** `user_credentials.credentials` と `oauth_apps.client_secret` が平文保存。DB 漏洩時にトークン・秘密鍵が直接漏洩するリスク。

| ID | タスク | 変更対象 | 備考 |
|----|--------|---------|------|
| S11-020 | 暗号化パッケージ作成 | `internal/crypto/` (新規) | AES-256-GCM。`Encrypt(plaintext, key)` / `Decrypt(ciphertext, keys)` |
| S11-021 | user_credentials の暗号化 | `internal/db/repo_credentials.go` | UpsertCredential で暗号化、GetCredential で復号 |
| S11-022 | oauth_apps の暗号化 | `internal/db/repo_oauth_apps.go` | admin API 設定時に暗号化、読み出し時に復号 |
| S11-023 | 環境変数 `CREDENTIAL_ENCRYPTION_KEY` 追加 | Render, .env.local | base64 encoded 32 bytes |
| S11-024 | 既存データの暗号化マイグレーションスクリプト | `scripts/migrate-encryption.go` (新規) | 全レコードを読み出し → 暗号化 → 更新 |
| S11-025 | キーバージョニング対応 | `internal/crypto/` | `{v: <version>, iv, ct}` 形式。ローテーション対応 |

---

### Phase 2: 環境整理 (優先度: 高)

| ID | タスク | 変更対象 | 備考 |
|----|--------|---------|------|
| S11-030 | Stripe Webhook Dashboard 設定 | Stripe Dashboard | endpoint URL を Go Server に設定 |
| S11-031 | トークン検証 API の認証必須化 | `apps/console/src/app/api/credentials/validate/` | Clerk 認証追加。SSRF 対策 |
| S11-032 | INTERNAL_SECRET 廃止 | Worker + Console | 使用箇所調査 → 不要なら削除 |
| S11-033 | SECONDARY_API_URL 廃止 | Worker `types.ts` | 未使用。削除 |
| S11-034 | expires_at 形式統一 | OAuth callback ルート (11 プロバイダ) | Unix timestamp (秒) に統一 |
| S11-035 | Worker spec の縮小 | `apps/worker/` | 共有スキーマの残骸削除 (Go Server spec が single source of truth) |

---

### Phase 3: テスト基盤 (優先度: 中)

| ID | タスク | 変更対象 | 備考 |
|----|--------|---------|------|
| S11-040 | authz middleware ユニットテスト | `middleware/authz_test.go` (新規) | CanAccessTool, WithinDailyLimit |
| S11-041 | broker/user.go ユニットテスト | `broker/user_test.go` (新規) | GetUserContext, RecordUsage |
| S11-042 | crypto パッケージ ユニットテスト | `internal/crypto/crypto_test.go` (新規) | Encrypt/Decrypt, キーローテーション |
| S11-043 | CI トリガーを push/PR に変更 | `.github/workflows/ci.yml` | workflow_dispatch → push + pull_request |
| S11-044 | `go test ./...` の全パス確認 | `apps/server/` | ビルド通過 + 既存テストのパス |

---

### Phase 4: 小タスク (優先度: 低、時間があれば)

| ID | タスク | 備考 |
|----|--------|------|
| S11-050 | Clerk DCR 有効化 | MCP クライアント (Claude Desktop 等) の OAuth 接続 |
| S11-051 | seed.sql の OAuth Apps 初期データ | dev 環境の初期化効率化 |
| S11-052 | 仕様書更新 (credit → subscription model) | Sprint 007 からの繰越し |

---

## 作業順序

```
Day 1:   Phase 1A (S11-001〜005) — API キー失効修正
Day 2:   Phase 1B (S11-010〜012) — OAuth state 署名
Day 3:   Phase 1C (S11-020〜023) — 暗号化パッケージ + 適用
Day 4:   Phase 1C (S11-024〜025) + Phase 2 (S11-030〜035) — マイグレーション + 環境整理
Day 5:   Phase 3 (S11-040〜044) — テスト基盤
Day 6:   バッファ + Phase 4 + E2E 検証
Day 7:   バッファ
```

---

## リスク

| リスク | 影響 | 対策 |
|--------|------|------|
| 暗号化導入で既存クレデンシャルが読めなくなる | 全ユーザーの外部サービス接続切断 | マイグレーションスクリプトのテスト環境検証。ロールバック手順を用意 |
| OAuth state 検証で既存フローが壊れる | OAuth 接続不能 | 段階的に 1 プロバイダずつ適用、検証後に全展開 |
| API キー DB 照合のレイテンシ増加 | MCP API レスポンス遅延 | キャッシュ (TTL 5 分) で照合結果を保持 |

---

## 完了条件

- [ ] revoke 済み API キーでのアクセスが 401 になる
- [ ] API キーにデフォルト有効期限 (90日) が設定される
- [ ] OAuth state が改ざん/リプレイされた場合に callback が拒否される
- [ ] DB 内の user_credentials, oauth_apps が暗号化されている
- [ ] `/api/credentials/validate` が認証必須になっている
- [ ] INTERNAL_SECRET, SECONDARY_API_URL が削除されている
- [ ] authz middleware + crypto パッケージのユニットテストが pass
- [ ] CI が push/PR トリガーで自動実行される
- [ ] `go test ./...` が全パスする

---

## 参考

- [sprint010-review.md](sprint010-review.md) - Sprint 010 レビュー
- [sprint010-backlog.md](diary/sprint010-backlog.md) - Sprint 010 バックログ (繰越し一覧)
- [day034-issue_report.md](day034-issue_report.md) - セキュリティ調査結果
- [day032-backlog.md](day032-backlog.md) - DAY032 バックログ (暗号化設計)
