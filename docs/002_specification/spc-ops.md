# MCPist 運用仕様書（spc-ops）

## ドキュメント管理情報

| 項目 | 値 |
|------|-----|
| Status | `draft` |
| Version | v1.0 (DAY8) |
| Note | Operations Specification |

---

## 概要

本ドキュメントは、MCPistの運用設計を定義する。

---

## 運用方針

### 運用原則

| 原則 | 説明 |
|------|------|
| 放置運用 | 日常的な手動介入を不要にする |
| 自動リトライ | 一時的なエラー（タイムアウト、レート制限等）は自動リトライ |
| 最小コスト | 無料枠で運用継続可能 |
| ベンダー分散 | 単一障害点の排除（Koyeb + Fly.io） |
| スケール時プラン変更のみ | スケール時は追加設計・工数なし、インフラプラン変更のみで対応 |

**注意:** ホスティングサービス（Koyeb, Supabase等）自体の障害時は復旧を待つ。

### 運用責任

| 領域 | 担当 | 頻度 |
|------|------|------|
| トークンリフレッシュ | 自動（SRV） | 都度 |
| ヘルスチェック | 自動（Cloudflare LB） | 30秒 |
| 障害対応 | 運用者 | 必要時 |
| セキュリティ更新 | 運用者 | 月次 |
| コスト監視 | 運用者 | 月次 |

---

## KPI目標

| 指標 | 目標 | 計算式 |
|------|------|--------|
| 可用性 | 99%（月間7.2時間以内のダウンタイム） | (成功リクエスト数 / 総リクエスト数) × 100 |
| エラー率 | < 1% | (5xxエラー数 / 総リクエスト数) × 100 |
| レスポンス時間 P95 | < 3秒 | 95パーセンタイルのレスポンス時間 |
| レスポンス時間 P99 | < 5秒 | 99パーセンタイルのレスポンス時間 |

※ 外部サービス（Koyeb, Supabase, Fly.io等）の障害時間は除外して計算

### Grafana Cloud クエリ例

```txt
# 可用性（直近24時間）
(
  sum(rate(mcpist_requests_total{status!~"5.."}[24h]))
  /
  sum(rate(mcpist_requests_total[24h]))
) * 100

# エラー率（直近1時間）
(
  sum(rate(mcpist_errors_total[1h]))
  /
  sum(rate(mcpist_requests_total[1h]))
) * 100

# レスポンス時間 P95
histogram_quantile(0.95, sum(rate(mcpist_request_duration_ms_bucket[5m])) by (le))
```

---

## 初回セットアップ

### 前提条件

```bash
# 各サービスへの登録とCLI
# Supabase (https://supabase.com)
npm install -g supabase
supabase login

# Koyeb (https://www.koyeb.com)
koyeb login

# Fly.io (https://fly.io)
flyctl auth login

# Vercel (https://vercel.com)
npm install -g vercel
vercel login

# Cloudflare (https://dash.cloudflare.com)
npm install -g wrangler
wrangler login

# Grafana Cloud (https://grafana.com)
# Web UIでアカウント作成、Prometheus/Loki設定
```

### セットアップ手順

```
1. リポジトリクローン
   └── git clone https://github.com/xxx/mcpist.git

2. Supabaseプロジェクト作成
   └── Auth設定（Google/GitHub等のソーシャルログイン有効化）
   └── mcpistスキーマ作成
   └── DBマイグレーション適用

3. 外部サービスOAuthアプリ登録
   └── Google, Atlassian等でOAuthアプリを作成
   └── Client ID / Secret を取得

4. API Gateway デプロイ（Cloudflare Worker）
   └── wrangler deploy
   └── 環境変数設定（JWT_SECRET, GATEWAY_SECRET）

5. MCP Server デプロイ（Koyeb Primary）
   └── リポジトリ連携
   └── 環境変数設定
   └── デプロイ実行

6. MCP Server デプロイ（Fly.io Standby）
   └── fly deploy
   └── 環境変数設定

7. Cloudflare Load Balancer 設定
   └── オリジンプール（Koyeb, Fly.io）
   └── ヘルスチェック設定

8. User Console デプロイ（Vercel）
   └── リポジトリ連携
   └── 環境変数設定

9. Grafana Cloud 設定
   └── Prometheus データソース
   └── Loki データソース
   └── ダッシュボード作成
   └── アラート設定

10. 動作確認
    └── /health エンドポイント確認
    └── OAuth連携テスト
    └── MCPツール実行テスト
```

---

## 日常運用

### 自動化タスク

| タスク | 実行主体 | 頻度 |
|--------|---------|------|
| トークンリフレッシュ | SRV (Token Vault) | 都度（期限切れ時） |
| ヘルスチェック | Cloudflare LB | 30秒間隔 |
| フェイルオーバー | Cloudflare LB | 自動（3回連続失敗時） |
| ログ収集 | Grafana Cloud | リアルタイム |

### コールドスタート対策

Koyeb/Fly.io Free Tierはアイドル時にスリープするため、Cloudflare LBのヘルスチェック（30秒間隔）で回避。

### ログ確認

```bash
# Grafana Cloud Lokiでログ確認
# クエリ例:

# エラーログ
{app="mcpist"} |= "error"

# 特定モジュールのログ
{app="mcpist"} | json | module="notion"

# 遅いリクエスト（3秒以上）
{app="mcpist"} | json | duration_ms > 3000
```

---

## 監視・アラート

### 監視項目

| 項目 | 閾値 | 重要度 |
|------|------|--------|
| CPU高負荷 | > 80% (5分間) | Warning |
| メモリ逼迫 | > 200MB | Critical |
| レイテンシ悪化 | P95 > 2秒 (5分間) | Warning |
| エラー率上昇 | 5xx > 5% (5分間) | Critical |
| オリジン停止 | /health 失敗 (3回連続) | Critical |
| Rate Limit多発 | > 100/min | Warning |
| トークンリフレッシュ失敗 | 発生時 | Critical |

### アラート通知

Grafana Cloud Alerting を使用:

| 重要度 | 通知先 | 対応 |
|--------|--------|------|
| Critical | メール | 即時対応 |
| Warning | ログのみ | ダッシュボードで確認 |

### Grafana ダッシュボード

```
┌─────────────────────────────────────────────────┐
│ MCPist Dashboard                                │
├─────────────────────────────────────────────────┤
│ ┌─────────────┐ ┌─────────────┐ ┌─────────────┐│
│ │ Requests/min│ │ Error Rate  │ │ P95 Latency ││
│ │     42      │ │    0.5%     │ │   1.2s      ││
│ └─────────────┘ └─────────────┘ └─────────────┘│
│                                                 │
│ ┌─────────────────────────────────────────────┐│
│ │ Request Timeline (1h)                       ││
│ │ ▂▃▅▇█▆▄▃▂▃▄▅▆▇█▇▆▅▄▃▂▃▄▅▆▇█▆▅▄▃▂          ││
│ └─────────────────────────────────────────────┘│
│                                                 │
│ ┌─────────────┐ ┌─────────────┐                │
│ │ Koyeb       │ │ Fly.io      │                │
│ │ ● Active    │ │ ○ Standby   │                │
│ └─────────────┘ └─────────────┘                │
│                                                 │
│ ┌─────────────────────────────────────────────┐│
│ │ Tool Calls by Module                        ││
│ │ notion:          ████████████ 50%           ││
│ │ google_calendar: ████████ 30%               ││
│ │ microsoft_todo:  █████ 20%                  ││
│ └─────────────────────────────────────────────┘│
└─────────────────────────────────────────────────┘
```

---

## 障害対応

### 障害分類

| 分類 | 症状 | 原因例 |
|------|------|--------|
| 認証エラー | 401/403 | JWT期限切れ、トークン無効 |
| 外部API障害 | 5xx | 外部サービス障害 |
| レート制限 | 429 | API呼び出し過多 |
| 内部エラー | 500 | バグ、設定ミス |
| オリジン障害 | 503 | Koyeb/Fly.io停止 |

### Runbook

#### 5xxエラー発生時

```
## アラート: 5xxエラー
### 影響: MCPサーバーが正常に応答できない
### 対応:
1. Grafanaでエラーログを確認: {app="mcpist"} | json | level="error"
2. 特定のモジュールに集中しているか確認
3. 外部サービスのステータスページを確認
4. 一時的なら自動リトライを待つ
5. 継続するならデプロイログを確認
### 根本対応: コード修正またはKoyeb再デプロイ
```

#### オリジン停止時

```
## アラート: ヘルスチェック失敗（3回連続）
### 影響: オリジンサーバーがダウン状態
### 対応:
1. Cloudflare LBがフェイルオーバーしているか確認
2. Koyeb/Fly.ioダッシュボードでサービス状態を確認
3. 最近のデプロイがあれば、前バージョンにロールバック
4. ホスティングサービス自体の障害なら復旧を待機
### 根本対応: 起動エラーの修正、リソース不足の確認
```

#### トークンリフレッシュ失敗時

```
## アラート: トークンリフレッシュ失敗
### 影響: 該当サービスへのアクセス不可
### 対応:
1. Grafanaでエラーログを確認
2. User Consoleで該当サービスのトークン状態を確認
3. 「再認可」を実行してOAuthフローをやり直す
### 根本対応: OAuthクライアント設定の見直し
```

### 復旧確認

```bash
# ヘルスチェック
curl https://api.mcpist.app/health

# 期待レスポンス
{"status":"ok","version":"1.0.0","origin":"koyeb"}
```

---

## セキュリティ運用

### 定期タスク

| タスク | 頻度 | 手順 |
|--------|------|------|
| 依存パッケージ更新 | 月次 | Dependabotアラート対応 |
| シークレットローテーション | 年次 or 漏洩時 | 下記手順参照 |
| アクセスログ監査 | 月次 | 不審なアクセス確認 |

### シークレットローテーション

#### JWT_SECRET ローテーション

```
1. 新しいJWT_SECRETを生成
2. Supabase Auth設定を更新
3. 既存セッションは期限切れで自動無効化
4. 再ログインを案内
```

#### GATEWAY_SECRET ローテーション

```
1. 新しいGATEWAY_SECRETを生成
2. Cloudflare Worker環境変数を更新
3. Koyeb/Fly.io環境変数を更新
4. 同時に更新しないと一時的に認証エラーが発生
```

### インシデント対応（トークン漏洩時）

```
1. 即時: User Consoleで全トークン削除
2. 即時: 外部サービス側でアプリ連携を解除
3. 調査: アクセスログで不正利用を確認
4. 復旧: OAuthアプリのClient Secret再生成
5. 復旧: 新しいSecret設定 → 再認可
```

---

## バックアップ・リストア

### バックアップ対象

| 対象 | 方法 | 保持期間 |
|------|------|---------|
| Supabase DB | 自動バックアップ（Supabase標準） | 7日 |
| 設定ファイル | Git | 永続 |
| 環境変数 | 手動エクスポート | 変更時 |

### 手動バックアップ

```bash
# Supabase DBエクスポート
supabase db dump -f backup.sql

# 環境変数エクスポート
# 各サービスのダッシュボードからエクスポート
```

### リストア手順

```
1. Supabase新プロジェクト作成（または既存をリセット）
2. DBスキーマ適用: supabase db push
3. バックアップインポート: supabase db restore
4. Koyeb/Fly.io再デプロイ
5. 環境変数再設定
6. OAuth再認可（トークンは再取得が必要）
```

---

## デプロイ

### 環境構成

| 環境       | Supabase/Render/Koyeb/Vercel | Cloudflare            | ドメイン           |
|------------|------------------------------|----------------------|-------------------|
| dev        | shiba.dog.leo.private        | shiba.dog.leo.private | dev.mcpist.app    |
| stage      | fukudamakoto.private         | shiba.dog.leo.private | stg.mcpist.app    |
| production | fukudamakoto.work            | shiba.dog.leo.private | cloud.mcpist.app  |

### DevOps フロー

```
┌─────────────────────────────────────────────────────────────────────────┐
│                           DevOps Flow                                   │
├─────────────────────────────────────────────────────────────────────────┤
│                                                                         │
│  local ──────────► dev ──────────► stg ──────────► prd                 │
│         自動CI/CD        手動            DNS切り替え                     │
│         (main push)    ディスパッチ       (Blue-Green)                   │
│                                                                         │
├─────────────────────────────────────────────────────────────────────────┤
│                                                                         │
│  local:  破壊的な開発、機能実装                                           │
│                                                                         │
│  dev:    自動デプロイ、ユーザーテスト、E2E検証                             │
│                                                                         │
│  stg:    prdデータをマイグレーション、本番相当の検証                        │
│          ※ stg = 次期prd環境として準備                                   │
│                                                                         │
│  prd:    DNS切り替えでstgをprdに昇格                                     │
│          ※ 旧prdは次のstgとして再利用 or 削除                            │
│                                                                         │
└─────────────────────────────────────────────────────────────────────────┘
```

### stg → prd 切り替え（Blue-Green Deployment）

**リリース時の操作:**

1. OpenTofuでDNSレコード切り替え（Cloudflare API）
2. 各サービスの環境変数更新（Supabase接続先等）
3. ヘルスチェック確認

**メリット:**
- 即時ロールバック可能（DNS切り戻し）
- stgで検証済みの環境がそのまま本番化
- コードマージ不要、デプロイ待ち時間なし

**自動化:**
- OpenTofuでDNS・環境変数を一括管理
- GitHub Actions手動ディスパッチで `tofu apply` 実行
- Cloudflare API / Supabase CLI で切り替え

### デプロイ対象

| サービス     | プラットフォーム     | デプロイ方法        |
|-------------|---------------------|-------------------|
| Console     | Vercel              | OpenTofu          |
| Worker      | Cloudflare Workers  | Wrangler + OpenTofu |
| Server      | Render (Primary)    | OpenTofu          |
| Server      | Koyeb (Secondary)   | OpenTofu          |
| OAuth Server| Supabase            | Supabase CLI      |
| DB Migration| Supabase            | Supabase CLI      |

### GitHub Actions ワークフロー構成

| ワークフロー | トリガー | 処理内容 |
|-------------|---------|---------|
| `deploy-dev.yml` | main push | OpenTofu apply (dev) |
| `promote-to-stg.yml` | 手動ディスパッチ | OpenTofu apply (stg) + prdデータマイグレーション |
| `promote-to-prd.yml` | 手動ディスパッチ | DNS切り替え + 環境変数更新 |
| `rollback-prd.yml` | 手動ディスパッチ | DNS切り戻し（緊急時） |

### DNS設定（Cloudflare管理）

| ドメイン | 環境 | 切り替え対象 |
|---------|------|-------------|
| dev.mcpist.app | dev | 固定 |
| stg.mcpist.app | stg | 固定 |
| cloud.mcpist.app | prd | **Blue-Green切り替え** |
| console.mcpist.app | prd | **Blue-Green切り替え** |

### デプロイフロー（従来）

```
GitHub Push (main)
    │
    ├─→ Koyeb: 自動デプロイ（GitHub連携）
    ├─→ Fly.io: GitHub Actions経由
    ├─→ Cloudflare Worker: wrangler deploy
    └─→ Vercel: 自動デプロイ（GitHub連携）
```

### デプロイ後検証（Smoke Test）

| テスト | 期待結果 |
|--------|----------|
| Koyeb + GATEWAY_SECRET | 200 |
| Fly.io + GATEWAY_SECRET | 200 |
| Worker経由（JWT付き） | 200 |
| 直接アクセス（SECRETなし） | 403 |

### ロールバック

```bash
# Koyeb: ダッシュボードから前バージョンを選択
# Fly.io:
flyctl deploy --image registry.fly.io/mcpist:previous

# Vercel: ダッシュボードから前デプロイメントを選択
```

---

## ポストモーテム

障害発生時は必ず記録を残す。

### テンプレート

```markdown
# ポストモーテム: [障害タイトル]

## 概要
- 発生日時: YYYY-MM-DD HH:MM - HH:MM
- 影響: [どのサービスが使えなかったか]
- 検知方法: [アラート / 手動発見]

## タイムライン
[時系列で事実を記載]

## 根本原因
[なぜ起きたか]

## 再発防止策
- [ ] [具体的なアクション]
```

### 保存場所

- `docs/postmortems/YYYY-MM-DD-title.md` としてリポジトリに保存

---

## 廃止手順

```
1. 外部サービス連携解除
   - 各外部サービスでOAuthアプリ連携を解除
   - Client Secretは破棄

2. データ削除
   - Supabase: プロジェクト削除
   - Koyeb: サービス削除
   - Fly.io: アプリ削除
   - Vercel: プロジェクト削除
   - Cloudflare: Worker削除
   - Grafana Cloud: データソース削除

3. シークレット破棄
   - すべてのシークレットを破棄

4. ドメイン解放（使用している場合）
```

---

## 関連ドキュメント

| ドキュメント | 内容 |
|-------------|------|
| [spc-sys.md](./spc-sys.md) | システム仕様書 |
| [spc-inf.md](spc-inf.md) | インフラストラクチャ仕様書 |
| [spc-tst.md](./spc-tst.md) | テスト仕様書 |
