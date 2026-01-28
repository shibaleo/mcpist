# Observability インタラクション仕様書（itr-OBS）

## ドキュメント管理情報

| 項目 | 値 |
|------|-----|
| Status | `draft` |
| Version | v1.0 |
| Note | Observability Interaction Specification |

---

## 概要

Observability（OBS）は、システム全体のログ収集・監視を担うコンポーネント。

主な責務：
- HTTPリクエストログの収集
- ツール実行ログの収集
- セキュリティイベントの収集
- X-Request-IDによるトレース

---

## 連携サマリー

| 相手 | 方向 | やり取り | 詳細 |
|------|------|----------|------|
| API Gateway | OBS ← GWY | HTTPリクエストログ受信 | [dtl-itr-GWY-OBS.md](./dtl-itr-GWY-OBS.md) |
| MCP Handler | OBS ← HDL | ツール実行ログ、セキュリティイベント受信 | [dtl-itr-HDL-OBS.md](./dtl-itr-HDL-OBS.md) |

---

## OBSが直接やり取りしないコンポーネント

| コンポーネント | 理由 |
|----------------|------|
| MCP Client (CLO/CLK) | GWY経由 |
| Auth Server (AUS) | 直接連携なし |
| Auth Middleware (AMW) | HDL経由 |
| Modules (MOD) | HDL経由 |
| Data Store (DST) | 直接連携なし |
| Token Vault (TVL) | 直接連携なし |
| User Console (CON) | 別アプリケーション |
| Session Manager (SSM) | 直接連携なし |
| Identity Provider (IDP) | 直接連携なし |
| External Auth Server (EAS) | 直接連携なし |
| External Service API (EXT) | 直接連携なし |
| Payment Service Provider (PSP) | 直接連携なし |

---

## 関連ドキュメント

| ドキュメント | 内容 |
|-------------|------|
| [spc-sys.md](../spc-sys.md) | システム仕様書 |
| [spc-itr.md](../spc-itr.md) | インタラクション仕様書 |
| [itr-GWY.md](./itr-GWY.md) | API Gateway詳細仕様 |
| [itr-HDL.md](./itr-HDL.md) | MCP Handler詳細仕様 |
