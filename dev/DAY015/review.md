# DAY015 レビュー

## 問題1: vault.secrets への UPDATE 権限問題

### なぜうまくいかなかったのか

**Supabase Vault の公式 API を使用せず、直接テーブル操作をしようとしたから。**

1. **公式 API の無視**
   - Supabase Vault には `vault.create_secret()` という公式 API が存在する
   - この API を使わず、`UPDATE vault.secrets SET ...` で直接テーブルを操作しようとした
   - 直接操作は権限や暗号化の問題が発生しやすい

2. **既存コードの確認不足**
   - 同様の処理を行う `upsert_service_token` RPC が既に存在していた
   - このRPCは `vault.create_secret()` を使用していた
   - 既存パターンを確認せずに新規実装した

### なぜうまくいったのか

**既存コードのパターンに従い、公式 API を使用したから。**

1. **既存パターンの発見**
   - 「他のRPCはどうやってVaultに保存しているの？」という質問がきっかけ
   - `upsert_service_token` RPC を調査し、`vault.create_secret()` を使用していることを発見

2. **公式 API の使用**
   - `vault.create_secret()` は Supabase Vault extension が提供する公式 API
   - 権限や暗号化が適切に処理される
   - DELETE + `vault.create_secret()` で更新を実現

### 再発防止策

1. **Supabase 機能は公式 API を使う**
   - Vault: `vault.create_secret()`, `vault.update_secret()`
   - 直接テーブル操作は避ける

2. **既存コードを先に確認する**
   - 新機能を実装する前に、同様の処理を行う既存コードを探す
   - 特に Supabase 固有の機能（Vault, Auth, Storage）は既存パターンに従う

3. **「なぜ動かないか」より「どうやって動いているか」**
   - 権限問題の原因を調べるより、既に動いているコードの実装を参考にする方が早い

---

## 問題2: OAuth App redirect_uri 設定問題

### なぜうまくいかなかったのか

1. **環境による設定の違い**
   - ローカル開発時に `http://localhost:3000/api/oauth/google/callback` で OAuth App を設定
   - 本番デプロイ後に `redirect_uri` を更新し忘れた

2. **設定の分散**
   - Google Cloud Console に redirect_uri を登録
   - `mcpist.oauth_apps` テーブルにも redirect_uri を保存
   - 両方が一致していないと `redirect_uri_mismatch` エラー

3. **エラーメッセージの不明確さ**
   - Google の `redirect_uri_mismatch` エラーは「どちらが間違っているか」を教えてくれない
   - アプリ側が送信している URI と、Google Console に登録されている URI の両方を確認する必要があった

### なぜうまくいったのか

1. **段階的な確認**
   - Google Cloud Console の設定を確認 → 正しかった
   - データベースの `oauth_apps` テーブルを確認 → ローカル用のままだった
   - 原因が特定できた

2. **コードの追跡**
   - `authorize/route.ts` で `credentials.redirect_uri` を使用していることを確認
   - これが `get_oauth_app_credentials` RPC から取得されることを確認
   - つまりデータベースの値が使われていることが判明

### 再発防止策

1. **環境変数で redirect_uri を管理**
   ```typescript
   const redirectUri = process.env.OAUTH_REDIRECT_BASE_URL + '/api/oauth/google/callback';
   ```
   - データベースではなく環境変数で管理することで、デプロイ環境ごとに自動的に切り替わる

2. **デプロイチェックリストの作成**
   - [ ] OAuth App の redirect_uri を本番 URL に更新
   - [ ] Google Cloud Console の Authorized redirect URIs を確認
   - [ ] 環境変数の確認（SUPABASE_URL, API keys など）

3. **OAuth 設定の一元管理**
   - redirect_uri は Google Cloud Console と oauth_apps テーブルの2箇所に存在
   - できれば1箇所で管理し、もう1箇所は参照するだけにする

---

## 学んだこと

1. **Supabase Vault は特殊**
   - 通常の PostgreSQL テーブルと異なり、`supabase_admin` が所有者
   - 公式 API（`vault.create_secret()` など）を使うのが安全

2. **既存コードは最高のドキュメント**
   - 「他の RPC はどうやって Vault に保存しているの？」という質問が突破口になった
   - 同じ問題を解決した既存コードを探すのが最も効率的

3. **エラーメッセージを鵜呑みにしない**
   - 「Success」と表示されても実際には失敗していることがある
   - 必ず結果を検証する（`SELECT relacl FROM pg_class...` など）

4. **OAuth は設定箇所が多い**
   - Provider 側（Google Cloud Console）
   - アプリ側（oauth_apps テーブル、環境変数）
   - 両方を確認する習慣をつける
