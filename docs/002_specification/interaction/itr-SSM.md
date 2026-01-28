# Session Manager インタラクション仕様書（itr-SSM）

## ドキュメント管理情報

| 項目 | 値 |
|------|-----|
| Status | `reviewed` |
| Version | v2.0 |
| Note | Session Manager Interaction Specification |

---

## 概要

Session Manager（SSM）は、ユーザーセッションとソーシャルログイン連携を管理するコンポーネント。

主な責務：
- ソーシャルログイン連携（Google, GitHub等）
- ユーザーID発行
- セッション管理

**実装:** Supabase Auth

---

## 連携サマリー

| 相手 | 方向 | やり取り | 詳細 |
|------|------|----------|------|
| Auth Server | SSM ↔ AUS | ユーザー認証連携 | [dtl-itr-AUS-SSM.md](./dtl-itr-AUS-SSM.md) |
| Identity Provider | SSM → IDP | ソーシャルログイン | [dtl-itr-IDP-SSM.md](./dtl-itr-IDP-SSM.md) |
| User Console | SSM ← CON | ソーシャルログイン | [dtl-itr-CON-SSM.md](./dtl-itr-CON-SSM.md) |

---

## セッション管理

| 項目 | 値 | 備考 |
|------|-----|------|
| Access Token有効期限 | 3600秒（1時間） | Supabase Authデフォルト |
| Refresh Token有効期限 | 無期限 | 1回のみ使用可能 |
| セッション有効期限 | 無期限 | サインアウトまで有効 |
| 同時セッション数 | 無制限 | - |

**実装:** Supabase Auth設定（実装範囲外）

---

## SSMが直接やり取りしないコンポーネント

| コンポーネント | 理由 |
|----------------|------|
| MCP Client (OAuth2.0) (CLO) | AUS経由 |
| MCP Client (API KEY) (CLK) | TVL経由 |
| API Gateway (GWY) | MCP通信専用 |
| Token Vault (TVL) | 外部サービストークン専用 |
| Auth Middleware (AMW) | MCP Server内部 |
| MCP Handler (HDL) | MCP Server内部 |
| Modules (MOD) | MCP Server内部 |
| User Console (CON) | SSMのログインUIを表示（直接連携ではない） |
| External Auth Server (EAS) | 外部サービス認証専用 |
| External Service API (EXT) | MOD経由 |
| Payment Service Provider (PSP) | DST経由 |

---

## 関連ドキュメント

| ドキュメント | 内容 |
|-------------|------|
| [spc-sys.md](../spc-sys.md) | システム仕様書 |
| [spc-itr.md](../spc-itr.md) | インタラクション仕様書 |
| [itr-AUS.md](./itr-AUS.md) | Auth Server詳細仕様 |
| [itr-IDP.md](./itr-IDP.md) | Identity Provider詳細仕様 |
| [itr-DST.md](./itr-DST.md) | Data Store詳細仕様 |
| [itr-CON.md](./itr-CON.md) | User Console詳細仕様 |
