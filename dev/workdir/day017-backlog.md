# DAY017 バックログ

## 概要

Sprint-006 の DAY017 時点のバックログ。仕様と実装の差分を管理。

---

## 仕様書・実装差分

### BL-010〜014（spc-*.md 系）

| ID | タスク | 対象ファイル | 備考 |
|----|--------|-------------|------|
| BL-010 | Rate Limit記述の更新 | spc-dsn.md | 実装では削除済み。「将来実装予定」に変更 |
| BL-011 | JWT `aud`チェック要件の整理 | spc-itf.md | 実装では明示チェックなし |
| BL-012 | MCP拡張エラーコード(2001-2005)の整理 | spc-itf.md | JSON-RPC標準コードのみに更新 |
| BL-013 | Console API設計の更新 | spc-itf.md | REST API→Supabase RPC方式に更新 |
| BL-014 | PSP Webhook仕様の整理 | spc-itf.md | Phase 1（Stripe）に合わせて更新 |

### BL-015〜020（実装不足）

| ID | タスク | 対象ファイル | 備考 |
|----|--------|-------------|------|
| BL-015 | enabled_modules 参照API実装 | dtl-itr-CON-DST.md | 仕様: 登録/更新可。実装: module_settingsテーブル未使用 |
| BL-016 | user_prompts 管理UI実装 | dtl-itr-CON-DST.md | 仕様: 登録/更新/削除可。実装: テーブルのみ、UI未実装 |
| BL-017 | usage_stats 参照API実装 | dtl-itr-CON-DST.md | 仕様: 参照のみ。実装: 未実装 |
| BL-018 | クレジット付与機能（CON→DST） | dtl-itr-CON-DST.md | 仕様: CONから任意整数分のクレジット付与。実装: トリガーで初期化のみ |
| BL-019 | ツール実行ログにuser_id追加 | dtl-itr-HDL-OBS.md | 仕様: user_id含む。実装: 未実装 |
| BL-020 | invalid_gateway_secretログ実装 | dtl-itr-HDL-OBS.md | 仕様: セキュリティイベント。実装: 未実装 |

---

## 参考

- [day017-plan.md](./day017-plan.md) - 計画
- [day017-worklog.md](./day017-worklog.md) - 作業ログ
- [day016-backlog.md](./day016-backlog.md) - 前日バックログ
- [sprint006.md](../sprint/sprint006.md) - スプリント計画
