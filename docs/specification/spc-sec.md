# MCPist セキュリティ仕様書（spc-sec）

## ドキュメント管理情報

| 項目 | 値 |
|------|-----|
| Status | `draft` |
| Version | v1.0 (DAY8) |
| Note | Security Specification |

---

## 概要

本ドキュメントは、MCPistのセキュリティ要件と対策を定義する。多層防御の設計思想に基づく。

---

## セキュリティ原則

| 原則 | 説明 |
|------|------|
| 多層防御 | 単一の対策に依存せず、複数の防御層を設ける |
| 最小権限 | 必要最小限の権限のみ付与 |
| 責任分離 | セキュリティ核心機能は実績あるサービスに委譲 |
| 透明性 | 設計・対策・判断理由を公開 |

---

## 認証・認可

### 認証方式

| 方式 | 用途 | 実装 |
|------|------|------|
| OAuth 2.1 Authorization Code Flow | LLMクライアント（Claude Code等） | Supabase Auth + Edge Function |
| Bearer Token（Long-lived） | API直接呼び出し | SHA-256ハッシュ検証 |
| ソーシャルログイン | User Console | Supabase Auth（Google, GitHub等） |

### JWT検証

| 項目 | 内容 |
|------|------|
| 署名アルゴリズム | RS256 |
| 検証方式 | Supabase Auth JWKS |
| issuer検証 | Supabase Auth URL |
| 有効期限 | OAuth 2.1仕様に準拠（1時間等） |

**許容事項**: JWTが漏洩した場合、有効期限までは悪用される可能性あり。これはOAuthの仕様上の制約。

### 認可

| 対象 | 方式 |
|------|------|
| DB行レベル | RLS（Row Level Security） |
| ツール実行 | Module有効/無効チェック |
| API Gateway | JWT検証 + Rate Limit |

---

## セキュリティリスクと対策

### リスク一覧

| リスク | 深刻度 | 対策 |
|--------|--------|------|
| プロンプトインジェクション | 中 | 多層防御（見せない・実行させない・検知） |
| X-User-IDヘッダー偽装 | 高 | Gateway Secret検証 |
| JWT偽装 | 高 | 署名検証（Supabase Auth）、短時間トークン |
| トークン漏洩（Token Vault） | 高 | Supabase Vault（AES-256-GCM）に委譲 |
| Rate Limitバイパス | 中 | 最大100%誤差を許容（Phase 1） |
| SQLインジェクション | 高 | Supabase SDK使用（生SQL禁止） |
| XSS | 中 | React標準エスケープ（dangerouslySetInnerHTML禁止） |

---

### プロンプトインジェクション対策

LLMがハルシネーションや悪意あるプロンプトにより、権限外のツールを呼び出すリスク。

**対策: 多層防御**

| Layer | 対策 | 説明 |
|-------|------|------|
| Layer 1 | 見せない | 権限のないモジュールはget_module_schemaで返却しない |
| Layer 2 | 実行させない | call_module_tool内で有効モジュールチェック |
| Layer 3 | 検知する | 権限外呼び出し試行をログに記録 |

---

### X-User-IDヘッダー偽装対策

攻撃者がAPI Gatewayを経由せず、直接オリジンにアクセスして`X-User-ID`ヘッダーを偽装するリスク。

**対策: Gateway Secret検証**

```
Worker → Origin 間で共有シークレットを検証

Origin側の検証:
  if r.Header.Get("X-Gateway-Secret") != os.Getenv("GATEWAY_SECRET") {
      return 403 Forbidden
  }
```

**シークレット配布:**

```
手動生成（openssl rand -hex 32）
    │
    ▼
GitHub Secrets に保存
    │
    ├─▶ Cloudflare Worker（wrangler secret put）
    ├─▶ Koyeb（環境変数）
    └─▶ Fly.io（flyctl secrets set）
```

---

### Rate Limitバイパス

Koyeb/Fly.io両方が稼働中の場合、Rate Limitカウンターが分散し、制限を超えるリクエストが通過するリスク。

**分析:**

| 制御 | 状況 |
|------|------|
| Burst制限 | Cloudflare KV（一元管理）→ バイパス不可 |
| Rate Limit | Origin側（メモリ + DB同期）→ 最大100%誤差 |

**許容理由:**
- 正常時はPrimaryのみにトラフィック集中（誤差0%）
- フェイルオーバー時のみ一時的に分散
- 最大120 req/minでも処理能力内
- 問題化したらRedis導入

---

## 暗号化

### 保存時暗号化

| データ | 暗号化方式 | 管理者 |
|--------|----------|--------|
| 外部サービストークン | AES-256-GCM | Supabase Vault |
| Long-lived Token | SHA-256ハッシュ | アプリケーション |
| DB全体 | TDE | Supabase（自動） |

### 通信暗号化

| 区間 | 暗号化 |
|------|--------|
| CLT ↔ API Gateway | TLS 1.3 |
| API Gateway ↔ Origin | TLS 1.3 |
| Origin ↔ Supabase | TLS 1.3 |
| Origin ↔ External APIs | TLS 1.2+ |

---

## 入力検証

| 検証項目 | 対策 |
|---------|------|
| SQLインジェクション | Supabase SDK使用（生SQL禁止） |
| XSS | React標準エスケープ（dangerouslySetInnerHTML禁止） |
| 大量データ | リクエストサイズ制限（Cloudflare Worker） |
| 不正JSON | スキーマバリデーション |

**方針:**
- 生SQLを書かない（Supabase SDK経由のみ）
- dangerouslySetInnerHTMLを使わない
- ユーザー入力は必ずバリデーション

---

## アクセス制御

### RLS（Row Level Security）

| テーブル | ポリシー |
|---------|---------|
| mcpist.users | user_id = auth.uid() |
| mcpist.oauth_tokens | user_id = auth.uid() |
| mcpist.subscriptions | user_id = auth.uid() |
| mcpist.usage | user_id = auth.uid() |
| mcpist.credits | user_id = auth.uid() |

### API Gateway

| チェック | 対象 |
|---------|------|
| JWT検証 | 全リクエスト |
| Rate Limit（IP単位） | 1000 req/min |
| Burst制限（ユーザー単位） | 5 req/s |

### Origin（MCP Server）

| チェック | 対象 |
|---------|------|
| Gateway Secret検証 | 全リクエスト |
| Rate Limit（ユーザー×プラン） | Free: 30 req/min |
| モジュール有効チェック | ツール実行時 |

---

## セキュリティ証明アプローチ

個人開発のためSOC-2等の第三者監査を受けられない。代替アプローチで信頼性を担保。

### 方針: 透明性 + 自動化 + 証跡

| アプローチ | 説明 |
|------------|------|
| 透明性 | 設計・対策・判断理由を公開 |
| 自動化 | CI/CDでセキュリティチェックを継続的に実行 |
| 証跡 | スキャン結果・対応履歴を記録・公開 |

### 自動化されたセキュリティチェック

| ツール | 用途 | Phase |
|--------|------|-------|
| Dependabot | 依存関係の脆弱性スキャン | Phase 1 |
| gosec | Go専用セキュリティリンター | Phase 1 |
| CodeQL / Semgrep | 静的解析（SAST） | Phase 2 |
| Trivy | コンテナイメージスキャン | Phase 2 |

### 第三者ツール活用

| サービス | コスト | 内容 |
|----------|--------|------|
| Mozilla Observatory | 無料 | HTTPヘッダー・TLS設定チェック |
| SSL Labs | 無料 | TLS設定の評価 |
| Cloudflare Security | 無料枠 | WAF、DDoS対策 |

---

## セキュリティ責任分界

### 責任の所在

| 機能 | 責任の所在 | 根拠 |
|------|-----------|------|
| JWT発行・検証 | Supabase | Supabase Auth標準機能 |
| トークン暗号化 | Supabase | Supabase Vault（AES-256-GCM） |
| アクセス制御（DB） | Supabase | RLS |
| 脆弱性対応（インフラ） | Supabase | SOC 2 Type II準拠 |
| DDoS対策 | Cloudflare | WAF、Rate Limit |
| アプリケーションロジック | **MCPist** | 自前実装 |

### Supabaseへの責任委譲

セキュリティ核心機能をSupabase（業界標準サービス）に委譲することで、Design Liabilityを最小化。

```
自前実装の範囲:
- UIとRPC制御フロー
- トークンリフレッシュ（HTTPリクエストのみ）
- ビジネスロジック

Supabaseに委譲:
- JWT発行・検証
- トークン暗号化・復号
- Row Level Security
```

---

## インシデント対応

### トークン漏洩時

```
1. 即時: User Consoleで全トークン削除
2. 即時: 外部サービス側でアプリ連携を解除
3. 調査: アクセスログで不正利用を確認
4. 復旧: OAuthアプリのClient Secret再生成
5. 復旧: 新しいSecret設定 → 再認可
6. 報告: ポストモーテム作成
```

### Gateway Secret漏洩時

```
1. 即時: 新しいGATEWAY_SECRETを生成
2. 即時: 全サービスに同時配布（Worker, Koyeb, Fly.io）
3. 調査: アクセスログで不正アクセス確認
4. 報告: ポストモーテム作成
```

---

## コンプライアンス

### OWASP Top 10対応

| 脆弱性 | 対策状況 |
|--------|----------|
| A01 Broken Access Control | RLS、モジュールチェック |
| A02 Cryptographic Failures | Supabase Vault、TLS |
| A03 Injection | パラメータ化クエリ |
| A04 Insecure Design | 多層防御、最小権限 |
| A05 Security Misconfiguration | セキュリティヘッダー、CSP |
| A06 Vulnerable Components | Dependabot |
| A07 Auth Failures | Supabase Auth、JWT検証 |
| A08 Software Integrity | CI/CD、署名検証 |
| A09 Logging Failures | Grafana Cloud |
| A10 SSRF | URLバリデーション |

---

## 関連ドキュメント

| ドキュメント | 内容 |
|-------------|------|
| [spc-sys.md](./spc-sys.md) | システム仕様書 |
| [spc-inf.md](spc-inf.md) | インフラストラクチャ仕様書 |
| [spc-tst.md](./spc-tst.md) | テスト仕様書（セキュリティテスト含む） |
