# MCPist セキュリティ設計

## 概要

MCPistのセキュリティリスクと対策。多層防御の設計思想に基づく。

関連ドキュメント:
- [dsn-permission-system.md](./dsn-permission-system.md) - 権限システム設計（Tool Sieve）
- [dsn-infrastructure.md](./dsn-infrastructure.md) - インフラ構成
- [dsn-subscription.md](./dsn-subscription.md) - Rate Limit設計

---

## セキュリティリスク一覧

| リスク | 深刻度 | 対策 | 状況 |
|--------|--------|------|------|
| プロンプトインジェクション | 中 | Tool Sieve（多層防御） | 対策済み |
| X-User-IDヘッダー偽装 | 高 | Gateway Secret検証 | 対策済み |
| JWT偽装 | 高 | 短時間トークン（OAuth仕様） | 許容 |
| Token Broker経由のトークン漏洩 | 高 | Supabase Vault（GoTrue） | 責任範囲外 |
| マルチインスタンスでRate Limitバイパス | 中 | 10分DB同期で最大100%誤差を許容 | 許容済み |

---

## プロンプトインジェクション対策

### リスク

LLMがハルシネーションや悪意あるプロンプトにより、権限外のツールを呼び出す。

### 対策: Tool Sieve（多層防御）

```
Layer 1: 見せない
  - 権限のないツールはget_module_schemaで返却しない
  - 攻撃対象が見えない = 攻撃できない

Layer 2: 実行させない
  - callメタツール内で権限チェック
  - 見えないツールを呼び出しても拒否

Layer 3: 検知する
  - 権限外ツールの呼び出し試行をログに記録
  - 正常ユーザーは見えないツールを呼ばない = 攻撃の可能性が高い
```

詳細は [dsn-permission-system.md](./dsn-permission-system.md) を参照。

---

## X-User-IDヘッダー偽装対策

### リスク

攻撃者がCloudflare Workerを経由せず、直接オリジンにアクセスして`X-User-ID`ヘッダーを偽装する。

### 対策: Gateway Secret検証

```
Worker → Origin 間で共有シークレットを検証

1. Gateway Secretを手動生成（デプロイ前）
2. GitHub Secretsに保存
3. GitHub Actions + Terraformで各サービスに配布
   - Cloudflare Worker: wrangler secret
   - Koyeb: 環境変数
   - Fly.io: secrets

Origin側の検証:
  if r.Header.Get("X-Gateway-Secret") != os.Getenv("GATEWAY_SECRET") {
      return 403 Forbidden
  }
```

### シークレット配布フロー

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

## JWT偽装対策

### リスク

攻撃者が有効なJWTを偽造または盗用する。

### 対策

| 対策 | 説明 |
|------|------|
| 署名検証 | Supabase AuthのJWKSで検証 |
| 短時間トークン | OAuth 2.0仕様に準拠（1時間等） |
| issuer検証 | 発行者が正しいか確認 |

**許容事項**: JWTが漏洩した場合、有効期限までは悪用される可能性あり。これはOAuthの仕様上の制約であり、許容する。

---

## Token Broker経由のトークン漏洩

### リスク

Token Broker（Supabase Edge Function）の脆弱性により、外部APIのアクセストークンが漏洩する。

### 対策

| 対策 | 説明 |
|------|------|
| Supabase Vault | トークンは暗号化保存 |
| Edge Functionの認証 | MCPサーバーからの呼び出しのみ許可 |

**責任範囲**: Supabase Vault（GoTrue）のセキュリティはSupabaseの責任。MCPistの責任範囲外。

---

## マルチインスタンスでRate Limitバイパス

### リスク

Koyeb/Fly.io両方が稼働中の場合、Rate Limitカウンターが分散し、制限を超えるリクエストが通過する。

### 分析

```
Burst制限: Cloudflare KV（一元管理）
  → バイパス不可

Rate Limit: Origin側（メモリ + DB同期）
  → 最大100%誤差の可能性あり
```

### 許容理由

| 条件 | 説明 |
|------|------|
| 正常時はPrimaryのみ | Koyeb健全時はトラフィックが集中、誤差0% |
| フェイルオーバー時のみ分散 | 一時的に両インスタンスに分散 |
| 120 req/minでも処理可能 | nanoインスタンスの処理能力内 |
| 問題化したらRedis導入 | Phase 1-2では考慮不要 |

**判断**: 10分DB同期で最大100%誤差を許容。厳密な保証（Redis）は技術的負債であり、Phase 1-2では導入しない。

詳細は [dsn-subscription.md](./dsn-subscription.md) を参照。

---

## セキュリティ証明アプローチ

個人開発のためSOC-2等の第三者監査を受けられない。代替として以下のアプローチで信頼性を担保する。

### 方針: 透明性 + 自動化 + 証跡

```
SOC-2がなくても信頼を得る方法:

1. 透明性: 設計・対策・判断理由を公開する
2. 自動化: CI/CDでセキュリティチェックを継続的に実行
3. 証跡: スキャン結果・対応履歴を記録・公開

「監査を受けていない」ではなく
「監査相当のチェックを自動化し、結果を公開している」
```

### セルフアセスメント

| 方法 | 説明 | Phase |
|------|------|-------|
| OWASP Top 10対応表 | 各脆弱性への対策状況を文書化 | Phase 1 |
| セキュリティ設計書公開 | 本ドキュメント（dsn-security.md） | Phase 1 |
| CIS Benchmarks | インフラ設定のベストプラクティス準拠 | Phase 2 |

### 自動化されたセキュリティチェック

CI/CDに組み込み、結果をバッジで公開:

| ツール | 用途 | Phase |
|--------|------|-------|
| Dependabot | 依存関係の脆弱性スキャン | Phase 1 |
| gosec | Go専用セキュリティリンター | Phase 1 |
| CodeQL / Semgrep | 静的解析（SAST） | Phase 2 |
| Trivy | コンテナイメージスキャン | Phase 2 |

### 透明性による信頼獲得

| アプローチ | 説明 | Phase |
|------------|------|-------|
| SECURITY.md | 脆弱性報告窓口・対応方針を公開 | Phase 1 |
| セキュリティポリシー | リポジトリに配置 | Phase 1 |
| インシデント対応手順 | 発生時の対応フローを事前公開 | Phase 2 |
| ADR/設計書 | 「なぜこの対策か」を説明 | Phase 1 |

### 第三者の活用（低コスト）

| サービス | コスト | 内容 |
|----------|--------|------|
| Mozilla Observatory | 無料 | HTTPヘッダー・TLS設定チェック |
| SSL Labs | 無料 | TLS設定の評価 |
| Cloudflare Security | 無料枠 | WAF、DDoS対策 |
| Bug Bountyプログラム | 成果報酬 | HackerOne等（Phase 2以降） |

---

## Phase 1 セキュリティチェックリスト

### 設計・対策

- [x] Tool Sieve実装（権限外ツールの非表示・実行拒否）
- [x] Gateway Secret設計（Worker→Origin間の認証）
- [x] JWT検証（Supabase Auth JWKS）
- [x] Rate Limitバイパスの許容判断
- [ ] Gateway Secretの生成・配布手順の文書化
- [ ] セキュリティログの設計（攻撃検知用）

### セキュリティ証明

- [ ] SECURITY.mdをリポジトリに配置
- [ ] GitHub Dependabot有効化
- [ ] gosecをCI/CDに追加
- [ ] OWASP Top 10対応表の作成

---

## 関連ドキュメント

- [dsn-permission-system.md](./dsn-permission-system.md) - 権限システム設計
- [dsn-infrastructure.md](./dsn-infrastructure.md) - インフラ構成
- [dsn-subscription.md](./dsn-subscription.md) - Rate Limit設計
- [dsn-load-management.md](./dsn-load-management.md) - 負荷対策設計
