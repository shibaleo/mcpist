---
title: MCPist 運用仕様書（spec-ops）
aliases:
  - spec-ops
  - MCPist-operations-specification
tags:
  - MCPist
  - specification
  - operations
document-type:
  - specification
document-class: specification
created: 2026-01-12T00:00:00+09:00
updated: 2026-01-14T00:00:00+09:00
---
# MCPist 運用仕様書（spec-ops）

## ドキュメント管理情報

| 項目 | 値 |
|------|-----|
| Status | `current` |
| Version | v3.0 (DAY5) |
| Base | DAY4からコピー |
| Note | DAY5で詳細化予定 |

---

本ドキュメントは、MCPistの運用設計を定義する。

---

## 1. 運用方針

### 1.1 運用原則

| 原則 | 説明 |
|------|------|
| 放置運用 | 日常的な手動介入を不要にする |
| 自動リトライ | 一時的なエラー（タイムアウト、レート制限等）は自動リトライ |
| 最小コスト | 無料枠で運用継続可能 |
| 個人管理 | 運用者 = ユーザー = 1人 |

**注意:** ホスティングサービス（Koyeb, Supabase等）自体の障害時は復旧を待つしかない。

### 1.2 運用責任

| 領域 | 責任者 | 頻度 |
|------|--------|------|
| トークン更新 | 自動 | 都度 |
| 障害対応 | ユーザー | 必要時 |
| セキュリティ更新 | ユーザー | 月次 |
| コスト監視 | ユーザー | 月次 |

### 1.3 可用性目標・KPI

#### 目標値

| 指標 | 目標 | 計算式 |
|------|------|--------|
| 可用性 | 99% (月間7.2時間以内のダウンタイム) | (成功リクエスト数 / 総リクエスト数) × 100 |
| エラー率 | < 1% | (5xxエラー数 / 総リクエスト数) × 100 |
| レスポンス時間 P95 | < 3秒 | 95パーセンタイルのレスポンス時間 |
| レスポンス時間 P99 | < 5秒 | 99パーセンタイルのレスポンス時間 |

※ 外部サービス（Koyeb, Supabase等）の障害時間は除外して計算

#### Grafana Cloud での計算

```
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

#### 月次レビュー

| 確認項目 | 方法 |
|---------|------|
| 可用性達成率 | Grafanaダッシュボードで月間集計 |
| 主要エラー | ログ検索で頻出エラーを特定 |
| 外部サービス障害 | 各サービスのステータスページ履歴 |

---

## 2. 初回セットアップ

### 2.1 前提条件

各サービスへの登録とCLIのインストール:

```bash
# Supabase (https://supabase.com)
npm install -g supabase
supabase login

# Koyeb (https://www.koyeb.com)
# https://www.koyeb.com/docs/cli/installation
koyeb login

# Vercel (https://vercel.com)
npm install -g vercel
vercel login

# Grafana Cloud (https://grafana.com) - 監視・アラート用
# Web UIでアカウント作成、Loki/IRM設定
```

※ 各サービスのWeb UIからも操作可能

### 2.2 セットアップ手順

```
1. リポジトリクローン
   └── git clone https://github.com/xxx/mcpist.git

2. Supabaseプロジェクト作成
   └── Auth設定（Google/GitHub等のソーシャルログイン有効化）
   └── DBスキーマ適用
   └── Edge Functions デプロイ

3. 外部サービスOAuthアプリ登録（タイプBのサービスのみ）
   └── Google, Atlassian等でOAuthアプリを作成
   └── Client ID / Secret を取得

4. MCPサーバーデプロイ（Koyeb）
   └── リポジトリ連携
   └── 環境変数設定
   └── デプロイ実行

5. 管理UIデプロイ（Vercel）
   └── リポジトリ連携
   └── 環境変数設定

6. allowed_usersにメールアドレス登録
   └── Supabase SQLエディタで実行

7. 管理UIにログイン → トークン登録
   └── タイプA: APIトークンを入力
   └── タイプB: OAuth認可フロー実行

8. MCPクライアント設定
   └── MCPサーバーURLを設定
   └── APIトークン or JWT設定
```

### 2.3 OAuthアプリ登録ガイド

#### Notion

| 項目 | 値 |
|------|-----|
| 登録URL | https://www.notion.so/my-integrations |
| タイプ | Internal Integration |
| Capabilities | Read/Update/Insert content |
| Redirect URI | `https://your-app.vercel.app/api/services/notion/oauth/callback` |

#### GitHub

| 項目 | 値 |
|------|-----|
| 登録URL | https://github.com/settings/developers |
| タイプ | OAuth App |
| Scopes | repo, read:user |
| Redirect URI | `https://your-app.vercel.app/api/services/github/oauth/callback` |

#### Jira / Confluence

| 項目 | 値 |
|------|-----|
| 登録URL | https://developer.atlassian.com/console/myapps/ |
| タイプ | OAuth 2.0 (3LO) |
| Scopes | read:jira-work, write:jira-work |
| Redirect URI | `https://your-app.vercel.app/api/services/jira/oauth/callback` |

#### Google

| 項目 | 値 |
|------|-----|
| 登録URL | https://console.cloud.google.com/apis/credentials |
| タイプ | OAuth 2.0 Client ID |
| Scopes | calendar.readonly, calendar.events |
| Redirect URI | `https://your-app.vercel.app/api/services/google/oauth/callback` |

---

## 3. 日常運用

### 3.1 自動化タスク

| タスク | 実行主体 | 頻度 |
|--------|---------|------|
| トークンリフレッシュ | Token Broker | 都度（期限切れ時） |
| ヘルスチェック | Koyeb | 30秒間隔 |
| ログローテーション | Grafana Cloud | 日次 |

### 3.2 コールドスタート対策

Koyeb Free Tierはアイドル時にスリープするため、定期的なヘルスチェックで回避:

```yaml
# GitHub Actions: .github/workflows/healthcheck.yml
name: Health Check
on:
  schedule:
    - cron: '*/10 * * * *'  # 10分間隔
jobs:
  healthcheck:
    runs-on: ubuntu-latest
    steps:
      - name: Ping MCP Server
        run: curl -f https://mcpist-xxx.koyeb.app/health
```

### 3.3 ログ確認

```bash
# Grafana Cloud Lokiでログ確認
# クエリ例:

# エラーログ
{app="mcpist"} |= "error"

# 特定モジュールのログ
{app="mcpist"} | json | module="notion"

# 遅いリクエスト
{app="mcpist"} | json | duration_ms > 3000
```

---

## 4. 障害対応

### 4.1 障害分類

| 分類 | 症状 | 原因例 |
|------|------|--------|
| 認証エラー | 401/403レスポンス | JWT期限切れ、トークン無効 |
| 外部API障害 | 5xxエラー | 外部サービス障害 |
| レート制限 | 429エラー | API呼び出し過多 |
| 内部エラー | 500エラー | バグ、設定ミス |

### 4.2 対応手順

#### 認証エラー（JWT関連）

```
1. JWTの有効期限を確認
2. Authサーバーで再ログイン
3. 新しいJWTをMCPクライアントに設定
```

#### 認証エラー（OAuthトークン関連）

```
1. 管理UIでトークン状態を確認
2. トークンが無効な場合:
   - 該当サービスの「再認可」を実行
3. リフレッシュトークンも無効な場合:
   - 外部サービス側でアプリ連携を解除
   - 管理UIから再度OAuth認可
```

#### 外部API障害

```
1. 外部サービスのステータスページを確認
2. 一時的障害の場合:
   - 自動リトライで復旧を待つ
3. 長時間障害の場合:
   - 該当モジュールを一時無効化
```

#### レート制限

```
1. ログでAPI呼び出し頻度を確認
2. batch使用でリクエスト数を削減
3. 必要に応じて外部サービスのプラン確認
```

### 4.3 復旧確認

```bash
# ヘルスチェック
curl https://mcpist-xxx.koyeb.app/health

# 期待レスポンス
{"status":"ok","version":"1.0.0"}
```

---

## 5. セキュリティ運用

### 5.1 定期タスク

| タスク | 頻度 | 手順 |
|--------|------|------|
| 依存パッケージ更新 | 月次 | Dependabotアラート対応 |
| シークレットローテーション | 年次 or 漏洩時 | 下記手順参照 |
| アクセスログ監査 | 月次 | 不審なアクセス確認 |

### 5.2 シークレットローテーション

#### JWT_SECRET ローテーション

```
1. 新しいJWT_SECRETを生成
2. Koyeb環境変数を更新
3. 既存セッションは期限切れで自動無効化
4. 再ログインを案内
```

#### OAuthトークンローテーション

```
1. 管理UIで該当サービスの「トークン削除」
2. 再度OAuth認可を実行
3. 新しいトークンがVaultに保存される
```

### 5.3 インシデント対応

#### トークン漏洩時

```
1. 即時: 管理UIで全トークン削除
2. 即時: 外部サービス側でアプリ連携を解除
3. 調査: アクセスログで不正利用を確認
4. 復旧: OAuthアプリのClient Secret再生成
5. 復旧: 新しいSecret設定 → 再認可
```

---

## 6. 監視・アラート

### 6.1 監視項目

| 項目 | 閾値 | 重要度 |
|------|------|--------|
| 5xxエラー | 発生時 | Critical |
| ヘルスチェック失敗 | 3回連続 | Critical |
| トークンリフレッシュ失敗 | 発生時 | Critical |
| レスポンス時間 P95 | > 5秒 (5分間) | Warning |
| 4xxエラー率 | > 10% (5分間) | Warning |

### 6.2 アラート通知

Grafana Cloud IRM（無料枠で利用可能）を使用:

| 重要度 | 通知先 | 対応 |
|--------|--------|------|
| Critical | メール | 即時対応 |
| Warning | ログのみ | ダッシュボードで確認 |

※ メール通知の有効/無効は管理UIで設定可能

### 6.3 Grafanaダッシュボード

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
│ ┌─────────────────────────────────────────────┐│
│ │ Tool Calls by Module                        ││
│ │ notion:    ████████████ 45%                 ││
│ │ github:    ████████ 30%                     ││
│ │ jira:      ████ 15%                         ││
│ │ other:     ██ 10%                           ││
│ └─────────────────────────────────────────────┘│
└─────────────────────────────────────────────────┘
```

---

## 7. バックアップ・リストア

### 7.1 バックアップ対象

| 対象 | 方法 | 保持期間 |
|------|------|---------|
| Supabase DB | 自動バックアップ | 7日 |
| 設定ファイル | Git | 永続 |
| 環境変数 | 手動エクスポート | 変更時 |

### 7.2 手動バックアップ

```bash
# 環境変数エクスポート
# Koyeb CLIまたはダッシュボードからエクスポート

# Supabase DBエクスポート
supabase db dump -f backup.sql

# OAuth設定バックアップ（Client ID/Secretは含めない）
# 管理UIから設定一覧をエクスポート
```

### 7.3 リストア手順

```
1. Supabase新プロジェクト作成（または既存をリセット）
2. DBスキーマ適用: supabase db push
3. バックアップインポート: supabase db restore
4. Koyeb再デプロイ
5. 環境変数再設定
6. OAuth再認可（トークンは再取得が必要）
```

---

## 8. 廃止手順

### 8.1 サービス廃止時

```
1. 外部サービス連携解除
   - 各外部サービスでOAuthアプリ連携を解除
   - Client Secretは破棄

2. データ削除
   - Supabase: プロジェクト削除
   - Koyeb: サービス削除
   - Vercel: プロジェクト削除
   - Grafana Cloud: データソース削除

3. シークレット破棄
   - すべてのシークレットを破棄

4. ドメイン解放（使用している場合）
```

---

## 9. トラブルシューティング

### 9.1 よくある問題

| 症状 | 原因 | 対処 |
|------|------|------|
| 「Token not found」 | OAuth未認可 | 管理UIで認可実行 |
| 「JWT expired」 | セッション切れ | 再ログイン |
| 「Rate limited」 | API呼び出し過多 | 待機 or batch使用 |
| レスポンスが遅い | コールドスタート | 待機 or ヘルスチェック設定 |
| 「Module not found」 | モジュール未有効化 | 管理UIで有効化 |

### 9.2 Runbook（アラート別対応手順）

#### 5xxエラー発生時

```
## アラート: 5xxエラー
### 影響: MCPサーバーが正常に応答できない
### 対応:
1. Grafanaでエラーログを確認: {app="mcpist"} | json | level="error"
2. 特定のモジュールに集中しているか確認
3. 外部サービスのステータスページを確認
4. 一時的なら自動リトライを待つ、継続するならデプロイログを確認
### 根本対応: コード修正またはKoyeb再デプロイ
```

#### ヘルスチェック失敗時

```
## アラート: ヘルスチェック失敗（3回連続）
### 影響: MCPサーバーがダウン状態
### 対応:
1. Koyebダッシュボードでサービス状態を確認
2. 最近のデプロイがあれば、前バージョンにロールバック
3. Koyeb自体の障害なら status.koyeb.com を確認して待機
### 根本対応: 起動エラーの修正、リソース不足の確認
```

#### トークンリフレッシュ失敗時

```
## アラート: トークンリフレッシュ失敗
### 影響: 該当サービスへのアクセス不可
### 対応:
1. Grafanaでエラーログを確認
2. 管理UIで該当サービスのトークン状態を確認
3. 「再認可」を実行してOAuthフローをやり直す
### 根本対応: OAuthクライアント設定の見直し、スコープ確認
```

### 9.3 デバッグ手順

```bash
# 1. ヘルスチェック
curl https://mcpist-xxx.koyeb.app/health

# 2. ログ確認（Grafana Cloud）
{app="mcpist"} | json | level="error"

# 3. トークン状態確認（管理UI）
# /services ページで各サービスの状態を確認

# 4. 外部API直接テスト
curl -H "Authorization: Bearer <token>" \
  https://api.notion.com/v1/users/me
```

### 9.4 ポストモーテム

障害発生時は必ず記録を残す。1人運用でも学習のため。

#### テンプレート

```markdown
# ポストモーテム: [障害タイトル]

## 概要
- 発生日時: YYYY-MM-DD HH:MM - HH:MM
- 影響: [どのサービスが使えなかったか]
- 検知方法: [アラート / 手動発見]

## 何が起きたか
[時系列で事実を記載]

## なぜ起きたか
[根本原因]

## 何を学んだか
[今回の教訓]

## 再発防止策
- [ ] [具体的なアクション]
```

#### 保存場所

- `docs/postmortems/YYYY-MM-DD-title.md` としてリポジトリに保存
- 同じ障害を繰り返さないための参照資料

---

## 10. 運用成熟ロードマップ（参考）

個人プロジェクトでも段階的に運用を成熟させるための指針。必須ではない。

### Phase 1: 基礎（実装済み）

- 可用性目標・KPI定義
- 監視・アラート設定
- 基本的な障害対応手順

### Phase 2: 運用の成熟

| 項目 | 説明 |
|------|------|
| Runbook整備 | 全アラートに対する対応手順を文書化（9.2節） |
| ポストモーテム習慣化 | 障害ごとに記録を残す（9.4節） |

### Phase 3: セキュリティの深化

| 項目 | 説明 |
|------|------|
| 暗号化詳細設計 | 鍵導出方法（PBKDF2等）、鍵ローテーションポリシー |
| 脅威モデリング | STRIDE等でトークン漏洩シナリオを洗い出す |

### Phase 4: 自動化と継続的改善

| 項目 | 説明 |
|------|------|
| CI/CDパイプライン | セキュリティスキャン、自動テスト、自動ロールバック |
| カオスエンジニアリング（軽量版） | 定期的に意図的に壊す（Supabase接続断、トークン無効化等） |

### Phase 5: ドキュメントの完成

| 項目 | 説明 |
|------|------|
| ADR継続 | 設計判断のたびにArchitecture Decision Recordを書く |
| アーキテクチャ図の維持 | C4モデル等でContext → Container → Component → Codeの各レベルを維持 |

---

## 11. 関連ドキュメント

| ドキュメント | 内容 |
|-------------|------|
| [要件仕様書](spec-req.md) | 要件定義 |
| [システム仕様書](spec-sys.md) | システム全体像 |
| [設計仕様書](spec-dsn.md) | 詳細設計 |
| [インフラ仕様書](spec-inf.md) | インフラ構成 |
