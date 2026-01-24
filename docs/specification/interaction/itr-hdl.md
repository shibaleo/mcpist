# MCP Handler インタラクション仕様書（itr-hdl）

## ドキュメント管理情報

| 項目 | 値 |
|------|-----|
| Status | `reviewed` |
| Version | v3.1 |
| Note | MCP Handler Interaction Specification (MCP Server内部) - REG統合版 |

---

## 概要

MCP Handler（HDL）は、MCPプロトコルを解釈し、モジュールを管理・実行するコンポーネント。

主な責務：
- JSON-RPC 2.0リクエストの解析
- MCPメソッド（tools, resources, prompts）のルーティング
- モジュールの登録・管理
- ユーザー設定に基づくモジュール・ツールのフィルタリング
- メタツール（get_module_schema, run, batch）の実装
- Modulesへのプリミティブ操作の委譲
- レスポンスの構築

**認可（Authorization）について:** HDLはDSTからユーザー設定を取得し、有効なモジュール・ツールのみを返却する。アカウント状態・クレジット残高のチェックもHDLで実行する。

**位置づけ:** MCP Server内部コンポーネント

**内部実装詳細:** [dtl-spc-hdl.md](./dtl-spc-hdl.md)

---

## 連携サマリー（spc-itrより）

| 相手 | 方向 | やり取り |
|------|------|----------|
| Auth Middleware | HDL ← AMW | 認証済みリクエスト受信 |
| Data Store | HDL → DST | ユーザー設定取得 |
| Modules | HDL → MOD | プリミティブ操作委譲 |

---

## 連携詳細

### AMW → HDL（認証済みリクエスト受信）

| 項目 | 内容 |
|------|------|
| プロトコル | 内部関数呼び出し |
| 入力 | JSON-RPC 2.0リクエスト + ユーザーコンテキスト |

**ユーザーコンテキスト（AMWから受け取る情報）:**

| フィールド | 説明 |
|-----------|------|
| user_id | 認証済みユーザーID |
| request_id | リクエスト追跡用ID |
| client_ip | クライアントIPアドレス |

---

### HDL → DST（ユーザー設定取得）

| 項目 | 内容 |
|------|------|
| トリガー | MCPメソッドリクエスト時 |
| 操作 | ユーザー設定・状態の取得 |

**取得する情報:**

| フィールド | 説明 |
|-----------|------|
| account_status | アカウント状態（active/suspended/disabled） |
| credit_balance | クレジット残高 |
| enabled_modules | 有効なモジュール一覧 |
| tool_settings | ツール単位の有効/無効設定 |
| user_prompts | ユーザー定義プロンプト |

**注:** プランによるモジュール制限は行わない。

**チェック項目:**
- account_statusがactive以外 → エラー
- credit_balanceが0以下 → エラー
- モジュールが無効 → エラー
- ツールが無効 → エラー

---

### HDL → MOD（プリミティブ操作委譲）

| 項目 | 内容 |
|------|------|
| トリガー | 権限チェック完了後 |
| 操作 | tools/resources/promptsの取得・実行 |

**実行コンテキスト（MODへ渡す情報）:**

| フィールド | 説明 |
|-----------|------|
| user_id | 認証済みユーザーID |
| module | 対象モジュール名 |
| primitive_type | プリミティブ種別（tool/resource/prompt） |
| primitive_name | プリミティブ名 |
| params | パラメータ |
| request_id | リクエスト追跡用ID |

---

## HDLが直接やり取りしないコンポーネント

| コンポーネント | 理由 |
|----------------|------|
| MCP Client (CLO/CLK) | GWY/AMW経由 |
| API Gateway (GWY) | AMW経由 |
| Auth Server (AUS) | GWY経由 |
| Session Manager (SSM) | DST経由 |
| Token Vault (TVL) | MOD経由 |
| User Console (CON) | 別アプリケーション |
| Identity Provider (IDP) | SSM経由 |
| External Auth Server (EAS) | CON経由 |
| External Service API (EXT) | MOD経由 |
| Payment Service Provider (PSP) | DST経由 |

---

## 関連ドキュメント

| ドキュメント | 内容 |
|-------------|------|
| [spc-sys.md](../spc-sys.md) | システム仕様書 |
| [spc-itr.md](../spc-itr.md) | インタラクション仕様書 |
| [dtl-spc-hdl.md](./dtl-spc-hdl.md) | MCP Handler詳細仕様 |
| [itr-amw.md](./itr-amw.md) | Auth Middleware詳細仕様 |
| [itr-dst.md](./itr-dst.md) | Data Store詳細仕様 |
| [itr-mod.md](./itr-mod.md) | Modules詳細仕様 |
| [itr-srv.md](./itr-srv.md) | MCP Server詳細仕様 |
