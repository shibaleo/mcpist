# Token Vault インターフェース仕様書（ifr-tvl）

## ドキュメント管理情報

| 項目 | 値 |
|------|-----|
| Status | `draft` |
| Version | v1.0 (DAY8) |
| Note | Token Vault Interface Specification |

---

## 概要

Token Vault（TVL）は、外部サービスのOAuthトークンを安全に管理するデータストア。

### 連携サマリー（spc-itrより）

| 相手 | 方向 | やり取り |
|------|------|----------|
| MCP Server | TVL ← SRV | トークンの取得 |
| User Console | TVL ← CON | OAuthトークン登録 |
| External API Server | TVL → EXT | トークンリフレッシュ |

---

## 連携詳細

### SRV → TVL（トークンの取得）

| 項目 | 内容 |
|------|------|
| トリガー | 外部API呼び出し時 |
| 操作 | サービス別トークン取得、有効期限確認 |

**フロー：**
1. SRVが外部APIリクエストを処理
2. TVLにuser_id + service_idで問い合わせ
3. トークンが有効期限内であればそのまま返却
4. 有効期限切れの場合はリフレッシュ後に返却
5. トークン未登録の場合はエラー返却

**取得する情報：**
- access_token
- token_type（通常は "Bearer"）
- 有効期限

---

### CON → TVL（OAuthトークン登録）

| 項目 | 内容 |
|------|------|
| トリガー | 外部サービス連携完了時 |
| 操作 | トークン新規登録、更新 |

**登録する情報：**
- user_id
- service_id（notion, github, jira, confluence, google_calendar, microsoft_todo）
- access_token
- refresh_token
- token_type
- expires_at
- scope

**フロー：**
1. ユーザーがCONで外部サービス連携
2. EXTからOAuthトークン取得
3. TVLにトークン情報を保存
4. 既存トークンがあれば上書き

---

### TVL → EXT（トークンリフレッシュ）

| 項目 | 内容 |
|------|------|
| トリガー | トークン取得時に有効期限切れを検出 |
| 操作 | refresh_tokenを使用した新トークン取得 |

**フロー：**
1. SRVからトークン取得リクエスト
2. access_tokenの有効期限切れを検出
3. EXTのトークンエンドポイントにリフレッシュリクエスト
4. 新しいaccess_token（+ refresh_token）を取得
5. TVL内のトークン情報を更新
6. 新しいaccess_tokenをSRVに返却

**リフレッシュ失敗時：**
- refresh_tokenも無効の場合はエラー返却
- ユーザーはCONで再連携が必要

---

## 関連ドキュメント

| ドキュメント | 内容 |
|-------------|------|
| [spc-sys.md](../spc-sys.md) | システム仕様書 |
| [spc-itr.md](../spc-itr.md) | インタラクション仕様書 |
| [itr-srv.md](./itr-srv.md) | MCP Server詳細仕様 |
| [itr-con.md](./itr-con.md) | User Console詳細仕様 |
| [ifr-ent.md](./ifr-ent.md) | Entitlement Store詳細仕様 |
