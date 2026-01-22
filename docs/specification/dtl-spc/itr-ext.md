# External API Server 詳細仕様書（itr-ext）

## ドキュメント管理情報

| 項目 | 値 |
|------|-----|
| Status | `draft` |
| Version | v1.0 (DAY8) |
| Note | External API Server Interaction Specification |

---

## 概要

External API Server（EXT）は、各モジュールがアクセスする外部サービスのAPIサーバーの総称。

### 連携サマリー（spc-itrより）

| 相手 | 方向 | やり取り |
|------|------|----------|
| Modules | EXT ← MOD | API呼び出し受付（HTTPS） |
| Token Vault | EXT ← TVL | トークンリフレッシュリクエスト受付 |
| User Console | EXT ← CON | OAuth認可フロー受付 |

---

## 認証方式

外部サービスの認証方式はサービスごとに異なる。認証方式の違いはToken Vault（TVL）が吸収し、MODは統一的なインターフェースでトークンを取得する。

| 認証方式 | 例 | 特徴 |
|----------|-----|------|
| OAuth 2.0 | Notion, Google Calendar | refresh_tokenによるトークン更新 |
| OAuth 1.0a | Twitter (X) | 署名ベース、トークン更新なし |
| 長期トークン | 一部サービス | APIキー、有効期限なしまたは長期 |

**共通:**
- トークン/認証情報はTVLで暗号化保存
- MODはTVLからトークンを取得してEXTにアクセス
- 認証方式・トークン形式の差異はTVLが吸収

---

### API呼び出し（MOD → EXT）

| 項目 | 内容 |
|------|------|
| プロトコル | HTTPS |
| データ形式 | JSON（サービスにより異なる） |
| 認証 | TVLから取得したBearer Token |

**フロー:**
1. MODがTVLにトークン取得リクエスト
2. TVLがaccess_tokenを返却（必要に応じてリフレッシュ）
3. MODがEXTにAPI呼び出し
4. EXTがレスポンスを返却

---

### トークンリフレッシュ（TVL → EXT）

| 項目 | 内容 |
|------|------|
| トリガー | access_token有効期限切れ |
| エンドポイント | サービスごとのトークンエンドポイント |
| 認証 | client_id + client_secret |

**リクエスト（共通形式）:**
```
POST /oauth/token
Content-Type: application/x-www-form-urlencoded

grant_type=refresh_token
&refresh_token={refresh_token}
&client_id={client_id}
&client_secret={client_secret}
```

**レスポンス:**
```json
{
  "access_token": "new_access_token",
  "refresh_token": "new_refresh_token",
  "expires_in": 3600,
  "token_type": "Bearer"
}
```

---

### OAuth認可フロー（CON → EXT）

| 項目 | 内容 |
|------|------|
| トリガー | ユーザーが外部サービス連携を開始 |
| フロー | OAuth 2.0 Authorization Code Flow |

**フロー:**
1. CONがEXTの認可エンドポイントにリダイレクト
2. ユーザーがEXTで認証・同意
3. EXTがCONのコールバックURLに認可コードを返却
4. CONがEXTのトークンエンドポイントでトークン交換
5. CONがTVLにトークンを保存

---

## 関連ドキュメント

| ドキュメント | 内容 |
|-------------|------|
| [spc-sys.md](../spc-sys.md) | システム仕様書 |
| [spc-itr.md](../spc-itr.md) | インタラクション仕様書 |
| [ifr-tvl.md](./ifr-tvl.md) | Token Vault詳細仕様 |
| [itr-con.md](./itr-con.md) | User Console詳細仕様 |
