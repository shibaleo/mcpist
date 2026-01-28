# Modules インタラクション仕様書（itr-MOD）

## ドキュメント管理情報

| 項目 | 値 |
|------|-----|
| Status | `reviewed` |
| Version | v2.1 |
| Note | Modules Interaction Specification (MCP Server内部) |

---

## 概要

Modules（MOD）は、外部サービス（Notion, Google Calendar等）との連携を実装する個別モジュールの集合。

主な責務：
- 外部サービスAPIの呼び出し
- Token Vaultからのトークン取得
- サービス固有のビジネスロジック実装
- エラーハンドリングとリトライ

**位置づけ:** MCP Server内部コンポーネント

---

## 連携サマリー

| 相手                   | 方向        | やり取り            | 詳細 |
| -------------------- | --------- | --------------- | ---- |
| MCP Handler          | MOD ← HDL | プリミティブ操作リクエスト受信 | [dtl-itr-HDL-MOD.md](./dtl-itr-HDL-MOD.md) |
| Token Vault          | MOD → TVL | トークン取得          | [dtl-itr-MOD-TVL.md](./dtl-itr-MOD-TVL.md) |
| Data Store           | MOD → DST | クレジット消費         | [dtl-itr-DST-MOD.md](./dtl-itr-DST-MOD.md) |
| External Service API | MOD → EXT | リソースアクセス（HTTPS） | [dtl-itr-EXT-MOD.md](./dtl-itr-EXT-MOD.md) |

---

## モジュール例

| モジュール | サービス | 主なツール |
|------------|----------|-----------|
| notion | Notion | search, create_page, update_page, get_database |
| google_calendar | Google Calendar | list_events, create_event, update_event, delete_event |
| microsoft_todo | Microsoft To Do | list_tasks, create_task, update_task, complete_task |
| zaim | Zaim | list_money, create_money |

---

## MODが直接やり取りしないコンポーネント

| コンポーネント | 理由 |
|----------------|------|
| MCP Client (CLO/CLK) | GWY/AMW/HDL経由 |
| API Gateway (GWY) | AMW経由 |
| Auth Server (AUS) | GWY経由 |
| Session Manager (SSM) | DST経由 |
| Auth Middleware (AMW) | HDL経由 |
| User Console (CON) | 別アプリケーション |
| Identity Provider (IDP) | SSM経由 |
| External Auth Server (EAS) | TVL経由（トークン取得のみ） |
| Payment Service Provider (PSP) | DST経由 |

---

## 関連ドキュメント

| ドキュメント | 内容 |
|-------------|------|
| [spc-sys.md](../spc-sys.md) | システム仕様書 |
| [spc-itr.md](../spc-itr.md) | インタラクション仕様書 |
| [itr-HDL.md](./itr-HDL.md) | MCP Handler詳細仕様 |
| [itr-DST.md](./itr-DST.md) | Data Store詳細仕様 |
| [itr-TVL.md](./itr-TVL.md) | Token Vault詳細仕様 |
| [itr-EXT.md](./itr-EXT.md) | External Service API詳細仕様 |
| [itr-SRV.md](./itr-SRV.md) | MCP Server詳細仕様 |
| [itf-mod.md](../dtl-spc/itf-mod.md) | モジュールインターフェース仕様 |
| [dsn-adt.md](../../003_design/dsn-adt.md) | 監査・請求・分析設計書 |
| [dsn-err.md](../../003_design/dsn-err.md) | エラーハンドリング設計書 |
