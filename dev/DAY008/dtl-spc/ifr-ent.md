# Entitlement Store インターフェース仕様書（ifr-ent）

## ドキュメント管理情報

| 項目 | 値 |
|------|-----|
| Status | `draft` |
| Version | v1.0 (DAY8) |
| Note | Entitlement Store Interface Specification |

---

## 概要

Entitlement Store（ENT）は、ユーザーの権限・設定情報を管理するデータストア。

### 連携サマリー（spc-itrより）

| 相手 | 方向 | やり取り |
|------|------|----------|
| Auth Server | ENT ← AUS | ユーザー情報の参照・作成 |
| MCP Server | ENT ← SRV | 権限情報の参照 |
| User Console | ENT ← CON | 設定の読み書き |
| Payment Service Provider | ENT ↔ PSP | 課金情報の同期 |

---

## 連携詳細

### AUS → ENT（ユーザー情報の参照・作成）

| 項目 | 内容 |
|------|------|
| トリガー | OAuth認証成功時 |
| 操作 | ユーザー存在確認、新規作成 |

**フロー：**
1. AUSがOAuth認証成功
2. ENTにuser_id（Supabase Auth UUID）で問い合わせ
3. 存在しなければ新規レコード作成（デフォルト設定）
4. 存在すればアカウント状態を返却

---

### SRV → ENT（権限情報の参照）

| 項目 | 内容 |
|------|------|
| トリガー | MCP Clientからのリクエスト受信時 |
| 操作 | アカウント状態確認、許可モジュール/ツール取得 |

**取得する情報：**
- アカウント状態（active / suspended / disabled）
- 許可モジュール一覧
- 各モジュールの許可ツール一覧
- 課金プラン

---

### CON → ENT（設定の読み書き）

| 項目 | 内容 |
|------|------|
| トリガー | ユーザーが設定画面で操作 |
| 操作 | モジュール有効/無効、課金プラン変更 |

**読み取り：**
- 現在の設定一覧
- 利用可能なプラン

**書き込み：**
- モジュール有効/無効の切り替え
- 課金プランの変更

---

### ENT ↔ PSP（課金情報の同期）

| 項目 | 内容 |
|------|------|
| トリガー | Webhook受信（PSP→ENT）、プラン変更（ENT→PSP） |
| 操作 | 課金状態の同期、サブスクリプション管理 |

**PSP → ENT（Webhook）：**
- サブスクリプション作成/更新/キャンセル
- 支払い成功/失敗
- 課金サイクル更新

**ENT → PSP（API呼び出し）：**
- Checkout Session作成（プラン購入）
- Customer Portal Session作成（プラン管理）
- サブスクリプションキャンセル

**同期する情報：**
- stripe_customer_id
- subscription_id
- current_plan
- billing_status（active / past_due / canceled）
- current_period_end

---

## 関連ドキュメント

| ドキュメント | 内容 |
|-------------|------|
| [spc-sys.md](../spc-sys.md) | システム仕様書 |
| [spc-itr.md](../spc-itr.md) | インタラクション仕様書 |
| [itr-aus.md](./itr-aus.md) | Auth Server詳細仕様 |
| [itr-srv.md](./itr-srv.md) | MCP Server詳細仕様 |
| [itr-con.md](./itr-con.md) | User Console詳細仕様 |
