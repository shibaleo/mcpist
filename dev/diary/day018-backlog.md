# DAY018 バックログ

## 日付

2026-01-29

---

## 新規追加

| ID | 内容 | 優先度 | 備考 |
|----|------|--------|------|
| BL-070 | database.types.ts 自動生成フロー整備 | 低 | 手動更新中。`npx supabase gen types` で自動化 |

---

## 引き継ぎ（DAY017 から）

| ID | 内容 | 状態 | 備考 |
|----|------|------|------|
| BL-011 | JWT `aud` チェック要件整理 | 未着手 | → DAY019 |
| BL-012 | MCP 拡張エラーコード整理 | 未着手 | → DAY019 |
| BL-013 | Console API 設計更新 | 未着手 | → DAY019 |
| BL-014 | PSP Webhook 仕様整理 | 未着手 | → DAY019 |
| BL-015 | enabled_modules 参照API実装 | 未着手 | Console ツール設定で一部実装済み |
| BL-016 | user_prompts 管理UI実装 | 未着手 | |
| BL-017 | usage_stats 参照API実装 | 未着手 | |
| BL-018 | クレジット付与機能（CON→DST） | 未着手 | |
| BL-019 | ツール実行ログにuser_id追加 | 未着手 | |
| BL-020 | invalid_gateway_secretログ実装 | 未着手 | |
| BL-060 | RFC 8707 Resource Indicators 対応 | 未着手 | |
| BL-061 | クレジット初期化をDBトリガーからアプリ層へ移行 | 未着手 | |

---

## 解消済み

| ID | 内容 | 解消方法 | 備考 |
|----|------|----------|------|
| BL-010 | Rate Limit記述の更新 | 仕様書から削除 | 将来実装予定として整理 |

---

## DAY019 への引き継ぎ

### 優先度: 高

- database.types.ts の RPC型定義を更新済み（手動）
- 本番デプロイ確認済み（Go Server、Worker経由E2E）

### 優先度: 中

- BL-011〜014: spc-itf.md 更新
- Observability 設計書作成

### 優先度: 低

- UI要求仕様書（S5-060〜062）
- CI自動化（S5-076〜077）
