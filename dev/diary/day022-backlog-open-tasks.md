# 未完了タスク一覧

> **注記:** このファイルの内容は `day022-backlog.md` へ転記済み（2026-02-03）

## 作成日

2026-01-29

---

## 未完了タスク（backlog集計）

| ID | 内容 | 状態 | 出典 | 備考 |
|----|------|------|------|------|
| S5-060 | UI要求仕様書作成 (spc-ui.md) | 未着手 | day017-backlog | 画面一覧・機能要件 |
| S5-061 | ユーザーフロー図作成 | 未着手 | day017-backlog | 主要フローの可視化 |
| S5-062 | 画面遷移図作成 | 未着手 | day017-backlog | 認証後のナビゲーション |
| S5-076 | CIにtools.json検証追加 | 未着手 | day017-backlog | Go出力との差分チェック自動化 |
| S5-077 | 未使用Goモジュール削除 | 未着手 | day017-backlog | クリーンアップ |
| D16-002 | 仕様書の実装追従更新 | 進行中 | day017-backlog | spec-impl-compare.md の差分を仕様書に反映 |
| D16-006 | E2Eテスト設計 | 未着手 | day017-backlog | OAuth認可フロー等 |
| BL-010 | Rate Limit記述の更新 | 未記載 | day017-backlog | 実装では削除済み。「将来実装予定」に変更 |
| BL-011 | JWT udチェック要件の整理 | 未記載 | day017-backlog | 実装では明示チェックなし |
| BL-012 | MCP拡張エラーコード(2001-2005)の整理 | 未記載 | day017-backlog | JSON-RPC標準コードのみに更新 |
| BL-013 | Console API設計の更新 | 未記載 | day017-backlog | REST API→Supabase RPC方式に更新 |
| BL-014 | PSP Webhook仕様の整理 | 未記載 | day017-backlog | Phase 1（Stripe）に合わせて更新 |
| BL-015 | enabled_modules 参照API実装 | 未記載 | day017-backlog | 仕様: 登録/更新可。実装: module_settingsテーブル未使用 |
| BL-016 | user_prompts 管理UI実装 | ✅完了 | day021 | Console でプロンプト作成・編集・削除・有効無効切替を実装 |
| BL-017 | usage_stats 参照API実装 | 未記載 | day017-backlog | 仕様: 参照のみ。実装: 未実装 |
| BL-018 | クレジット付与機能（CON→DST） | 未記載 | day017-backlog | 仕様: CONから任意整数分のクレジット付与。実装: トリガーで初期化のみ |
| BL-019 | ツール実行ログにuser_id追加 | 未記載 | day017-backlog | 仕様: user_id含む。実装: 未実装 |
| BL-020 | invalid_gateway_secretログ実装 | 未記載 | day017-backlog | 仕様: セキュリティイベント。実装: 未実装 |
| BL-060 | RFC 8707 Resource Indicators 対応 | 未記載 | day017-backlog | MCP OAuth仕様推奨。Supabase Auth対応状況要確認 |
| BL-061 | クレジット初期化をDBトリガーからアプリ層へ移行 | 未記載 | day017-backlog | ビジネスロジックはアプリ層（CON→DST）が適切 |
| BL-NEW | 管理者画面でツールバッジ表示期間/対象の管理 | 未記載 | backlog-open-tasks | tools.jsonではなく管理側で制御 |
| BL-070 | オブザーバビリティ設計 | 未着手 | day021 | ログ、メトリクス、トレーシング |
| BL-071 | セキュリティ設計 | 未着手 | day021 | JWT aud チェック、rate limit、脅威モデリング |
| BL-072 | UI最適化 | 未着手 | day021 | レスポンシブ対応、エラーハンドリング改善 |
| BL-073 | オンボーディング改善 | 未着手 | day021 | 初回ユーザーガイド、チュートリアル |
| BL-074 | ライトモード色定義見直し | 未着手 | day021 | 現在:rootがダーク基調のまま。適切なライトテーマ色を定義する |
| BL-075 | 仕様書整備（JWT aud/MCP エラーコード） | 未着手 | day021 | BL-011/BL-012 の具体的な対応 |
| BL-076 | Microsoft To Do モジュール実装 | 未着手 | day021 | Google Tasks同様のタスク管理機能 |
| BL-077 | Google Calendar 日本の祝日対応 | 未着手 | day021 | 祝日カレンダー取得・表示機能 |

---

## 参照

- day017-backlog.md
- day018-backlog.md（新規/引き継ぎは未記載）

