# Sprint 005 レビュー

## 基本情報

| 項目 | 値 |
|------|-----|
| スプリント番号 | SPRINT-005 |
| 期間 | 2026-01-25 〜 2026-01-27 |
| マイルストーン | M4: RPC実装・リファクタリング・モジュール拡張 |
| 作業日数 | DAY014 〜 DAY016（3日間） |

---

## 目標と結果

**目標:** RPC関数の実装・統合、モジュール拡張（Airtable）、インフラ自動化

**結果:** 目標の中核（RPC基盤、OAuth基盤、モジュール拡張）は達成。加えて計画外のセキュリティ・Observability 基盤を構築した。UI要件定義（Phase 4）と仕様書更新は未着手のまま。

| Phase | 達成率 | 備考 |
|-------|--------|------|
| Phase 1: RPC関数実装 | 100% | 17 RPC 完了 |
| Phase 2: RPC呼び出しリファクタ | 100% | Console/Worker/Go 全て統一 |
| Phase 3: パスルーティング設計 | 100% | |
| Phase 4: UI要件定義 | 0% | 未着手 |
| Phase 5: ツール設定API | 75% | CI検証・クリーンアップが残 |
| Phase 6: モジュール拡張 | 100% | Airtable + Microsoft To Do |
| Phase 7: OAuth トークンリフレッシュ | 100% | Google + Microsoft |

---

## スプリントを通じた判断と学び

### DAY015: Supabase Vault との付き合い方

vault.secrets を直接 UPDATE しようとして権限エラーに遭遇した。既存の `upsert_service_token` RPC が `vault.create_secret()` を使っていることを発見し、同じパターン（DELETE + create_secret）で解決。

**教訓:** Supabase の内部テーブルは公式 API を使う。そして「なぜ動かないか」を調べるより「どうやって動いているか」を既存コードから学ぶ方が早い。

### DAY015: OAuth 設定の分散問題

Google OAuth の redirect_uri が DB にローカル開発用のまま残っていて `redirect_uri_mismatch` エラー。Provider 側（Google Cloud Console）とアプリ側（oauth_apps テーブル）の2箇所に同じ情報が散らばっていた。

**教訓:** OAuth は設定箇所が多い。デプロイ時のチェックリストが必要。理想は環境変数で一元管理。

### DAY016: MCP 仕様への寄せ方

独自フィールド（`defaultEnabled` / `dangerous`）を MCP 仕様の `annotations` に一本化した。「独自の方が柔軟では」という迷いがあったが、MCP仕様が安定期に入った今、独自拡張の維持コスト（Console と Go Server の両方でマッピング）が上回ると判断。

### DAY016: セキュリティのエラーメッセージ設計

Batch 権限チェックで「クライアントにどこまで返すか」を設計した。結論は、クライアントには曖昧なメッセージ、サーバーログに具体的な情報。Layer 1 (Filter) で非表示のツールを LLM が呼ぶ時点で異常であり、ツール名を返すのは攻撃者への情報提供になる。

**教訓:** 「誰に向けた情報か」を常に区別する。

### DAY016: Batch creditCost=0 脆弱性

Batch は「実行後にクレジット消費」する設計だったため、残高不足でもツール実行できてしまう脆弱性があった。事前に `TotalCredits() < toolCount` で検証するように修正。

**教訓:** 「後で精算」の設計はセキュリティホールになりやすい。

### DAY016: 技術提案の評価方法

gRPC 移行構想を評価した。Google Cloud ブログだけでなく、参照先の PR（#1936 → reject）、仕様レベルの議論（SEP-1319, SEP-1352）、通信パターンの本質的ミスマッチ（Adrian Cole の指摘）まで掘り下げて REJECTED 判定。

**教訓:** ブログ記事だけで判断しない。公式SDKの動向と仕様レベルの前提条件を先に確認する。

---

## 反省

### 計画の精度

DAY016 で計画外タスクが7件発生した。D16-001（Annotations 移行）を起点に「Go Server 側も統一すべき → メタツールも動的化すべき → Observability も必要」と連鎖的に拡大した。技術的には正しい判断の連鎖だったが、事前に依存関係を洗い出す時間を取れば計画に含められたはず。

### 仕様書の先送り

仕様書更新（D16-002）を3日連続で先送りした。Sprint-005 では一貫して「実装 > 仕様書」の優先判断をしたが、乖離が拡大している。Sprint-006 では最優先にする。

### Phase 4 の未着手

UI要件定義は3日間一度も着手しなかった。実装が先行して動くものがある以上、仕様を後から書く形になる。「実装が正」の方針で Sprint-004 の仕様群と整合させる必要がある。

---

## 達成できなかったことと影響

| 未達 | 影響 | 対応 |
|------|------|------|
| Phase 4: UI要件定義 | UI仕様が存在しない | Sprint-006 Phase 5 で対応（低優先） |
| 仕様書の実装追従更新 | BL-010〜014 の5項目が未修正。仕様書の信頼性低下 | Sprint-006 Phase 1 で最優先対応 |
| E2E テスト設計 | テスト基盤なしでリリース継続 | Sprint-006 Phase 2 で対応 |
| CI tools.json 検証 | 手動運用で代替可能 | Sprint-006 Phase 3 で対応 |

---

## Sprint-006 に向けて

Sprint-005 まで「作る」フェーズが続いた。RPC基盤、OAuth基盤、8モジュール115ツール、セキュリティ多層防御、Observability — 機能面の基盤は揃った。

Sprint-006 では「整える」フェーズに移行する。仕様と実装の乖離解消、テスト基盤構築、CI/CD 整備が主題。

---

## 参考

- [sprint005.md](./sprint005.md) - Sprint 計画書
- DAY014: [worklog](../diary/day014-review.md) / [review](../diary/day014-review.md)
- DAY015: [worklog](../diary/day015-worklog.md) / [review](../diary/day015-review.md)
- DAY016: [worklog](../workdir/day016-worklog.md) / [review](../workdir/day016-review.md)
- [day016-backlog.md](../workdir/day016-backlog.md) - バックログ
