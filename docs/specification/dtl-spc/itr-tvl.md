# Token Vault インタラクション仕様書（itr-tvl）

## ドキュメント管理情報

| 項目 | 値 |
|------|-----|
| Status | `draft` |
| Version | v2.0 |
| Note | Token Vault Interaction Specification |

---

## 概要

Token Vault（TVL）は、外部サービスのOAuthトークン・API KEYを安全に管理するデータストア。

主な責務：
- 外部サービスのOAuthトークン保存・取得
- トークンリフレッシュの自動実行
- API KEY認証の提供
- トークンの暗号化保存

---

## 連携サマリー（spc-itrより）

| 相手 | 方向 | やり取り |
|------|------|----------|
| MCP Client (API KEY) | TVL ← CLK | API KEY認証受付 |
| API Gateway | TVL → GWY | API KEY提供 |
| Data Store | TVL ← DST | ユーザーID共有 |
| User Console | TVL ← CON | トークン登録 |
| External Auth Server | TVL ← EAS | 認証（トークン受信） |
| Modules | TVL → MOD | トークン提供 |

---

## 連携詳細

### CLK → TVL（API KEY認証受付）

| 項目 | 内容 |
|------|------|
| プロトコル | HTTPS（GWY経由） |
| 用途 | MCP Client (API KEY) の認証 |

**フロー:**
1. CLKがAPI KEYでリクエスト
2. GWYがTVLにAPI KEY検証を依頼
3. TVLがAPI KEYからユーザーIDを特定
4. ユーザーID（またはエラー）を返却

**API KEY形式:**
```
mcpist_{random_string}
```

---

### TVL → GWY（API KEY検証）

| 項目 | 内容 |
|------|------|
| 用途 | API KEYハッシュによる検証結果の返却 |
| 方向 | GWYからの検証リクエストに応答 |

**セキュリティ設計:**

API KEYは平文で保存せず、SHA256ハッシュで保存・検証する。

```
保存: SHA256(api_key) → TVL
検証: GWYがSHA256(api_key)を計算し、TVLに問い合わせ
```

これにより：
- TVLのDBが漏洩しても平文API KEYは取得不可
- GWYのメモリダンプでも平文API KEYは漏洩しない（ハッシュ計算後即破棄）

**検証リクエスト:**
```json
{
  "api_key_hash": "sha256_xxx..."
}
```

**検証レスポンス（成功）:**
```json
{
  "valid": true,
  "user_id": "user-123"
}
```

**検証レスポンス（失敗）:**
```json
{
  "valid": false,
  "error": "invalid_api_key"
}
```

---

### DST ← TVL（ユーザーID共有）

| 項目 | 内容 |
|------|------|
| 用途 | トークン管理のユーザー紐付け |
| 方向 | DSTのuser_id体系を使用 |

TVLはDSTと同じuser_id（Supabase Auth UUID）を使用してトークンを管理する。

---

### CON → TVL（トークン登録）

| 項目 | 内容 |
|------|------|
| トリガー | 外部サービス連携完了時 |
| 操作 | トークン新規登録、更新 |

**登録する情報:**
- user_id
- service_id（notion, google_calendar, microsoft_todo等）
- access_token
- refresh_token
- token_type
- expires_at
- scope

**フロー:**
1. ユーザーがCONで外部サービス連携
2. EASからOAuthトークン取得
3. TVLにトークン情報を保存（暗号化）
4. 既存トークンがあれば上書き

---

### EAS → TVL（認証トークン受信）

| 項目 | 内容 |
|------|------|
| トリガー | OAuth認可フロー完了時 |
| 操作 | トークン保存 |

CONが取得したトークンをTVLに保存する。EASとTVLの直接通信はなく、CON経由で行われる。

---

### TVL → MOD（トークン提供）

| 項目 | 内容 |
|------|------|
| トリガー | 外部API呼び出し時 |
| 操作 | サービス別トークン取得、有効期限確認 |

**フロー:**
1. MODが外部APIリクエストを処理
2. TVLにuser_id + service_idで問い合わせ
3. トークンが有効期限内であればそのまま返却
4. 有効期限切れの場合はリフレッシュ後に返却
5. トークン未登録の場合はエラー返却

**取得リクエスト:**
```json
{
  "user_id": "user-123",
  "service": "notion"
}
```

**取得レスポンス（成功）:**
```json
{
  "user_id": "user-123",
  "service": "notion",
  "long_term_token": "ntn_xxx",
  "oauth_token": null
}
```

**トークン優先度:**
1. `oauth_token` が存在すれば使用（ユーザー固有の権限）
2. `oauth_token` がなければ `long_term_token` を使用（共有/固定権限）

---

## トークンリフレッシュ

### TVL → EXT（トークンリフレッシュ）

| 項目 | 内容 |
|------|------|
| トリガー | トークン取得時に有効期限切れを検出 |
| 操作 | refresh_tokenを使用した新トークン取得 |

**フロー:**
1. MODからトークン取得リクエスト
2. access_tokenの有効期限切れを検出
3. EXTのトークンエンドポイントにリフレッシュリクエスト
4. 新しいaccess_token（+ refresh_token）を取得
5. TVL内のトークン情報を更新
6. 新しいaccess_tokenをMODに返却

**リフレッシュリクエスト（共通形式）:**
```
POST /oauth/token
Content-Type: application/x-www-form-urlencoded

grant_type=refresh_token
&refresh_token={refresh_token}
&client_id={client_id}
&client_secret={client_secret}
```

**リフレッシュ失敗時:**
- refresh_tokenも無効の場合はエラー返却
- ユーザーはCONで再連携が必要

---

## サービス別トークン形式

| Service | long_term_token | oauth_token |
|---------|-----------------|-------------|
| notion | Internal Integration Token (`ntn_xxx`) | OAuth Access Token |
| google_calendar | - | OAuth Access Token |
| microsoft_todo | - | OAuth Access Token |

---

## TVLが直接やり取りしないコンポーネント

| コンポーネント | 理由 |
|----------------|------|
| MCP Client (OAuth2.0) (CLO) | AUS経由で認証 |
| Auth Server (AUS) | OAuth2.0はAUS担当 |
| Session Manager (SSM) | DST経由 |
| Auth Middleware (AMW) | GWY経由 |
| MCP Handler (HDL) | MOD経由 |
| Identity Provider (IDP) | SSM経由 |
| Payment Service Provider (PSP) | DST経由 |

---

## 関連ドキュメント

| ドキュメント | 内容 |
|-------------|------|
| [spc-sys.md](../spc-sys.md) | システム仕様書 |
| [spc-itr.md](../spc-itr.md) | インタラクション仕様書 |
| [itf-tvl.md](./itf-tvl.md) | Token Vault API仕様 |
| [itr-gwy.md](./itr-gwy.md) | API Gateway詳細仕様 |
| [itr-con.md](./itr-con.md) | User Console詳細仕様 |
| [itr-mod.md](./itr-mod.md) | Modules詳細仕様 |
