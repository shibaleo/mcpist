# DAY010 振り返り（Review）

## 基本情報

| 項目 | 値 |
|------|-----|
| 日付 | 2026-01-19 〜 2026-01-20 |
| スプリント | SPRINT-002 |

---

## 躓いたポイントと解決策

### 1. Windows で `sh -c` が動作しない

**状況:**
```json
"start:server": "sh -c 'cd apps/server && go run ./cmd/server'"
```
→ Windows では `sh` コマンドが存在せずエラー

**解決:**
Go の `-C` フラグを使用してクロスプラットフォーム対応
```json
"start:server": "go run -C apps/server ./cmd/server"
```

**学び:**
- npm scripts でシェルコマンドを使う場合は OS 依存に注意
- 言語やツールの built-in オプションを優先する

---

### 2. Next.js と Go Server のポート競合

**状況:**
`.env.local` に `PORT=8089` を設定
→ Next.js も Go Server も同じ `PORT` 環境変数を読み込む
→ 両方が 8089 で起動しようとして競合

**解決:**
`dotenv-cli -v PORT=3000` で Next.js 用に上書き
```json
"start:console": "dotenv -v PORT=3000 -e .env.local -- pnpm --filter @mcpist/console dev"
```

**学び:**
- 環境変数名は汎用的すぎると競合する（`PORT` など）
- `dotenv-cli` の `-v` オプションで特定のコマンドにのみ上書き可能
- Turborepo にはこの機能がない（dotenv-cli を使う）

---

### 3. Supabase スキーマが存在しない

**状況:**
```
ERROR: schema "mcpist" does not exist
```

**原因:**
Supabase が不完全な状態で起動していた（マイグレーション未実行）

**解決:**
```bash
supabase stop --no-backup
supabase start
```
→ クリーンな状態からマイグレーションを再実行

**学び:**
- Supabase の状態がおかしいときは `--no-backup` で完全リセット
- Docker Desktop が起動していないとサイレントに失敗することがある

---

### 4. Supabase OAuth Server がローカルで使えない

**状況:**
Supabase Dashboard で OAuth Server (BETA) を発見
→ `supabase/config.toml` に `[auth.oauth]` を追加
→ エラー: `'auth' has invalid keys: oauth`

**原因:**
OAuth Server は Supabase Cloud のみの機能。OSS 版には含まれていない。

**解決:**
環境変数 `ENVIRONMENT` で本番/開発を切り替え
```typescript
// apps/console/src/lib/env.ts
export const useSupabaseOAuthServer = isProduction
```

- 開発: カスタム OAuth 実装を使用
- 本番: Supabase OAuth Server にリダイレクト/プロキシ

**学び:**
- Supabase の機能は OSS 版と Cloud 版で差がある
- OAuth Server: Cloud のみ
- Vault: OSS でも利用可能
- 環境で機能を切り替える設計が有効

---

### 5. consent ページのパス変更

**状況:**
Supabase OAuth Server の設定画面で Authorization URL を設定
→ `/auth/consent` ではなく `/oauth/consent` にする必要があった

**解決:**
```
apps/console/src/app/auth/consent/
→ apps/console/src/app/oauth/consent/
```

**学び:**
- 本番サービスの仕様に合わせてパスを設計する
- Supabase OAuth Server は `/oauth/*` を想定している

---

## 今日学んだこと

### dotenv-cli の活用

| オプション | 説明 |
|-----------|------|
| `-e .env.local` | 読み込む env ファイルを指定 |
| `-v KEY=value` | 環境変数を上書き |
| `--` | 以降をコマンドとして実行 |

モノレポで環境変数を一元管理しつつ、アプリごとに上書きできる。

### Go の `-C` フラグ

```bash
go run -C apps/server ./cmd/server
```

`-C` でディレクトリを変更してからコマンドを実行。
`cd` と違ってクロスプラットフォーム対応。

### 環境による機能切り替え

```typescript
export const isDevelopment = process.env.ENVIRONMENT === 'development'
export const isProduction = process.env.ENVIRONMENT === 'production'
export const useSupabaseOAuthServer = isProduction
```

同一コードベースで開発/本番の挙動を切り替える。
環境変数は `ENVIRONMENT` のみ変更すれば OK。

---

## 改善できるポイント

### 1. 環境変数のドキュメント化

`.env.example` にすべての環境変数とその用途を記載する。
特に `PORT` のような競合しやすい変数は注意書きが必要。

### 2. エラーハンドリングの改善

`dotenv-cli` で環境変数が見つからない場合のエラーメッセージが分かりにくい。
起動時に必須変数をチェックするスクリプトがあると良い。

### 3. Supabase の状態確認

`supabase status` で現在の状態を確認してから起動するスクリプトがあると便利。

---

## 次回に向けて

- [ ] JWT 認証フローの E2E テスト
- [ ] 本番環境デプロイ
- [ ] Supabase OAuth Server の Dashboard 設定
- [ ] エラーハンドリングの改善
