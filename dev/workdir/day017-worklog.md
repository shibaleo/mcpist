# DAY017 作業ログ

## 日付

2026-01-28

---

## 作業内容

### D17-001a: spc-dsn.md 更新（設計仕様書）

実装に合わせて spc-dsn.md を v1.0 → v2.0 に更新。Status: `reviewed`

| 変更箇所                               | 旧                                         | 新                                                                 |
| ---------------------------------- | ----------------------------------------- | ----------------------------------------------------------------- |
| Go バージョン                           | 1.21+                                     | 1.23+                                                             |
| Next.js                            | 14+                                       | 15+ / React 19                                                    |
| デプロイ先                              | Koyeb (Primary) / Fly.io (Standby)        | Render (Primary) / Koyeb (Secondary)                              |
| SRV内部: Module Registry (REG)       | 独立コンポーネント                                 | 廃止。ルーティングは MCP Handler (HDL) の責務                                  |
| SRV内部: auth/, entitlement/, vault/ | 独立ディレクトリ                                  | middleware/, store/ に統合。Entitlement Store 概念廃止                    |
| SRV内部: Observability               | なし                                        | observability/loki.go 追加                                          |
| Go 外部ライブラリ                         | jwt, supabase-go, prometheus              | 全削除。標準ライブラリのみ                                                     |
| Worker 責務                          | 旧仕様の責務一覧                                  | APIキー検証、OAuth Discovery、フェイルオーバー等を追加。Rate Limit/Burst は課金対象外として維持 |
| Console ライブラリ                      | @stripe/stripe-js                         | 削除（未導入）。Radix UI, jose, lucide-react 等を追加                         |
| Data Store セクション                   | Entitlement Store / Token Vault (ENT/TVL) | Data Store (Supabase PostgreSQL) に統一                              |
| Auth Server                        | OAuth (Google, GitHub)                    | OAuth (Google, Apple, Microsoft, GitHub)                          |
| ローカル開発                             | Docker Compose                            | 削除（不使用）。Air は残存                                                   |
| ブランチ戦略                             | 変更なし                                      | feature → dev → main を維持（現在は未確立のため main 直接運用）                     |
| リポジトリ構成                            | auth/, entitlement/, vault/, registry.go  | middleware/, modules/, store/, observability/, httpclient/        |
| Status typo                        | `proprsed`                                | `proposed` → `reviewed`                                           |

### D17-001b: spc-inf.md 更新（インフラ仕様書）

実装に合わせて spc-inf.md を v2.0 → v3.0 に更新。Status: `reviewed`

| 変更箇所 | 旧 | 新 |
|----------|-----|-----|
| 章立て | コンポーネントとインフラのマッピング（機能層別） | レイヤー別インフラ構成（canvas と統一） |
| レイヤー名 | クライアント層、ゲートウェイ層、MCPサーバー層等 | Consumer Layer, Edge Service Layer, Compute Layer 等 |
| Edge Service Layer | Worker, KV | Worker, KV, DNS, OAuth Discovery, Observability を追加 |
| Compute Layer | 新設 | MCP Server (Primary/Secondary) + Docker Container |
| Backend Platform Layer | Auth + DB Backend層 | OAuth Server, Session Manager, Data Store, Token Vault |
| Delivery Layer | なし | 新設。GitHub (Actions) + DockerHub |
| Observability Layer | なし | 新設。Grafana Cloud (Loki) |
| Payment Layer | 独立レイヤー | 廃止。External Integration Layer に統合 |
| External Integration Layer | 外部サービス | PSP (Stripe), IdP, External Auth Server, External API を統合 |
| インフラサービス一覧 | DockerHub なし | DockerHub 追加 |
| Stripe 表記 | 従量課金 | 決済手数料のみ（月額料金なし、3.6%） |
| データストア | 個別テーブル列挙 | スキーマレベル + spc-tbl.md 参照 |
| ASCII 構成図 | 旧レイヤー名 | canvas と統一したレイヤー名に更新 |
| コスト表 | DockerHub なし | DockerHub $0 追加 |

### D17-001c: grh-infrastructure.canvas 作成

インフラ構成図を Obsidian canvas 形式で新規作成。

- 8レイヤー構成: Consumer, Edge Service, Compute, Backend Platform, UI, Delivery, Observability, External Integration
- インフラサービスのみ記載（論理コンポーネントは排除）
- グループ間エッジでデータフローを表現
- spc-inf.md と完全に整合

### D17-001d: spc-itf.md → interaction/ ファイル整合性検証・整備

spc-itr.md で定義されたコンポーネント間の連携ペアと、interaction/ ディレクトリ内のファイル群の整合性を検証し、不足分の追加・孤立分の削除・命名規則統一を実施。

#### 1. spc-itr.md vs dtl-itr ファイルの整合性検証

spc-itr.md から一意なペア（24件）を抽出し、dtl-itr ファイル（21件）と比較。

**不足 7件（dtl-itr ファイル作成）:**

| ID          | ファイル               | 連携内容                   |
| ----------- | ------------------ | ---------------------- |
| ITR-REL-022 | dtl-itr-AMW-DST.md | AMW→DST ユーザーコンテキスト取得   |
| ITR-REL-023 | dtl-itr-AUS-DST.md | AUS↔DST ユーザーID共有（トリガー） |
| ITR-REL-024 | dtl-itr-DST-GWY.md | GWY→DST APIキー検証        |
| ITR-REL-025 | dtl-itr-DST-SSM.md | SSM→DST ユーザー情報登録・参照    |
| ITR-REL-026 | dtl-itr-DST-TVL.md | DST→TVL ユーザー紐付け        |
| ITR-REL-027 | dtl-itr-GWY-OBS.md | GWY→OBS HTTPリクエストログ    |
| ITR-REL-028 | dtl-itr-HDL-OBS.md | HDL→OBS ツール実行ログ        |

**孤立 4件:**

| ファイル | 対応 |
|---------|------|
| dtl-itr-DST-MOD.md | 削除（spc-itr.md に対応ペアなし） |
| dtl-itr-EAS-TVL.md | 削除（spc-itr.md に対応ペアなし） |
| dtl-itr-GWY-TVL.md | 削除（spc-itr.md に対応ペアなし） |
| dtl-itr-AUS-SSM.md | 残存（spc-itr.md に AUS-SSM 追記で整合） |

#### 2. spc-itr.md への AUS-SSM 追記

AUS セクション（§4）と SSM セクション（§13）に「ユーザー登録時の関数トリガー」連携を追記。

#### 3. idx-itr-rel.md 更新

- ITR-REL-022〜028 を追加
- ITR-REL-005 (GWY-TVL)、ITR-REL-011 (MOD-DST)、ITR-REL-013 (TVL-EAS) を削除
- 28件 → 25件
- カテゴリ「データストア・インフラ連携（022-028）」新設

#### 4. itr-xxx.md → itr-XXX.md リネーム

16ファイルを `git mv`（2段階）で大文字リネーム。idx-itr-rel.md、dtl-itr-*.md、itr-*.md 内のリンク参照を一括更新。

#### 5. itr-OBS.md 作成・itr-SRV.md 削除

- itr-OBS.md 新規作成（GWY・HDLからのログ受信）
- itr-SRV.md 削除（SRV は抽象化コンポーネント、canvas ではグループノード `grp-srv`。グループノードはファイルに反映しない方針）
- itr-reg.md 削除（REG は spc-itr.md のコンポーネント一覧に不在）

#### 6. grh-componet-interactions.canvas ID整理

- ノードID: ハッシュ値・英語名 → コンポーネント略称（CLO, GWY, AMW 等）、グループは `grp-xxx`
- エッジID: `e1`, `e2` 等 → `CLO-GWY`, `GWY-AMW` 等のペア名形式
- AUS-SSM の重複エッジ（通常 + bidirectional）を bidirectional 1本に統合

#### 7. 最終整合性

| 対応関係 | 数量 | 一致 |
|----------|------|------|
| canvas コンポーネントノード ↔ itr-XXX.md | 16 = 16 | OK |
| canvas エッジ ↔ dtl-itr-XXX-YYY.md | 25 = 25 | OK |
| idx-itr-rel.md ITR-REL 件数 | 25 | OK |

---

## コミット履歴

| コミット | メッセージ |
|----------|-----------|
| - | - |

---

## タスク進捗

| ID      | タスク                             | 状態                                                                                  |
| ------- | ------------------------------- | ----------------------------------------------------------------------------------- |
| D17-001 | 仕様書の実装追従更新 (BL-010〜014 + 設計書作成) | **完了** (spc-dsn.md, spc-inf.md, spc-itr.md, spc-itf.md, interaction/ 整備, canvas 整理) |
| D17-002 | Stripe 商品・価格設定                  | 未着手                                                                                 |
| D17-003 | サインアップ時の無料クレジット付与               | 未着手                                                                                 |
| D17-004 | Checkout Session API 実装         | 未着手                                                                                 |
| D17-005 | Webhook ハンドラ実装                  | 未着手                                                                                 |
| D17-006 | E2E テスト設計書作成                    | 未着手                                                                                 |

---

## バックログ更新

| ID | タスク | 状態 | 備考 |
|----|--------|------|------|
| - | - | - | - |
