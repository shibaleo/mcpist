# DAY017 レビュー

## 日付

2026-01-28

---

## 実施したコミット (8件)

| コミット | 種別 | 内容 |
|---------|------|------|
| ba948e6 | feat | MCP Tool Annotations 採用、メタツール動的登録 |
| 9903e77 | feat | Batch事前権限チェック、Loki Observability統合 |
| c0bd91a | feat | /mcp → /connections リネーム、tools.json更新 |
| 48f4ef4 | docs | sprint005-review, sprint006計画, day016ドキュメント |
| ade98d1 | fix | 有効ツール0件のモジュールを非表示化 |
| 03d2b37 | docs | 仕様書v2.0更新、interaction整合性確保 |
| 3d6bb58 | docs | dtl-itr 7件更新、canvas整理、バックログ追加 |
| 83e4059 | docs | 軽微修正 |

---

## 発見された課題

### 1. 詳細仕様と実装の整合作業がつらかった

仕様書が実装に追従できておらず、大量の更新が必要だった。特にinteractionファイル群（dtl-itr-*.md）の整合性確保に時間がかかった。

### 2. 責務境界の誤認

- クレジットチェックの責務が AMW にあると思っていたが、実際は HDL
- credits初期化がDBトリガー（AUS-DST）にあったが、アプリ層（CON-DST）が適切

### 3. RFC 8707 Resource Indicators 未対応

MCP OAuth仕様で推奨されているが未実装（BL-060として起票）。Supabase Auth の対応状況も未確認。

### 4. セキュリティ情報の曖昧なエラーメッセージ

9903e77 で「クライアントには曖昧なエラー、詳細はサーバー側ログ」を実装。これ自体は正しいが、ドキュメント化が不十分だった。

---

## 良かった点

1. **Canvasグラフデータによる整合性管理** - コンポーネント仕様とitr-XXX.mdファイル、インタラクション仕様とdtl-itr-XXX-YYY.mdを対応させ、数的に比較することで整合的なドキュメント体系を構築できた。今後も継続する
2. **Loki統合による可観測性向上** - X-Request-ID でリクエスト追跡可能に
3. **Batch事前チェック** - All-or-Nothing で部分実行を防止
4. **MCP Annotations採用** - 独自フィールドから標準仕様へ移行

---

## 残課題（DAY018以降）

| ID | 内容 |
|----|------|
| draft 5件 | dtl-itr-DST-GWY, DST-PSP, DST-SSM, DST-TVL, GWY-OBS のレビュー |
| BL-060 | RFC 8707 Resource Indicators 対応 |
| BL-061 | クレジット初期化をDBトリガーからアプリ層へ移行 |

---

## 判断と学び

### Supabase Vault をインフラとして Data Store と分離した理由

Supabase Vault は同一の Supabase（PostgreSQL）基盤上で動作し、内部的には pgsodium 等を用いた暗号化機構によりシークレットを保護する。とはいえ、インフラ仕様書（spc-inf.md）および canvas（grh-infrastructure.canvas）では **Token Vault を Data Store とは別コンポーネント**として扱う。

**分離の目的:** シークレット管理（Secrets）という論理境界を明確にし、将来の移行・運用の影響範囲を切り出すため。

**理由:**
- Supabase の管理UI上、Vault は独立した管理画面・機能として提供され、運用上も別サービス単位として扱える
- 将来的に外部サービストークン管理を Composio / Nango / Auth0 等の専用サービスへ移行する可能性がある
- インフラ移行や障害対応時の影響範囲（業務データ vs シークレット）を明確化でき、変更容易性が高い
- 論理的な役割（業務データ保存 vs シークレット暗号化保存）が異なるため、実体が同一基盤でも分離する妥当性がある


---

## 反省

### 計画 vs 実績の差異分析

| ID | タスク | 計画 | 実績 | 差異 |
|----|--------|------|------|------|
| D17-001 | 仕様書の実装追従更新 | BL-010〜014 修正 + 設計書3件作成 | **完了**（大幅拡大） | spc-dsn, spc-inf, spc-itr, spc-itf, interaction/整備, canvas整理 |
| D17-002 | Stripe 商品・価格設定 | Stripe Dashboard 設定 | 未着手 | D17-001 の作業量増大により未着手 |
| D17-003 | サインアップ時の無料クレジット付与 | free_credits=1000 | 未着手 | 同上 |
| D17-004 | Checkout Session API 実装 | Edge Function 作成 | 未着手 | 同上 |
| D17-005 | Webhook ハンドラ実装 | add_paid_credits 連携 | 未着手 | 同上 |
| D17-006 | E2E テスト設計書作成 | tst-e2e.md 作成 | 未着手 | 優先度低 |

**完了率: 1/6 (17%)**

### D17-001 のスコープ拡大

計画では BL-010〜014 の5項目修正と設計書3件（Tool Sieve, Observability, Annotations）の作成を想定していたが、実際には以下に拡大した：

| 作業 | 計画 | 実績 |
|------|------|------|
| spc-dsn.md 更新 | 含まない | v1.0→v2.0 大幅更新 |
| spc-inf.md 更新 | 含まない | v2.0→v3.0 レイヤー構成刷新 |
| grh-infrastructure.canvas | 含まない | 新規作成（8レイヤー構成） |
| interaction/ 整合性検証 | 含まない | 25ファイル検証・7件新規・4件削除 |
| itr-XXX.md リネーム | 含まない | 16ファイル大文字リネーム |
| grh-componet-interactions.canvas | 含まない | ノード/エッジID整理 |
| dtl-itr-*.md 更新 | 含まない | 9件 v2.0 更新（20/25 reviewed） |

### 計画精度の課題

1. **見積もり不足**: 仕様書の実装追従更新を「BL-010〜014 の5項目修正」と見積もったが、実際には関連する仕様書全体の整合性確保が必要だった
2. **依存関係の見落とし**: interaction/整備を開始したことで、canvas整理・リネーム・インデックス更新が連鎖的に発生
3. **並行タスクの非現実性**: 仕様書整備とStripe実装を同日に計画したが、前者だけで1日を要した

### 改善案

- 仕様書整備は独立した日に完結させる
- Stripe実装はDAY018に再スケジュール
